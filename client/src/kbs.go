package src

// Ingestion of Holonix KBS ("Knowledge-Based System") batch outputs — see
// client/data/Holo_Files/*_kbs_output.json for real samples. This is the
// demand side of a /get_options request: one file describes one product
// batch, every strategy the KBS judged feasible for it, and for each
// strategy the full step DAG (predecessors/successors + per-step service
// requirements) already computed by the KBS. It carries nothing about
// providers/capabilities — that half of the pipeline (techProviderSearch,
// MatchCapability) is untouched and reads the generalized Mongo
// collections as before.
//
// Verified structurally identical (same keys, same nesting, at every
// level) across all three sample product types (Motor, Clutch,
// SolidTool) — one parser handles all of them. What varies between
// product types is purely data: step/operation vocabulary, requirement
// names, and whether requirementValue is a number or a bare string
// (e.g. "mid" for a corrosion level). That data-level variation is why
// UseCase.KBSAttributeAliases exists instead of a hardcoded mapping here.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"github.com/gin-gonic/gin"
)

// KBSOutput is the root shape of one kbs_output.json file.
type KBSOutput struct {
	TenantID           string        `json:"tenantId"`
	BatchID            string        `json:"batchId"`
	BatchCardinality   int           `json:"batchCardinality"`
	ProductType        string        `json:"productType"`
	InputProduct       KBSProduct    `json:"inputProduct"`
	FeasibleStrategies []KBSStrategy `json:"feasibleStrategies"`
}

type KBSProduct struct {
	EfficiencyClass string `json:"efficencyClass"` // sic — matches the KBS's own spelling
	ProductClass    string `json:"productClass"`
}

type KBSStrategy struct {
	StrategyName  string     `json:"strategyName"`
	OutputProduct KBSProduct `json:"outputProduct"`
	Steps         []KBSStep  `json:"steps"`
}

type KBSStep struct {
	CurrentStepName       string            `json:"currentStepName"`
	CurrentStepType       string            `json:"currentStepType"`
	AppliedToArtifact     string            `json:"appliedToArtifact"`
	BelongingToMacroPhase string            `json:"beloningToMacroPhase"` // sic
	BelongingToRoute      string            `json:"beloningToRoute"`      // sic
	Predecessors          []string          `json:"predecessors"`
	Successors            []string          `json:"successors"`
	Service               KBSServiceWrapper `json:"service"`
}

type KBSServiceWrapper struct {
	ServiceChaining string       `json:"serviceChaining"`
	Services        []KBSService `json:"services"`
}

type KBSService struct {
	ServiceName         string           `json:"serviceName"`
	ServiceRequirements []KBSRequirement `json:"serviceRequirements"`
}

type KBSRequirement struct {
	RequirementName  string          `json:"requirementName"`
	RequirementValue json.RawMessage `json:"requirementValue"`
	RequirementUoM   *string         `json:"requirementUoM"`
}

// LoadKBSOutput reads and parses one kbs_output.json file.
func LoadKBSOutput(path string) (*KBSOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading KBS output %s: %w", path, err)
	}
	var out KBSOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing KBS output %s: %w", path, err)
	}
	return &out, nil
}

// kbsStepOperation derives the DSM's generic operation name (e.g.
// "Disassembly", "AssemblyTesting") from a step's currentStepName, which is
// always "<appliedToArtifact>-<operationSuffix>" with the suffix already
// camelCase — Title-casing its first letter is the only transformation
// needed. Verified against every step in all three sample files: the
// prefix invariant holds everywhere.
func kbsStepOperation(step KBSStep) (string, error) {
	prefix := step.AppliedToArtifact + "-"
	if !strings.HasPrefix(step.CurrentStepName, prefix) {
		return "", fmt.Errorf("step %q does not start with appliedToArtifact prefix %q", step.CurrentStepName, prefix)
	}
	suffix := strings.TrimPrefix(step.CurrentStepName, prefix)
	if suffix == "" {
		return "", fmt.Errorf("step %q has no operation suffix after its artifact prefix", step.CurrentStepName)
	}
	return strings.ToUpper(suffix[:1]) + suffix[1:], nil
}

var kbsTrailingDigits = regexp.MustCompile(`\d+$`)

// kbsNormalizeRequirementKey strips the trailing instance number from a KBS
// requirementName (e.g. "axisHeightMotor001" -> "axisheightmotor"),
// lowercased for a case-insensitive lookup in KBSAttributeAliases.
func kbsNormalizeRequirementKey(raw string) string {
	return strings.ToLower(kbsTrailingDigits.ReplaceAllString(raw, ""))
}

// kbsFormatRequirementValue renders a KBS requirementValue (a JSON number
// or a bare string, e.g. "mid" for a corrosion level) plus its optional
// unit into the single display string the rest of the matching pipeline
// expects (Requirements is map[string][]string). numeric_min-ruled
// attributes parse this with a leading-number regex that ignores
// formatting/units entirely, so exact spacing doesn't matter there; for
// attributes with no rule this string is compared for exact equality, so
// it must match the capability side's own formatting.
func kbsFormatRequirementValue(raw json.RawMessage, uom *string) (string, error) {
	var asNumber float64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		s := strconv.FormatFloat(asNumber, 'f', -1, 64)
		if uom != nil && *uom != "" {
			s += " " + *uom
		}
		return s, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString, nil
	}
	return "", fmt.Errorf("requirementValue %s is neither a number nor a string", raw)
}

// kbsStepRequirements merges every serviceRequirement across all of a
// step's candidate services into one Requirements map, translating each
// KBS requirementName through uc.KBSAttributeAliases. A step's
// serviceChaining can offer several alternative services (an OR — e.g.
// "opticalMicroscopeService OR digitalMicroscopeService"), but in every
// sample seen so far at most one alternative actually carries
// requirements, so a plain union across services is safe. If two
// services ever declare the same attribute with different values, that's
// a real modeling conflict (which alternative's number should the
// requirement be?) and this returns an error for it rather than silently
// picking one.
func kbsStepRequirements(uc execdb.UseCase, step KBSStep) (Requirements, error) {
	reqs := Requirements{}
	for _, svc := range step.Service.Services {
		for _, r := range svc.ServiceRequirements {
			key := kbsNormalizeRequirementKey(r.RequirementName)
			field, ok := uc.KBSAttributeAliases[key]
			if !ok {
				return nil, fmt.Errorf("step %q: no kbsAttributeAliases entry for requirement %q (normalized %q) — add one to the %q use case config",
					step.CurrentStepName, r.RequirementName, key, uc.ID)
			}
			value, err := kbsFormatRequirementValue(r.RequirementValue, r.RequirementUoM)
			if err != nil {
				return nil, fmt.Errorf("step %q requirement %q: %w", step.CurrentStepName, r.RequirementName, err)
			}
			if existing, ok := reqs[field]; ok && existing[0] != value {
				return nil, fmt.Errorf("step %q: conflicting values for %q across alternative services (%q vs %q)",
					step.CurrentStepName, field, existing[0], value)
			}
			reqs[field] = []string{value}
		}
	}
	return reqs, nil
}

// KBSStrategyToDAG converts one feasibleStrategy's steps into the
// map[string]DAGNode shape a /get_options route already expects. The map
// key stays the KBS's own unique currentStepName (not the generic
// operation), so every node — including the many sub-artifact nodes that
// share the same operation within one strategy — keeps its own identity,
// its own predecessors/successors, and its own requirements; Operation is
// what matching actually looks capability up by.
func KBSStrategyToDAG(uc execdb.UseCase, strategy KBSStrategy) (map[string]DAGNode, error) {
	dag := make(map[string]DAGNode, len(strategy.Steps))
	for _, step := range strategy.Steps {
		if _, exists := dag[step.CurrentStepName]; exists {
			return nil, fmt.Errorf("strategy %q: duplicate step name %q", strategy.StrategyName, step.CurrentStepName)
		}
		op, err := kbsStepOperation(step)
		if err != nil {
			return nil, fmt.Errorf("strategy %q: %w", strategy.StrategyName, err)
		}
		reqs, err := kbsStepRequirements(uc, step)
		if err != nil {
			return nil, fmt.Errorf("strategy %q: %w", strategy.StrategyName, err)
		}
		dag[step.CurrentStepName] = DAGNode{
			Prerequisites: append([]string{}, step.Predecessors...),
			Requirements:  reqs,
			Operation:     op,
			Universal:     len(step.Service.Services) == 0,
		}
	}
	return dag, nil
}

// KBSOutputToOptimizationInput converts one Holonix KBS batch output into
// the OptimizationInput /get_options already accepts, with one Routes
// entry per feasibleStrategy — exactly the DSM's existing "multiple
// candidate routes per request" shape, so a route-with-a-redesign-step
// (e.g. Motor's "Upgrade" strategy) needs no special-casing here: it's
// just another route, built the same way as every other.
func KBSOutputToOptimizationInput(kbs *KBSOutput, startingPoint []float64, optimizationMode string) (OptimizationInput, error) {
	uc, err := execdb.GetUseCaseByKBSProductType(kbs.ProductType)
	if err != nil {
		return OptimizationInput{}, fmt.Errorf("no use case declares kbsProductType %q (from batch %q): %w", kbs.ProductType, kbs.BatchID, err)
	}

	routes := make(map[string]map[string]DAGNode, len(kbs.FeasibleStrategies))
	for _, strat := range kbs.FeasibleStrategies {
		dag, err := KBSStrategyToDAG(uc, strat)
		if err != nil {
			return OptimizationInput{}, err
		}
		routes[strat.StrategyName] = dag
	}

	return OptimizationInput{
		UseCase: uc.ID,
		ProcessRequirements: ProcessRequirements{
			ProductMatch: kbs.InputProduct.EfficiencyClass,
			Quantity:     kbs.BatchCardinality,
		},
		Routes:           routes,
		StartingPoint:    startingPoint,
		OptimizationMode: optimizationMode,
	}, nil
}

// KBSOptimizationRequest is the envelope POST /get_options accepts when the
// caller wants to submit a real Holonix KBS batch output directly instead
// of a hand-built OptimizationInput. The "kbs" field is the KBS's own
// output verbatim — a real integration can paste a kbs_output.json file in
// there completely unmodified. StartingPoint/OptimizationMode travel
// alongside it rather than inside it: the KBS describes the product batch
// (what needs to happen, to what, in what order), never where the request
// originates geographically or which optimization mode to run — that's
// request-specific context the KBS has no notion of.
type KBSOptimizationRequest struct {
	StartingPoint    []float64 `json:"starting_point"`
	OptimizationMode string    `json:"optimization_mode"`
	KBS              KBSOutput `json:"kbs"`
}

// BindOptimizationInput reads one /get_options request body and returns the
// OptimizationInput to run, accepting either shape the endpoint supports:
//   - the existing hand-built OptimizationInput JSON (routes/requirements/...),
//     unchanged — detected by the absence of a "kbs" key, and
//   - a KBSOptimizationRequest envelope — detected by the presence of one —
//     converted via KBSOutputToOptimizationInput.
//
// Reads the body itself (via c.GetRawData, not c.ShouldBindJSON) because
// which target struct to unmarshal into isn't known until the body's own
// shape has been peeked at.
func BindOptimizationInput(c *gin.Context) (OptimizationInput, error) {
	raw, err := c.GetRawData()
	if err != nil {
		return OptimizationInput{}, fmt.Errorf("reading request body: %w", err)
	}

	var probe struct {
		KBS json.RawMessage `json:"kbs"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && len(probe.KBS) > 0 {
		var envelope KBSOptimizationRequest
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return OptimizationInput{}, fmt.Errorf("invalid KBS request envelope: %w", err)
		}
		return KBSOutputToOptimizationInput(&envelope.KBS, envelope.StartingPoint, envelope.OptimizationMode)
	}

	var request OptimizationInput
	if err := json.Unmarshal(raw, &request); err != nil {
		return OptimizationInput{}, fmt.Errorf("invalid JSON format: %w", err)
	}
	return request, nil
}

package src

// Generates and reads IDTA 02020 Capability Description AAS documents for
// the DSM validation dataset's 12 providers, following the exact template
// of the 5 existing hand-written samples at
// client/data/NewData/electric-motors/V2/aas-instancies/CapabilityDescription-ElectricMotorProvider-{000..004}.json
// (same nesting: AAS shell -> submodel -> CapabilitySet -> one
// CapabilityContainer per operation -> Capability + CapabilityComment +
// PropertySet, each PropertySet holding one PropertyContainer per field).
// This is the format the project's AAS-based actor identification is
// meant to run against, as opposed to the generalized Mongo
// providers/capabilities collections the CSV-based pipeline uses.
//
// Two boilerplate strings in the existing 5 files are constant across
// every operation and every provider — the CapabilityContainer's own
// description ("Offered recoating capability of tool provider N.") and
// the Capability's own description/semanticId ("Restores or renews the
// cutting-tool coating...", ".../semantic/capability/recoating") — clearly
// copy-pasted from an earlier cutting-tool "recoating" capability template
// and never parameterized per operation. Reproduced here verbatim for
// structural consistency with the existing 5, not because it's accurate.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
)

// validationAASDir is where ProcessDirectoryValidate reads provider
// capability files from — overridable so tests can point it at the
// repo-relative path (go test's cwd is the package directory) while the
// running server uses the same ./data/... convention ACTOR_DIRECTORY and
// friends already rely on (client/data gets copied to ./data in the
// container — see Dockerfile.client).
func validationAASDir() string {
	return getEnvOrDefault("VALIDATION_AAS_DIR", "./data/dsm_validation_dataset/aas-instancies")
}

// validationDatasetOperations is the fixed operation set
// motor_providers_clustered.csv carries one {Op}Equipments/{Op}Energy/
// {Op}Cost column triple for — identical to the real electric-motor use
// case's operations list.
var validationDatasetOperations = []string{
	"Reuse", "AssemblyInspection", "Inspection", "Punching", "Design", "Grinding",
	"Replace", "AssemblyForManufacturingStep", "Disassembly", "AssemblyTesting",
	"DimensionalTesting", "Turning", "Reassembly",
}

func isBlankCSVCell(v string) bool { return v == "" || v == "N/A" }

func parseCSVFloatPtr(raw string) *float64 {
	if isBlankCSVCell(raw) {
		return nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil
	}
	return &f
}

func csvAttrValueToList(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			out = append(out, csvAttrValueToList(e)...)
		}
		return out
	default:
		return nil
	}
}

func parseCSVEquipmentDict(raw string) ([]execdb.Equipment, error) {
	if isBlankCSVCell(raw) {
		return nil, nil
	}
	jsonish := strings.ReplaceAll(raw, "'", "\"")
	var parsed map[string]map[string]any
	if err := json.Unmarshal([]byte(jsonish), &parsed); err != nil {
		return nil, fmt.Errorf("equipment cell %q: %w", raw, err)
	}
	equipment := make([]execdb.Equipment, 0, len(parsed))
	for name, attrs := range parsed {
		if name == "" {
			continue
		}
		attributes := make(map[string][]string, len(attrs))
		for k, v := range attrs {
			attributes[k] = csvAttrValueToList(v)
		}
		equipment = append(equipment, execdb.Equipment{Name: name, Attributes: attributes})
	}
	return equipment, nil
}

// parseValidationProvidersGrouped parses motor_providers_clustered.csv —
// same column shape the real importer reads — into one ProviderRecord per
// row plus a providerID -> operation -> CapabilityRecord index, which is
// exactly the shape GenerateValidationAASFiles needs to build one AAS
// document per provider.
func parseValidationProvidersGrouped(csvPath string) ([]execdb.ProviderRecord, map[string]map[string]execdb.CapabilityRecord, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening validation providers CSV: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("reading validation providers CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("empty validation providers CSV: %s", csvPath)
	}
	header := records[0]

	var providers []execdb.ProviderRecord
	capsByProvider := make(map[string]map[string]execdb.CapabilityRecord)

	for _, rec := range records[1:] {
		row := make(map[string]string, len(header))
		for i, col := range header {
			if i < len(rec) {
				row[col] = rec[i]
			}
		}
		providerID := row["IdProvider"]
		lat, _ := strconv.ParseFloat(row["Latitude"], 64)
		lon, _ := strconv.ParseFloat(row["Longitude"], 64)

		ops := make(map[string]execdb.CapabilityRecord)
		var offered []string
		for _, op := range validationDatasetOperations {
			eqRaw := row[op+"Equipments"]
			if isBlankCSVCell(eqRaw) {
				continue
			}
			equipment, err := parseCSVEquipmentDict(eqRaw)
			if err != nil {
				return nil, nil, fmt.Errorf("provider %s op %s: %w", providerID, op, err)
			}
			offered = append(offered, op)
			ops[op] = execdb.CapabilityRecord{
				Operation: op,
				Energy:    parseCSVFloatPtr(row[op+"Energy"]),
				Cost:      parseCSVFloatPtr(row[op+"Cost"]),
				Equipment: equipment,
			}
		}
		capsByProvider[providerID] = ops

		providers = append(providers, execdb.ProviderRecord{
			ProviderID: providerID,
			Name:       row["Name"],
			Country:    row["Country"],
			Location:   []float64{lat, lon},
			Operations: offered,
		})
	}
	return providers, capsByProvider, nil
}

// ── Node builders (generation) ──────────────────────────────────────────

type aasLangText struct {
	Language string `json:"language"`
	Text     string `json:"text"`
}

type aasKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type aasRef struct {
	Type string   `json:"type"`
	Keys []aasKey `json:"keys"`
}

type aasQualifier struct {
	SemanticID aasRef `json:"semanticId"`
	Type       string `json:"type"`
	ValueType  string `json:"valueType"`
	Value      string `json:"value"`
}

type aasProperty struct {
	IDShort     string        `json:"idShort"`
	DisplayName []aasLangText `json:"displayName"`
	Description []aasLangText `json:"description"`
	SemanticID  aasRef        `json:"semanticId"`
	ValueType   string        `json:"valueType"`
	Value       string        `json:"value"`
	ModelType   string        `json:"modelType"`
}

type aasMultiLangProp struct {
	IDShort     string        `json:"idShort"`
	DisplayName []aasLangText `json:"displayName"`
	Description []aasLangText `json:"description"`
	SemanticID  aasRef        `json:"semanticId"`
	Value       []aasLangText `json:"value"`
	ModelType   string        `json:"modelType"`
}

type aasCapability struct {
	IDShort     string         `json:"idShort"`
	DisplayName []aasLangText  `json:"displayName"`
	Description []aasLangText  `json:"description"`
	SemanticID  aasRef         `json:"semanticId"`
	Qualifiers  []aasQualifier `json:"qualifiers"`
	ModelType   string         `json:"modelType"`
}

type aasCollection struct {
	IDShort     string            `json:"idShort"`
	DisplayName []aasLangText     `json:"displayName"`
	Description []aasLangText     `json:"description"`
	SemanticID  aasRef            `json:"semanticId"`
	Value       []json.RawMessage `json:"value"`
	ModelType   string            `json:"modelType"`
}

type aasSpecificAssetID struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type aasAssetInfo struct {
	AssetKind        string               `json:"assetKind"`
	GlobalAssetID    string               `json:"globalAssetId"`
	AssetType        string               `json:"assetType"`
	SpecificAssetIDs []aasSpecificAssetID `json:"specificAssetIds"`
}

type aasModelRef struct {
	Type string   `json:"type"`
	Keys []aasKey `json:"keys"`
}

type aasShell struct {
	IDShort          string        `json:"idShort"`
	DisplayName      []aasLangText `json:"displayName"`
	Description      []aasLangText `json:"description"`
	ID               string        `json:"id"`
	AssetInformation aasAssetInfo  `json:"assetInformation"`
	Submodels        []aasModelRef `json:"submodels"`
	ModelType        string        `json:"modelType"`
}

type aasSubmodel struct {
	IDShort          string            `json:"idShort"`
	DisplayName      []aasLangText     `json:"displayName"`
	Description      []aasLangText     `json:"description"`
	ID               string            `json:"id"`
	Kind             string            `json:"kind"`
	SemanticID       aasRef            `json:"semanticId"`
	SubmodelElements []json.RawMessage `json:"submodelElements"`
	ModelType        string            `json:"modelType"`
}

type aasCapabilityDoc struct {
	AssetAdministrationShells []aasShell    `json:"assetAdministrationShells"`
	Submodels                 []aasSubmodel `json:"submodels"`
}

func lang(text string) []aasLangText { return []aasLangText{{Language: "en", Text: text}} }

func extRef(url string) aasRef {
	return aasRef{Type: "ExternalReference", Keys: []aasKey{{Type: "GlobalReference", Value: url}}}
}

func mustRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("aas_capability: marshaling a well-formed internal node: %v", err))
	}
	return b
}

func propertyNode(idShort, display, desc, valueType, value string) json.RawMessage {
	return mustRaw(aasProperty{
		IDShort: idShort, DisplayName: lang(display), Description: lang(desc),
		SemanticID: extRef("https://remanet-project.eu/semantic/capability-property/" + idShort),
		ValueType:  valueType, Value: value, ModelType: "Property",
	})
}

func multiLangNode(idShort, display, desc, semanticURL, text string) json.RawMessage {
	return mustRaw(aasMultiLangProp{
		IDShort: idShort, DisplayName: lang(display), Description: lang(desc),
		SemanticID: extRef(semanticURL), Value: lang(text), ModelType: "MultiLanguageProperty",
	})
}

func collectionNode(idShort, display, desc, semanticURL string, children ...json.RawMessage) json.RawMessage {
	return mustRaw(aasCollection{
		IDShort: idShort, DisplayName: lang(display), Description: lang(desc),
		SemanticID: extRef(semanticURL), Value: children, ModelType: "SubmodelElementCollection",
	})
}

func capabilityNode(idShort, display string) json.RawMessage {
	return mustRaw(aasCapability{
		IDShort: idShort, DisplayName: lang(display),
		// Verbatim boilerplate from the existing 5 — see file header.
		Description: lang("Restores or renews the cutting-tool coating using the available coating equipment."),
		SemanticID:  extRef("https://remanet-project.eu/semantic/capability/recoating"),
		Qualifiers: []aasQualifier{{
			SemanticID: extRef("https://admin-shell.io/idta/CapabilityDescription/CapabilityRoleQualifier/Offered/1/0"),
			Type:       "Offered", ValueType: "xs:boolean", Value: "true",
		}},
		ModelType: "Capability",
	})
}

// propertyContainer builds one PropertyContainer_<key> wrapper: a
// collection holding the actual leaf Property plus its PropertyComment —
// exactly the shape every field (CapabilityID, OperationType, ProviderID,
// ...) uses in the existing 5 files.
func propertyContainer(key, display, desc, valueType, value, commentText string) json.RawMessage {
	prop := propertyNode(key, display, desc, valueType, value)
	comment := multiLangNode("PropertyComment", "Property Comment",
		"Human-readable information about the capability property.",
		"https://admin-shell.io/idta/CapabilityDescription/PropertyComment/1/0", commentText)
	return collectionNode("PropertyContainer_"+key, display, desc,
		"https://admin-shell.io/idta/CapabilityDescription/PropertyContainer/1/0", prop, comment)
}

// aasOperationData is the per-operation input the generator needs —
// already-formatted display text and the equipment dict, mirroring what
// one non-blank {Op}Equipments CSV cell carries for one provider.
type aasOperationData struct {
	Key           string // e.g. "AssemblyTesting" — matches the DSM's Operation name
	Display       string // e.g. "Assembly Testing" — human-readable OperationType text
	EquipTypes    string // semicolon-joined, humanized equipment/service names
	EquipSpecJSON string // the equipment dict as a JSON string (EquipmentSpecifications)
	Energy        *float64
	Cost          *float64
}

func formatOptionalFloat(f *float64) string {
	if f == nil {
		return "0"
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

func capabilityContainerForOperation(op aasOperationData, providerIDStr, providerName, country string, lat, lon float64) json.RawMessage {
	cap := capabilityNode(op.Key+"Capability", op.Display+" Capability")
	comment := multiLangNode("CapabilityComment", "Capability Comment",
		"Human-readable description of the offered capability.",
		"https://admin-shell.io/idta/CapabilityDescription/CapabilityComment/1/0",
		fmt.Sprintf("%s capability provided by %s in %s. Equipment: %s.", op.Display, providerName, country, op.EquipTypes))

	propSet := collectionNode("PropertySet_"+op.Key, op.Display+" Properties",
		"Properties describing the offered recoating capability.",
		"https://admin-shell.io/idta/CapabilityDescription/PropertySet/1/0",
		propertyContainer("CapabilityID", "CapabilityID", "Stable identifier of this offered capability.",
			"xs:string", fmt.Sprintf("CAP-%s-%s", providerIDStr, strings.ToUpper(op.Key)),
			"Stable identifier of this offered capability."),
		propertyContainer("OperationType", "OperationType", "Type of remanufacturing or support operation offered by the provider.",
			"xs:string", op.Display, "Type of remanufacturing or support operation offered by the provider."),
		propertyContainer("ProviderID", "ProviderID", "Identifier of the provider in the source dataset.",
			"xs:integer", providerIDStr, "Identifier of the provider in the source dataset."),
		propertyContainer("ProviderCountry", "ProviderCountry", "Country code of the provider.",
			"xs:string", country, "Country code of the provider."),
		propertyContainer("ProviderLatitude", "ProviderLatitude", "Latitude of the provider location.",
			"xs:double", strconv.FormatFloat(lat, 'f', -1, 64), "Latitude of the provider location."),
		propertyContainer("ProviderLongitude", "ProviderLongitude", "Longitude of the provider location.",
			"xs:double", strconv.FormatFloat(lon, 'f', -1, 64), "Longitude of the provider location."),
		propertyContainer("ProviderName", "ProviderName", "Name of the technology provider in the source dataset.",
			"xs:string", providerName, "Name of the technology provider in the source dataset."),
		propertyContainer("EquipmentTypes", "EquipmentTypes", "Semicolon-separated list of equipment available to realize the capability.",
			"xs:string", op.EquipTypes, "Semicolon-separated list of equipment available to realize the capability."),
		propertyContainer("EnergyPerOperation", "EnergyPerOperation", "Energy value associated with one execution of the offered capability.",
			"xs:double", formatOptionalFloat(op.Energy), "Energy value associated with one execution of the offered capability. The source dataset does not specify the measurement unit."),
		propertyContainer("CostPerOperation", "CostPerOperation", "Cost value associated with one execution of the offered capability.",
			"xs:double", formatOptionalFloat(op.Cost), "Cost value associated with one execution of the offered capability. The source dataset does not specify the currency."),
		propertyContainer("EquipmentSpecifications", "EquipmentSpecifications", "Normalized source equipment dictionary, including supplied capability limits and parameters.",
			"xs:string", op.EquipSpecJSON, "Normalized source equipment dictionary, including supplied capability limits and parameters."),
	)

	return collectionNode("CapabilityContainer_"+op.Key, op.Display,
		"Offered recoating capability of tool provider "+providerIDStr+".",
		"https://admin-shell.io/idta/CapabilityDescription/CapabilityContainer/1/0", cap, comment, propSet)
}

// BuildValidationCapabilityDoc assembles one provider's full AAS
// CapabilityDescription document — same shape as the existing 5, scoped
// under a distinct "electric-motor-validation-provider" id/file namespace
// so it never collides with the real providers 0-4.
func BuildValidationCapabilityDoc(providerIDStr, providerName, country string, lat, lon float64, operations []aasOperationData) ([]byte, error) {
	var containers []json.RawMessage
	for _, op := range operations {
		containers = append(containers, capabilityContainerForOperation(op, providerIDStr, providerName, country, lat, lon))
	}
	capSet := collectionNode("CapabilitySet_ElectricMotorServices", providerName+" Electric Motor Services",
		"Set of cutting-tool service capabilities offered by this provider.",
		"https://admin-shell.io/idta/CapabilityDescription/CapabilitySet/1/0", containers...)

	submodelID := "https://remanet-project.eu/ids/submodel/capability-description/electric-motor-validation-provider/" + providerIDStr
	submodel := aasSubmodel{
		IDShort:     "CapabilityDescription_ElectricMotorValidationProvider_" + providerIDStr,
		DisplayName: lang("Electric Motor Capability Description for " + providerName),
		Description: lang("Instance of IDTA 02020 Capability Description 1.0 for electric-motor remanufacturing services (DSM validation dataset)."),
		ID:          submodelID,
		Kind:        "Instance",
		SemanticID:  extRef("https://admin-shell.io/idta/SubmodelTemplate/CapabilityDescription/1/0"),
		SubmodelElements: []json.RawMessage{
			capSet,
		},
		ModelType: "Submodel",
	}

	shell := aasShell{
		IDShort:     "ElectricMotorValidationProviderCapabilityAAS_" + providerIDStr,
		DisplayName: lang(providerName + " Electric Motor Capabilities (validation dataset)"),
		Description: lang("Capability AAS for " + providerName + ", generated from the RemaNet DSM validation dataset (motor_providers_clustered.csv)."),
		ID:          "https://remanet-project.eu/ids/aas/electric-motor-validation-provider/" + providerIDStr,
		AssetInformation: aasAssetInfo{
			AssetKind:     "Instance",
			GlobalAssetID: "https://remanet-project.eu/ids/asset/electric-motor-validation-provider/" + providerIDStr,
			AssetType:     "ElectricMotorServiceProvider",
			SpecificAssetIDs: []aasSpecificAssetID{
				{Name: "ProviderID", Value: providerIDStr},
				{Name: "ProviderName", Value: providerName},
				{Name: "Country", Value: country},
			},
		},
		Submodels: []aasModelRef{{Type: "ModelReference", Keys: []aasKey{{Type: "Submodel", Value: submodelID}}}},
		ModelType: "AssetAdministrationShell",
	}

	doc := aasCapabilityDoc{
		AssetAdministrationShells: []aasShell{shell},
		Submodels:                 []aasSubmodel{submodel},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// humanizeServiceName turns a CSV/JSON equipment-dict key like
// "AssemblyDisassemblyService" into "Assembly Disassembly Service" —
// purely decorative (feeds only the human-readable EquipmentTypes/
// CapabilityComment text, never the machine-matched EquipmentSpecifications
// JSON), so an approximate split is fine.
var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func humanizeServiceName(name string) string {
	spaced := camelBoundary.ReplaceAllString(name, "$1 $2")
	spaced = strings.ReplaceAll(spaced, "Cnc", "CNC")
	spaced = strings.ReplaceAll(spaced, "Fea", "FEA")
	spaced = strings.ReplaceAll(spaced, "Cad", "CAD")
	spaced = strings.ReplaceAll(spaced, "Iec", "IEC")
	spaced = strings.ReplaceAll(spaced, "Cmm", "CMM")
	return spaced
}

// ── Generation entry point ──────────────────────────────────────────────

// GenerateValidationAASFiles reads motor_providers_clustered.csv and writes
// one CapabilityDescription AAS JSON file per provider into outDir,
// following the exact template of the 5 existing hand-written samples.
func GenerateValidationAASFiles(csvPath, outDir string) (int, error) {
	providers, capsByProvider, err := parseValidationProvidersGrouped(csvPath)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, fmt.Errorf("creating output dir: %w", err)
	}

	for _, p := range providers {
		var ops []aasOperationData
		for _, op := range validationDatasetOperations {
			c, ok := capsByProvider[p.ProviderID][op]
			if !ok {
				continue
			}
			equipSpecJSON, serviceNames, err := equipmentToSpecJSONAndNames(c.Equipment)
			if err != nil {
				return 0, fmt.Errorf("provider %s op %s: %w", p.ProviderID, op, err)
			}
			display := operationDisplayText[op]
			if display == "" {
				display = op
			}
			ops = append(ops, aasOperationData{
				Key: op, Display: display,
				EquipTypes:    strings.Join(serviceNames, "; "),
				EquipSpecJSON: equipSpecJSON,
				Energy:        c.Energy, Cost: c.Cost,
			})
		}

		doc, err := BuildValidationCapabilityDoc(p.ProviderID, p.Name, p.Country, p.Location[0], p.Location[1], ops)
		if err != nil {
			return 0, fmt.Errorf("provider %s: building AAS doc: %w", p.ProviderID, err)
		}
		idNum, _ := strconv.Atoi(p.ProviderID)
		fname := fmt.Sprintf("CapabilityDescription-ElectricMotorValidationProvider-%03d.json", idNum)
		if err := os.WriteFile(filepath.Join(outDir, fname), doc, 0o644); err != nil {
			return 0, fmt.Errorf("writing %s: %w", fname, err)
		}
	}
	return len(providers), nil
}

// operationDisplayText mirrors the OperationType text used by the 5
// existing AAS files, extracted from them directly (see conversation —
// read every PropertyContainer_OperationType value across all 5 samples).
var operationDisplayText = map[string]string{
	"AssemblyTesting":              "Assembly Testing",
	"Reassembly":                   "Reassembly",
	"Reuse":                        "Reuse",
	"AssemblyForManufacturingStep": "Assembly for Manufacturing",
	"Grinding":                     "Grinding",
	"Replace":                      "Component Replacement",
	"Design":                       "Motor Redesign",
	"DimensionalTesting":           "Dimensional Testing",
	"Turning":                      "Turning",
	"Punching":                     "Lamination Punching and Stacking",
	"Inspection":                   "Inspection",
	"AssemblyInspection":           "Assembly Inspection",
	"Disassembly":                  "Disassembly",
}

func equipmentToSpecJSONAndNames(equipment []execdb.Equipment) (string, []string, error) {
	spec := make(map[string]map[string]any, len(equipment))
	var names []string
	for _, e := range equipment {
		attrs := make(map[string]any, len(e.Attributes))
		for k, v := range e.Attributes {
			if len(v) == 1 {
				attrs[k] = v[0]
			} else {
				attrs[k] = v
			}
		}
		spec[e.Name] = attrs
		if e.Name != "" {
			names = append(names, humanizeServiceName(e.Name))
		}
	}
	if len(spec) == 0 {
		spec = map[string]map[string]any{"": {}}
		names = []string{"Available; equipment type not specified"}
	}
	b, err := json.Marshal(spec)
	if err != nil {
		return "", nil, err
	}
	return string(b), names, nil
}

// ── Parsing (actor identification sourced from AAS files) ───────────────

// LoadAASCapabilityProviders reads every CapabilityDescription-*.json file
// in dir and returns one ProviderRecord plus its CapabilityRecords per
// file — the same shapes techProviderSearch/matchProviderCapabilities
// already consume, so no downstream matching code needs to change: only
// where this data comes from does.
func LoadAASCapabilityProviders(dir, useCaseID string) ([]execdb.ProviderRecord, []execdb.CapabilityRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("reading AAS directory %s: %w", dir, err)
	}

	var providers []execdb.ProviderRecord
	var capabilities []execdb.CapabilityRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		provider, caps, err := parseAASCapabilityFile(path, useCaseID)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		providers = append(providers, provider)
		capabilities = append(capabilities, caps...)
	}
	return providers, capabilities, nil
}

// parseAASCapabilityFile walks one CapabilityDescription AAS document's
// tree generically (map[string]any, not static structs) because the AAS
// "value" field means a different shape depending on modelType
// (SubmodelElementCollection -> []element, Property -> scalar string,
// MultiLanguageProperty -> []langText, Capability -> no value at all) —
// far simpler to navigate dynamically than to fight that polymorphism
// with static types on the read side.
func parseAASCapabilityFile(path, useCaseID string) (execdb.ProviderRecord, []execdb.CapabilityRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return execdb.ProviderRecord{}, nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return execdb.ProviderRecord{}, nil, fmt.Errorf("invalid JSON: %w", err)
	}

	shells, _ := doc["assetAdministrationShells"].([]any)
	if len(shells) == 0 {
		return execdb.ProviderRecord{}, nil, fmt.Errorf("no assetAdministrationShells")
	}
	shell, _ := shells[0].(map[string]any)
	assetInfo, _ := shell["assetInformation"].(map[string]any)
	specificIDs, _ := assetInfo["specificAssetIds"].([]any)

	var providerID, providerName, country string
	for _, raw := range specificIDs {
		entry, _ := raw.(map[string]any)
		name, _ := entry["name"].(string)
		value, _ := entry["value"].(string)
		switch name {
		case "ProviderID":
			providerID = value
		case "ProviderName":
			providerName = value
		case "Country":
			country = value
		}
	}
	if providerID == "" {
		return execdb.ProviderRecord{}, nil, fmt.Errorf("missing ProviderID in specificAssetIds")
	}

	submodels, _ := doc["submodels"].([]any)
	if len(submodels) == 0 {
		return execdb.ProviderRecord{}, nil, fmt.Errorf("no submodels")
	}
	submodel, _ := submodels[0].(map[string]any)
	submodelElements, _ := submodel["submodelElements"].([]any)
	if len(submodelElements) == 0 {
		return execdb.ProviderRecord{}, nil, fmt.Errorf("no submodelElements")
	}
	capSet, _ := submodelElements[0].(map[string]any)
	containers, _ := capSet["value"].([]any)

	var lat, lon float64
	haveLocation := false
	var operations []string
	var capabilities []execdb.CapabilityRecord

	for _, rawContainer := range containers {
		container, _ := rawContainer.(map[string]any)
		idShort, _ := container["idShort"].(string)
		op := strings.TrimPrefix(idShort, "CapabilityContainer_")
		if op == idShort {
			continue // not a capability container
		}
		children, _ := container["value"].([]any)
		var propSet map[string]any
		for _, rawChild := range children {
			child, _ := rawChild.(map[string]any)
			if cidShort, _ := child["idShort"].(string); strings.HasPrefix(cidShort, "PropertySet_") {
				propSet = child
				break
			}
		}
		if propSet == nil {
			return execdb.ProviderRecord{}, nil, fmt.Errorf("operation %s: no PropertySet found", op)
		}
		propContainers, _ := propSet["value"].([]any)

		fields := make(map[string]string)
		for _, rawPC := range propContainers {
			pc, _ := rawPC.(map[string]any)
			pcIDShort, _ := pc["idShort"].(string)
			key := strings.TrimPrefix(pcIDShort, "PropertyContainer_")
			leafChildren, _ := pc["value"].([]any)
			for _, rawLeaf := range leafChildren {
				leaf, _ := rawLeaf.(map[string]any)
				if mt, _ := leaf["modelType"].(string); mt == "Property" {
					if v, ok := leaf["value"].(string); ok {
						fields[key] = v
					}
				}
			}
		}

		if !haveLocation {
			if v, err := strconv.ParseFloat(fields["ProviderLatitude"], 64); err == nil {
				lat = v
			}
			if v, err := strconv.ParseFloat(fields["ProviderLongitude"], 64); err == nil {
				lon = v
			}
			haveLocation = true
		}

		equipment, err := parseAASEquipmentSpecJSON(fields["EquipmentSpecifications"])
		if err != nil {
			return execdb.ProviderRecord{}, nil, fmt.Errorf("operation %s: %w", op, err)
		}

		operations = append(operations, op)
		capabilities = append(capabilities, execdb.CapabilityRecord{
			ID:         useCaseID + ":" + providerID + ":" + op,
			UseCase:    useCaseID,
			ProviderID: providerID,
			Operation:  op,
			Energy:     parseAASFloatPtr(fields["EnergyPerOperation"]),
			Cost:       parseAASFloatPtr(fields["CostPerOperation"]),
			Equipment:  equipment,
		})
	}

	provider := execdb.ProviderRecord{
		ID:         useCaseID + ":" + providerID,
		UseCase:    useCaseID,
		ProviderID: providerID,
		Name:       providerName,
		Country:    country,
		Location:   []float64{lat, lon},
		Operations: operations,
	}
	return provider, capabilities, nil
}

func parseAASFloatPtr(raw string) *float64 {
	if raw == "" {
		return nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &f
}

// parseAASEquipmentSpecJSON parses an EquipmentSpecifications value — a
// proper JSON object string (unlike the CSV's Python-dict-repr cells),
// e.g. {"AssemblyDisassemblyService":{"RatedPower":"7.500 kW"}} — into the
// same []execdb.Equipment shape the CSV path produces, so MatchCapability
// sees identical structures regardless of source.
func parseAASEquipmentSpecJSON(raw string) ([]execdb.Equipment, error) {
	if raw == "" {
		return nil, nil
	}
	var parsed map[string]map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("EquipmentSpecifications %q: %w", raw, err)
	}
	equipment := make([]execdb.Equipment, 0, len(parsed))
	for name, attrs := range parsed {
		if name == "" {
			continue
		}
		attributes := make(map[string][]string, len(attrs))
		for k, v := range attrs {
			attributes[k] = csvAttrValueToList(v)
		}
		equipment = append(equipment, execdb.Equipment{Name: name, Attributes: attributes})
	}
	return equipment, nil
}

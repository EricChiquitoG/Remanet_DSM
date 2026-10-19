package src

// A golden-master validation test: a hand-picked 12-provider CSV
// (client/data/dsm_validation_dataset/motor_providers_clustered.csv) plus
// two ground-truth provider x step compatibility matrices computed
// independently and stored in capability_matrices.xlsx. This test seeds
// those 12 providers into a throwaway use case, runs the real
// techProviderSearch against the real KBSStrategyToDAG-built DAG for both
// of the Motor batch's strategies, and diffs the result against the
// ground truth cell by cell — proving the whole KBS-ingestion +
// matchmaking pipeline (including the Universal-step rule) reproduces an
// independently computed answer, not just "doesn't crash."
//
// The ground-truth matrices below were extracted by reading
// capability_matrices.xlsx's raw XML directly (no xlsx library in this
// module) and verified cell-by-cell against the sheet's own
// ReconditionStepsLegend/UpgradeStepsLegend/Infos sheets before being
// copied here.

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

const clusteredProvidersCSV = "../data/dsm_validation_dataset/motor_providers_clustered.csv"

var validationOperations = []string{
	"Reuse", "AssemblyInspection", "Inspection", "Punching", "Design", "Grinding",
	"Replace", "AssemblyForManufacturingStep", "Disassembly", "AssemblyTesting",
	"DimensionalTesting", "Turning", "Reassembly",
}

// providerOrder is the row order of both the CSV and the ground-truth
// matrices (IdProvider 0..11).
var providerOrder = []string{
	"COST_01_AssemblyHub", "COST_02_DesignPunching", "COST_03_QualityTesting", "COST_04_IntegratedWorks",
	"ENERGY_01_AssemblyHub", "ENERGY_02_DesignPunching", "ENERGY_03_QualityTesting", "ENERGY_04_IntegratedWorks",
	"LOCAL_01_AssemblyHub", "LOCAL_02_DesignPunching", "LOCAL_03_QualityTesting", "LOCAL_04_IntegratedWorks",
}

// reconditionGroundTruth[providerIdx][stepIdx], steps in the same order as
// reconditionStrategy's steps[] array (ReconditionStepsLegend, sheet3).
var reconditionGroundTruth = [][]int{
	{1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1},
	{0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1},
	{0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1},
	{0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

// upgradeGroundTruth[providerIdx][stepIdx], steps in Upgrade's steps[]
// array order (UpgradeStepsLegend, sheet4).
var upgradeGroundTruth = [][]int{
	{1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1},
	{1, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1},
	{1, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	{1, 1, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1},
	{1, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 1},
	{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
}

func validationIsBlank(v string) bool { return v == "" || v == "N/A" }

func validationParseFloatPtr(raw string) *float64 {
	if validationIsBlank(raw) {
		return nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil
	}
	return &f
}

func validationAttrValueToList(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			out = append(out, validationAttrValueToList(e)...)
		}
		return out
	default:
		return nil
	}
}

func validationParseEquipmentDict(raw string) ([]execdb.Equipment, error) {
	if validationIsBlank(raw) {
		return nil, nil
	}
	jsonish := strings.ReplaceAll(raw, "'", "\"")
	var parsed map[string]map[string]any
	if err := json.Unmarshal([]byte(jsonish), &parsed); err != nil {
		return nil, err
	}
	equipment := make([]execdb.Equipment, 0, len(parsed))
	for name, attrs := range parsed {
		if name == "" {
			continue
		}
		attributes := make(map[string][]string, len(attrs))
		for k, v := range attrs {
			attributes[k] = validationAttrValueToList(v)
		}
		equipment = append(equipment, execdb.Equipment{Name: name, Attributes: attributes})
	}
	return equipment, nil
}

// loadClusteredProviders parses motor_providers_clustered.csv — same
// column shape as the real motor_providers.csv the importer reads — into
// ProviderRecord/CapabilityRecord pairs scoped to useCaseID.
func loadClusteredProviders(t *testing.T, useCaseID string) ([]execdb.ProviderRecord, []execdb.CapabilityRecord) {
	t.Helper()
	f, err := os.Open(clusteredProvidersCSV)
	if err != nil {
		t.Fatalf("opening clustered providers CSV: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("reading clustered providers CSV: %v", err)
	}
	header := records[0]

	var providers []execdb.ProviderRecord
	var capabilities []execdb.CapabilityRecord
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

		var offered []string
		for _, op := range validationOperations {
			eqRaw := row[op+"Equipments"]
			if validationIsBlank(eqRaw) {
				continue
			}
			equipment, err := validationParseEquipmentDict(eqRaw)
			if err != nil {
				t.Fatalf("provider %s op %s: %v", providerID, op, err)
			}
			offered = append(offered, op)
			capabilities = append(capabilities, execdb.CapabilityRecord{
				ID:         useCaseID + ":" + providerID + ":" + op,
				UseCase:    useCaseID,
				ProviderID: providerID,
				Operation:  op,
				Energy:     validationParseFloatPtr(row[op+"Energy"]),
				Cost:       validationParseFloatPtr(row[op+"Cost"]),
				Equipment:  equipment,
			})
		}
		providers = append(providers, execdb.ProviderRecord{
			ID:         useCaseID + ":" + providerID,
			UseCase:    useCaseID,
			ProviderID: providerID,
			Name:       row["Name"],
			Country:    row["Country"],
			Location:   []float64{lat, lon},
			Operations: offered,
		})
	}
	return providers, capabilities
}

// TestKBSValidation_MotorBatch2AgainstGroundTruth is the actual validation
// example: seeds the 12 clustered providers, runs the real matching
// pipeline against both of the Motor batch's strategies, and diffs every
// (provider, step) cell against the independently computed ground truth.
func TestKBSValidation_MotorBatch2AgainstGroundTruth(t *testing.T) {
	requireMongoForKBS(t)

	testUseCaseID := "kbs-validation-motor"
	uc := electricMotorUseCaseForKBS() // same KBSAttributeAliases as the real electric-motor config
	uc.ID = testUseCaseID
	uc.Operations = validationOperations
	if err := execdb.UpsertUseCase(uc); err != nil {
		t.Fatalf("seeding validation use case: %v", err)
	}
	providers, capabilities := loadClusteredProviders(t, testUseCaseID)
	if len(providers) != 12 {
		t.Fatalf("parsed %d providers from clustered CSV, want 12", len(providers))
	}
	for _, p := range providers {
		if err := execdb.UpsertProvider(p); err != nil {
			t.Fatalf("seeding provider %s: %v", p.ID, err)
		}
	}
	for _, c := range capabilities {
		if err := execdb.UpsertCapability(c); err != nil {
			t.Fatalf("seeding capability %s: %v", c.ID, err)
		}
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		execdb.UseCaseCollection.DeleteOne(ctx, bson.M{"_id": testUseCaseID})
		execdb.ProviderCollection.DeleteMany(ctx, bson.M{"useCase": testUseCaseID})
		execdb.CapabilityCollection.DeleteMany(ctx, bson.M{"useCase": testUseCaseID})
	}()

	kbs, err := LoadKBSOutput(motorKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}

	cases := []struct {
		strategyName string
		groundTruth  [][]int
	}{
		{"reconditionStrategy", reconditionGroundTruth},
		{"Upgrade", upgradeGroundTruth},
	}

	for _, tc := range cases {
		t.Run(tc.strategyName, func(t *testing.T) {
			var strat *KBSStrategy
			for i := range kbs.FeasibleStrategies {
				if kbs.FeasibleStrategies[i].StrategyName == tc.strategyName {
					strat = &kbs.FeasibleStrategies[i]
				}
			}
			if strat == nil {
				t.Fatalf("strategy %q not found", tc.strategyName)
			}

			dag, err := KBSStrategyToDAG(uc, *strat)
			if err != nil {
				t.Fatalf("KBSStrategyToDAG: %v", err)
			}
			steps := make([]StepEntry, 0, len(dag))
			for name, node := range dag {
				steps = append(steps, StepEntry{StepName: name, Operation: node.Operation, Universal: node.Universal, Requirements: node.Requirements})
			}

			resp, err := techProviderSearch(testUseCaseID, steps)
			if err != nil {
				t.Fatalf("techProviderSearch: %v", err)
			}

			matched := make(map[string]map[string]bool, len(resp.Responses)) // providerName -> stepName -> matched
			for _, r := range resp.Responses {
				m := make(map[string]bool, len(r.Processes))
				for _, p := range r.Processes {
					m[p.ProcessName] = true
				}
				matched[r.uid] = m
			}

			mismatches := 0
			for providerIdx, providerName := range providerOrder {
				for stepIdx, step := range strat.Steps {
					want := tc.groundTruth[providerIdx][stepIdx] == 1
					got := matched[providerName][step.CurrentStepName]
					if want != got {
						mismatches++
						t.Errorf("%s / %s / step %d (%s): want %v, got %v",
							tc.strategyName, providerName, stepIdx, step.CurrentStepName, want, got)
					}
				}
			}
			t.Logf("%s: %d providers x %d steps checked, %d mismatches", tc.strategyName, len(providerOrder), len(strat.Steps), mismatches)
		})
	}
}

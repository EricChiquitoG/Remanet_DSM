package src

import (
	"context"
	"os"
	"testing"
	"time"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

const motorKBSFile = "../data/Holo_Files/T5_Motor_batch-2_N-10_DSP-2_kbs_output.json"
const clutchKBSFile = "../data/Holo_Files/T5_Clutch_batch-3_N-10_DSP-3_kbs_output.json"
const solidToolKBSFile = "../data/Holo_Files/T5_SolidTool_batch-4_N-10_DSP-4_kbs_output.json"

// electricMotorUseCaseForKBS mirrors the real client/data/usecases/electric-motor.json
// config (AttributeRules included — without them, AxisHeight/RatedPower/
// ShaftDiameter would fall back to exact-string matching instead of
// numeric_min, which breaks any real capability matching against this use
// case, not just KBS-sourced requirements).
func electricMotorUseCaseForKBS() execdb.UseCase {
	return execdb.UseCase{
		ID: "electric-motor",
		KBSAttributeAliases: map[string]string{
			"axisheightmotor":    "AxisHeight",
			"ratedpowermotor":    "RatedPower",
			"shaftdiametermotor": "ShaftDiameter",
		},
		AttributeRules: map[string]execdb.AttributeRule{
			"AxisHeight":          {Type: "numeric_min"},
			"RatedPower":          {Type: "numeric_min"},
			"ShaftLength":         {Type: "numeric_min"},
			"ShaftDiameter":       {Type: "numeric_min"},
			"ShaftToleranceClass": {Type: "ordinal", Order: []string{"bad", "mid", "good"}},
		},
		ExistenceOnlyOperations: []string{"AssemblyTesting", "Reuse", "Replace"},
	}
}

func TestKBSStepOperation(t *testing.T) {
	cases := []struct {
		name      string
		stepName  string
		artifact  string
		wantOp    string
		expectErr bool
	}{
		{"disassembly", "motor001-disassembly", "motor001", "Disassembly", false},
		{"camelCase suffix", "motor001-assemblyTesting", "motor001", "AssemblyTesting", false},
		{"underscored artifact", "bearing_001-visualInspection", "bearing_001", "VisualInspection", false},
		{"design", "reluctanceMotorRotorPart-design", "reluctanceMotorRotorPart", "Design", false},
		{"mismatched artifact prefix", "motor001-disassembly", "bearing001", "", true},
		{"empty suffix", "motor001-", "motor001", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			op, err := kbsStepOperation(KBSStep{CurrentStepName: c.stepName, AppliedToArtifact: c.artifact})
			if c.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got operation %q", op)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if op != c.wantOp {
				t.Errorf("got %q, want %q", op, c.wantOp)
			}
		})
	}
}

func TestKBSNormalizeRequirementKey(t *testing.T) {
	cases := map[string]string{
		"axisHeightMotor001":       "axisheightmotor",
		"ratedPowerMotor001":       "ratedpowermotor",
		"corrosionLevelHousing001": "corrosionlevelhousing",
		"torqueCapacityClutch001":  "torquecapacityclutch",
		"clutchOuterDiameter001":   "clutchouterdiameter",
	}
	for raw, want := range cases {
		if got := kbsNormalizeRequirementKey(raw); got != want {
			t.Errorf("kbsNormalizeRequirementKey(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestKBSFormatRequirementValue(t *testing.T) {
	uom := func(s string) *string { return &s }

	cases := []struct {
		name      string
		raw       string // raw JSON for requirementValue
		uom       *string
		want      string
		expectErr bool
	}{
		{"number with unit", "320.0", uom("mm"), "320 mm", false},
		{"fractional number with unit", "5.5", uom("kW"), "5.5 kW", false},
		{"number with nil unit", "100.0", nil, "100", false},
		{"bare string, no unit", `"mid"`, nil, "mid", false},
		{"invalid shape", "[1,2,3]", nil, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := kbsFormatRequirementValue([]byte(c.raw), c.uom)
			if c.expectErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestKBSStrategyToDAG_MotorRecondition proves the actual bug this whole
// mechanism exists to fix: reconditionStrategy has five separate
// Inspection-family steps on five different sub-artifacts (bearing, rotor,
// stator, rotorLamination, shaft). Keying the DAG by bare operation name
// (as the pipeline did before DAGNode gained Operation) would have the
// later ones silently overwrite the earlier ones in the map. This checks
// all of them survive as distinct nodes with their own predecessors and
// their own Operation tag.
func TestKBSStrategyToDAG_MotorRecondition(t *testing.T) {
	kbs, err := LoadKBSOutput(motorKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	var strat *KBSStrategy
	for i := range kbs.FeasibleStrategies {
		if kbs.FeasibleStrategies[i].StrategyName == "reconditionStrategy" {
			strat = &kbs.FeasibleStrategies[i]
		}
	}
	if strat == nil {
		t.Fatal("reconditionStrategy not found in sample file")
	}

	uc := electricMotorUseCaseForKBS()
	dag, err := KBSStrategyToDAG(uc, *strat)
	if err != nil {
		t.Fatalf("KBSStrategyToDAG: %v", err)
	}
	if len(dag) != len(strat.Steps) {
		t.Fatalf("got %d DAG nodes, want %d (one per step — a collision would shrink this)", len(dag), len(strat.Steps))
	}

	disassembly, ok := dag["motor001-disassembly"]
	if !ok {
		t.Fatal("missing motor001-disassembly node")
	}
	if disassembly.Operation != "Disassembly" {
		t.Errorf("motor001-disassembly: Operation = %q, want %q", disassembly.Operation, "Disassembly")
	}
	if got, want := disassembly.Requirements["RatedPower"], []string{"5.5 kW"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("motor001-disassembly: RatedPower = %v, want %v", got, want)
	}
	if got, want := disassembly.Requirements["AxisHeight"], []string{"100 mm"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("motor001-disassembly: AxisHeight = %v, want %v", got, want)
	}

	// The five same-operation, different-artifact Inspection-family nodes:
	// each must exist, each must carry the Inspection operation it was
	// given in the source data, each with its own (different) predecessor.
	inspectionNodes := map[string][]string{
		"bearing001-inspection":         {"motor001-disassembly"},
		"rotor001-assemblyInspection":   {"motor001-disassembly"},
		"stator001-assemblyInspection":  {"motor001-disassembly"},
		"rotorLamination001-inspection": {"rotor001-disassembly"},
		"shaft001-inspection":           {"rotor001-disassembly"},
	}
	for name, wantPrereqs := range inspectionNodes {
		node, ok := dag[name]
		if !ok {
			t.Errorf("missing node %q — collision with another same-operation step?", name)
			continue
		}
		if len(node.Prerequisites) != 1 || node.Prerequisites[0] != wantPrereqs[0] {
			t.Errorf("%s: Prerequisites = %v, want %v", name, node.Prerequisites, wantPrereqs)
		}
	}
	if dag["bearing001-inspection"].Operation != "Inspection" {
		t.Errorf("bearing001-inspection: Operation = %q, want %q", dag["bearing001-inspection"].Operation, "Inspection")
	}
	if dag["rotorLamination001-inspection"].Operation != "Inspection" {
		t.Errorf("rotorLamination001-inspection: Operation = %q, want %q", dag["rotorLamination001-inspection"].Operation, "Inspection")
	}
}

// TestKBSStrategyToDAG_SolidTool proves the SolidTool file (the cutting-tool
// use case's KBS output, per the mapping that productType "SolidTool" is
// cutting-tool's KBS identity) converts cleanly with zero
// kbsAttributeAliases entries: both its steps (inspection -> replace) carry
// no serviceRequirements in this sample, so no alias lookup is ever
// triggered.
func TestKBSStrategyToDAG_SolidTool(t *testing.T) {
	kbs, err := LoadKBSOutput(solidToolKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	if len(kbs.FeasibleStrategies) != 1 {
		t.Fatalf("got %d feasibleStrategies, want 1", len(kbs.FeasibleStrategies))
	}
	strat := kbs.FeasibleStrategies[0]

	uc := execdb.UseCase{ID: "cutting-tool", KBSAttributeAliases: map[string]string{}}
	dag, err := KBSStrategyToDAG(uc, strat)
	if err != nil {
		t.Fatalf("KBSStrategyToDAG: %v", err)
	}
	if len(dag) != 2 {
		t.Fatalf("got %d DAG nodes, want 2", len(dag))
	}

	inspection, ok := dag["solidTool001-inspection"]
	if !ok {
		t.Fatal("missing solidTool001-inspection node")
	}
	if inspection.Operation != "Inspection" {
		t.Errorf("solidTool001-inspection: Operation = %q, want %q", inspection.Operation, "Inspection")
	}
	if len(inspection.Prerequisites) != 0 {
		t.Errorf("solidTool001-inspection: Prerequisites = %v, want none (it's the route's root)", inspection.Prerequisites)
	}

	replace, ok := dag["solidTool001-replace"]
	if !ok {
		t.Fatal("missing solidTool001-replace node")
	}
	if replace.Operation != "Replace" {
		t.Errorf("solidTool001-replace: Operation = %q, want %q", replace.Operation, "Replace")
	}
	if len(replace.Prerequisites) != 1 || replace.Prerequisites[0] != "solidTool001-inspection" {
		t.Errorf("solidTool001-replace: Prerequisites = %v, want [solidTool001-inspection]", replace.Prerequisites)
	}
}

// TestKBSStrategyToDAG_MissingAlias proves the converter fails loudly
// rather than guessing when a use case's kbsAttributeAliases doesn't cover
// a requirement the KBS actually sent — the state Clutch is in today (no
// use case has claimed its kbsProductType, since that's being deferred).
func TestKBSStrategyToDAG_MissingAlias(t *testing.T) {
	kbs, err := LoadKBSOutput(clutchKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	uc := execdb.UseCase{ID: "clutch", KBSAttributeAliases: map[string]string{}}
	if _, err := KBSStrategyToDAG(uc, kbs.FeasibleStrategies[0]); err == nil {
		t.Error("expected an error for a requirement with no alias entry, got nil")
	}
}

// requireMongoForKBS only needs Mongo — unlike requireMongo (used by the
// provider-matching tests), KBSOutputToOptimizationInput never touches
// Redis, so it shouldn't be skipped just because Redis isn't reachable.
func requireMongoForKBS(t *testing.T) {
	t.Helper()
	if os.Getenv("MONGODB_URL") == "" {
		t.Skip("MONGODB_URL not set — skipping live MongoDB integration test")
	}
	if execdb.Client == nil {
		if err := execdb.InitMongoDB(); err != nil {
			t.Skipf("MongoDB not reachable: %v", err)
		}
	}
}

// TestKBSOutputToOptimizationInput_LiveMotor exercises the full converter
// end to end against a real Mongo use-case document (upserted and cleaned
// up by this test, so it doesn't depend on — or disturb — whatever the
// importer has actually seeded for "electric-motor").
func TestKBSOutputToOptimizationInput_LiveMotor(t *testing.T) {
	requireMongoForKBS(t)

	testUseCaseID := "kbs-test-motor"
	sentinelProductType := "Motor-test-sentinel" // avoids colliding with the real electric-motor doc's kbsProductType "Motor"
	uc := electricMotorUseCaseForKBS()
	uc.ID = testUseCaseID
	uc.KBSProductType = sentinelProductType
	if err := execdb.UpsertUseCase(uc); err != nil {
		t.Fatalf("seeding test use case: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := execdb.UseCaseCollection.DeleteOne(ctx, bson.M{"_id": testUseCaseID}); err != nil {
			t.Logf("cleanup: failed to delete test use case %s: %v", testUseCaseID, err)
		}
	}()

	kbs, err := LoadKBSOutput(motorKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	kbs.ProductType = sentinelProductType

	input, err := KBSOutputToOptimizationInput(kbs, []float64{53.0, -8.0}, "forward")
	if err != nil {
		t.Fatalf("KBSOutputToOptimizationInput: %v", err)
	}

	if input.UseCase != testUseCaseID {
		t.Errorf("UseCase = %q, want %q", input.UseCase, testUseCaseID)
	}
	if input.ProcessRequirements.ProductMatch != "IE3" {
		t.Errorf("ProductMatch = %q, want %q", input.ProcessRequirements.ProductMatch, "IE3")
	}
	if input.ProcessRequirements.Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", input.ProcessRequirements.Quantity)
	}
	if _, ok := input.Routes["reconditionStrategy"]; !ok {
		t.Error("missing reconditionStrategy route")
	}
	if _, ok := input.Routes["Upgrade"]; !ok {
		t.Error("missing Upgrade route")
	}
}

// TestKBSOutputToOptimizationInput_LiveSolidTool mirrors the Motor test
// above, but for the cutting-tool use case under its KBS identity
// "SolidTool".
func TestKBSOutputToOptimizationInput_LiveSolidTool(t *testing.T) {
	requireMongoForKBS(t)

	testUseCaseID := "kbs-test-cutting"
	sentinelProductType := "SolidTool-test-sentinel" // avoids colliding with the real cutting-tool doc's kbsProductType "SolidTool"
	uc := execdb.UseCase{ID: testUseCaseID, KBSProductType: sentinelProductType, KBSAttributeAliases: map[string]string{}}
	if err := execdb.UpsertUseCase(uc); err != nil {
		t.Fatalf("seeding test use case: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := execdb.UseCaseCollection.DeleteOne(ctx, bson.M{"_id": testUseCaseID}); err != nil {
			t.Logf("cleanup: failed to delete test use case %s: %v", testUseCaseID, err)
		}
	}()

	kbs, err := LoadKBSOutput(solidToolKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	kbs.ProductType = sentinelProductType

	input, err := KBSOutputToOptimizationInput(kbs, []float64{53.0, -8.0}, "forward")
	if err != nil {
		t.Fatalf("KBSOutputToOptimizationInput: %v", err)
	}
	if input.UseCase != testUseCaseID {
		t.Errorf("UseCase = %q, want %q", input.UseCase, testUseCaseID)
	}
	if input.ProcessRequirements.Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", input.ProcessRequirements.Quantity)
	}
	route, ok := input.Routes["replaceStrategy"]
	if !ok {
		t.Fatal("missing replaceStrategy route")
	}
	if len(route) != 2 {
		t.Errorf("got %d nodes in replaceStrategy, want 2", len(route))
	}
}

// TestKBSOutputToOptimizationInput_UnknownProductType proves Clutch fails
// loudly today — no use case claims its kbsProductType, since Clutch
// support is explicitly being deferred — rather than silently matching the
// wrong (or no) use case.
func TestKBSOutputToOptimizationInput_UnknownProductType(t *testing.T) {
	requireMongoForKBS(t)

	kbs, err := LoadKBSOutput(clutchKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	if _, err := KBSOutputToOptimizationInput(kbs, []float64{53.0, -8.0}, "forward"); err == nil {
		t.Error("expected an error for productType \"Clutch\" (no use case declares it yet), got nil")
	}
}

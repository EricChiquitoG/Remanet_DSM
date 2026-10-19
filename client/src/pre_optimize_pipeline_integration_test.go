package src

// Runs everything ProcessDirectory + OptimizationHandler do up to (but not
// including) the gRPC call to the optimizer, against real seeded Mongo
// data for one real batch — proving actor selection, cost/energy/distance
// matrices, and the final JSON blob are all built correctly before
// anything is sent for optimization.
//
// Requires MongoDB + Redis; skips when MONGODB_URL isn't set (same
// convention as techprovider_integration_test.go).
//
// Run:
//
//	MONGODB_URL="mongodb://erichi:secret@localhost:27017/myapp?authSource=admin" \
//	MONGODB_DB=myapp go test -run TestPreOptimizePipeline -v

import (
	"context"
	"testing"
	"time"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

// productCatalogFindOne fetches a CatalogProduct's attributes/quantity by
// _id without pulling in the full execdb.CatalogProduct type (only what
// this file's tests need).
func productCatalogFindOne(t *testing.T, id string, out any) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return execdb.ProductCatalogCollection.FindOne(ctx, bson.M{"_id": id}).Decode(out)
}

// buildAndInspectRoute replicates ProcessDirectory's actor-selection and
// matrix-construction steps (skipping the HTTP layer, customerSearch — a
// single hardcoded "end" boundary stands in for its forward-mode fallback
// — and the final gRPC send) for one request, logging everything so the
// result can be inspected.
func runPreOptimizePipeline(t *testing.T, label string, request OptimizationInput) {
	t.Helper()
	t.Logf("=== %s ===", label)

	uniqueSteps := FlattenUniqueStepsWithData(request)
	TPResponses, err := techProviderSearch(request.UseCase, uniqueSteps)
	if err != nil {
		t.Logf("techProviderSearch error: %v", err)
	}

	t.Logf("-- candidate actors selected --")
	for _, r := range TPResponses.Responses {
		t.Logf("  %s @ %v", r.uid, r.location)
		for _, p := range r.Processes {
			t.Logf("      %-14s cost=%-10.4g energy=%-10.4g", p.ProcessName, p.Cost, p.Energy)
		}
	}

	resultCollection := ConvertTPToResultCollection(TPResponses)
	resultCollection.Results = append(resultCollection.Results,
		Result{ContactName: "end_destination", Matches: []string{"end"}, Location: request.StartingPoint},
		Result{ContactName: "origin", Matches: []string{"origin"}, Location: request.StartingPoint},
	)

	costMatrix, err := CreateCostMatrixFromResults(resultCollection, &TPResponses)
	if err != nil {
		t.Fatalf("CreateCostMatrixFromResults: %v", err)
	}
	energyMatrix, err := CreateEnergyMatrixFromResults(resultCollection, &TPResponses)
	if err != nil {
		t.Fatalf("CreateEnergyMatrixFromResults: %v", err)
	}
	distMatrix := DistanceMatrixConstructor(resultCollection, nil)

	t.Logf("-- cost matrix --")
	for actor, byProc := range costMatrix {
		t.Logf("  %s: %v", actor, byProc)
	}
	t.Logf("-- energy matrix --")
	for actor, byProc := range energyMatrix {
		t.Logf("  %s: %v", actor, byProc)
	}

	for routeName, dag := range request.Routes {
		prDAG := DAGToPrerequisites(dag)
		sequence, err := TopologicalSort(dag)
		if err != nil {
			t.Fatalf("route %s: invalid DAG: %v", routeName, err)
		}
		processes := append([]string{"origin"}, sequence...)
		processes = append(processes, "end")

		jsonData := exportToJson(processes, costMatrix, energyMatrix, distMatrix, augmentWithBoundaryNodes(prDAG))
		t.Logf("-- route %s: final payload that would be sent for optimization --", routeName)
		t.Logf("  %v", jsonData)

		// Feasibility check a human can read directly: for every real step
		// (not origin/end), is there at least one actor with cost>0?
		t.Logf("-- feasibility per step --")
		for _, step := range sequence {
			feasible := false
			for actor, byProc := range costMatrix {
				if byProc[step] > 0 {
					feasible = true
					t.Logf("  %-16s OK — e.g. %s (cost=%.4g)", step, actor, byProc[step])
					break
				}
			}
			if !feasible {
				t.Logf("  %-16s INFEASIBLE — no actor has a nonzero cost for this step", step)
			}
		}
	}
}

func TestPreOptimizePipeline_ElectricMotorBatch000(t *testing.T) {
	requireMongo(t)

	// Real values from product_catalog:electric-motor:batch_000 and
	// owners:electric-motor:0 (MotorOwner-000, IE). Route: Disassembly ->
	// Replace (bearing, RealAssessment=Bad) -> Reassembly ->
	// AssemblyTesting, mirroring the batch's actual condition.
	request := OptimizationInput{
		UseCase:             "electric-motor",
		ProcessRequirements: ProcessRequirements{ProductMatch: "IE4", Quantity: 22},
		StartingPoint:       []float64{53.06771992011326, -8.066015435216345},
		OptimizationMode:    "forward",
		Routes: map[string]map[string]DAGNode{
			"MotorRefurb_batch_000": {
				"Disassembly": {Prerequisites: []string{}, Requirements: Requirements{
					"AxisHeight": {"355"}, "RatedPower": {"315"},
				}},
				"Replace": {Prerequisites: []string{"Disassembly"}, Requirements: Requirements{}},
				"Reassembly": {Prerequisites: []string{"Replace"}, Requirements: Requirements{
					"AxisHeight": {"355"}, "RatedPower": {"315"},
				}},
				"AssemblyTesting": {Prerequisites: []string{"Reassembly"}, Requirements: Requirements{}},
			},
		},
	}
	runPreOptimizePipeline(t, "electric-motor batch_000 (AxisHeight=355, RatedPower=315)", request)
}

// TestPreOptimizePipeline_FirstFiveMotorBatches runs the same
// Disassembly -> Replace -> Reassembly -> AssemblyTesting route against the
// first 5 real electric-motor batches, each with its own actual
// AxisHeight/RatedPower as the Disassembly/Reassembly requirement, and
// reports how many providers matched per step — a direct, multi-batch
// version of the single-batch feasibility check above.
func TestPreOptimizePipeline_FirstFiveMotorBatches(t *testing.T) {
	requireMongo(t)

	batches := []string{"batch_000", "batch_001", "batch_002", "batch_003", "batch_004"}

	for _, batch := range batches {
		var product struct {
			Quantity   int               `bson:"quantity"`
			Attributes map[string]string `bson:"attributes"`
		}
		if err := productCatalogFindOne(t, "electric-motor:"+batch, &product); err != nil {
			t.Fatalf("%s: %v", batch, err)
		}
		axisHeight := product.Attributes["AxisHeight"]
		ratedPower := product.Attributes["RatedPower"]

		steps := []StepEntry{
			{StepName: "Disassembly", Requirements: Requirements{"AxisHeight": {axisHeight}, "RatedPower": {ratedPower}}},
			{StepName: "Replace", Requirements: Requirements{}},
			{StepName: "Reassembly", Requirements: Requirements{"AxisHeight": {axisHeight}, "RatedPower": {ratedPower}}},
			{StepName: "AssemblyTesting", Requirements: Requirements{}},
		}
		resp, err := techProviderSearch("electric-motor", steps)
		if err != nil {
			t.Fatalf("%s: techProviderSearch: %v", batch, err)
		}

		counts := map[string]int{}
		for _, r := range resp.Responses {
			for _, p := range r.Processes {
				counts[p.ProcessName]++
			}
		}
		t.Logf("%-10s qty=%-4d AxisHeight=%-5s RatedPower=%-6s -> Disassembly=%-3d Replace=%-3d Reassembly=%-3d AssemblyTesting=%-3d",
			batch, product.Quantity, axisHeight, ratedPower,
			counts["Disassembly"], counts["Replace"], counts["Reassembly"], counts["AssemblyTesting"])
	}
}

func TestPreOptimizePipeline_ElectricMotorBatch083(t *testing.T) {
	requireMongo(t)

	// batch_083: AxisHeight=80, RatedPower=1.5 — small enough to be within
	// every provider's declared Disassembly/Reassembly capacity, unlike
	// batch_000. Run side by side to separate "the pipeline is broken" from
	// "this particular batch has no capable provider."
	request := OptimizationInput{
		UseCase:             "electric-motor",
		ProcessRequirements: ProcessRequirements{ProductMatch: "IE1", Quantity: 1},
		StartingPoint:       []float64{53.06771992011326, -8.066015435216345},
		OptimizationMode:    "forward",
		Routes: map[string]map[string]DAGNode{
			"MotorRefurb_batch_083": {
				"Disassembly": {Prerequisites: []string{}, Requirements: Requirements{
					"AxisHeight": {"80"}, "RatedPower": {"1.5"},
				}},
				"Replace": {Prerequisites: []string{"Disassembly"}, Requirements: Requirements{}},
				"Reassembly": {Prerequisites: []string{"Replace"}, Requirements: Requirements{
					"AxisHeight": {"80"}, "RatedPower": {"1.5"},
				}},
				"AssemblyTesting": {Prerequisites: []string{"Reassembly"}, Requirements: Requirements{}},
			},
		},
	}
	runPreOptimizePipeline(t, "electric-motor batch_083 (AxisHeight=80, RatedPower=1.5)", request)
}

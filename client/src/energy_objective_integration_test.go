package src

// Fast gRPC round-trip check that the "energy" objective actually survives
// the real wire: Go builds a tiny problem with a nonzero EnergyMatrix,
// sends it to a live optimizer over gRPC, and checks the returned
// Solution.Energy matches by-hand expectation. Complements:
//   - matching.go's pure MatchCapability tests (no DB, no network)
//   - techprovider_integration_test.go (live Mongo, no optimizer)
//   - optimizer/verify_optimizer.py (live pymoo, no gRPC)
// This is the one that actually exercises the proto wire for the new field.
//
// Requires a running optimizer; skips (not fails) when OPTIMIZER_ADDR isn't
// set, same convention as evaluation_test.go.
//
// Run:
//
//	OPTIMIZER_ADDR=localhost:50061 go test -run TestEnergyObjective -v

import (
	"os"
	"testing"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
)

func TestEnergyObjectiveSurvivesGRPCRoundTrip(t *testing.T) {
	addr := os.Getenv("OPTIMIZER_ADDR")
	if addr == "" {
		t.Skip("OPTIMIZER_ADDR not set — skipping live optimizer round trip")
	}

	conn, err := connectToServer(addr)
	if err != nil {
		t.Fatalf("cannot connect to optimizer at %s: %v", addr, err)
	}
	defer conn.Close()
	client := pb.NewSubmissionServiceClient(conn)

	// One actor, one step, one location. Manufacturing cost sums every step
	// in the route including the origin/end boundary nodes (cost 1 each,
	// same sentinel CreateCostMatrixFromResults uses), so the only
	// feasible assignment totals 10 (Step1) + 1 (origin) + 1 (end) = 12,
	// with energy 7.5 (boundary nodes carry no energy). Anything other than
	// exactly that for every returned solution is a wiring bug.
	jsonData := map[string]interface{}{
		"processes": map[string][]string{
			"origin": {},
			"Step1":  {"origin"},
			"end":    {"Step1"},
		},
		"CostMatrix": map[string]map[string]float64{
			"Actor":  {"Step1": 10},
			"origin": {"origin": 1},
			"end":    {"end": 1},
		},
		"EnergyMatrix": map[string]map[string]float64{
			"Actor": {"Step1": 7.5},
		},
		"matrix": map[string]map[string]float64{
			"Actor":  {"Actor": 0, "origin": 0, "end": 0},
			"origin": {"Actor": 0, "origin": 0, "end": 0},
			"end":    {"Actor": 0, "origin": 0, "end": 0},
		},
	}

	res, err := sendOptimize(jsonData, client)
	if err != nil {
		t.Fatalf("sendOptimize failed: %v", err)
	}
	if res.ErrorMessage != "" {
		t.Fatalf("optimizer returned an error: %s", res.ErrorMessage)
	}
	if len(res.Solutions) == 0 {
		t.Fatal("expected at least one feasible solution")
	}
	for _, sol := range res.Solutions {
		if sol.Energy != 7.5 {
			t.Errorf("expected Energy=7.5 for the only feasible assignment, got %f", sol.Energy)
		}
		if sol.ManufacturingCost != 12 {
			t.Errorf("expected ManufacturingCost=12, got %f", sol.ManufacturingCost)
		}
	}
}

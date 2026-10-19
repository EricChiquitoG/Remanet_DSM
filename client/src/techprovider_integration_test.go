package src

// Integration check against a live MongoDB instance seeded by
// client/cmd/importdata. Proves techProviderSearch actually matches real
// capability data end-to-end — not just the pure MatchCapability unit
// tests in matching_test.go, which never touch the DB.
//
// Requires MongoDB; skips (not fails) when MONGODB_URL isn't set, same
// convention as evaluation_test.go's OPTIMIZER_ADDR gate.
//
// Run:
//
//	MONGODB_URL="mongodb://erichi:secret@localhost:27017/myapp?authSource=admin" \
//	MONGODB_DB=myapp go test -run TestTechProviderSearch -v

import (
	"os"
	"testing"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
)

func requireMongo(t *testing.T) {
	t.Helper()
	if os.Getenv("MONGODB_URL") == "" {
		t.Skip("MONGODB_URL not set — skipping live MongoDB integration test")
	}
	if execdb.Client == nil {
		if err := execdb.InitMongoDB(); err != nil {
			t.Skipf("MongoDB not reachable: %v", err)
		}
	}
	// techProviderSearch caches each match via SaveTPResponse (Redis).
	if RedisClient == nil {
		if err := InitRedis(); err != nil {
			t.Skipf("Redis not reachable: %v", err)
		}
	}
}

func TestTechProviderSearch_ElectricMotorMatchesRealCapability(t *testing.T) {
	requireMongo(t)

	// Design's AxisHeight equipment value is a fixed constant (250 mm)
	// shared by every provider that declares it at all — true in both the
	// V2 and V3 datasets, just under different provider IDs/names each
	// time the data is regenerated. Check for any match rather than a
	// hardcoded provider identity, so this doesn't break on the next
	// reseed.
	steps := []StepEntry{
		{StepName: "Design", Requirements: Requirements{"AxisHeight": {"250 mm"}}},
		{StepName: "AssemblyTesting", Requirements: Requirements{}},
	}
	resp, err := techProviderSearch("electric-motor", steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var sawDesignMatch bool
	for _, r := range resp.Responses {
		for _, p := range r.Processes {
			if p.ProcessName == "Design" {
				sawDesignMatch = true
			}
		}
	}
	if !sawDesignMatch {
		t.Error("expected at least one provider to match Design with AxisHeight=250 mm")
	}
}

func TestTechProviderSearch_ElectricMotorRejectsMismatchedValue(t *testing.T) {
	requireMongo(t)

	// No provider's Design equipment declares a 9999 mm axis height —
	// MatchCapability should reject all of them for this step.
	steps := []StepEntry{
		{StepName: "Design", Requirements: Requirements{"AxisHeight": {"9999 mm"}}},
	}
	resp, _ := techProviderSearch("electric-motor", steps)
	for _, r := range resp.Responses {
		for _, p := range r.Processes {
			if p.ProcessName == "Design" {
				t.Errorf("provider %s should not match Design with an impossible AxisHeight", r.uid)
			}
		}
	}
}

func TestTechProviderSearch_CuttingToolCoatingAttributeMatch(t *testing.T) {
	requireMongo(t)

	steps := []StepEntry{
		{StepName: "Coating", Requirements: Requirements{"CoatingType": {"TiN"}}},
	}
	resp, err := techProviderSearch("cutting-tool", steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Responses) == 0 {
		t.Error("expected at least one cutting-tool provider offering TiN coating")
	}
}

func TestTechProviderSearch_CuttingToolExistenceOnlyOperation(t *testing.T) {
	requireMongo(t)

	// Inspection has no equipment attributes for any cutting-tool provider
	// and is declared existence-only — any requirement should still match
	// every provider that offers Inspection at all.
	steps := []StepEntry{
		{StepName: "Inspection", Requirements: Requirements{"AnythingAtAll": {"whatever"}}},
	}
	resp, err := techProviderSearch("cutting-tool", steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Responses) == 0 {
		t.Error("expected existence-only Inspection to match regardless of the (unsatisfiable) requirement")
	}
}

func TestTechProviderSearch_UnknownUseCaseErrors(t *testing.T) {
	requireMongo(t)

	_, err := techProviderSearch("not-a-real-use-case", []StepEntry{{StepName: "Anything"}})
	if err == nil {
		t.Error("expected an error for an unknown use case")
	}
}

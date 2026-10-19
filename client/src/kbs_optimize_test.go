package src

// Runs the real DSM pipeline all the way to a live optimizer and back,
// using the same clustered-provider data the matching validation already
// proved correct (kbs_validation_test.go). Unlike that test, which stops
// at the capability matrix, this one calls OptimizationHandler — the exact
// function ProcessDirectory uses for /get_options — so everything between
// "who can do each step" and "here is the chosen assignment" (cost/energy
// matrices, topological sort, the DAG JSON payload, the gRPC round trip)
// is the real production code path, not a reimplementation of it.
//
// Requires MongoDB and a reachable optimizer (OPTIMIZER_ADDR); skips if
// either is unavailable, same convention as the other integration tests.

import (
	"context"
	"os"
	"testing"
	"time"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"go.mongodb.org/mongo-driver/bson"
)

func TestKBSOptimize_MotorBatch2RealPipeline(t *testing.T) {
	requireMongoForKBS(t)
	if os.Getenv("OPTIMIZER_ADDR") == "" {
		t.Skip("OPTIMIZER_ADDR not set — skipping live optimizer run")
	}
	// Best-effort: techProviderSearch/OptimizationHandler cache through
	// Redis when it's available (both degrade to a logged warning via
	// cache.go's nil-guards when it's not), so connect it here too when
	// REDIS_ADDR/REDIS_PASSWORD are set — matching docker-compose's own
	// redis service — so this test exercises that real path as well
	// instead of silently skipping it.
	if RedisClient == nil && os.Getenv("REDIS_ADDR") != "" {
		if err := InitRedis(); err != nil {
			t.Logf("Redis not reachable, caching will be skipped: %v", err)
		}
	}

	testUseCaseID := "kbs-optimize-motor"
	uc := electricMotorUseCaseForKBS()
	uc.ID = testUseCaseID
	if err := execdb.UpsertUseCase(uc); err != nil {
		t.Fatalf("seeding use case: %v", err)
	}
	providers, capabilities := loadClusteredProviders(t, testUseCaseID)
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

	// Build the OptimizationInput the same way KBSOutputToOptimizationInput
	// would, but targeting the throwaway use case directly (that function
	// resolves by kbsProductType, which this scratch use case doesn't
	// declare — the DAG-building half, KBSStrategyToDAG, is identical
	// either way).
	routes := make(map[string]map[string]DAGNode, len(kbs.FeasibleStrategies))
	for _, strat := range kbs.FeasibleStrategies {
		dag, err := KBSStrategyToDAG(uc, strat)
		if err != nil {
			t.Fatalf("KBSStrategyToDAG %q: %v", strat.StrategyName, err)
		}
		routes[strat.StrategyName] = dag
	}
	request := OptimizationInput{
		UseCase: testUseCaseID,
		ProcessRequirements: ProcessRequirements{
			ProductMatch: kbs.InputProduct.EfficiencyClass,
			Quantity:     kbs.BatchCardinality,
		},
		Routes:           routes,
		StartingPoint:    []float64{53.0, -8.0},
		OptimizationMode: "forward",
	}

	// From here on, every call is the real function ProcessDirectory uses.
	uniqueSteps := FlattenUniqueStepsWithData(request)
	tpResponses, err := techProviderSearch(request.UseCase, uniqueSteps)
	if err != nil {
		t.Fatalf("techProviderSearch: %v", err)
	}

	resultCollection := ConvertTPToResultCollection(tpResponses)
	resultCollection.Results = append(resultCollection.Results,
		Result{ContactName: "end", Matches: []string{"end"}, Location: request.StartingPoint},
		Result{ContactName: "origin", Matches: []string{"origin"}, Location: request.StartingPoint},
	)

	costMatrix, err := CreateCostMatrixFromResults(resultCollection, &tpResponses)
	if err != nil {
		t.Fatalf("CreateCostMatrixFromResults: %v", err)
	}
	energyMatrix, err := CreateEnergyMatrixFromResults(resultCollection, &tpResponses)
	if err != nil {
		t.Fatalf("CreateEnergyMatrixFromResults: %v", err)
	}
	distMatrix := DistanceMatrixConstructor(resultCollection, nil)

	response, err := OptimizationHandler(request, distMatrix, costMatrix, energyMatrix)
	if err != nil {
		t.Fatalf("OptimizationHandler: %v", err)
	}

	for _, result := range response.Results {
		t.Logf("=== route with sequence %v ===", result.ProcessSequence)
		if result.Error != "" {
			t.Errorf("route failed: %s", result.Error)
			continue
		}
		if len(result.OptimizationJSON.Solutions) == 0 {
			t.Errorf("route produced zero solutions")
			continue
		}
		for _, sol := range result.OptimizationJSON.Solutions {
			t.Logf("  solution %d: transportCost=%.2f manufacturingCost=%.2f energy=%.2f",
				sol.SolNumber, sol.TransportCost, sol.ManufacturingCost, sol.Energy)
			for _, a := range sol.Assignments {
				t.Logf("    %-45s -> %s", a.ProcessName, a.ActorID)
			}
		}
	}
}

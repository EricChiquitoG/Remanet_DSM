package src

// Proves actor identification sourced from real AAS CapabilityDescription
// files (client/data/dsm_validation_dataset/aas-instancies/) reproduces
// the exact same independently-computed ground truth
// (capability_matrices.xlsx) that the Mongo/CSV-backed path already proved
// correct in kbs_validation_test.go — same providerOrder,
// reconditionGroundTruth, upgradeGroundTruth. Unlike that test, this one
// needs no MongoDB and no Redis at all: AASTechProviderSearch reads
// providers/capabilities straight from disk.

import (
	"testing"
)

const validationAASTestDir = "../data/dsm_validation_dataset/aas-instancies"

func TestAASValidation_MotorBatch2AgainstGroundTruth(t *testing.T) {
	uc := electricMotorUseCaseForKBS()

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

			resp, err := AASTechProviderSearch(uc, validationAASTestDir, steps)
			if err != nil {
				t.Fatalf("AASTechProviderSearch: %v", err)
			}

			matched := make(map[string]map[string]bool, len(resp.Responses))
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
			t.Logf("%s: %d providers x %d steps checked (AAS-sourced), %d mismatches", tc.strategyName, len(providerOrder), len(strat.Steps), mismatches)
		})
	}
}

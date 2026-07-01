package src

import (
	"strings"
	"testing"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
)

// ---- processOptimizationResponse ------------------------------------------

func TestProcessOptimizationResponse_ErrorMessage(t *testing.T) {
	res := &pb.OptimizationResponse{
		ErrorMessage: "something went wrong",
	}
	_, err := processOptimizationResponse(res)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error should contain server message, got: %v", err)
	}
}

func TestProcessOptimizationResponse_NoSolutions(t *testing.T) {
	res := &pb.OptimizationResponse{}
	results, err := processOptimizationResponse(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Solutions) != 0 {
		t.Errorf("expected 0 solutions, got %d", len(results.Solutions))
	}
}

func TestProcessOptimizationResponse_NormalAssignments(t *testing.T) {
	res := &pb.OptimizationResponse{
		Solutions: []*pb.Solution{
			{
				Assignments: []*pb.ProcessAssignment{
					{ProcessName: "Cleaning", UserId: "ActorA"},
					{ProcessName: "Reassembly", UserId: "ActorB"},
				},
				TransportCost:     100.5,
				ManufacturingCost: 200.0,
			},
		},
	}
	results, err := processOptimizationResponse(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Solutions) != 1 {
		t.Fatalf("expected 1 solution, got %d", len(results.Solutions))
	}
	sol := results.Solutions[0]
	if sol.TransportCost != 100.5 {
		t.Errorf("expected TransportCost=100.5, got %f", sol.TransportCost)
	}
	if sol.ManufacturingCost != 200.0 {
		t.Errorf("expected ManufacturingCost=200.0, got %f", sol.ManufacturingCost)
	}
	if len(sol.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(sol.Assignments))
	}
	if sol.Assignments[0].ProcessName != "Cleaning" || sol.Assignments[0].ActorID != "ActorA" {
		t.Errorf("first assignment wrong: %+v", sol.Assignments[0])
	}
	if sol.Assignments[1].ProcessName != "Reassembly" || sol.Assignments[1].ActorID != "ActorB" {
		t.Errorf("second assignment wrong: %+v", sol.Assignments[1])
	}
}

func TestProcessOptimizationResponse_FiltersBoundaryNodes(t *testing.T) {
	// "origin" and "end" are internal optimizer artifacts and must not appear
	// in the returned assignments.
	res := &pb.OptimizationResponse{
		Solutions: []*pb.Solution{
			{
				Assignments: []*pb.ProcessAssignment{
					{ProcessName: "origin", UserId: "origin_actor"},
					{ProcessName: "Cleaning", UserId: "ActorA"},
					{ProcessName: "Reassembly", UserId: "ActorB"},
					{ProcessName: "end", UserId: "end_customer"},
				},
				TransportCost:     50,
				ManufacturingCost: 130,
			},
		},
	}
	results, err := processOptimizationResponse(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sol := results.Solutions[0]
	for _, a := range sol.Assignments {
		if a.ProcessName == "origin" || a.ProcessName == "end" {
			t.Errorf("boundary node %q must be filtered out of assignments", a.ProcessName)
		}
	}
	if len(sol.Assignments) != 2 {
		t.Errorf("expected 2 real assignments after filtering, got %d", len(sol.Assignments))
	}
}

func TestProcessOptimizationResponse_AssignmentOrderPreserved(t *testing.T) {
	// The order from the proto (topological) must be preserved in the slice.
	processes := []string{"Cleaning", "Inspection", "Reassembly", "Certification"}
	protoAssignments := make([]*pb.ProcessAssignment, len(processes))
	for i, p := range processes {
		protoAssignments[i] = &pb.ProcessAssignment{ProcessName: p, UserId: "ActorX"}
	}
	res := &pb.OptimizationResponse{
		Solutions: []*pb.Solution{
			{Assignments: protoAssignments, TransportCost: 0, ManufacturingCost: 0},
		},
	}
	results, err := processOptimizationResponse(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sol := results.Solutions[0]
	for i, p := range processes {
		if sol.Assignments[i].ProcessName != p {
			t.Errorf("position %d: expected %s, got %s", i, p, sol.Assignments[i].ProcessName)
		}
	}
}

func TestProcessOptimizationResponse_MultipleSolutions(t *testing.T) {
	res := &pb.OptimizationResponse{
		Solutions: []*pb.Solution{
			{Assignments: []*pb.ProcessAssignment{{ProcessName: "Cleaning", UserId: "A"}},
				TransportCost: 10, ManufacturingCost: 100},
			{Assignments: []*pb.ProcessAssignment{{ProcessName: "Cleaning", UserId: "B"}},
				TransportCost: 20, ManufacturingCost: 80},
		},
	}
	results, err := processOptimizationResponse(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Solutions) != 2 {
		t.Fatalf("expected 2 solutions, got %d", len(results.Solutions))
	}
	if results.Solutions[0].SolNumber != 0 || results.Solutions[1].SolNumber != 1 {
		t.Errorf("SolNumbers should be 0 and 1, got %d and %d",
			results.Solutions[0].SolNumber, results.Solutions[1].SolNumber)
	}
}

func TestProcessOptimizationResponse_OnlyBoundaryNodes(t *testing.T) {
	// If every assignment is a boundary node, the result has 0 real assignments.
	res := &pb.OptimizationResponse{
		Solutions: []*pb.Solution{
			{
				Assignments: []*pb.ProcessAssignment{
					{ProcessName: "origin", UserId: "origin_actor"},
					{ProcessName: "end", UserId: "end_customer"},
				},
			},
		},
	}
	results, err := processOptimizationResponse(res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Solutions[0].Assignments) != 0 {
		t.Errorf("expected 0 assignments after filtering all boundary nodes")
	}
}

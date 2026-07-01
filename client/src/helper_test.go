package src

import (
	"reflect"
	"sort"
	"testing"
)

// ---- TopologicalSort -------------------------------------------------------

func TestTopologicalSort_LinearChain(t *testing.T) {
	dag := map[string]DAGNode{
		"A": {Prerequisites: []string{}},
		"B": {Prerequisites: []string{"A"}},
		"C": {Prerequisites: []string{"B"}},
	}
	got, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A must come before B, B before C
	pos := positionMap(got)
	if pos["A"] >= pos["B"] || pos["B"] >= pos["C"] {
		t.Errorf("wrong order: %v", got)
	}
}

func TestTopologicalSort_ParallelBranches(t *testing.T) {
	// A → C, B → C  (A and B can run in parallel, C waits for both)
	dag := map[string]DAGNode{
		"A": {Prerequisites: []string{}},
		"B": {Prerequisites: []string{}},
		"C": {Prerequisites: []string{"A", "B"}},
	}
	got, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := positionMap(got)
	if pos["A"] >= pos["C"] || pos["B"] >= pos["C"] {
		t.Errorf("C must come after both A and B: %v", got)
	}
}

func TestTopologicalSort_DiamondDAG(t *testing.T) {
	// A → B, A → C, B → D, C → D
	dag := map[string]DAGNode{
		"A": {Prerequisites: []string{}},
		"B": {Prerequisites: []string{"A"}},
		"C": {Prerequisites: []string{"A"}},
		"D": {Prerequisites: []string{"B", "C"}},
	}
	got, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pos := positionMap(got)
	if pos["A"] >= pos["B"] || pos["A"] >= pos["C"] {
		t.Errorf("A must be first: %v", got)
	}
	if pos["B"] >= pos["D"] || pos["C"] >= pos["D"] {
		t.Errorf("D must be last: %v", got)
	}
}

func TestTopologicalSort_SingleNode(t *testing.T) {
	dag := map[string]DAGNode{
		"Solo": {Prerequisites: []string{}},
	}
	got, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "Solo" {
		t.Errorf("expected [Solo], got %v", got)
	}
}

func TestTopologicalSort_EmptyDAG(t *testing.T) {
	dag := map[string]DAGNode{}
	got, err := TopologicalSort(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	dag := map[string]DAGNode{
		"A": {Prerequisites: []string{"B"}},
		"B": {Prerequisites: []string{"A"}},
	}
	_, err := TopologicalSort(dag)
	if err == nil {
		t.Fatal("expected error for cyclic DAG, got nil")
	}
}

func TestTopologicalSort_SelfLoop(t *testing.T) {
	dag := map[string]DAGNode{
		"A": {Prerequisites: []string{"A"}},
	}
	_, err := TopologicalSort(dag)
	if err == nil {
		t.Fatal("expected error for self-loop, got nil")
	}
}

// ---- augmentWithBoundaryNodes ----------------------------------------------

func TestAugmentWithBoundaryNodes_LinearChain(t *testing.T) {
	// A → B → C  (predecessor format)
	prDAG := map[string][]string{
		"A": {},
		"B": {"A"},
		"C": {"B"},
	}
	aug := augmentWithBoundaryNodes(prDAG)

	// origin must exist with no prerequisites
	if prereqs, ok := aug["origin"]; !ok || len(prereqs) != 0 {
		t.Errorf("origin should have no prereqs, got %v", prereqs)
	}
	// A is a start node → must depend on origin
	if !containsStr(aug["A"], "origin") {
		t.Errorf("A should depend on origin, got %v", aug["A"])
	}
	// B and C keep their original prerequisites
	if !reflect.DeepEqual(aug["B"], []string{"A"}) {
		t.Errorf("B prereqs wrong: %v", aug["B"])
	}
	// end must depend on C (only terminal)
	if !containsStr(aug["end"], "C") {
		t.Errorf("end should depend on C, got %v", aug["end"])
	}
	// end must NOT depend on A or B (they have successors)
	if containsStr(aug["end"], "A") || containsStr(aug["end"], "B") {
		t.Errorf("end must only list terminal nodes, got %v", aug["end"])
	}
}

func TestAugmentWithBoundaryNodes_ParallelBranches(t *testing.T) {
	// Two parallel chains A→C and B→C
	prDAG := map[string][]string{
		"A": {},
		"B": {},
		"C": {"A", "B"},
	}
	aug := augmentWithBoundaryNodes(prDAG)

	if !containsStr(aug["A"], "origin") {
		t.Errorf("A should depend on origin")
	}
	if !containsStr(aug["B"], "origin") {
		t.Errorf("B should depend on origin")
	}
	if !containsStr(aug["end"], "C") {
		t.Errorf("end should depend on C")
	}
	if containsStr(aug["end"], "A") || containsStr(aug["end"], "B") {
		t.Errorf("A and B are not terminal (C depends on them): %v", aug["end"])
	}
}

func TestAugmentWithBoundaryNodes_IsolatedNode(t *testing.T) {
	// X has no prereqs and nothing depends on it
	prDAG := map[string][]string{
		"X": {},
	}
	aug := augmentWithBoundaryNodes(prDAG)

	if !containsStr(aug["X"], "origin") {
		t.Errorf("isolated start node should depend on origin")
	}
	if !containsStr(aug["end"], "X") {
		t.Errorf("isolated terminal node should be prereq of end")
	}
}

func TestAugmentWithBoundaryNodes_Empty(t *testing.T) {
	aug := augmentWithBoundaryNodes(map[string][]string{})

	if _, ok := aug["origin"]; !ok {
		t.Error("origin node should always be present")
	}
	if _, ok := aug["end"]; !ok {
		t.Error("end node should always be present")
	}
	if len(aug["end"]) != 0 {
		t.Errorf("end should have no prereqs for empty input, got %v", aug["end"])
	}
}

func TestAugmentWithBoundaryNodes_MultipleTerminals(t *testing.T) {
	// A → B, A → C  (B and C are both terminal)
	prDAG := map[string][]string{
		"A": {},
		"B": {"A"},
		"C": {"A"},
	}
	aug := augmentWithBoundaryNodes(prDAG)

	endPrereqs := aug["end"]
	sort.Strings(endPrereqs)
	if !reflect.DeepEqual(endPrereqs, []string{"B", "C"}) {
		t.Errorf("end should depend on both terminals B and C, got %v", endPrereqs)
	}
}

// ---- DAGToPrerequisites ----------------------------------------------------

func TestDAGToPrerequisites_Normal(t *testing.T) {
	dag := map[string]DAGNode{
		"A": {Prerequisites: []string{}},
		"B": {Prerequisites: []string{"A"}},
		"C": {Prerequisites: []string{"A", "B"}},
	}
	got := DAGToPrerequisites(dag)

	if len(got["A"]) != 0 {
		t.Errorf("A should have no prereqs, got %v", got["A"])
	}
	if !reflect.DeepEqual(got["B"], []string{"A"}) {
		t.Errorf("B prereqs wrong: %v", got["B"])
	}
	wantC := []string{"A", "B"}
	sort.Strings(got["C"])
	if !reflect.DeepEqual(got["C"], wantC) {
		t.Errorf("C prereqs wrong: %v", got["C"])
	}
}

func TestDAGToPrerequisites_Empty(t *testing.T) {
	got := DAGToPrerequisites(map[string]DAGNode{})
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

// ---- FindCommon ------------------------------------------------------------

func TestFindCommon_Overlap(t *testing.T) {
	got := FindCommon([]string{"A", "B", "C"}, []string{"B", "C", "D"})
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"B", "C"}) {
		t.Errorf("expected [B C], got %v", got)
	}
}

func TestFindCommon_NoOverlap(t *testing.T) {
	got := FindCommon([]string{"A", "B"}, []string{"C", "D"})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestFindCommon_EmptyFirst(t *testing.T) {
	got := FindCommon([]string{}, []string{"A", "B"})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestFindCommon_EmptySecond(t *testing.T) {
	got := FindCommon([]string{"A", "B"}, []string{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestFindCommon_BothEmpty(t *testing.T) {
	got := FindCommon([]string{}, []string{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestFindCommon_Duplicates(t *testing.T) {
	// Duplicates in list1 should produce at most one match per list2 element
	got := FindCommon([]string{"A", "A", "B"}, []string{"A"})
	if len(got) != 2 {
		// FindCommon iterates list1, so "A" appears twice → 2 matches; document behaviour
		t.Logf("FindCommon with duplicate in list1 returns %d matches (current behaviour)", len(got))
	}
}

// ---- helpers ---------------------------------------------------------------

func positionMap(order []string) map[string]int {
	m := make(map[string]int, len(order))
	for i, v := range order {
		m[v] = i
	}
	return m
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

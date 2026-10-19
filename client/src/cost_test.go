package src

import (
	"math"
	"testing"
)

// ---- Haversine -------------------------------------------------------------

func TestHaversine_SamePoint(t *testing.T) {
	d := Haversine([]float64{48.8566, 2.3522}, []float64{48.8566, 2.3522})
	if math.Abs(d) > 1e-9 {
		t.Errorf("same point should be ~0, got %g", d)
	}
}

func TestHaversine_EquatorialOneDegree(t *testing.T) {
	// One degree of longitude along the equator ≈ 111.195 km
	d := Haversine([]float64{0, 0}, []float64{0, 1})
	if math.Abs(d-111.195) > 0.5 {
		t.Errorf("expected ~111.2 km, got %f", d)
	}
}

func TestHaversine_Symmetric(t *testing.T) {
	p1 := []float64{48.8566, 2.3522}  // Paris
	p2 := []float64{51.5074, -0.1278} // London
	d1 := Haversine(p1, p2)
	d2 := Haversine(p2, p1)
	if math.Abs(d1-d2) > 0.001 {
		t.Errorf("Haversine should be symmetric: %f vs %f", d1, d2)
	}
}

func TestHaversine_ParisLondon(t *testing.T) {
	// Paris ↔ London ≈ 341 km
	d := Haversine([]float64{48.8566, 2.3522}, []float64{51.5074, -0.1278})
	if math.Abs(d-341) > 5 {
		t.Errorf("Paris-London expected ~341 km, got %f", d)
	}
}

func TestHaversine_AlwaysNonNegative(t *testing.T) {
	cases := [][4]float64{
		{-33.8688, 151.2093, 35.6762, 139.6503}, // Sydney → Tokyo
		{40.7128, -74.0060, 55.7558, 37.6173},   // New York → Moscow
		{0, 0, 0, 180},                          // equatorial half-globe
	}
	for _, c := range cases {
		d := Haversine([]float64{c[0], c[1]}, []float64{c[2], c[3]})
		if d < 0 {
			t.Errorf("distance should never be negative, got %f", d)
		}
	}
}

// ---- CreateCostMatrixFromResults -------------------------------------------

func emptyTP() *responsesTP {
	return &responsesTP{}
}

func TestCreateCostMatrix_OriginEndGetCostOne(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "origin_actor", Matches: []string{"origin"}, Location: []float64{0, 0}},
		{ContactName: "end_actor", Matches: []string{"end"}, Location: []float64{1, 1}},
	}}
	matrix, err := CreateCostMatrixFromResults(rc, emptyTP())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matrix["origin_actor"]["origin"] != 1 {
		t.Errorf("origin actor should have cost 1 for origin, got %f", matrix["origin_actor"]["origin"])
	}
	if matrix["end_actor"]["end"] != 1 {
		t.Errorf("end actor should have cost 1 for end, got %f", matrix["end_actor"]["end"])
	}
}

func TestCreateCostMatrix_TechProviderCosts(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "ActorA", Matches: []string{"Cleaning", "Reassembly"}, Location: []float64{0, 0}},
	}}
	tp := &responsesTP{Responses: []techProviderResponse{
		{uid: "ActorA", Processes: []process{
			{ProcessName: "Cleaning", Cost: 50},
			{ProcessName: "Reassembly", Cost: 80},
		}},
	}}
	matrix, err := CreateCostMatrixFromResults(rc, tp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matrix["ActorA"]["Cleaning"] != 50 {
		t.Errorf("expected Cleaning=50, got %f", matrix["ActorA"]["Cleaning"])
	}
	if matrix["ActorA"]["Reassembly"] != 80 {
		t.Errorf("expected Reassembly=80, got %f", matrix["ActorA"]["Reassembly"])
	}
}

func TestCreateCostMatrix_EmptyMatchesNoPanic(t *testing.T) {
	// Before the fix, this would panic with index out of range on Matches[0]
	rc := ResultCollection{Results: []Result{
		{ContactName: "ghost", Matches: []string{}, Location: []float64{0, 0}},
	}}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked on empty Matches: %v", r)
		}
	}()
	_, err := CreateCostMatrixFromResults(rc, emptyTP())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCostMatrix_EmptyResultCollection(t *testing.T) {
	rc := ResultCollection{}
	matrix, err := CreateCostMatrixFromResults(rc, emptyTP())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matrix) != 0 {
		t.Errorf("expected empty matrix, got %v", matrix)
	}
}

func TestCreateCostMatrix_RedesignCost(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "designer", Matches: []string{"Redesign"}, Location: []float64{0, 0}},
	}}
	matrix, err := CreateCostMatrixFromResults(rc, emptyTP())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matrix["designer"]["Redesign"] != 100 {
		t.Errorf("Redesign should have hardcoded cost 100, got %f", matrix["designer"]["Redesign"])
	}
}

// ---- CreateEnergyMatrixFromResults ------------------------------------------

func TestCreateEnergyMatrix_TechProviderEnergy(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "ActorA", Matches: []string{"Cleaning", "Reassembly"}, Location: []float64{0, 0}},
	}}
	tp := &responsesTP{Responses: []techProviderResponse{
		{uid: "ActorA", Processes: []process{
			{ProcessName: "Cleaning", Cost: 50, Energy: 12.5},
			{ProcessName: "Reassembly", Cost: 80, Energy: 7.25},
		}},
	}}
	matrix, err := CreateEnergyMatrixFromResults(rc, tp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matrix["ActorA"]["Cleaning"] != 12.5 {
		t.Errorf("expected Cleaning energy=12.5, got %f", matrix["ActorA"]["Cleaning"])
	}
	if matrix["ActorA"]["Reassembly"] != 7.25 {
		t.Errorf("expected Reassembly energy=7.25, got %f", matrix["ActorA"]["Reassembly"])
	}
}

func TestCreateEnergyMatrix_OriginEndRedesignAreZero(t *testing.T) {
	// Unlike cost, boundary/Redesign entries don't need a nonzero sentinel —
	// the optimizer's capability gating reads CostMatrix only.
	rc := ResultCollection{Results: []Result{
		{ContactName: "origin_actor", Matches: []string{"origin"}, Location: []float64{0, 0}},
		{ContactName: "end_actor", Matches: []string{"end"}, Location: []float64{1, 1}},
		{ContactName: "designer", Matches: []string{"Redesign"}, Location: []float64{0, 0}},
	}}
	matrix, err := CreateEnergyMatrixFromResults(rc, emptyTP())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matrix["origin_actor"]["origin"] != 0 {
		t.Errorf("origin actor should have energy 0, got %f", matrix["origin_actor"]["origin"])
	}
	if matrix["end_actor"]["end"] != 0 {
		t.Errorf("end actor should have energy 0, got %f", matrix["end_actor"]["end"])
	}
	if matrix["designer"]["Redesign"] != 0 {
		t.Errorf("Redesign should have energy 0, got %f", matrix["designer"]["Redesign"])
	}
}

func TestCreateEnergyMatrix_EmptyMatchesNoPanic(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "ghost", Matches: []string{}, Location: []float64{0, 0}},
	}}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked on empty Matches: %v", r)
		}
	}()
	_, err := CreateEnergyMatrixFromResults(rc, emptyTP())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- DistanceMatrixConstructor ---------------------------------------------

func TestDistanceMatrix_SelfDistanceIsZero(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "A", Location: []float64{48.8566, 2.3522}},
	}}
	dm := DistanceMatrixConstructor(rc, nil)
	if dm["A"]["A"] != 0 {
		t.Errorf("self-distance should be 0, got %f", dm["A"]["A"])
	}
}

func TestDistanceMatrix_Symmetric(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "Paris", Location: []float64{48.8566, 2.3522}},
		{ContactName: "London", Location: []float64{51.5074, -0.1278}},
	}}
	dm := DistanceMatrixConstructor(rc, nil)
	if math.Abs(dm["Paris"]["London"]-dm["London"]["Paris"]) > 0.001 {
		t.Errorf("distance matrix should be symmetric: %f vs %f",
			dm["Paris"]["London"], dm["London"]["Paris"])
	}
}

func TestDistanceMatrix_SpinRedesignAlwaysZero(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "SPIN_REDESIGN", Location: []float64{48.8566, 2.3522}},
		{ContactName: "ActorA", Location: []float64{51.5074, -0.1278}},
	}}
	dm := DistanceMatrixConstructor(rc, nil)
	if dm["SPIN_REDESIGN"]["ActorA"] != 0 {
		t.Errorf("SPIN_REDESIGN → any should be 0, got %f", dm["SPIN_REDESIGN"]["ActorA"])
	}
	if dm["ActorA"]["SPIN_REDESIGN"] != 0 {
		t.Errorf("any → SPIN_REDESIGN should be 0, got %f", dm["ActorA"]["SPIN_REDESIGN"])
	}
}

func TestDistanceMatrix_AllActorsPresent(t *testing.T) {
	rc := ResultCollection{Results: []Result{
		{ContactName: "A", Location: []float64{0, 0}},
		{ContactName: "B", Location: []float64{0, 1}},
		{ContactName: "C", Location: []float64{1, 0}},
	}}
	dm := DistanceMatrixConstructor(rc, nil)
	for _, from := range []string{"A", "B", "C"} {
		for _, to := range []string{"A", "B", "C"} {
			if _, ok := dm[from][to]; !ok {
				t.Errorf("missing entry dm[%s][%s]", from, to)
			}
		}
	}
}

func TestDistanceMatrix_KnownDistance(t *testing.T) {
	// One degree of longitude on the equator ≈ 111.2 km
	rc := ResultCollection{Results: []Result{
		{ContactName: "origin", Location: []float64{0, 0}},
		{ContactName: "dest", Location: []float64{0, 1}},
	}}
	dm := DistanceMatrixConstructor(rc, nil)
	if math.Abs(dm["origin"]["dest"]-111.195) > 0.5 {
		t.Errorf("expected ~111.2 km, got %f", dm["origin"]["dest"])
	}
}

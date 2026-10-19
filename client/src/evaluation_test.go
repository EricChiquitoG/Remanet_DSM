package src

// Evaluation test for the DSM optimization pipeline.
//
// What it does:
//   Builds cost and distance matrices using the same Go functions as production
//   (CreateCostMatrixFromResults, DistanceMatrixConstructor), then sends the
//   problem to the optimizer container via gRPC and records wall-clock time.
//
// Prerequisites:
//   docker compose up optimizer          # only the optimizer is needed
//
// Run:
//   OPTIMIZER_ADDR=localhost:50060 go test -run TestEvaluation -v -timeout 120m

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
)

// ── Embedded actor catalogue ──────────────────────────────────────────────────
// Mirrors directory.json and the AAS mock data values.

type evalActorDef struct {
	name      string
	lat       float64
	lon       float64
	cost      float64 // cost_per_hour from AAS
	processes []string
}

var evalActorPool = []evalActorDef{
	// General remanufacturers — disassembly, assembly, and handling steps
	{"Dissasembler0", 65.584816, 22.156704, 58.61, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler1", 21.584816, 2.156704, 45.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler2", 65.584816, 13.220000, 52.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler3", 59.913869, 10.752245, 68.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler4", 48.137154, 11.576124, 55.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler5", 41.385064, 2.173403, 88.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler6", 52.229676, 21.012229, 35.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	{"Dissasembler7", 45.464664, 9.188540, 62.00, []string{"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"}},
	// Precision machinists — shaft and stator work
	{"Shaft0", 41.220000, 22.154000, 54.34, []string{"Grinding", "Turning", "StatorRewind"}},
	{"Shaft1", 42.584816, 21.156704, 70.00, []string{"Grinding", "Turning", "StatorRewind"}},
	{"Shaft2", 69.584816, 22.156704, 48.00, []string{"Grinding", "Turning", "StatorRewind"}},
	{"Shaft3", 45.748234, 4.846710, 65.00, []string{"Grinding", "Turning", "StatorRewind"}},
	{"Shaft4", 50.075538, 14.437800, 52.00, []string{"Grinding", "Turning", "StatorRewind"}},
	{"Shaft5", 52.370216, 4.895168, 32.00, []string{"Grinding", "Turning", "StatorRewind"}},
	{"Shaft6", 60.169857, 24.938379, 75.00, []string{"Grinding", "Turning", "StatorRewind"}},
	// Lamination manufacturers — rotor core stamping
	{"Lamination0", 55.584816, 9.100000, 54.10, []string{"Punching"}},
	{"Lamination1", 60.100000, 19.100000, 70.00, []string{"Punching"}},
	{"Lamination2", 40.220000, 29.100000, 45.00, []string{"Punching"}},
	{"Lamination3", 55.676097, 12.568337, 40.00, []string{"Punching"}},
	{"Lamination4", 38.722252, -9.139337, 30.00, []string{"Punching"}},
}

// ── DAG definitions ───────────────────────────────────────────────────────────
// {process: prerequisites}  — matches the DAGNode.Prerequisites field format.
// origin/end are added by augmentWithBoundaryNodes.

type dagDef = map[string][]string

var evalDAGs = map[string]dagDef{
	"P_2proc": {
		"Cleaning":      {},
		"Certification": {"Cleaning"},
	},
	"P_4proc": {
		"Cleaning":      {},
		"Disassembly":   {"Cleaning"},
		"Reassembly":    {"Disassembly"},
		"Certification": {"Reassembly"},
	},
	"P_5proc": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"Reassembly":          {"BearingSubstitution"},
		"Certification":       {"Reassembly"},
	},
	"P_7proc": {
		"Cleaning":      {},
		"Disassembly":   {"Cleaning"},
		"Turning":       {"Disassembly"},
		"Punching":      {"Disassembly"},
		"Grinding":      {"Turning"},
		"Reassembly":    {"Grinding", "Punching"},
		"Certification": {"Reassembly"},
	},
	"P_9proc": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"Redesign":            {"Disassembly"},
		"Punching":            {"Redesign"},
		"Turning":             {"Redesign"},
		"Grinding":            {"Turning"},
		"Reassembly":          {"BearingSubstitution", "Punching", "Grinding"},
		"Certification":       {"Reassembly"},
	},

	// Strategy scenarios — aligned with paper S1 and S2, plus illustrative alternatives.
	//
	// S1 (Refurbishment): three parallel branches after Disassembly — bearing
	// substitution, stator rewind, and shaft precision (Turning→Grinding) — all
	// converging at Reassembly. Rotor core is retained (no Redesign/Punching).
	"S1_refurbishment": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"StatorRewind":        {"Disassembly"},
		"Turning":             {"Disassembly"},
		"Grinding":            {"Turning"},
		"Reassembly":          {"BearingSubstitution", "StatorRewind", "Grinding"},
		"Certification":       {"Reassembly"},
	},
	// S2 (Functional Upgrade): S1 branches plus a Redesign branch from Disassembly
	// that drives both Punching (new rotor laminations) and Turning→Grinding (shaft
	// rework to redesigned tolerances). StatorRewind remains independent of Redesign.
	"S2_upgrade": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"Redesign":            {"Disassembly"},
		"Punching":            {"Redesign"},
		"StatorRewind":        {"Disassembly"},
		"Turning":             {"Redesign"},
		"Grinding":            {"Turning"},
		"Reassembly":          {"BearingSubstitution", "Punching", "StatorRewind", "Grinding"},
		"Certification":       {"Reassembly"},
	},
	// S3 (Shaft Precision Only): illustrative mechanical-only path — no stator or
	// rotor work; exercises only general remanufacturers and precision machinists.
	"S3_shaft": {
		"Cleaning":      {},
		"Disassembly":   {"Cleaning"},
		"Turning":       {"Disassembly"},
		"Grinding":      {"Turning"},
		"Reassembly":    {"Grinding"},
		"Certification": {"Reassembly"},
	},
	// S4 (Rotor Core Renewal): new lamination stack only — no stator or shaft work;
	// exercises general remanufacturers, lamination manufacturers, and the redesign node.
	"S4_rotor": {
		"Cleaning":      {},
		"Disassembly":   {"Cleaning"},
		"Redesign":      {"Disassembly"},
		"Punching":      {"Redesign"},
		"Reassembly":    {"Punching"},
		"Certification": {"Reassembly"},
	},

	"B_linear": {
		"Cleaning":      {},
		"Disassembly":   {"Cleaning"},
		"Turning":       {"Disassembly"},
		"Punching":      {"Turning"},
		"Grinding":      {"Punching"},
		"Reassembly":    {"Grinding"},
		"Certification": {"Reassembly"},
	},
	"B_fork_join": {
		"Cleaning":      {},
		"Disassembly":   {"Cleaning"},
		"Turning":       {"Cleaning"},
		"Punching":      {"Cleaning"},
		"Reassembly":    {"Disassembly", "Turning", "Punching"},
		"Certification": {"Reassembly"},
	},
	"B_double_fork": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"Grinding":            {"Disassembly"},
		"BearingSubstitution": {"Cleaning"},
		"Turning":             {"BearingSubstitution"},
		"Punching":            {"BearingSubstitution"},
		"Reassembly":          {"Grinding", "Turning", "Punching"},
		"Certification":       {"Reassembly"},
	},
	"B_wide_fanout": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Cleaning"},
		"Turning":             {"Cleaning"},
		"Punching":            {"Cleaning"},
		"Grinding":            {"Cleaning"},
		"Reassembly":          {"Disassembly", "BearingSubstitution", "Turning", "Punching", "Grinding"},
		"Certification":       {"Reassembly"},
	},

	// T: topology comparison — each fork-join paired with its flattened sequential equivalent.
	// Fork variants use the same DAG keys as S1/S2 where applicable.

	// T_1branch: 1 additional parallel branch (2 paths) — minimal fork-join.
	"T_1branch_fork": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"StatorRewind":        {"Disassembly"},
		"Reassembly":          {"BearingSubstitution", "StatorRewind"},
		"Certification":       {"Reassembly"},
	},
	"T_1branch_flat": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"StatorRewind":        {"BearingSubstitution"},
		"Reassembly":          {"StatorRewind"},
		"Certification":       {"Reassembly"},
	},

	// T_2branch: 2 additional parallel branches (3 paths).
	"T_2branch_fork": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"StatorRewind":        {"Disassembly"},
		"Grinding":            {"Disassembly"},
		"Reassembly":          {"BearingSubstitution", "StatorRewind", "Grinding"},
		"Certification":       {"Reassembly"},
	},
	"T_2branch_flat": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"StatorRewind":        {"BearingSubstitution"},
		"Grinding":            {"StatorRewind"},
		"Reassembly":          {"Grinding"},
		"Certification":       {"Reassembly"},
	},

	// T_S1_flat: S1 Refurbishment with all branches serialized.
	// (fork variant reuses S1_refurbishment)
	"T_S1_flat": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"StatorRewind":        {"BearingSubstitution"},
		"Turning":             {"StatorRewind"},
		"Grinding":            {"Turning"},
		"Reassembly":          {"Grinding"},
		"Certification":       {"Reassembly"},
	},

	// T_S2_flat: S2 Functional Upgrade with all branches serialized.
	// (fork variant reuses S2_upgrade)
	"T_S2_flat": {
		"Cleaning":            {},
		"Disassembly":         {"Cleaning"},
		"BearingSubstitution": {"Disassembly"},
		"Redesign":            {"BearingSubstitution"},
		"Punching":            {"Redesign"},
		"StatorRewind":        {"Punching"},
		"Turning":             {"StatorRewind"},
		"Grinding":            {"Turning"},
		"Reassembly":          {"Grinding"},
		"Certification":       {"Reassembly"},
	},
}

// ── Scenario definitions ──────────────────────────────────────────────────────

type evalScenario struct {
	id          string
	dim         string
	dagKey      string
	maxActors   int // 0 = use all eligible actors
	nRoutes     int
	hasRedesign bool
}

var evalScenarios = []evalScenario{
	// A: actor count — how solution count/quality scales with pool size
	{"A_1actor", "A", "P_5proc", 1, 1, false},
	{"A_2actors", "A", "P_5proc", 2, 1, false},
	{"A_4actors", "A", "P_5proc", 4, 1, false},
	{"A_6actors", "A", "P_5proc", 6, 1, false},
	{"A_8actors", "A", "P_5proc", 0, 1, false},

	// P: process count — how time scales with workflow depth / complexity
	{"P_2proc", "P", "P_2proc", 0, 1, false},
	{"P_4proc", "P", "P_4proc", 0, 1, false},
	{"P_5proc", "P", "P_5proc", 0, 1, false},
	{"P_7proc", "P", "P_7proc", 0, 1, false},
	{"P_9proc", "P", "P_9proc", 0, 1, true},

	// R: route count — parallel goroutine strategies
	{"R_1route", "R", "P_5proc", 0, 1, false},
	{"R_2routes", "R", "P_5proc", 0, 2, false},
	{"R_4routes", "R", "P_5proc", 0, 4, false},
	{"R_8routes", "R", "P_5proc", 0, 8, false},

	// B: branch topology — how DAG shape affects optimization
	{"B_linear", "B", "B_linear", 0, 1, false},
	{"B_fork_join", "B", "B_fork_join", 0, 1, false},
	{"B_double_fork", "B", "B_double_fork", 0, 1, false},
	{"B_wide_fanout", "B", "B_wide_fanout", 0, 1, false},

	// S: paper strategies — S1 (fork-join) and S2 (depth-2 branching) from the use case
	{"S1_refurbishment", "S", "S1_refurbishment", 0, 1, false},
	{"S2_functional_upgrade", "S", "S2_upgrade", 0, 1, true},

	// T: topology — each fork-join paired with its flattened sequential counterpart
	{"T_1branch_fork", "T", "T_1branch_fork", 0, 1, false},
	{"T_1branch_flat", "T", "T_1branch_flat", 0, 1, false},
	{"T_2branch_fork", "T", "T_2branch_fork", 0, 1, false},
	{"T_2branch_flat", "T", "T_2branch_flat", 0, 1, false},
	{"T_S1_fork", "T", "S1_refurbishment", 0, 1, false},
	{"T_S1_flat", "T", "T_S1_flat", 0, 1, false},
	{"T_S2_fork", "T", "S2_upgrade", 0, 1, true},
	{"T_S2_flat", "T", "T_S2_flat", 0, 1, true},
}

// ── Matrix construction helpers ───────────────────────────────────────────────

// buildActorPool selects actors whose process capabilities intersect the DAG
// (capped to maxActors eligible actors), adds boundary and optional redesign
// actors, and returns the ResultCollection and responsesTP that the production
// matrix functions expect.
func buildActorPool(dag dagDef, maxActors int, hasRedesign bool, startingPoint []float64) (ResultCollection, responsesTP) {
	dagProcesses := make(map[string]bool)
	for proc := range dag {
		dagProcesses[proc] = true
	}

	rc := ResultCollection{}
	tp := responsesTP{}

	rc.Results = append(rc.Results, Result{
		ContactName: "origin",
		Matches:     []string{"origin"},
		Location:    startingPoint,
	})
	rc.Results = append(rc.Results, Result{
		ContactName: "end_destination",
		Matches:     []string{"end"},
		Location:    startingPoint,
	})

	count := 0
	for _, actor := range evalActorPool {
		relevant := false
		for _, p := range actor.processes {
			if dagProcesses[p] {
				relevant = true
				break
			}
		}
		if !relevant {
			continue
		}
		if maxActors > 0 && count >= maxActors {
			continue
		}
		count++

		rc.Results = append(rc.Results, Result{
			ContactName: actor.name,
			Matches:     actor.processes,
			Location:    []float64{actor.lat, actor.lon},
		})

		procs := make([]process, 0, len(actor.processes))
		for _, p := range actor.processes {
			procs = append(procs, process{
				ProcessID:   fmt.Sprintf("%s-%s", actor.name, p),
				ProcessName: p,
				Cost:        actor.cost,
			})
		}
		tp.Responses = append(tp.Responses, techProviderResponse{
			uid:       actor.name,
			location:  []float64{actor.lat, actor.lon},
			Processes: procs,
		})
	}

	if hasRedesign {
		rc.Results = append(rc.Results, Result{
			ContactName: "SPIN_REDESIGN",
			Matches:     []string{"Redesign"},
			Location:    startingPoint,
		})
	}

	return rc, tp
}

// buildProblemJSON constructs the JSON payload for the optimizer's gRPC Optimize call.
func buildProblemJSON(dag dagDef, rc ResultCollection, tp responsesTP) (map[string]interface{}, error) {
	costMatrix, err := CreateCostMatrixFromResults(rc, &tp)
	if err != nil {
		return nil, err
	}
	energyMatrix, err := CreateEnergyMatrixFromResults(rc, &tp)
	if err != nil {
		return nil, err
	}
	distMatrix := DistanceMatrixConstructor(rc, nil)
	augmented := augmentWithBoundaryNodes(dag)
	return exportToJson(nil, costMatrix, energyMatrix, distMatrix, augmented), nil
}

// ── gRPC call (own timeout, no Redis) ────────────────────────────────────────

func evalOptimize(client pb.SubmissionServiceClient, jsonData map[string]interface{}, timeout time.Duration) (*pb.OptimizationResponse, time.Duration, error) {
	data, err := json.Marshal(jsonData)
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := &pb.OptimizationRequest{JsonProblemData: string(data)}
	t0 := time.Now()
	resp, err := client.Optimize(ctx, req)
	return resp, time.Since(t0), err
}

// ── CSV output ────────────────────────────────────────────────────────────────

type evalRow struct {
	Dimension         string
	ScenarioID        string
	Run               int
	WallTimeS         float64
	NSolutions        int
	NActorsTotal      int
	NProcesses        int
	NRoutes           int
	SolNumber         int
	TransportCost     float64
	ManufacturingCost float64
	Assignments       string
	Error             string
}

func writeCSVRows(w *csv.Writer, rows []evalRow) {
	for _, r := range rows {
		_ = w.Write([]string{
			r.Dimension,
			r.ScenarioID,
			fmt.Sprintf("%d", r.Run),
			fmt.Sprintf("%.3f", r.WallTimeS),
			fmt.Sprintf("%d", r.NSolutions),
			fmt.Sprintf("%d", r.NActorsTotal),
			fmt.Sprintf("%d", r.NProcesses),
			fmt.Sprintf("%d", r.NRoutes),
			fmt.Sprintf("%d", r.SolNumber),
			fmt.Sprintf("%.4f", r.TransportCost),
			fmt.Sprintf("%.4f", r.ManufacturingCost),
			r.Assignments,
			r.Error,
		})
	}
	w.Flush()
}

// ── Main test ─────────────────────────────────────────────────────────────────

func TestEvaluation(t *testing.T) {
	addr := os.Getenv("OPTIMIZER_ADDR")
	if addr == "" {
		t.Skip("OPTIMIZER_ADDR not set — skipping evaluation (set to localhost:50060)")
	}

	const (
		runsPerScenario = 3
		optTimeout      = 10 * time.Minute
	)
	startingPoint := []float64{20.464664, 9.18854}

	// Connect to optimizer
	conn, err := connectToServer(addr)
	if err != nil {
		t.Fatalf("cannot connect to optimizer at %s: %v", addr, err)
	}
	defer conn.Close()
	client := pb.NewSubmissionServiceClient(conn)

	// Output CSV
	outDir := filepath.Join("..", "..", "evaluation", "results")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Logf("warning: could not create %s, writing to current dir: %v", outDir, err)
		outDir = "."
	}
	ts := time.Now().Format("20060102_150405")
	csvPath := filepath.Join(outDir, fmt.Sprintf("evaluation_%s.csv", ts))
	f, err := os.Create(csvPath)
	if err != nil {
		t.Fatalf("cannot create CSV: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{
		"dimension", "scenario_id", "run",
		"wall_time_s", "n_solutions", "n_actors_total", "n_processes", "n_routes",
		"sol_number", "transport_cost", "manufacturing_cost", "assignments", "error",
	})
	w.Flush()

	t.Logf("Writing results to %s", csvPath)

	for _, sc := range evalScenarios {
		dag := evalDAGs[sc.dagKey]
		rc, tp := buildActorPool(dag, sc.maxActors, sc.hasRedesign, startingPoint)

		// Count technical actors (not origin/end)
		nActors := 0
		for _, r := range rc.Results {
			if r.Matches[0] != "origin" && r.Matches[0] != "end" {
				nActors++
			}
		}
		nProc := len(dag)

		t.Logf("▶ [%s] %s  (%d proc, %d actors, %d route(s))",
			sc.dim, sc.id, nProc, nActors, sc.nRoutes)

		jsonData, err := buildProblemJSON(dag, rc, tp)
		if err != nil {
			t.Errorf("%s: buildProblemJSON failed: %v", sc.id, err)
			continue
		}

		for run := 1; run <= runsPerScenario; run++ {
			var rows []evalRow

			if sc.nRoutes == 1 {
				resp, elapsed, err := evalOptimize(client, jsonData, optTimeout)
				rows = collectRows(sc, run, elapsed, nActors, nProc, resp, err)
			} else {
				// N parallel routes — each gets its own goroutine, mirrors Go server behaviour
				type routeResult struct {
					resp    *pb.OptimizationResponse
					elapsed time.Duration
					err     error
				}
				results := make([]routeResult, sc.nRoutes)
				var wg sync.WaitGroup
				t0 := time.Now()
				for i := 0; i < sc.nRoutes; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						resp, elapsed, err := evalOptimize(client, jsonData, optTimeout)
						results[idx] = routeResult{resp, elapsed, err}
					}(i)
				}
				wg.Wait()
				wallTime := time.Since(t0)

				// Use the first route's solutions for the CSV solution rows; log all timings
				logParts := make([]string, sc.nRoutes)
				for i, r := range results {
					logParts[i] = fmt.Sprintf("%.2fs", r.elapsed.Seconds())
				}

				rows = collectRows(sc, run, wallTime, nActors, nProc, results[0].resp, results[0].err)
				// Override wall time in all rows with the actual concurrent wall time
				for i := range rows {
					rows[i].WallTimeS = wallTime.Seconds()
				}
				t.Logf("  ✓ run %d: wall=%.2fs  routes=[%v]  %d solutions",
					run, wallTime.Seconds(), logParts, rows[0].NSolutions)
			}

			if sc.nRoutes == 1 {
				t.Logf("  ✓ run %d: %.2fs  %d solutions", run, rows[0].WallTimeS, rows[0].NSolutions)
			}
			writeCSVRows(w, rows)
		}
	}

	t.Logf("Evaluation complete. Results: %s", csvPath)
}

// collectRows converts an optimizer response into CSV rows.
func collectRows(sc evalScenario, run int, elapsed time.Duration, nActors, nProc int,
	resp *pb.OptimizationResponse, callErr error) []evalRow {

	base := evalRow{
		Dimension:    sc.dim,
		ScenarioID:   sc.id,
		Run:          run,
		WallTimeS:    elapsed.Seconds(),
		NActorsTotal: nActors,
		NProcesses:   nProc,
		NRoutes:      sc.nRoutes,
	}

	if callErr != nil {
		base.Error = callErr.Error()
		base.NSolutions = -1
		return []evalRow{base}
	}

	results, err := processOptimizationResponse(resp)
	if err != nil {
		base.Error = err.Error()
		base.NSolutions = -1
		return []evalRow{base}
	}

	base.NSolutions = len(results.Solutions)
	if len(results.Solutions) == 0 {
		return []evalRow{base}
	}

	rows := make([]evalRow, len(results.Solutions))
	for i, sol := range results.Solutions {
		assigned, _ := json.Marshal(sol.Assignments)
		r := base
		r.SolNumber = sol.SolNumber
		r.TransportCost = sol.TransportCost
		r.ManufacturingCost = sol.ManufacturingCost
		r.Assignments = string(assigned)
		rows[i] = r
	}
	return rows
}

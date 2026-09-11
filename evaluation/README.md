# DSM Evaluation Scenarios

All scenarios POST to `http://localhost:8080/process_directory` with `Content-Type: application/json`.

---

## Dimension 1 — Actor count (combination space)

**Scenario file:** `scenarios/P3_5proc_linearA_baseline.json` (fixed — 5 linear Class A steps)  
**Variable:** swap `client/data/directory.json` with a file from `directories/` before each run.

| Run | Directory file            | Class A actors | Theoretical combinations (5 steps) |
|-----|---------------------------|----------------|-------------------------------------|
| A1  | directory_1actor.json     | 1              | 1¹ = 1                             |
| A2  | directory_2actors.json    | 2              | 2⁵ = 32                            |
| A3  | directory_4actors.json    | 4              | 4⁵ = 1 024                         |
| A4  | directory_6actors.json    | 6              | 6⁵ = 7 776                         |
| A5  | `client/data/directory.json` (full) | 8   | 8⁵ = 32 768                        |

> After swapping the directory file, restart the Docker client service so the new file is loaded:
> `docker compose restart client`

---

## Dimension 2 — Process count (workflow depth)

All use the full directory (8A / 7C / 5D actors), 1 route.

| Scenario | File                          | Processes | Classes     | Approx. combinations          |
|----------|-------------------------------|-----------|-------------|-------------------------------|
| P1       | P1_2proc_linearA.json         | 2         | A           | 8² = 64                       |
| P2       | P2_4proc_linearA.json         | 4         | A           | 8⁴ = 4 096                    |
| P3       | P3_5proc_linearA_baseline.json| 5         | A           | 8⁵ = 32 768  ← baseline       |
| P4       | P4_7proc_mixed.json           | 7         | A + C + D   | 8⁵ × 7² × 5 ≈ 1 M            |
| P5       | P5_9proc_full.json            | 9         | A+C+D+Redesign | 8⁶ × 7² × 5 × 1 ≈ 8 M     |

---

## Dimension 3 — Strategy count (goroutine parallelism)

All routes contain the same 5-step linear Class A DAG (same search space per route). Wall-clock time should plateau as goroutines saturate available CPU cores.

| Scenario | File                  | Routes (goroutines) |
|----------|-----------------------|---------------------|
| R1       | R1_single_route.json  | 1                   |
| R2       | R2_dual_routes.json   | 2                   |
| R4       | R4_quad_routes.json   | 4                   |
| R8       | R8_octo_routes.json   | 8                   |

---

## Dimension 4 — Branch topology (DAG structure)

Same process mix and actor pool, only the prerequisite structure changes. Isolates the effect of parallel branches on solution space and transport cost computation.

| Scenario | File                      | Nodes | Shape                                                      |
|----------|---------------------------|-------|------------------------------------------------------------|
| B1       | B1_linear_7step.json      | 7     | Linear chain: A→A→C→D→C→A→A                               |
| B2       | B2_single_fork_join.json  | 6     | Single fork: A→{A, C, D}→A→A                              |
| B3       | B3_double_fork.json       | 8     | Two parallel 2-step chains merge at Reassembly             |
| B4       | B4_wide_fanout.json       | 8     | Fan-out of 5 parallel processes between Cleaning→Reassembly|

> B1 and B4 have the same 7 processes and the same per-process actor pools — combination count is identical. What changes is the transport cost structure in the optimizer, which affects which solutions dominate.

---

## Determinism check

Run any single scenario 3× with no changes in between. The `seed=3` in the NSGA2 optimizer makes results deterministic. All three runs must return identical `processSequence`, `assignments`, `transportCost`, and `manufacturingCost` values.

```bash
# Example — run P3 three times and diff the results
curl -s -X POST http://localhost:8080/process_directory \
  -H "Content-Type: application/json" \
  -d @evaluation/scenarios/P3_5proc_linearA_baseline.json > run1.json

curl -s -X POST http://localhost:8080/process_directory \
  -H "Content-Type: application/json" \
  -d @evaluation/scenarios/P3_5proc_linearA_baseline.json > run2.json

diff run1.json run2.json   # should be empty
```

#!/usr/bin/env python3
"""
Sanity-check for opt_DAG.py.

Defines a minimal 2-actor / 2-step problem where every valid assignment can be
enumerated by hand.  Computes expected transport cost, manufacturing cost,
makespan, and energy for each assignment, identifies the Pareto front, then
runs NSGA2 on the same problem and checks whether its output matches.
"""
import sys, os, json, math, itertools
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import opt_DAG as opt
import numpy as np
from pymoo.algorithms.moo.nsga2 import NSGA2
from pymoo.operators.crossover.ux import UniformCrossover
from pymoo.operators.mutation.pm import PolynomialMutation
from pymoo.optimize import minimize
from pymoo.termination import get_termination


def haversine(lat1, lon1, lat2, lon2):
    R = 6371
    r = math.pi / 180
    la1, lo1, la2, lo2 = lat1*r, lon1*r, lat2*r, lon2*r
    dlat, dlon = la2-la1, lo2-lo1
    a = math.sin(dlat/2)**2 + math.cos(la1)*math.cos(la2)*math.sin(dlon/2)**2
    return R * 2 * math.atan2(math.sqrt(a), math.sqrt(1-a))


# ---------------------------------------------------------------------------
# Problem definition
# ---------------------------------------------------------------------------
# Locations [lat, lon]
locs = {
    'ActorA':   (0.0, 1.0),   # equidistant between origin and end_dest
    'ActorB':   (5.0, 0.0),   # far away → high transport, low cost
    'end_dest': (0.0, 2.0),
    'origin':   (0.0, 0.0),
}

# Cost matrix:  ActorA is expensive but centrally located,
#               ActorB is cheap but far → creates genuine transport/cost trade-off
cost_matrix = {
    'ActorA':   {'Cleaning': 200.0, 'Reassembly': 200.0},
    'ActorB':   {'Cleaning':   5.0, 'Reassembly':   5.0},
    'origin':   {'origin': 1.0},
    'end_dest': {'end':    1.0},
}

# Energy matrix: deliberately inverted vs. cost — ActorB is cheap AND far,
# but also energy-hungry, while ActorA is expensive/central but energy-light.
# This creates a genuine 4-way trade-off so the test actually exercises
# energy as an optimized objective, not just a passthrough value.
energy_matrix = {
    'ActorA':   {'Cleaning':  2.0, 'Reassembly':  2.0},
    'ActorB':   {'Cleaning': 20.0, 'Reassembly': 20.0},
    'origin':   {'origin': 0.0},
    'end_dest': {'end':    0.0},
}

# Augmented DAG (origin and end boundary nodes included)
dag = {
    'origin':     [],
    'Cleaning':   ['origin'],
    'Reassembly': ['Cleaning'],
    'end':        ['Reassembly'],
}

actors = sorted(locs.keys())   # alphabetical – mirrors Python's user_location_names sort

# Build distance matrix using the same Haversine formula as Go
dist = {}
for a in actors:
    dist[a] = {}
    for b in actors:
        dist[a][b] = 0.0 if a == b else haversine(*locs[a], *locs[b])

print("=== Distance Matrix (km) ===")
for a in actors:
    row = "  " + a + ": " + ", ".join(f"{b}={dist[a][b]:.2f}" for b in actors)
    print(row)

problem_json_str = json.dumps({
    'processes':    dag,
    'CostMatrix':   cost_matrix,
    'EnergyMatrix': energy_matrix,
    'matrix':       dist,
})

# ---------------------------------------------------------------------------
# Load via opt_DAG (identical path to what the gRPC server uses)
# ---------------------------------------------------------------------------
(mfg_route, users_data, transport_matrix, loaded_cost, loaded_energy,
 n_steps, n_users, users, workflow_dag) = \
    opt.load_problem_data_from_json_string(problem_json_str)

print(f"\n=== Problem Setup ===")
print(f"Manufacturing route (topological order): {mfg_route}")
print(f"Users (alphabetically sorted):           {users}")
print(f"n_steps={n_steps}, n_users={n_users}")

problem = opt.ManufacturingProblemDAG(
    mfg_route, users_data, transport_matrix, loaded_cost, loaded_energy,
    n_steps, n_users, users, workflow_dag)

# ---------------------------------------------------------------------------
# Enumerate ALL valid assignments
# ---------------------------------------------------------------------------
valid_per_step = []
print("\n=== Capability per Step ===")
for step_idx in range(n_steps):
    capable = [u for u in range(n_users) if problem.user_step_capable[u, step_idx]]
    valid_per_step.append(capable)
    print(f"  Step {step_idx} ({mfg_route[step_idx]:12s}): "
          f"{[users[u] for u in capable]}")

all_assignments = list(itertools.product(*valid_per_step))
print(f"\nTotal valid assignments: {len(all_assignments)}")

print("\n=== Manual Cost Computation ===")
manual_solutions = []
for assignment in all_assignments:
    x = np.array(assignment)
    manuf = sum(problem.user_step_manuf_cost[x[i], i] for i in range(n_steps))
    energy = sum(problem.user_step_energy[x[i], i] for i in range(n_steps))
    transport = opt.calculate_total_transport_cost(
        workflow_dag, mfg_route, x, transport_matrix)
    makespan, _ = opt.calculate_makespan(
        workflow_dag, mfg_route, x, problem.user_step_manuf_cost, transport_matrix)
    label = " -> ".join(f"{mfg_route[i]}@{users[x[i]]}" for i in range(n_steps))
    manual_solutions.append((label, transport, manuf, makespan, energy))
    print(f"  {label}")
    print(f"    Transport={transport:8.2f}  Manuf={manuf:8.2f}  Makespan={makespan:8.2f}  Energy={energy:8.2f}")

# ---------------------------------------------------------------------------
# Identify Pareto front from manual solutions
# ---------------------------------------------------------------------------
def is_dominated(s, others):
    t1, m1, k1, e1 = s[1], s[2], s[3], s[4]
    for o in others:
        t2, m2, k2, e2 = o[1], o[2], o[3], o[4]
        if (t2 <= t1 and m2 <= m1 and k2 <= k1 and e2 <= e1
                and (t2 < t1 or m2 < m1 or k2 < k1 or e2 < e1)):
            return True
    return False

pareto_manual = [s for s in manual_solutions
                 if not is_dominated(s, manual_solutions)]

print("\n=== Expected Pareto Front (manual) ===")
for s in sorted(pareto_manual, key=lambda x: x[1]):
    print(f"  {s[0]}")
    print(f"    Transport={s[1]:8.2f}  Manuf={s[2]:8.2f}  Makespan={s[3]:8.2f}  Energy={s[4]:8.2f}")

# ---------------------------------------------------------------------------
# Run NSGA2 optimizer (smaller pop for speed in testing)
# ---------------------------------------------------------------------------
print("\n=== Running NSGA2 Optimizer ===")
algorithm = NSGA2(
    pop_size=100,
    sampling=opt.CapableUserSampling(),
    crossover=UniformCrossover(prob=0.7),
    mutation=PolynomialMutation(prob=1.0 / problem.n_var, eta=40, vtype=int),
    eliminate_duplicates=True,
)
termination = get_termination("n_gen", 200)
res = minimize(problem, algorithm, termination, seed=42,
               save_history=False, verbose=False)

feasible = [ind for ind in res.pop
            if ind.G is None or np.all(ind.G <= 0)]
print(f"Feasible individuals in final population: {len(feasible)}")

# Collect unique solutions from optimizer output
seen = {}
for ind in feasible:
    x = ind.X
    label = " -> ".join(f"{mfg_route[i]}@{users[x[i]]}" for i in range(n_steps))
    key = label
    if key not in seen:
        seen[key] = (label, ind.F[0], ind.F[1], ind.F[2], ind.F[3])

opt_solutions = list(seen.values())
pareto_opt = [s for s in opt_solutions if not is_dominated(s, opt_solutions)]

print("\n=== Optimizer Pareto Front ===")
for s in sorted(pareto_opt, key=lambda x: x[1]):
    print(f"  {s[0]}")
    print(f"    Transport={s[1]:8.2f}  Manuf={s[2]:8.2f}  Makespan={s[3]:8.2f}  Energy={s[4]:8.2f}")

# ---------------------------------------------------------------------------
# Cross-check each optimizer solution against manual computation
# ---------------------------------------------------------------------------
print("\n=== Cross-Check: Optimizer Costs vs Manual Recomputation ===")
ok = True
for label, t_opt, m_opt, k_opt, e_opt in pareto_opt:
    # Find matching manual solution
    match = next((s for s in manual_solutions if s[0] == label), None)
    if match is None:
        print(f"  ✗ UNKNOWN assignment returned by optimizer: {label}")
        ok = False
        continue
    _, t_man, m_man, k_man, e_man = match
    transport_ok = abs(t_opt - t_man) < 0.01
    manuf_ok     = abs(m_opt - m_man) < 0.01
    makespan_ok  = abs(k_opt - k_man) < 0.01
    energy_ok    = abs(e_opt - e_man) < 0.01
    status = "✓" if (transport_ok and manuf_ok and makespan_ok and energy_ok) else "✗"
    print(f"  {status} {label}")
    if not transport_ok:
        print(f"      Transport mismatch: optimizer={t_opt:.2f}  expected={t_man:.2f}")
        ok = False
    if not manuf_ok:
        print(f"      Manuf mismatch:     optimizer={m_opt:.2f}  expected={m_man:.2f}")
        ok = False
    if not makespan_ok:
        print(f"      Makespan mismatch:  optimizer={k_opt:.2f}  expected={k_man:.2f}")
        ok = False
    if not energy_ok:
        print(f"      Energy mismatch:    optimizer={e_opt:.2f}  expected={e_man:.2f}")
        ok = False

# ---------------------------------------------------------------------------
# Check that all expected Pareto solutions were found
# ---------------------------------------------------------------------------
print("\n=== Coverage Check ===")
opt_labels = {s[0] for s in pareto_opt}
for s in pareto_manual:
    found = s[0] in opt_labels
    print(f"  {'✓' if found else '✗'} {'FOUND' if found else 'MISSING'}: {s[0]}")
    if not found:
        ok = False

print(f"\n{'='*50}")
print(f"RESULT: {'PASS ✓' if ok else 'FAIL ✗'}")
print(f"{'='*50}")

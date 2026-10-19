import numpy as np
from pymoo.core.problem import ElementwiseProblem
import random
import json
from pymoo.algorithms.moo.nsga2 import NSGA2
from pymoo.operators.crossover.ux import UniformCrossover
from pymoo.operators.mutation.pm import PolynomialMutation
from pymoo.optimize import minimize
from pymoo.operators.sampling.rnd import IntegerRandomSampling
from pymoo.termination import get_termination
from pymoo.core.sampling import Sampling
from collections import deque, defaultdict


# --- Data Loading Function ---
def load_problem_data_from_json_string(filestr):
    """
    Loads problem data supporting DAG workflows with PREDECESSOR notation.
    JSON must contain:
    - 'matrix': location transport costs
    - 'processes': dict where each process lists its predecessors
    - 'CostMatrix': manufacturing cost per process per location
    - 'EnergyMatrix': energy per process per location (optional; defaults to
      0 for any (location, process) pair it doesn't cover — unlike
      CostMatrix, a missing or zero entry never makes an assignment
      infeasible, it just contributes nothing to the energy objective)

    Example:
    {
        "processes": {
            "S1": [],                          // Starting node
            "S2": ["S1"],                      // S2 needs S1 first
            "S5": ["S4.1", "S4.21", "S4.22"]  // S5 needs all three
        }
    }
    """
    data = json.loads(filestr)

    # Extract data from JSON
    matrix_data = data["matrix"]
    cost_matrix_data = data["CostMatrix"]
    energy_matrix_data = data.get("EnergyMatrix", {})

    # Get processes - must be a dict with predecessor notation
    processes_data = data.get("processes", {})
    
    if not isinstance(processes_data, dict):
        raise ValueError(
            "The 'processes' field must be a dictionary with predecessor notation. "
            "Example: {'S2': ['S1'], 'S3': ['S2']}"
        )
    
    # --- Build DAG Structure from Predecessor Notation ---
    predecessor_dict = processes_data
    all_processes = set(predecessor_dict.keys())
    
    # Collect all processes mentioned in predecessor lists
    for predecessors in predecessor_dict.values():
        if isinstance(predecessors, list):
            all_processes.update(predecessors)
        elif predecessors:  # Non-empty, non-list value
            raise ValueError(
                f"Predecessor values must be lists. Found: {predecessors}"
            )
    
    # Convert predecessor notation to successor notation (internal format)
    workflow_dag = convert_predecessors_to_successors(predecessor_dict, all_processes)

    # Use topological order so chromosome position matches execution order.
    manufacturing_route = topological_sort(workflow_dag, all_processes)

    # --- Validate basic structure ---
    if not isinstance(matrix_data, dict) or not isinstance(cost_matrix_data, dict):
        print("Error: JSON file should contain 'matrix' and 'CostMatrix' objects.")
        exit()

    # --- Define Users/Actors as Locations ---
    user_location_names = list(matrix_data.keys())
    user_location_names.sort()
    num_users = len(user_location_names)

    if num_users == 0:
        print("Error: No users (locations) found in the 'matrix' object keys.")
        exit()

    users_data = [{"id": loc, "location": loc} for loc in user_location_names]
    users = [user for user in user_location_names]

    _location_map = {name: i for i, name in enumerate(user_location_names)}

    # --- Process Location Transport Data from 'matrix' ---
    _location_transport_costs = np.full((num_users, num_users), float('inf'))

    for row_name, row_data in matrix_data.items():
        row_idx = _location_map.get(row_name)
        if row_idx is None:
            continue

        if not isinstance(row_data, dict):
            print(f"Error: Value for location '{row_name}' in 'matrix' is not an object.")
            exit()

        for col_name, cost in row_data.items():
            col_idx = _location_map.get(col_name)
            if col_idx is None:
                continue

            if isinstance(cost, (int, float)):
                _location_transport_costs[row_idx, col_idx] = cost
            else:
                print(f"Warning: Non-numeric cost for {row_name} to {col_name}. Setting to inf.")
                _location_transport_costs[row_idx, col_idx] = float('inf')

    _transport_costs_matrix_users = _location_transport_costs

    # --- Process Cost Matrix Data ---
    loaded_cost_matrix = cost_matrix_data

    # Validation
    process_names = set(manufacturing_route)
    for loc, process_costs in loaded_cost_matrix.items():
        if loc not in _location_map:
            print(f"Warning: Location '{loc}' in CostMatrix but not in matrix keys.")
        if not isinstance(process_costs, dict):
            print(f"Error: Value for location '{loc}' in CostMatrix is not an object.")
            exit()
        for process, cost in process_costs.items():
            if not isinstance(cost, (int, float)):
                print(f"Warning: Non-numeric cost for location '{loc}', process '{process}'.")

    # --- Process Energy Matrix Data (optional) ---
    # Unlike CostMatrix, a missing/non-numeric entry is not an error — it
    # just means 0 energy is tracked for that (location, process) pair, and
    # never affects feasibility.
    loaded_energy_matrix = energy_matrix_data if isinstance(energy_matrix_data, dict) else {}
    for loc, process_energies in loaded_energy_matrix.items():
        if not isinstance(process_energies, dict):
            print(f"Warning: Value for location '{loc}' in EnergyMatrix is not an object; ignoring.")

    num_steps = len(manufacturing_route)

    return (manufacturing_route, users_data, _transport_costs_matrix_users,
            loaded_cost_matrix, loaded_energy_matrix, num_steps, num_users, users, workflow_dag)


def convert_predecessors_to_successors(predecessor_dict, all_processes):
    """
    Convert predecessor-based workflow to successor-based workflow (internal format).
    
    Predecessor format (input):
        {"S2": ["S1"], "S3": ["S2"], "S5": ["S4.1", "S4.21", "S4.22"]}
        
    Successor format (output - internal use):
        {"S1": ["S2"], "S2": ["S3"], "S4.1": ["S5"], "S4.21": ["S5"], "S4.22": ["S5"]}
    """
    successor_dict = {process: [] for process in all_processes}
    
    for process, predecessors in predecessor_dict.items():
        if not isinstance(predecessors, list):
            predecessors = [predecessors]
        
        for pred in predecessors:
            if pred in successor_dict:
                successor_dict[pred].append(process)
            else:
                # Predecessor not in all_processes, add it
                successor_dict[pred] = [process]
    
    return successor_dict


# --- DAG Helper Functions ---
def topological_sort(workflow_dag, all_processes):
    """
    Performs topological sort on the workflow DAG.
    Returns list of processes in execution order.
    """
    in_degree = {process: 0 for process in all_processes}
    
    # Calculate in-degrees
    for process, successors in workflow_dag.items():
        for successor in successors:
            if successor in in_degree:
                in_degree[successor] += 1
    
    # Find all nodes with no incoming edges
    queue = deque([p for p in all_processes if in_degree[p] == 0])
    sorted_order = []
    
    while queue:
        current = queue.popleft()
        sorted_order.append(current)
        
        # Reduce in-degree for successors
        for successor in workflow_dag.get(current, []):
            if successor in in_degree:
                in_degree[successor] -= 1
                if in_degree[successor] == 0:
                    queue.append(successor)
    
    if len(sorted_order) != len(all_processes):
        raise ValueError("Workflow contains a cycle or disconnected processes!")
    
    return sorted_order


def get_predecessors(workflow_dag, process):
    """
    Returns list of all predecessor processes for a given process.
    """
    predecessors = []
    for pred, successors in workflow_dag.items():
        if process in successors:
            predecessors.append(pred)
    return predecessors


def calculate_makespan(workflow_dag, process_list, assignment, 
                       user_step_manuf_cost, transport_costs_matrix):
    """
    Calculate the total makespan (completion time) for a DAG workflow.
    Uses critical path method.
    
    Returns: (makespan, is_feasible)
    """
    # Create process to index mapping
    process_to_idx = {p: i for i, p in enumerate(process_list)}
    
    # Track earliest start and finish times
    start_times = {}
    finish_times = {}
    
    # Topological sort to process in correct order
    try:
        sorted_processes = topological_sort(workflow_dag, process_list)
    except ValueError as e:
        return float('inf'), False
    
    for process in sorted_processes:
        process_idx = process_to_idx[process]
        assigned_user = assignment[process_idx]
        
        # Get all predecessors
        predecessors = get_predecessors(workflow_dag, process)
        
        if not predecessors:
            # Starting process - no predecessors
            start_time = 0.0
        else:
            # Must wait for ALL predecessors to complete AND transport time
            max_arrival_time = 0.0
            for pred in predecessors:
                pred_idx = process_to_idx[pred]
                pred_user = assignment[pred_idx]
                pred_finish = finish_times[pred]
                
                # Add transport time if different locations
                if pred_user != assigned_user:
                    transport_cost = transport_costs_matrix[pred_user, assigned_user]
                    if transport_cost == float('inf'):
                        return float('inf'), False
                    arrival_time = pred_finish + transport_cost
                else:
                    arrival_time = pred_finish
                
                max_arrival_time = max(max_arrival_time, arrival_time)
            
            start_time = max_arrival_time
        
        # Calculate finish time
        manufacturing_time = user_step_manuf_cost[assigned_user, process_idx]
        if manufacturing_time == float('inf'):
            return float('inf'), False
        
        finish_time = start_time + manufacturing_time
        
        start_times[process] = start_time
        finish_times[process] = finish_time
    
    # Makespan is the maximum finish time
    makespan = max(finish_times.values()) if finish_times else 0.0
    return makespan, True


def calculate_total_transport_cost(workflow_dag, process_list, assignment, 
                                   transport_costs_matrix):
    """
    Calculate total transportation cost based on actual edges in the DAG.
    """
    process_to_idx = {p: i for i, p in enumerate(process_list)}
    total_transport_cost = 0.0
    
    for process, successors in workflow_dag.items():
        if process not in process_to_idx:
            continue
        
        process_idx = process_to_idx[process]
        current_user = assignment[process_idx]
        
        for successor in successors:
            if successor not in process_to_idx:
                continue
            
            successor_idx = process_to_idx[successor]
            next_user = assignment[successor_idx]
            
            # Only add transport cost if different locations
            if current_user != next_user:
                cost = transport_costs_matrix[current_user, next_user]
                if cost == float('inf'):
                    return float('inf')
                total_transport_cost += cost
    
    return total_transport_cost


# --- Manufacturing Problem with DAG Support ---
class ManufacturingProblemDAG(ElementwiseProblem):
    def __init__(self, manufacturing_route, users_data, transport_costs_matrix_users,
                 loaded_cost_matrix, energy_matrix, num_steps, num_users, users, workflow_dag):
        self.manufacturing_route = manufacturing_route
        self.users_data = users_data
        self.users = users
        self.transport_costs_matrix_users = transport_costs_matrix_users
        self.loaded_cost_matrix = loaded_cost_matrix
        self.energy_matrix = energy_matrix
        self.num_steps = num_steps
        self.num_users = num_users
        self.workflow_dag = workflow_dag

        # Precompute capability and manufacturing cost matrices
        self.user_step_manuf_cost = np.full((self.num_users, self.num_steps), float('inf'))
        self.user_step_capable = np.full((self.num_users, self.num_steps), False)
        # Energy defaults to 0, not inf: unlike cost, a missing/zero entry
        # never makes an assignment infeasible — it just adds nothing to
        # the energy objective.
        self.user_step_energy = np.zeros((self.num_users, self.num_steps))

        for uid, user in enumerate(self.users):
            costsForUser = self.loaded_cost_matrix.get(user, {})
            energyForUser = self.energy_matrix.get(user, {})

            for stepid, processName in enumerate(self.manufacturing_route):
                costProcess = costsForUser.get(processName)

                if costProcess is not None and costProcess > 0:
                    self.user_step_capable[uid, stepid] = True
                    self.user_step_manuf_cost[uid, stepid] = costProcess
                else:
                    self.user_step_capable[uid, stepid] = False
                    self.user_step_manuf_cost[uid, stepid] = float('inf')

                energyProcess = energyForUser.get(processName)
                if isinstance(energyProcess, (int, float)):
                    self.user_step_energy[uid, stepid] = energyProcess

        super().__init__(n_var=self.num_steps,
                         n_obj=4,  # Transport Cost, Manufacturing Cost, Makespan, Energy
                         n_constr=self.num_steps,
                         xl=0,
                         xu=self.num_users - 1,
                         vtype=int)

    def _evaluate(self, x, out, *args, **kwargs):
        total_manufacturing_cost = 0.0
        total_transport_cost = 0.0
        makespan = 0.0
        total_energy = 0.0
        constraints = np.zeros(self.num_steps)

        # 1. Check capability constraints and calculate manufacturing cost
        for step_idx in range(self.num_steps):
            user_idx = x[step_idx]

            if not self.user_step_capable[user_idx, step_idx]:
                constraints[step_idx] = 1.0
                total_manufacturing_cost = float('inf')
                total_transport_cost = float('inf')
                makespan = float('inf')
                total_energy = float('inf')
                break
            else:
                cost = self.user_step_manuf_cost[user_idx, step_idx]
                if cost == float('inf'):
                    constraints[step_idx] = 1.0
                    total_manufacturing_cost = float('inf')
                    total_transport_cost = float('inf')
                    makespan = float('inf')
                    total_energy = float('inf')
                    break
                total_manufacturing_cost += cost
                total_energy += self.user_step_energy[user_idx, step_idx]

        # 2. Calculate transport cost and makespan if feasible
        if total_manufacturing_cost != float('inf'):
            total_transport_cost = calculate_total_transport_cost(
                self.workflow_dag, self.manufacturing_route, x,
                self.transport_costs_matrix_users
            )

            makespan, is_feasible = calculate_makespan(
                self.workflow_dag, self.manufacturing_route, x,
                self.user_step_manuf_cost, self.transport_costs_matrix_users
            )
            
            if not is_feasible:
                total_transport_cost = float('inf')
                makespan = float('inf')

        out["F"] = [total_transport_cost, total_manufacturing_cost, makespan, total_energy]
        out["G"] = constraints


# --- Sampling ---
class CapableUserSampling(Sampling):
    def _do(self, problem, n_samples, **kwargs):
        X = np.full((n_samples, problem.n_var), -1, dtype=int)
        for i in range(n_samples):
            for j in range(problem.n_var):
                capable_users_for_step_j = np.where(problem.user_step_capable[:, j])[0]

                if len(capable_users_for_step_j) == 0:
                    process_name = problem.manufacturing_route[j]
                    raise ValueError(
                        f"No capable user for process: {process_name} (Step {j}). "
                        f"Check CostMatrix."
                    )

                X[i, j] = random.choice(capable_users_for_step_j)
        return X
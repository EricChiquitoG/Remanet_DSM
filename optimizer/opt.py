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


# --- Data Loading Function ---
def load_problem_data_from_json_string(filestr):
    """
    Loads problem data assuming locations are the users/actors.
    JSON contains 'matrix' (location transport), 'processes' (route),
    and 'CostMatrix' (manuf cost per process per location).
    """
    data = json.loads(filestr)

    # Extract data from JSON
    matrix_data = data["matrix"]
    manufacturing_route = data["processes"] # Route comes from 'processes' list
    cost_matrix_data = data["CostMatrix"] # Load the cost matrix

    # --- Validate basic structure ---
    if not isinstance(matrix_data, dict) or not isinstance(manufacturing_route, list) or not isinstance(cost_matrix_data, dict):
         print("Error: JSON file should contain top-level object 'matrix', list 'processes', and object 'CostMatrix'.")
         exit()

    # --- Define Users/Actors as Locations ---
    # The keys of the matrix (or CostMatrix) define the locations, which are now the users/actors
    user_location_names = list(matrix_data.keys()) # Assume matrix keys are the set of users (locations)
    user_location_names.sort() # Ensure consistent ordering if needed later, though map handles it
    num_users = len(user_location_names)

    if num_users == 0:
         print("Error: No users (locations) found in the 'matrix' object keys.")
         exit()

    # Create the users_data list where each user's ID and location is the location name
    users_data = [{"id": loc, "location": loc} for loc in user_location_names]
    users = [user for user in user_location_names]
    print(users)
    print(matrix_data)
    print(manufacturing_route)
    print(cost_matrix_data)

    # Create the location map (Location Name -> User Index)
    # Since users are locations, and we order them consistently, location index = user index
    _location_map = {name: i for i, name in enumerate(user_location_names)}

    # --- Process Location Transport Data from 'matrix' ---
    _location_transport_costs = np.full((num_users, num_users), float('inf')) # Matrix size is now num_users x num_users

    for row_name, row_data in matrix_data.items():
        row_idx = _location_map.get(row_name) # Get user index for this location
        if row_idx is None: continue # Should not happen if _location_map comes from keys

        if not isinstance(row_data, dict):
             print(f"Error: Value for location '{row_name}' in 'matrix' is not an object.")
             exit()

        for col_name, cost in row_data.items():
            col_idx = _location_map.get(col_name) # Get user index for target location
            if col_idx is None: continue # Skip if target location not in map/users

            if isinstance(cost, (int, float)):
                 # Store cost directly in the user-to-user matrix
                 _location_transport_costs[row_idx, col_idx] = cost
            else:
                 print(f"Warning: Non-numeric cost found for {row_name} to {col_name} in matrix: {cost}. Setting to inf.")
                 _location_transport_costs[row_idx, col_idx] = float('inf')

    # Now _location_transport_costs is directly the user-to-user transport matrix
    _transport_costs_matrix_users = _location_transport_costs

    # --- Process Cost Matrix Data ---
    # Store the CostMatrix for later lookup in the problem class's init
    loaded_cost_matrix = cost_matrix_data

    # Optional: Basic validation of CostMatrix keys/structure against expected users (locations) and processes
    process_names = set(manufacturing_route)
    for loc, process_costs in loaded_cost_matrix.items():
        if loc not in _location_map:
             print(f"Warning: Location '{loc}' found in CostMatrix but not in matrix keys (users list).")
        if not isinstance(process_costs, dict):
             print(f"Error: Value for location '{loc}' in CostMatrix is not an object.")
             exit()
        for process, cost in process_costs.items():
            if not isinstance(process, str):
                 print(f"Error: Process key '{process}' for location '{loc}' in CostMatrix is not a string.")
                 exit()
            # Optional: Check if process is in manufacturing_route if you want strict validation
            # if process not in process_names:
            #      print(f"Warning: Process '{process}' found in CostMatrix but not in 'processes' list.")
            if not isinstance(cost, (int, float)):
                 print(f"Warning: Non-numeric cost found for location '{loc}', process '{process}' in CostMatrix: {cost}. This might be treated as infeasible.")


    num_steps = len(manufacturing_route)
    return manufacturing_route, users_data, _transport_costs_matrix_users, loaded_cost_matrix, num_steps, num_users, users


# --- End of Data Loading Function ---


# --- Update the ManufacturingProblem class to use the loaded cost matrix ---

class ManufacturingProblem(ElementwiseProblem):
    def __init__(self, manufacturing_route, users_data, transport_costs_matrix_users, loaded_cost_matrix, num_steps, num_users, users):
        self.manufacturing_route = manufacturing_route
        self.users_data = users_data # user_data will contain {"id": loc, "location": loc}
        self.users = users
        self.transport_costs_matrix_users = transport_costs_matrix_users # This is now directly loc-to-loc matrix
        self.loaded_cost_matrix = loaded_cost_matrix # Store the loaded cost matrix
        self.num_steps = num_steps
        self.num_users = num_users # num_users is number of locations

        # Precompute capability and manufacturing cost matrices
        # This precomputation now uses the loaded_cost_matrix and the user's identity (which is their location)
        self.user_step_manuf_cost = np.full((self.num_users, self.num_steps), float('inf'))
        self.user_step_capable = np.full((self.num_users, self.num_steps), False)


        for uid, user in enumerate(self.users):
            costsForUser = self.loaded_cost_matrix.get(user,{})
            
            for stepid, processName in enumerate(self.manufacturing_route):
                costProcess = costsForUser.get(processName)

                if (costProcess is not None and costProcess > 0): # Assuming cost > 0 for capability):
                    self.user_step_capable[uid, stepid] = True
                    self.user_step_manuf_cost[uid, stepid] = costProcess
                else:
                    # Location is not capable of this process
                    self.user_step_capable[uid, stepid] = False
                    self.user_step_manuf_cost[uid, stepid] = float('inf') # Cannot perform step, cost is infinite

        super().__init__(n_var=self.num_steps,
                         n_obj=2,  # Transportation Cost, Manufacturing Cost
                         n_constr=self.num_steps,  # One capability constraint per step
                         xl=0,
                         xu=self.num_users - 1,
                         vtype=int) # Variables are integer user (location) indices

    def _evaluate(self, x, out, *args, **kwargs):
        total_manufacturing_cost = 0.0
        total_transport_cost = 0.0
        constraints = np.zeros(self.num_steps) # Constraint violations

        # 1. Calculate Manufacturing Cost & Check Capability Constraints
        for step_idx in range(self.num_steps):
            user_idx = x[step_idx] 

            # Check capability and get cost from the precomputed matrices
            if not self.user_step_capable[user_idx, step_idx]:
                constraints[step_idx] = 1.0  # Constraint violated: assigned location is not capable of this process
                total_manufacturing_cost = float('inf') # Penalize heavily
                break # No need to calculate further if a step is impossible
            else:
                # Cost is already looked up and stored in precomputed matrix
                cost = self.user_step_manuf_cost[user_idx, step_idx]
                if cost == float('inf'): # Should not happen if precomputation was correct, but safety check
                     constraints[step_idx] = 1.0 # Treat as violation if cost is inf
                     total_manufacturing_cost = float('inf')
                     break
                total_manufacturing_cost += cost

        # If manufacturing already made it infeasible, transport cost is also effectively infinite
        if total_manufacturing_cost == float('inf'):
            total_transport_cost = float('inf')
        else:
            # 2. Calculate Transportation Cost - Uses the user-to-user transport matrix (which is loc-to-loc)
            for i in range(self.num_steps - 1):
                user_idx_current_step = x[i] # Index of the location assigned to current step
                user_idx_next_step = x[i+1] # Index of the location assigned to next step

                # Defensive check for valid indices
                if not (0 <= user_idx_current_step < self.num_users and 0 <= user_idx_next_step < self.num_users):
                     total_transport_cost = float('inf') # Invalid index during transport calc
                     break # Solution is infeasible

                # No transport cost if the same user (location) performs consecutive steps
                if user_idx_current_step != user_idx_next_step:
                    # Lookup cost from the loc-to-loc matrix (which is stored as _transport_costs_matrix_users)
                    cost = self.transport_costs_matrix_users[user_idx_current_step, user_idx_next_step]
                    if cost == float('inf'): # Implies an impossible transport route
                        total_transport_cost = float('inf') # Path is impossible
                        break
                    total_transport_cost += cost

        out["F"] = [total_transport_cost, total_manufacturing_cost]
        out["G"] = constraints


# --- Sampling, Algorithm Setup, Execution, and Results Printing ---
# These parts remain largely the same, operating on the problem object



class CapableUserSampling(Sampling):
    def _do(self, problem, n_samples, **kwargs):
        X = np.full((n_samples, problem.n_var), -1, dtype=int)
        for i in range(n_samples):
            for j in range(problem.n_var):  # For each step (process name)
                # Find all users (locations) capable of performing the process for step j
                capable_users_for_step_j = np.where(problem.user_step_capable[:, j])[0]

                if len(capable_users_for_step_j) == 0:
                    # This check is crucial - ensures the problem is solvable regarding capabilities for EACH STEP
                    process_name = problem.manufacturing_route[j]
                    raise ValueError(f"No capable user (location) found for process: {process_name} (Step {j}). Check CostMatrix.")

                X[i, j] = random.choice(capable_users_for_step_j)
        return X


""" 
# Initialize the problem with loaded data
problem = ManufacturingProblem(manufacturing_route, users_data, _transport_costs_matrix_users, loaded_cost_matrix, num_steps, num_users, users)

# Define the algorithm
algorithm = NSGA2(
    pop_size=100,
    sampling=CapableUserSampling(), # Uses precomputed capability (based on CostMatrix)
    crossover=UniformCrossover(prob=0.9),
    mutation=PolynomialMutation(prob=1.0/problem.n_var, eta=20, vtype=int),
    eliminate_duplicates=True
)

# Define termination criterion
termination = get_termination("n_gen", 400)

# Run the optimization
res = minimize(
    problem,
    algorithm,
    termination,
    seed=1,
    save_history=True,
    verbose=True
)

# Get objective values and decision variables from the result
objectives = res.F
solutions_user_indices = res.X # These indices now represent assigned locations

if objectives is not None and len(objectives) > 0:
    # Sort by the first objective (transportation cost), then by the second (manufacturing cost)
    sorted_indices = np.lexsort((objectives[:, 1], objectives[:, 0])) # Sort by Manufacturing then Transport

    print("Pareto-Optimal Solutions (Sorted by Transportation Cost, then Manufacturing Cost):")
    for i in sorted_indices:
        user_sequence_indices = solutions_user_indices[i] # Indices of assigned locations
         # Add a check for valid indices before displaying location IDs
        if np.any(user_sequence_indices < 0) or np.any(user_sequence_indices >= num_users):
             print(f"  Solution {i}: Contains invalid user/location indices ({user_sequence_indices}). Skipping display.")
             continue

        # Get the location names from the user_data list
        location_sequence_names = [users_data[user_idx]['id'] for user_idx in user_sequence_indices]

        print(f"  Solution {i}:")
        print(f"    Location Sequence (Indices): {user_sequence_indices}")
        print(f"    Location Sequence (Names):   {location_sequence_names}")
        print(f"    Transport Cost:    {objectives[i, 0]:.2f}")
        print(f"    Manufacturing Cost: {objectives[i, 1]:.2f}")
else:
    print("No feasible solutions found or optimization did not yield results.") """
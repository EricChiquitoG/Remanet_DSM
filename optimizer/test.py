import numpy as np
from pymoo.core.problem import ElementwiseProblem
import random

# 1. Manufacturing Route (fixed sequence of operations)
manufacturing_route = ["casting", "machining", "painting", "assembly"]
num_steps = len(manufacturing_route)

# 2. Users Data
users_data = [
    {"id": "UserA", "location": "Loc1", "capabilities": {"casting", "machining"}, "manufacturing_costs": {"casting": 100, "machining": 150}},
    {"id": "UserB", "location": "Loc2", "capabilities": {"machining", "painting"}, "manufacturing_costs": {"machining": 140, "painting": 90}},
    {"id": "UserC", "location": "Loc3", "capabilities": {"casting", "painting", "assembly"}, "manufacturing_costs": {"casting": 110, "painting": 80, "assembly": 200}},
    {"id": "UserD", "location": "Loc1", "capabilities": {"assembly", "machining"}, "manufacturing_costs": {"assembly": 190, "machining": 160}}
]
num_users = len(users_data)

# 3. Precompute Transportation Costs (User Index to User Index)
# Example: _transport_costs_matrix_users[user_idx1][user_idx2] = cost
# This matrix should be pre-calculated based on user locations.
# For this example, let's assume locations "Loc1", "Loc2", "Loc3" map to indices 0, 1, 2
_location_map = {"Loc1": 0, "Loc2": 1, "Loc3": 2}
_location_transport_costs = np.array([
    [0, 50, 70],  # Loc1 to Loc1, Loc2, Loc3
    [50, 0, 40],  # Loc2 to Loc1, Loc2, Loc3
    [70, 40, 0]   # Loc3 to Loc1, Loc2, Loc3
])
_transport_costs_matrix_users = np.full((num_users, num_users), float('inf'))
for i in range(num_users):
    for j in range(num_users):
        loc_i_str = users_data[i]['location']
        loc_j_str = users_data[j]['location']
        # Handle cases where a location might not be in _location_map if new locations are added
        loc_i_idx = _location_map.get(loc_i_str)
        loc_j_idx = _location_map.get(loc_j_str)

        if loc_i_idx is not None and loc_j_idx is not None:
            _transport_costs_matrix_users[i, j] = _location_transport_costs[loc_i_idx, loc_j_idx]
        else: # One of the locations is not in the predefined map.
             _transport_costs_matrix_users[i, j] = float('inf') # Or handle error




class ManufacturingProblem(ElementwiseProblem):
    def __init__(self, manufacturing_route, users_data, transport_costs_matrix_users):
        self.manufacturing_route = manufacturing_route
        self.users_data = users_data
        self.transport_costs_matrix_users = transport_costs_matrix_users
        self.num_steps = len(manufacturing_route)
        self.num_users = len(users_data)

        # Precompute capability and manufacturing cost matrices for efficiency
        self.user_step_manuf_cost = np.full((self.num_users, self.num_steps), float('inf'))
        self.user_step_capable = np.full((self.num_users, self.num_steps), False)

        #_______Not needed as its the capability map
        for user_idx, user_info in enumerate(self.users_data):
            for step_idx, step_name in enumerate(self.manufacturing_route):
                if step_name in user_info['capabilities']:
                    self.user_step_capable[user_idx, step_idx] = True
                    cost = user_info['manufacturing_costs'].get(step_name)
                    if cost is not None:
                        self.user_step_manuf_cost[user_idx, step_idx] = cost
                    else:
                        # Capable but no cost defined: treat as error or highly expensive
                        self.user_step_manuf_cost[user_idx, step_idx] = float('inf')


        super().__init__(n_var=self.num_steps,
                         n_obj=2,  # Transportation Cost, Manufacturing Cost
                         n_constr=self.num_steps,  # One capability constraint per step
                         xl=0,
                         xu=self.num_users - 1,
                         vtype=int) # Variables are integer user indices

    def _evaluate(self, x, out, *args, **kwargs):
        # x is a 1D numpy array: [user_idx_for_step0, user_idx_for_step1, ...]
        total_manufacturing_cost = 0.0
        total_transport_cost = 0.0
        constraints = np.zeros(self.num_steps) # Constraint violations

        # 1. Calculate Manufacturing Cost & Check Capability Constraints
        for step_idx in range(self.num_steps):
            user_idx = x[step_idx] # User assigned to this step

            if not self.user_step_capable[user_idx, step_idx]:
                constraints[step_idx] = 1.0  # Constraint violated
                # Penalize objectives heavily if a user is not capable
                total_manufacturing_cost = float('inf')
                # No need to calculate further if a step is impossible
                break
            else:
                cost = self.user_step_manuf_cost[user_idx, step_idx]
                if cost == float('inf'): # Capable, but cost is inf (e.g. not defined properly)
                    constraints[step_idx] = 1.0 # Treat as violation
                    total_manufacturing_cost = float('inf')
                    break
                total_manufacturing_cost += cost
        
        # If manufacturing already made it infeasible, transport cost is also effectively infinite
        if total_manufacturing_cost == float('inf'):
            total_transport_cost = float('inf')
        else:
            # 2. Calculate Transportation Cost
            for i in range(self.num_steps - 1):
                user_idx_current_step = x[i]
                user_idx_next_step = x[i+1]

                # No transport cost if the same user performs consecutive steps
                if user_idx_current_step != user_idx_next_step:
                    cost = self.transport_costs_matrix_users[user_idx_current_step, user_idx_next_step]
                    if cost == float('inf'): # Should not happen if matrix is well-defined
                        total_transport_cost = float('inf') # Path is impossible
                        break
                    total_transport_cost += cost
        
        out["F"] = [total_transport_cost, total_manufacturing_cost]
        # Pymoo expects G <= 0 for feasible solutions.
        # A positive value in 'constraints' means violation.
        out["G"] = constraints


from pymoo.core.sampling import Sampling

class CapableUserSampling(Sampling):
    def _do(self, problem, n_samples, **kwargs):
        X = np.full((n_samples, problem.n_var), -1, dtype=int)
        for i in range(n_samples):
            for j in range(problem.n_var):  # For each step
                # Find all users capable of performing step j
                capable_users_for_step_j = np.where(problem.user_step_capable[:, j])[0]
                if len(capable_users_for_step_j) == 0:
                    # This implies an ill-defined problem: no user can do this step.
                    # Handle by raising error or assigning a random user (will be caught by constraints)
                    raise ValueError(f"No capable user found for step: {problem.manufacturing_route[j]}")
                X[i, j] = random.choice(capable_users_for_step_j)
        return X
    
""" from pymoo.core.mutation import Mutation

class CapableUserMutation(Mutation):
    def __init__(self, prob_var=None): # prob_var: probability of mutating each variable (step) in an individual
        super().__init__()
        # If prob_var is None, it can be set to 1.0 / problem.n_var later
        self.prob_var = prob_var

    def _do(self, problem, X, **kwargs):
        Y = X.copy() # Output array
        prob_mutation = self.prob_var if self.prob_var is not None else (1.0 / problem.n_var)

        for i in range(X.shape[0]):  # For each individual in X
            for j in range(X.shape[1]):  # For each variable (step) in the individual
                if random.random() < prob_mutation:
                    current_user_idx = Y[i, j]
                    capable_users_for_step_j = np.where(problem.user_step_capable[:, j])[0]
                    
                    # Find alternative capable users (excluding the current one, if possible)
                    alternative_capable_users = [u for u in capable_users_for_step_j if u != current_user_idx]
                    
                    if len(alternative_capable_users) > 0:
                        Y[i, j] = random.choice(alternative_capable_users)
                    elif len(capable_users_for_step_j) > 0: # Only one capable user (the current one) or all options are the same
                        Y[i, j] = capable_users_for_step_j[0] # Re-assign (no effective change or ensures it's set)
                    # If no capable users at all for this step (should be caught by sampling/problem def)
                    # the gene remains, and the solution will be penalized by constraints.
        return Y """
    
from pymoo.algorithms.moo.nsga2 import NSGA2
from pymoo.operators.crossover.ux import UniformCrossover # Suitable for integer vectors
# from pymoo.operators.crossover.sbx import SBX # More for real-valued
# from pymoo.operators.crossover.point import TwoPointCrossover
from pymoo.operators.mutation.pm import PolynomialMutation # Can work for integers if vtype is int
from pymoo.operators.sampling.rnd import IntegerRandomSampling
from pymoo.optimize import minimize
from pymoo.termination import get_termination

# Initialize the problem
problem = ManufacturingProblem(manufacturing_route, users_data, _transport_costs_matrix_users)

# Define the algorithm
algorithm = NSGA2(
    pop_size=100,
    sampling=CapableUserSampling(),  # Use custom sampling
    crossover=UniformCrossover(prob=0.9), # Crossover probability
    #mutation=CapableUserMutation(prob_var=0.1), # Per-variable mutation probability (e.g., 10%)
    # If not using custom mutation:
    mutation=PolynomialMutation(prob=1.0/problem.n_var, eta=20, vtype=int), # Adjust eta
    eliminate_duplicates=True
)

# Define termination criterion (e.g., number of generations)
termination = get_termination("n_gen", 200) # Example: 200 generations

# Run the optimization
res = minimize(
    problem,
    algorithm,
    termination,
    seed=1, # For reproducibility
    save_history=True,
    verbose=True # Print progress
)

# Get objective values and decision variables from the result
objectives = res.F
solutions_user_indices = res.X

if objectives is not None and len(objectives) > 0:
    # Sort by the first objective (transportation cost), then by the second (manufacturing cost)
    sorted_indices = np.lexsort((objectives[:, 1], objectives[:, 0]))

    print("Pareto-Optimal Solutions (Sorted by Transportation Cost, then Manufacturing Cost):")
    for i in sorted_indices:
        user_sequence_indices = solutions_user_indices[i]
        user_sequence_ids = [users_data[user_idx]['id'] for user_idx in user_sequence_indices]

        print(f"  Solution {i}:")
        print(f"    User Sequence (Indices): {user_sequence_indices}")
        print(f"    User Sequence (IDs):     {user_sequence_ids}")
        print(f"    Transport Cost:    {objectives[i, 0]:.2f}")
        print(f"    Manufacturing Cost: {objectives[i, 1]:.2f}")
        # You can also print the actual route taken:
        # route_taken = [(manufacturing_route[step_idx], users_data[user_idx]['id']) for step_idx, user_idx in enumerate(user_sequence_indices)]
        # print(f"    Detailed Route: {route_taken}")
else:
    print("No feasible solutions found or optimization did not yield results.")
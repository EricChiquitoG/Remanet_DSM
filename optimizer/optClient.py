# Import your generated gRPC files
import DSM_pb2
import DSM_pb2_grpc
import time
import grpc
import numpy as np
import opt
from concurrent import futures
from pymoo.algorithms.moo.nsga2 import NSGA2
from pymoo.operators.crossover.ux import UniformCrossover
from pymoo.operators.mutation.pm import PolynomialMutation
from pymoo.optimize import minimize
from pymoo.operators.sampling.rnd import IntegerRandomSampling
from pymoo.termination import get_termination

class SubmissionServicer(DSM_pb2_grpc.SubmissionServiceServicer):
    """Implements the gRPC service methods."""

    def Optimize(self, request, context):
        """Handles the Optimize RPC call."""
        print("Received optimization request.")

        try:
            manufacturing_route, users_data, transport_costs_matrix_users, loaded_cost_matrix, num_steps, num_users, users = opt.load_problem_data_from_json_string(request.json_problem_data)


            print("safe")
            # 2. Initialize the problem
            problem = opt.ManufacturingProblem(manufacturing_route, users_data, transport_costs_matrix_users, loaded_cost_matrix, num_steps, num_users, users)

            # 3. Define the algorithm (can be configured via request if needed)
            algorithm = NSGA2(
                pop_size=300,
                sampling=opt.CapableUserSampling(),
                crossover=UniformCrossover(prob=0.7),
                mutation=PolynomialMutation(prob=1.0/problem.n_var, eta=40, vtype=int),
                eliminate_duplicates=True
            )

            # 4. Define termination (can be configured via request)
            termination = get_termination("n_gen", 400)

            # 5. Run the optimization
            print("Running optimization...")
            res_minimize = minimize(
                problem,
                algorithm,
                termination,
                seed=3,
                save_history=False, # Set to False for gRPC server to save memory
                verbose=False,
                # return_least_constrained=True # Remove this if it's causing issues or not recognized
            )

            print("Optimization finished.")

            response = DSM_pb2.OptimizationResponse()

            #Entirety of the pareto population, change if needed to filter just a few or one ---> so far open
            final_population = res_minimize.pop

            # Filter unfeasible solutions
            feasible_solutions = [
                ind for ind in final_population
                if ind.G is None or np.all(ind.G <= 0)
            ]
            #Sort solutions
            # May be a good idea to establish a limit for the ammount of options we return ______TO DO!!
            sorted_solutions = sorted(
                feasible_solutions,
                key=lambda ind: (ind.get("rank"), -ind.get("crowding"))
            )

            if len(sorted_solutions) > 0:
                for ind in sorted_solutions:
                    # Access attributes directly from the Individual object
                    user_sequence_indices = ind.X
                    objectives = ind.F

                    if np.any(user_sequence_indices < 0) or np.any(user_sequence_indices >= num_users):
                         print(f"Warning: Skipping invalid solution with indices: {user_sequence_indices}")
                         continue # Skip this solution

                    user_sequence_ids = [users_data[user_idx]['id'] for user_idx in user_sequence_indices]

                    solution_proto = DSM_pb2.Solution(
                        user_indices=user_sequence_indices.tolist(), # Convert numpy array to list
                        user_ids=user_sequence_ids,
                        transport_cost=objectives[0], # First objective
                        manufacturing_cost=objectives[1] # Second objective

                    )
                    response.solutions.append(solution_proto)
            else:
                response.error_message = "Optimization did not find any feasible solutions in the final population."
                print(response.error_message)


        except Exception as e:
            # Catch any errors during data loading or optimization
            response = DSM_pb2.OptimizationResponse()
            response.error_message = f"Optimization failed: {e}"
            print(f"Optimization failed: {e}")
            # You might want to set a gRPC status code here as well
            # context.set_code(grpc.StatusCode.INTERNAL)
            # context.set_details(str(e))


        return response

def serve():
    """Starts the gRPC server."""
    print("Loading...") # This prints before server setup
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=50))
    DSM_pb2_grpc.add_SubmissionServiceServicer_to_server(
        SubmissionServicer(), server)
    server.add_insecure_port('[::]:50060')
    print("Starting gRPC server on port 50060...") # This prints before server.start()
    server.start()
    print("Server is running and waiting for requests on port 500...")
    try:
        while True:
            time.sleep(86400) # Keep server running for a day
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    serve()
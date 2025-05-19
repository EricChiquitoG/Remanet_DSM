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
            # 1. Load data from the request JSON string
            # Pass the hardcoded users_data_actors list
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
            res = minimize(
                problem,
                algorithm,
                termination,
                seed=3,
                save_history=True, # No need to save history for gRPC response
                verbose=False # Don't print progress to server console
            )
            print("Optimization finished.")

            # 6. Format results into the gRPC response message
            response = DSM_pb2.OptimizationResponse()

            if res.F is not None and len(res.F) > 0:
                # Sort results (optional, but good for consistent output)
                sorted_indices = np.lexsort((res.F[:, 1], res.F[:, 0])) # Sort by Manufacturing then Transport

                for i in sorted_indices:
                    user_sequence_indices = res.X[i]
                    # Ensure indices are valid before looking up IDs
                    if np.any(user_sequence_indices < 0) or np.any(user_sequence_indices >= num_users):
                         print(f"Warning: Skipping invalid solution with indices: {user_sequence_indices}")
                         continue # Skip this solution

                    user_sequence_ids = [users_data[user_idx]['id'] for user_idx in user_sequence_indices]

                    solution_proto = DSM_pb2.Solution(
                        user_indices=user_sequence_indices.tolist(), # Convert numpy array to list for proto
                        user_ids=user_sequence_ids,
                        transport_cost=res.F[i, 0],
                        manufacturing_cost=res.F[i, 1]
                    )
                    response.solutions.append(solution_proto)
            else:
                response.error_message = "Optimization did not find any feasible solutions."
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
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    DSM_pb2_grpc.add_SubmissionServiceServicer_to_server(
        SubmissionServicer(), server)
    server.add_insecure_port('[::]:50060')
    print("Starting gRPC server on port 50060...") # This prints before server.start()
    server.start()
    # Add this line AFTER server.start()
    print("Server is running and waiting for requests on port 500...")
    try:
        while True:
            time.sleep(86400) # Keep server running for a day
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    serve()
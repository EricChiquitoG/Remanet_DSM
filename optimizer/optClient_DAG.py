# Import your generated gRPC files
import DSM_pb2
import DSM_pb2_grpc
import time
import grpc
import numpy as np
import opt_DAG as opt
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
        """Handles the Optimize RPC call with DAG predecessor support."""
        print("Received optimization request.")

        try:
            # Load data - now returns workflow_dag as well
            (manufacturing_route, users_data, transport_costs_matrix_users, 
             loaded_cost_matrix, num_steps, num_users, users, workflow_dag) = \
                opt.load_problem_data_from_json_string(request.json_problem_data)

            print("Data loaded successfully")
            print(f"Workflow with {num_steps} processes")
            
            # Initialize the DAG-aware problem
            problem = opt.ManufacturingProblemDAG(
                manufacturing_route, users_data, transport_costs_matrix_users, 
                loaded_cost_matrix, num_steps, num_users, users, workflow_dag
            )

            # Define the algorithm
            algorithm = NSGA2(
                pop_size=400,
                sampling=opt.CapableUserSampling(),
                crossover=UniformCrossover(prob=0.7),
                mutation=PolynomialMutation(prob=1.0/problem.n_var, eta=40, vtype=int),
                eliminate_duplicates=True
            )

            # Define termination
            termination = get_termination("n_gen", 400)

            # Run the optimization
            print("Running optimization...")
            res_minimize = minimize(
                problem,
                algorithm,
                termination,
                seed=3,
                save_history=False,
                verbose=False,
            )

            print("Optimization finished.")

            response = DSM_pb2.OptimizationResponse()

            # Get final population
            final_population = res_minimize.pop

            # Filter feasible solutions
            feasible_solutions = [
                ind for ind in final_population
                if ind.G is None or np.all(ind.G <= 0)
            ]
            
            # Sort solutions
            sorted_solutions = sorted(
                feasible_solutions,
                key=lambda ind: (ind.get("rank"), -ind.get("crowding"))
            )

            maxSol =5

            if len(sorted_solutions) > 0:
                for ind in sorted_solutions[:maxSol]:
                    user_sequence_indices = ind.X
                    objectives = ind.F  # Now 3 objectives: transport, manuf, makespan

                    if np.any(user_sequence_indices < 0) or np.any(user_sequence_indices >= num_users):
                        print(f"Warning: Skipping invalid solution with indices: {user_sequence_indices}")
                        continue

                    assignments = [
                        DSM_pb2.ProcessAssignment(
                            process_name=manufacturing_route[i],
                            user_id=users_data[user_idx]['id']
                        )
                        for i, user_idx in enumerate(user_sequence_indices)
                    ]

                    solution_proto = DSM_pb2.Solution(
                        assignments=assignments,
                        transport_cost=objectives[0],
                        manufacturing_cost=objectives[1],
                    )
                    
                    response.solutions.append(solution_proto)
            else:
                response.error_message = "Optimization did not find any feasible solutions in the final population."
                print(response.error_message)

        except Exception as e:
            response = DSM_pb2.OptimizationResponse()
            response.error_message = f"Optimization failed: {e}"
            print(f"Optimization failed: {e}")
            import traceback
            traceback.print_exc()

        return response


def serve():
    """Starts the gRPC server."""
    print("Loading DAG-enabled optimization server...")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=50))
    DSM_pb2_grpc.add_SubmissionServiceServicer_to_server(
        SubmissionServicer(), server)
    server.add_insecure_port('[::]:50060')
    print("Starting gRPC server on port 50060...")
    server.start()
    print("Server is running and waiting for requests on port 50060...")
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == '__main__':
    serve()
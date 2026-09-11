# Import your generated gRPC files
import DSM_pb2
import DSM_pb2_grpc
import os
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


def _run_optimization(json_problem_data: str) -> list:
    """
    Runs a single NSGA-II optimization in a worker process.

    Must be a module-level function so multiprocessing can pickle it.
    Returns plain dicts so the result crosses the process boundary without
    any proto dependency in the worker.
    """
    (manufacturing_route, users_data, transport_costs_matrix_users,
     loaded_cost_matrix, num_steps, num_users, users, workflow_dag) = \
        opt.load_problem_data_from_json_string(json_problem_data)

    problem = opt.ManufacturingProblemDAG(
        manufacturing_route, users_data, transport_costs_matrix_users,
        loaded_cost_matrix, num_steps, num_users, users, workflow_dag
    )

    algorithm = NSGA2(
        pop_size=400,
        sampling=opt.CapableUserSampling(),
        crossover=UniformCrossover(prob=0.7),
        mutation=PolynomialMutation(prob=1.0/problem.n_var, eta=40, vtype=int),
        eliminate_duplicates=True
    )

    res_minimize = minimize(
        problem,
        algorithm,
        get_termination("n_gen", 400),
        seed=3,
        save_history=False,
        verbose=False,
    )

    feasible = [
        ind for ind in res_minimize.pop
        if ind.G is None or np.all(ind.G <= 0)
    ]
    sorted_solutions = sorted(
        feasible,
        key=lambda ind: (ind.get("rank"), -ind.get("crowding"))
    )

    results = []
    for ind in sorted_solutions[:5]:
        indices = ind.X
        if np.any(indices < 0) or np.any(indices >= num_users):
            continue
        results.append({
            "assignments": [
                {"process_name": manufacturing_route[i], "user_id": users_data[idx]["id"]}
                for i, idx in enumerate(indices)
            ],
            "transport_cost": float(ind.F[0]),
            "manufacturing_cost": float(ind.F[1]),
        })

    return results


_process_pool: futures.ProcessPoolExecutor | None = None


class SubmissionServicer(DSM_pb2_grpc.SubmissionServiceServicer):
    """Implements the gRPC service methods."""

    def Optimize(self, request, context):
        """Handles the Optimize RPC call with DAG predecessor support."""
        print("Received optimization request.")

        try:
            future = _process_pool.submit(_run_optimization, request.json_problem_data)
            solutions_data = future.result()

            response = DSM_pb2.OptimizationResponse()

            if solutions_data:
                for sol in solutions_data:
                    response.solutions.append(DSM_pb2.Solution(
                        assignments=[
                            DSM_pb2.ProcessAssignment(**a) for a in sol["assignments"]
                        ],
                        transport_cost=sol["transport_cost"],
                        manufacturing_cost=sol["manufacturing_cost"],
                    ))
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
    global _process_pool

    port = os.environ.get("OPTIMIZER_PORT", "50060")
    max_workers = int(os.environ.get("OPTIMIZER_WORKERS", os.cpu_count() or 4))

    print(f"Loading DAG-enabled optimization server (process pool: {max_workers} workers)...")
    _process_pool = futures.ProcessPoolExecutor(max_workers=max_workers)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=50))
    DSM_pb2_grpc.add_SubmissionServiceServicer_to_server(SubmissionServicer(), server)
    server.add_insecure_port(f'[::]:{port}')
    print(f"Starting gRPC server on port {port}...")
    server.start()
    print(f"Server is running and waiting for requests on port {port}...")
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)
        _process_pool.shutdown(wait=False)


if __name__ == '__main__':
    serve()
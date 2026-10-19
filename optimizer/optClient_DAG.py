# Import your generated gRPC files
import DSM_pb2
import DSM_pb2_grpc
import os
import time
import grpc
import multiprocessing
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
     loaded_cost_matrix, loaded_energy_matrix, num_steps, num_users, users, workflow_dag) = \
        opt.load_problem_data_from_json_string(json_problem_data)

    problem = opt.ManufacturingProblemDAG(
        manufacturing_route, users_data, transport_costs_matrix_users,
        loaded_cost_matrix, loaded_energy_matrix, num_steps, num_users, users, workflow_dag
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
            # ind.F[2] is makespan — computed and used for ranking but not
            # surfaced here, same as before this change.
            "energy": float(ind.F[3]),
        })

    return results


def _warm_worker() -> int:
    """
    Task used only to force a worker process to actually fork/spawn and
    sit busy long enough that the pool can't hand every other warm-up task
    to the same already-idle worker — without the sleep, a fast worker can
    finish and pick up all of its siblings' tasks too, making it look like
    only one worker ever started. Must be a module-level function so it
    can be pickled and sent to the worker. Returns the worker's pid so
    startup can log that distinct processes really came up.
    """
    time.sleep(1)
    return os.getpid()


_process_pool: futures.ProcessPoolExecutor | None = None


class SubmissionServicer(DSM_pb2_grpc.SubmissionServiceServicer):
    """Implements the gRPC service methods."""

    def Optimize(self, request, context):
        """Handles the Optimize RPC call with DAG predecessor support."""
        print("Received optimization request.", flush=True)

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
                        energy=sol["energy"],
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

    print(f"Loading DAG-enabled optimization server (process pool: {max_workers} workers)...", flush=True)
    # mp_context="spawn" explicitly, rather than trusting the platform
    # default: a spawned worker execs into a clean interpreter and never
    # inherits the parent's memory/file descriptors, so it can't end up
    # holding a corrupted copy of gRPC's own sockets/threads the way a
    # forked worker could.
    _process_pool = futures.ProcessPoolExecutor(
        max_workers=max_workers,
        mp_context=multiprocessing.get_context("spawn"),
    )
    # Force every worker to actually start now, while this process has no
    # gRPC state yet, rather than lazily on first real request — gRPC's
    # own guidance is "fork (or spawn) before gRPC exists, never after";
    # ProcessPoolExecutor otherwise starts new workers on demand, which
    # could happen well after grpc.server() below is already live.
    warm_pids = [f.result(timeout=60) for f in [
        _process_pool.submit(_warm_worker) for _ in range(max_workers)
    ]]
    print(f"Process pool warmed up, {len(set(warm_pids))} distinct worker pids: {warm_pids}", flush=True)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=50))
    DSM_pb2_grpc.add_SubmissionServiceServicer_to_server(SubmissionServicer(), server)
    server.add_insecure_port(f'[::]:{port}')
    print(f"Starting gRPC server on port {port}...", flush=True)
    server.start()
    print(f"Server is running and waiting for requests on port {port}...", flush=True)
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)
        _process_pool.shutdown(wait=False)


if __name__ == '__main__':
    serve()
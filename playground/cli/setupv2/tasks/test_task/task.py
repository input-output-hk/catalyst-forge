"""
Simple test tasks to verify the setup v2 system.
"""

from cli.setupv2 import task
from .config import TestTaskConfig
import time


@task(
    "test_base",
    provides={"value": int, "message": str},
    config_keys=["test_task"],
    retry_delays=[2, 5],
)
def setup_test_base(ctx, cfg, log):
    """Basic test task that provides values for other tasks."""
    # Parse configuration with validation and defaults
    test_config = TestTaskConfig.model_validate(cfg.get("test_task", {}))

    log.write("Starting test_base task\n")
    log.write("This task has no dependencies\n")

    if test_config.enable_delay:
        time.sleep(1)

    log.write("Test base task completed\n")
    return {"value": test_config.test_value, "message": test_config.test_message}


@task(
    "test_dependent",
    requires=["test_base.value", "test_base.message"],
    provides={"result": str},
    config_keys=["test_task"],
)
def setup_test_dependent(ctx, cfg, log):
    """Test task that depends on test_base."""
    # Parse configuration with validation and defaults
    test_config = TestTaskConfig.model_validate(cfg.get("test_task", {}))

    log.write("Starting test_dependent task\n")
    log.write(f"Received value from test_base: {ctx['test_base.value']}\n")
    log.write(f"Received message from test_base: {ctx['test_base.message']}\n")

    if test_config.enable_delay:
        time.sleep(1)

    result = f"Processed: {ctx['test_base.message']} with value {ctx['test_base.value']}"
    log.write(f"Result: {result}\n")

    return {"result": result}


@task("test_parallel_1", provides={"data": str}, config_keys=["test_task"])
def setup_test_parallel_1(ctx, cfg, log):
    """First parallel test task."""
    # Parse configuration with validation and defaults
    test_config = TestTaskConfig.model_validate(cfg.get("test_task", {}))

    log.write("Starting test_parallel_1\n")

    if test_config.enable_delay:
        time.sleep(2)

    log.write("test_parallel_1 completed\n")
    return {"data": "parallel_1_data"}


@task("test_parallel_2", provides={"data": str}, config_keys=["test_task"])
def setup_test_parallel_2(ctx, cfg, log):
    """Second parallel test task."""
    # Parse configuration with validation and defaults
    test_config = TestTaskConfig.model_validate(cfg.get("test_task", {}))

    log.write("Starting test_parallel_2\n")

    if test_config.enable_delay:
        time.sleep(2)

    log.write("test_parallel_2 completed\n")
    return {"data": "parallel_2_data"}


@task(
    "test_final",
    requires=["test_dependent.result", "test_parallel_1.data", "test_parallel_2.data"],
    provides={"summary": str},
    config_keys=[],
)
def setup_test_final(ctx, cfg, log):
    """Final task that depends on multiple other tasks."""

    log.write("Starting test_final task\n")
    log.write(f"Dependent result: {ctx['test_dependent.result']}\n")
    log.write(f"Parallel 1 data: {ctx['test_parallel_1.data']}\n")
    log.write(f"Parallel 2 data: {ctx['test_parallel_2.data']}\n")

    summary = "Final summary with all data collected"
    log.write(f"Summary: {summary}\n")

    return {"summary": summary}

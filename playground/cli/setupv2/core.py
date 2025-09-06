"""
Core task runner with dependency resolution and parallel execution.
"""

import networkx as nx
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from types import MappingProxyType
import shutil
from typing import Dict, Any, Callable, List, Optional
import time
import threading

_tasks: Dict[str, Dict[str, Any]] = {}
_context: Dict[str, Any] = {}
_context_lock = threading.Lock()


def task(
    name: str,
    requires: Optional[List[str]] = None,
    provides: Optional[Dict[str, type]] = None,
    config_keys: Optional[List[str]] = None,
    timeout_sec: int = 600,
    retry_delays: Optional[List[int]] = None,
):
    """
    Register a task with automatic namespacing and validation.

    Args:
        name: Unique task name
        requires: List of required context keys from other tasks
        provides: Dict of keys this task provides with their expected types
        config_keys: List of config keys this task needs access to
        timeout_sec: Maximum time for task execution
        retry_delays: List of delays (in seconds) between retry attempts
    """

    def decorator(fn: Callable):
        provides_dict = provides or {}
        namespaced_provides = {f"{name}.{k}": v for k, v in provides_dict.items()}

        _tasks[name] = {
            "fn": fn,
            "requires": requires or [],
            "provides": namespaced_provides,
            "config_keys": config_keys or [],
            "timeout": timeout_sec,
            "retry_delays": retry_delays or [10],
            "status": "pending",
        }
        return fn

    return decorator


def _filter_config(cfg: Any, keys: List[str]) -> Dict[str, Any]:
    """
    Extract only requested keys from config, ensuring they exist.

    Args:
        cfg: Full configuration object
        keys: List of dot-separated key paths to extract

    Returns:
        Filtered config dict with only requested keys

    Raises:
        ValueError: If a required config key is not found
    """
    filtered: Dict[str, Any] = {}
    for key_path in keys:
        parts = key_path.split(".")
        value = cfg

        for i, part in enumerate(parts):
            if isinstance(value, dict):
                if part not in value:
                    raise ValueError(f"Config key '{key_path}' not found at '{part}'")
                value = value.get(part)
            elif hasattr(value, part):
                value = getattr(value, part)
            else:
                raise ValueError(f"Config key '{key_path}' not found (not a dict at '{part}')")

            if value is None and i < len(parts) - 1:
                raise ValueError(f"Config key '{key_path}' not found (None at '{part}')")

        current = filtered
        for part in parts[:-1]:
            if part not in current:
                current[part] = {}
            current = current[part]
        current[parts[-1]] = value

    return filtered if keys else cfg


def _validate_result(result: Dict[str, Any], expected: Dict[str, type]) -> None:
    """
    Validate task result matches expected types.

    Args:
        result: Task return value
        expected: Expected keys and their types (without namespace prefix)

    Raises:
        ValueError: If validation fails
    """
    for key, expected_type in expected.items():
        if key not in result:
            raise ValueError(f"Task must provide '{key}'")
        if result[key] is None:
            raise ValueError(f"Task provided None for required key '{key}'")
        if not isinstance(result[key], expected_type):
            raise ValueError(
                f"Task provided wrong type for '{key}': "
                f"expected {expected_type.__name__}, got {type(result[key]).__name__}"
            )


def run_task_sync(name: str, cfg: Any, log_dir: Path) -> Dict[str, Any]:
    """
    Execute a single task with retries (synchronous).

    Args:
        name: Task name to execute
        cfg: Configuration object
        log_dir: Directory for task logs

    Returns:
        Task result dictionary

    Raises:
        RuntimeError: If task fails after all retry attempts
    """
    task_info = _tasks[name]

    with _context_lock:
        ctx = MappingProxyType(_context.copy())

    for req in task_info["requires"]:
        if req not in ctx:
            raise ValueError(f"Task '{name}' requires '{req}' but it's not available")

    try:
        filtered_cfg = _filter_config(cfg, task_info["config_keys"])
    except ValueError as e:
        print(f"⚙️  {name}... ✗")
        print(f"  ❌ Config Error: {e}")
        task_info["status"] = "failed"
        raise

    log_file = log_dir / f"{name}.log"

    retry_delays = task_info["retry_delays"]
    last_error = None

    for attempt in range(len(retry_delays) + 1):
        if attempt > 0:
            print(f"⚙️  {name} (retry {attempt}/{len(retry_delays)})...", end="", flush=True)
        else:
            print(f"⚙️  {name}...", end="", flush=True)

        try:
            with open(log_file, "a") as log:
                if attempt > 0:
                    log.write(f"\n\n=== RETRY {attempt} ===\n\n")

                result = task_info["fn"](ctx, filtered_cfg, log)

            if result:
                expected_types = {k.split(".")[-1]: v for k, v in task_info["provides"].items()}
                _validate_result(result, expected_types)

                namespaced_result = {f"{name}.{k}": v for k, v in result.items()}
                with _context_lock:
                    _context.update(namespaced_result)

            print(" ✓")
            task_info["status"] = "completed"
            return result or {}

        except Exception as e:
            print(" ✗")
            last_error = str(e)

        if attempt < len(retry_delays):
            time.sleep(retry_delays[attempt])

    print(f"  📁 Logs: {log_file}")
    print(f"  ❌ Error: {last_error}")
    task_info["status"] = "failed"
    raise RuntimeError(f"Task '{name}' failed after {len(retry_delays) + 1} attempts")


def _resolve_dependencies(tasks: Dict[str, Dict], requested: List[str]) -> set:
    """
    Resolve all transitive dependencies for the requested tasks.

    Args:
        tasks: Task registry
        requested: List of task names the user wants to run

    Returns:
        Set of all tasks that need to run (requested + dependencies)
    """
    # Build provides map for all tasks
    provides_map = {}
    for name, task_info in tasks.items():
        for key in task_info["provides"]:
            provides_map[key] = name

    # Start with requested tasks
    needed = set(requested)
    to_process = list(requested)

    # Recursively find all dependencies
    while to_process:
        current = to_process.pop()
        for req in tasks[current]["requires"]:
            if req in provides_map:
                provider = provides_map[req]
                if provider not in needed:
                    needed.add(provider)
                    to_process.append(provider)

    return needed


def compute_execution_levels(
    tasks: Dict[str, Dict], only: Optional[List[str]] = None
) -> List[List[str]]:
    """
    Compute parallel execution levels using networkx for dependency resolution.

    Args:
        tasks: Task registry dictionary
        only: Optional list of specific tasks to run

    Returns:
        List of task groups that can run in parallel

    Raises:
        ValueError: If there are circular dependencies or missing providers
    """
    G = nx.DiGraph()

    if only:
        # Resolve all dependencies for the requested tasks
        active = _resolve_dependencies(tasks, only)
    else:
        active = set(tasks.keys())

    provides_map: Dict[str, str] = {}
    for name in active:
        for key in tasks[name]["provides"]:
            if key in provides_map:
                raise ValueError(
                    f"Contract conflict: '{key}' provided by both '{provides_map[key]}' and '{name}'"
                )
            provides_map[key] = name

    for name in active:
        G.add_node(name)

    for name in active:
        for req in tasks[name]["requires"]:
            if req not in provides_map:
                available = list(provides_map.keys())
                raise ValueError(
                    f"Task '{name}' requires '{req}' but no task provides it. "
                    f"Available keys: {', '.join(sorted(available))}"
                )
            provider = provides_map[req]
            if provider in active:
                G.add_edge(provider, name)
            else:
                if only:
                    raise ValueError(
                        f"Task '{name}' requires '{req}' from task '{provider}', "
                        f"but '{provider}' is not in --only list"
                    )

    if not nx.is_directed_acyclic_graph(G):
        cycles = list(nx.simple_cycles(G))
        raise ValueError(f"Circular dependency detected: {' -> '.join(cycles[0] + [cycles[0][0]])}")

    return [list(gen) for gen in nx.topological_generations(G)]


def run_all(cfg: Any, only: Optional[List[str]] = None, dry_run: bool = False):
    """
    Run all tasks respecting dependencies and parallelism.

    Args:
        cfg: Configuration object
        only: Optional list of specific tasks to run
        dry_run: If True, show execution plan without running tasks
    """
    if only:
        unknown = set(only) - set(_tasks.keys())
        if unknown:
            available = sorted(_tasks.keys())
            raise ValueError(
                f"Unknown tasks: {', '.join(sorted(unknown))}. "
                f"Available tasks: {', '.join(available)}"
            )

        # Show what dependencies were automatically included
        resolved = _resolve_dependencies(_tasks, only)
        if len(resolved) > len(only):
            added = sorted(resolved - set(only))
            print(f"📋 Including dependencies: {', '.join(added)}")

    levels = compute_execution_levels(_tasks, only)

    total_tasks = sum(len(level) for level in levels)
    print(f"📋 Execution plan: {len(levels)} levels, {total_tasks} tasks")
    for i, level in enumerate(levels, 1):
        print(f"  Level {i}: {', '.join(level)}")
    print()

    if dry_run:
        print("✨ Dry run complete (no tasks executed)")
        return

    # Get the playground directory (two levels up from setupv2)
    playground_dir = Path(__file__).resolve().parent.parent.parent
    log_dir = playground_dir / "logs" / "latest"
    if log_dir.exists():
        shutil.rmtree(log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)

    for level_num, level_tasks in enumerate(levels, 1):
        with ThreadPoolExecutor(max_workers=len(level_tasks)) as executor:
            futures = {
                executor.submit(run_task_sync, name, cfg, log_dir): name for name in level_tasks
            }

            failed = False
            for future in as_completed(futures):
                name = futures[future]
                try:
                    future.result()
                except Exception:
                    failed = True
                    print(f"  ⚠️  Task '{name}' failed, will stop after level {level_num} completes")

            if failed:
                remaining_levels = levels[level_num:]
                if remaining_levels:
                    remaining_tasks = [task for level in remaining_levels for task in level]
                    print(f"\n❌ Stopping execution due to failures at level {level_num}")
                    print(f"   Skipping: {', '.join(remaining_tasks)}")
                raise RuntimeError("One or more tasks failed")

    print("\n✅ All tasks completed successfully")


def list_tasks():
    """
    List all registered tasks with their contracts.

    Returns:
        Formatted string showing task dependencies and provisions
    """
    if not _tasks:
        return "No tasks registered"

    lines = []
    for name, info in sorted(_tasks.items()):
        requires = ", ".join(info["requires"]) if info["requires"] else ""

        provides_with_types = []
        for key, type_hint in info["provides"].items():
            provides_with_types.append(f"{key}: {type_hint.__name__}")
        provides = ", ".join(provides_with_types) if provides_with_types else ""

        lines.append(f"{name}: requires({requires}) → provides({provides})")

    return "\n".join(lines)

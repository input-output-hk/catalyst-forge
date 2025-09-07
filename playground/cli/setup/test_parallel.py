#!/usr/bin/env python3
"""
Test script to verify parallel execution of base infrastructure tasks.
"""

import sys

sys.path.insert(0, "/Users/josh/work/catalyst-forge/playground")

from cli.setup import compute_execution_levels, load_tasks
from cli.setup.core import _tasks


def test_parallel_execution():
    """Test that base infrastructure tasks run in parallel."""

    print("Testing Setup V2 Parallel Execution")
    print("=" * 50)

    # Load all tasks
    load_tasks()

    # Filter to just our base infrastructure tasks
    base_tasks = ["postgres", "cert-manager", "localstack", "registry"]

    # Compute execution levels
    levels = compute_execution_levels(_tasks, only=base_tasks)

    print("\n📋 Execution Plan Analysis:")
    print(f"  Total levels: {len(levels)}")
    print(f"  Total tasks: {sum(len(level) for level in levels)}")

    for i, level in enumerate(levels, 1):
        print(f"\n  Level {i}: {len(level)} tasks")
        for task_name in level:
            task_info = _tasks[task_name]
            print(f"    - {task_name}")
            print(f"      Requires: {task_info['requires'] if task_info['requires'] else 'None'}")
            print(
                f"      Provides: {list(task_info['provides'].keys()) if task_info['provides'] else 'None'}"
            )

    print("\n✅ Test Results:")
    if len(levels) == 1 and len(levels[0]) == 4:
        print("  ✓ All 4 base infrastructure tasks are in Level 1")
        print("  ✓ They will execute in PARALLEL")
        print("  ✓ No dependencies between them")
        return True
    else:
        print("  ✗ Tasks are not properly configured for parallel execution")
        return False


if __name__ == "__main__":
    success = test_parallel_execution()

    print("\n" + "=" * 50)
    if success:
        print("🎉 Phase 4 base infrastructure tasks are correctly configured!")
        print("   They will run in parallel, reducing setup time significantly.")
    else:
        print("❌ There's an issue with the task configuration.")
        sys.exit(1)

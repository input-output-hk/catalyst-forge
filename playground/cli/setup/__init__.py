"""
Setup - Contract-based task runner for Catalyst Forge playground.

A minimal, dependency-based task runner that enables parallel execution
while maintaining simplicity for platform engineers.
"""

from .core import (
    task,
    run_all,
    compute_execution_levels,
    list_tasks,
    _resolve_dependencies,
)
from .tools import helm, kubectl, http, load_tasks, load_values

__all__ = [
    "task",
    "run_all",
    "compute_execution_levels",
    "list_tasks",
    "_resolve_dependencies",
    "helm",
    "kubectl",
    "http",
    "load_tasks",
    "load_values",
]

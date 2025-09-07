"""Utility helpers for logging, command execution, and filesystem.

This module provides small utilities used by the Playground v2 CLI including
structured logging helpers, a typed subprocess runner, and simple path helpers.
"""

from __future__ import annotations

import os
import subprocess
from pathlib import Path
from typing import Any  # noqa: F401  (used for type hints in some dynamic contexts)


def which(name: str) -> str | None:
    """Return the full path to an executable if it exists in PATH.

    Args:
        name: Name of the executable to locate.

    Returns:
        Absolute path of the executable if found; otherwise None.
    """
    for path in os.environ.get("PATH", "").split(os.pathsep):
        candidate = Path(path) / name
        if candidate.exists() and os.access(candidate, os.X_OK):
            return str(candidate)
    return None


def require_cmd(name: str) -> None:
    """Ensure an executable exists in PATH or exit.

    Args:
        name: Name of the executable to require.

    Raises:
        SystemExit: If the command is not found in PATH.
    """
    if not which(name):
        err(f"Required command not found: {name}")
        raise SystemExit(1)


def err(message: str) -> None:
    """Log an error message and exit.

    Args:
        message: The message to emit.
    """
    from .logging import get_logger

    get_logger("console").error(message)


# The old run() helper has been replaced by playground.cli.runner.CommandRunner


def get_repo_root(start: Path | None = None) -> Path:
    """Return the git repository root directory.

    Tries `git rev-parse --show-toplevel`. If that fails, walks up from the
    provided start (or CWD) until a `.git` directory is found. Falls back to
    the current working directory if no repository marker is found.

    Args:
        start: Optional starting path to search from.

    Returns:
        Absolute Path to the repository root (or CWD as a fallback).
    """
    try:
        out = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        p = (out.stdout or "").strip()
        if p:
            return Path(p)
    except Exception:
        pass

    current = (start or Path.cwd()).resolve()
    for parent in [current, *current.parents]:
        if (parent / ".git").exists():
            return parent
    return current

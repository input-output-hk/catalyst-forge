"""Utility helpers for logging, command execution, and filesystem.

This module provides small utilities used by the Playground v2 CLI including
structured logging helpers, a typed subprocess runner, and simple path helpers.
"""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path
from typing import Any  # noqa: F401  (used for type hints in some dynamic contexts)


def log(message: str) -> None:
    """Log an informational message to stdout.

    Args:
        message: The message to emit.
    """
    print(f"[INFO] {message}")


def warn(message: str) -> None:
    """Log a warning message to stderr.

    Args:
        message: The message to emit.
    """
    print(f"[WARN] {message}", file=sys.stderr)


def err(message: str) -> None:
    """Log an error message to stderr.

    Args:
        message: The message to emit.
    """
    print(f"[ERROR] {message}", file=sys.stderr)


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


# The old run() helper has been replaced by playground.cli.runner.CommandRunner


def get_client_cert_paths(cert_dir: Path, base_name: str) -> dict[str, Path]:
    """Return paths for client TLS artifacts used by Earthly CLI.

    Args:
        cert_dir: Directory where client certs are stored.
        base_name: Base filename (without suffix) for the client cert and key.

    Returns:
        Dict with keys: 'cert', 'key', 'ca'.
    """
    cert_path = cert_dir / f"{base_name}.pem"
    key_path = cert_dir / f"{base_name}-key.pem"
    # The CA path is typically the mkcert CAROOT rootCA.pem; callers may copy it locally.
    ca_path = cert_dir / "rootCA.pem"
    return {"cert": cert_path, "key": key_path, "ca": ca_path}


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

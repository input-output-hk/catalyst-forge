#!/usr/bin/env python3
"""Playground v2 CLI for local environment lifecycle and setup tasks.

This CLI provides commands to manage a local development environment, run
pluggable setup tasks, and deploy services to a local k3d Kubernetes cluster.
"""

from __future__ import annotations

from pathlib import Path

import typer

from .k3d import (
    cluster_exists,
    delete_cluster,
)
from .models import ClusterSummary  # re-export for tests/importers
from .config import ConfigState, get_default_config_path, load_config
import logging
from .utils import get_repo_root
from .logging import configure_logging, get_logger


# Use git to find repository root, fallback to parent of cli directory
REPO_ROOT = get_repo_root(Path(__file__).resolve().parent)

app = typer.Typer(
    help="CLI helpers for the Playground v2 local environment",
    pretty_exceptions_enable=False,
    pretty_exceptions_show_locals=False,
)

# Re-export public API for downstream imports/tests
__all__: list[str] = ["ClusterSummary", "app"]
dns_app = typer.Typer(help="DNS helpers (CoreDNS pinning)")
app.add_typer(dns_app, name="dns")


@app.callback()
def _load_global_config(
    ctx: typer.Context,
    pretty: bool = typer.Option(
        False,
        "--pretty",
        help="Enable pretty exceptions (with no locals)",
        rich_help_panel="Global",
    ),
    config: Path | None = typer.Option(
        None,
        "--config",
        help="Path to the playground config file (defaults to playground/config.cue)",
    ),
    debug: bool = typer.Option(
        False,
        "--debug",
        help="Enable debug logging to console",
        rich_help_panel="Global",
    ),
) -> None:
    """Load the playground configuration once and store it in the app context.

    Subcommands may retrieve the resolved path and parsed model via `ctx.obj`.
    """
    # Apply pretty exception preference early
    if pretty:
        try:
            app.pretty_exceptions_enable = True
            app.pretty_exceptions_show_locals = False
        except Exception:
            pass

    # Configure global logging
    playground_dir = REPO_ROOT.parent
    log_dir = playground_dir / "logs" / "latest"
    configure_logging(
        log_dir=log_dir,
        console_level=logging.DEBUG if debug else logging.INFO,
        file_level=logging.DEBUG,
    )

    config_path = config if config is not None else get_default_config_path(REPO_ROOT)
    cfg, raw = load_config(config_path)
    ctx.obj = ConfigState(path=config_path, config=cfg, raw=raw)

    logger = get_logger("console")
    logger.info(f"Using configuration: {config_path}")


@app.command("setup")
def setup(
    ctx: typer.Context,
    dry_run: bool = typer.Option(
        False,
        "--dry-run",
        help="Show execution plan without running tasks",
    ),
    only: list[str] = typer.Option(
        None,
        "--only",
        help="Run only specific task(s). Repeat flag to specify multiple.",
    ),
    list_: bool = typer.Option(
        False,
        "--list",
        help="List available tasks with their contracts and exit",
    ),
) -> None:
    """Run environment setup tasks using the contract-based system.

    This system enables parallel execution based on explicit dependency
    contracts between tasks.
    """
    state: ConfigState | None = getattr(ctx, "obj", None)
    if state is None:
        raise RuntimeError("Config state not loaded")

    # Import setup module
    try:
        from cli.setup import run_all, list_tasks, load_tasks
    except ImportError as e:
        raise SystemExit(f"Failed to import setup: {e}")

    # Get logger for user-facing messages
    logger = get_logger("console")

    # Load all tasks from the tasks/ directory
    logger.info("Loading setup tasks...")
    load_tasks()

    # Handle --list flag
    if list_:
        logger.info("Available tasks and their contracts:")
        logger.info("-" * 60)
        task_list = list_tasks()
        if task_list == "No tasks registered":
            logger.info("No tasks available. Add tasks to setupv2/tasks/ directory.")
        else:
            # For multi-line output, we need to log each line
            for line in task_list.split("\n"):
                if line.strip():
                    logger.info(line)
        return

    # Run tasks
    try:
        run_all(state.raw, only=only or None, dry_run=dry_run)
    except Exception as e:
        raise SystemExit(f"Setup failed: {e}")


@app.command("up")
def up(
    ctx: typer.Context,
) -> None:
    """Bring up the full local test environment using setupv2.

    This runs all setup tasks in the correct dependency order with parallel execution
    where possible. Uses the new contract-based setupv2 system.
    """
    state: ConfigState | None = getattr(ctx, "obj", None)
    if state is None:
        raise RuntimeError("Config state not loaded")

    # Import setup module
    try:
        from cli.setup import run_all, load_tasks
    except ImportError as e:
        raise SystemExit(f"Failed to import setup: {e}")

    # Get logger for user-facing messages
    logger = get_logger("console")

    # Load all tasks from the tasks/ directory
    logger.info("Loading setup tasks...")
    load_tasks()

    # Run all tasks with setupv2
    try:
        run_all(state.raw)
    except Exception as e:
        raise SystemExit(f"Setup failed: {e}")


@app.command("down")
def k3d_down(name: str = typer.Option("forge", help="Cluster name")) -> None:
    """Destroy the k3d cluster and detach resources."""
    logger = get_logger("console")
    if cluster_exists(name):
        delete_cluster(name)
        logger.info(f"Cluster '{name}' deleted.")
    else:
        logger.info(f"Cluster '{name}' not found; nothing to do.")


if __name__ == "__main__":
    app()

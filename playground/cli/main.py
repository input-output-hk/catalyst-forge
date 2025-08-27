#!/usr/bin/env python3
"""Playground v2 CLI for local environment lifecycle and setup tasks.

This CLI provides commands to manage a local development environment, run
pluggable setup tasks, and deploy services to a local k3d Kubernetes cluster.
"""

from __future__ import annotations

from pathlib import Path

import typer

from .k3d_ops import (
    cluster_exists,
    delete_cluster,
)
from .models import ClusterSummary  # re-export for tests/importers
from .config import ConfigState, get_default_config_path, load_config
import importlib
import pkgutil
from .setup.registry import registry
from .utils import get_repo_root, log
from .db import DatabaseRoot, KratosSeeder, HydraSeeder, ForgeSeeder, TemporalSeeder
from .db.base import DatabaseConfig
from .deploy import deploy_service
from .deps import Deps

# Resolve the playground root (one level up from this file's directory)
BASE_DIR = Path(__file__).resolve().parent.parent

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

    repo_root = get_repo_root(BASE_DIR)
    config_path = config if config is not None else get_default_config_path(repo_root)
    cfg, raw = load_config(config_path)
    ctx.obj = ConfigState(path=config_path, config=cfg, raw=raw)
    log(f"Using configuration: {config_path}")


@app.command("migrate")
def migrate(
    pg_host: str = typer.Option("127.0.0.1", help="PostgreSQL host"),
    pg_port: int = typer.Option(5432, help="PostgreSQL port"),
    pg_user: str = typer.Option("postgres", help="PostgreSQL admin user"),
    pg_password: str = typer.Option("postgres", help="PostgreSQL admin password"),
) -> None:
    """Initialize PostgreSQL for dependent apps (Kratos, Hydra).

    Idempotent: safe to re-run. Extensible via additional seeders.
    """
    root = DatabaseRoot(
        DatabaseConfig(
            host=pg_host,
            port=pg_port,
            user=pg_user,
            password=pg_password,
            admin_db="postgres",
        )
    )
    # Run seeders
    for seeder in (KratosSeeder(), HydraSeeder(), ForgeSeeder(), TemporalSeeder()):
        seeder.run(root)
    log("Database migration completed.")


@app.command("deploy")
def deploy(
    ctx: typer.Context,
    service: str = typer.Argument(..., help="Service name under services/ (e.g., api)"),
    kubeconfig: Path | None = typer.Option(
        None,
        "--kubeconfig",
        help="Path to kubeconfig (defaults to playground/kubeconfig or $KUBECONFIG)",
    ),
    no_manifest: bool = typer.Option(
        False, "--no-manifest", help="Do not print the rendered manifest before apply"
    ),
) -> None:
    """Build, render, and deploy a service to the local cluster (Earthly + kubectl)."""
    state: ConfigState | None = getattr(ctx, "obj", None)
    config_path = state.path if state is not None else None
    deploy_service(
        service_name=service,
        kubeconfig_opt=kubeconfig,
        show_manifest=not no_manifest,
        config_path=config_path,
    )


@app.command("setup")
def setup(
    ctx: typer.Context,
    only: list[str] = typer.Option(
        None,
        "--only",
        help="Run only specific setup task(s). Repeat flag to specify multiple.",
    ),
    list_: bool = typer.Option(False, "--list", help="List available setup tasks and exit"),
) -> None:
    """Run environment setup tasks. Idempotent and configurable via config.cue."""
    state: ConfigState | None = getattr(ctx, "obj", None)
    if state is None:
        raise RuntimeError("Config state not loaded")

    # Auto-import all setup submodules to trigger their registration
    import playground.cli.setup as setup_pkg

    for mod in pkgutil.iter_modules(setup_pkg.__path__, setup_pkg.__name__ + "."):
        importlib.import_module(mod.name)

    # Resolve task metas
    metas = list(registry.all())
    if list_:
        for m in sorted(metas, key=lambda k: (k.priority, k.name)):
            log(f"{m.name} (prio={m.priority}) - {m.description}")
        return

    only_set = set(only or [])
    if only:
        unknown = [n for n in only_set if all(m.name != n for m in metas)]
        if unknown:
            raise SystemExit(
                f"Unknown setup task(s): {', '.join(unknown)}. Available: {', '.join(sorted(m.name for m in metas))}"
            )
        metas = [m for m in metas if m.name in only_set]

    # Order by priority then name
    metas.sort(key=lambda m: (m.priority, m.name))

    ran_any = False
    deps = Deps(state)
    for meta in metas:
        task = meta.factory(state, deps)
        if task is None:
            log(f"[INFO] Skipping '{meta.name}' (not applicable)")
            continue
        log(f"[INFO] Running '{meta.name}' (prio={meta.priority}) — {meta.description}")
        task.run()
        ran_any = True

    if not ran_any:
        log("[INFO] No setup tasks to run")


@app.command("down")
def k3d_down(name: str = typer.Option("forge", help="Cluster name")) -> None:
    """Destroy the k3d cluster and detach resources."""
    if cluster_exists(name):
        delete_cluster(name)
        log(f"Cluster '{name}' deleted.")
    else:
        log(f"Cluster '{name}' not found; nothing to do.")


if __name__ == "__main__":
    app()

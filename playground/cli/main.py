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
import importlib
import pkgutil
from .setup.registry import registry
from .utils import get_repo_root, log

## (no direct db imports here)
from .deps import Deps

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

    config_path = config if config is not None else get_default_config_path(REPO_ROOT)
    cfg, raw = load_config(config_path)
    ctx.obj = ConfigState(path=config_path, config=cfg, raw=raw)
    log(f"Using configuration: {config_path}")


@app.command("setup")
def setup(
    ctx: typer.Context,
    only: list[str] = typer.Option(
        None,
        "--only",
        help="Run only specific setup task(s). Repeat flag to specify multiple.",
    ),
    task_arg: list[str] = typer.Option(
        None,
        "--task-arg",
        help="Set a runtime arg for a task: <task>.<key>=<value>. Repeatable.",
    ),
    task_args_file: Path | None = typer.Option(
        None,
        "--task-args-file",
        help='Path to JSON file with runtime args, shaped as {"<task>": { ... }}.',
    ),
    list_: bool = typer.Option(False, "--list", help="List available setup tasks and exit"),
) -> None:
    """Run environment setup tasks. Idempotent and configurable via config.cue."""
    state: ConfigState | None = getattr(ctx, "obj", None)
    if state is None:
        raise RuntimeError("Config state not loaded")

    # Auto-import all setup submodules to trigger their registration
    import cli.setup as setup_pkg

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

    # Parse runtime task args (namespaced) and attach to state
    def _parse_value(val: str):
        import json as _json

        v = val.strip()
        if not v:
            return ""
        if (v.startswith("{") and v.endswith("}")) or (v.startswith("[") and v.endswith("]")):
            try:
                return _json.loads(v)
            except Exception:
                return v
        lower = v.lower()
        if lower in ("true", "false"):
            return lower == "true"
        try:
            if v.isdigit() or (v.startswith("-") and v[1:].isdigit()):
                return int(v)
        except Exception:
            pass
        if "," in v:
            return [p for p in (s.strip() for s in v.split(",")) if p]
        return v

    runtime: dict[str, dict[str, object]] = {}
    import json as _json

    if task_args_file is not None:
        try:
            data = _json.loads(task_args_file.read_text())
            if isinstance(data, dict):
                for tname, tvals in data.items():
                    if isinstance(tname, str) and isinstance(tvals, dict):
                        runtime.setdefault(tname, {}).update(tvals)
        except Exception:
            pass
    for item in task_arg or []:
        if "=" not in item:
            continue
        lhs, rhs = item.split("=", 1)
        if "." not in lhs:
            continue
        tname, key = lhs.split(".", 1)
        if not tname or not key:
            continue
        runtime.setdefault(tname, {})[key] = _parse_value(rhs)

    # Attach runtime map to config state for task consumption
    state.runtime = runtime

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


@app.command("setupv2")
def setupv2(
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
    """Run environment setup tasks using the v2 contract-based system.

    This is a new setup system that enables parallel execution based on
    explicit dependency contracts between tasks.
    """
    state: ConfigState | None = getattr(ctx, "obj", None)
    if state is None:
        raise RuntimeError("Config state not loaded")

    # Import setupv2 module
    try:
        from cli.setupv2 import run_all, list_tasks, load_tasks
    except ImportError as e:
        raise SystemExit(f"Failed to import setupv2: {e}")

    # Load all tasks from the tasks/ directory
    print("[INFO] Loading setupv2 tasks...")
    load_tasks()

    # Handle --list flag
    if list_:
        print("\nAvailable tasks and their contracts:")
        print("-" * 60)
        task_list = list_tasks()
        if task_list == "No tasks registered":
            print("No tasks available. Add tasks to setupv2/tasks/ directory.")
        else:
            print(task_list)
        return

    # Run tasks
    try:
        run_all(state.config, only=only or None, dry_run=dry_run)
    except Exception as e:
        raise SystemExit(f"Setup failed: {e}")


@app.command("up")
def up(
    ctx: typer.Context,
    helmfile_action: str = typer.Option(
        "apply",
        "--helmfile-action",
        help="Helmfile action to use for all helmfile steps (apply or sync)",
    ),
) -> None:
    """Bring up the full local test environment in the correct order.

    This orchestrates existing setup tasks with targeted helmfile steps.
    """
    state: ConfigState | None = getattr(ctx, "obj", None)
    if state is None:
        raise RuntimeError("Config state not loaded")

    # Normalize helmfile action
    action = (helmfile_action or "apply").strip().lower()
    if action not in {"apply", "sync"}:
        action = "apply"

    # Auto-import all setup submodules to ensure task registration
    import cli.setup as setup_pkg

    for mod in pkgutil.iter_modules(setup_pkg.__path__, setup_pkg.__name__ + "."):
        importlib.import_module(mod.name)

    # Build index of tasks by name
    metas = {m.name: m for m in registry.all()}

    def _run_task(name: str, overrides: dict[str, object] | None = None) -> None:
        meta = metas.get(name)
        if meta is None:
            raise SystemExit(f"Required setup task '{name}' is not registered")

        # Ensure runtime mapping exists
        if state.runtime is None:
            state.runtime = {}
        # Save and apply namespaced runtime overrides
        previous = (
            dict(state.runtime.get(name, {})) if isinstance(state.runtime.get(name), dict) else {}
        )
        if overrides:
            merged = {**previous, **overrides}
            state.runtime[name] = merged
        try:
            deps = Deps(state)
            task = meta.factory(state, deps)
            if task is None:
                log(f"[INFO] Skipping '{name}' (not applicable)")
                return
            log(f"[INFO] Running '{name}' — {meta.description}")
            task.run()
        finally:
            # Restore previous runtime scope
            if previous:
                state.runtime[name] = previous
            elif name in (state.runtime or {}):
                try:
                    del state.runtime[name]  # type: ignore[index]
                except Exception:
                    pass

    # 1) k3d cluster
    _run_task("k3d")

    # 2) Generate TLS/Earthly config
    _run_task("generate")

    # 3) Helmfile staged applies (foundational components)
    foundational = [
        "cert-manager",
        "trust-manager",
        "envoy-gateway",
        "registry",
        "postgres",
        "localstack",
        "external-secrets",
    ]
    for release in foundational:
        _run_task("helmfile", {"action": action, "only": release})

    # 4) CoreDNS wildcard pin
    _run_task("dns")

    # 5) Database migrations
    _run_task("migrate")

    # 6) Remaining helmfile releases (app/service layer)
    remaining = ["gitea", "mailpit", "kratos", "hydra", "temporal", "argocd"]
    _run_task("helmfile", {"action": action, "only": remaining})

    # 7) Hydra client setup
    _run_task("hydra")

    # 8) Deploy all internal services
    _run_task("deploy")


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

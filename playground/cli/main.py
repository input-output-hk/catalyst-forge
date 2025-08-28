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

## (no direct db imports here)
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


## migrate subcommand has been removed in favor of setup task 'migrate'


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

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Optional

from ..config import ConfigState
from ..utils import err, log
from ..deploy import deploy_service
from .base import SetupTask
from .registry import register_setup_task


@dataclass
class DeploySetup(SetupTask):
    state: ConfigState

    def run(self) -> None:
        # All services defined under deployments in the typed config
        services = sorted(self.state.config.deployments.keys())
        if not services:
            log("No deployments defined in config; skipping deploy")
            return

        # Runtime overrides
        runtime = getattr(self.state, "runtime", None) or {}
        dep_rt = runtime.get("deploy", {}) if isinstance(runtime, dict) else {}

        only = dep_rt.get("only") if isinstance(dep_rt, dict) else None
        skip = dep_rt.get("skip") if isinstance(dep_rt, dict) else None
        kubeconfig_str = dep_rt.get("kubeconfig") if isinstance(dep_rt, dict) else None
        show_manifest_rt = dep_rt.get("show_manifest") if isinstance(dep_rt, dict) else None

        if isinstance(only, str):
            only_set = {only}
        elif isinstance(only, list):
            only_set = {str(x) for x in only}
        else:
            only_set = set()

        if isinstance(skip, str):
            skip_set = {skip}
        elif isinstance(skip, list):
            skip_set = {str(x) for x in skip}
        else:
            skip_set = set()

        available = set(services)
        unknown_only = [n for n in only_set if n not in available]
        unknown_skip = [n for n in skip_set if n not in available]
        if unknown_only or unknown_skip:
            err(
                "Unknown deployment(s): "
                + ", ".join(sorted(set(unknown_only + unknown_skip)))
                + ". Available: "
                + ", ".join(sorted(available))
            )
            raise SystemExit(1)

        selected = [s for s in services if (not only_set or s in only_set) and s not in skip_set]
        if not selected:
            log("No deployments selected after filtering; nothing to do")
            return

        kubeconfig: Optional[Path] = None
        if isinstance(kubeconfig_str, str) and kubeconfig_str.strip():
            kubeconfig = Path(kubeconfig_str).expanduser().resolve()

        show_manifest: bool = True
        if isinstance(show_manifest_rt, bool):
            show_manifest = show_manifest_rt

        for svc in selected:
            log(f"Deploying service '{svc}'...")
            deploy_service(
                service_name=svc,
                kubeconfig_opt=kubeconfig,
                show_manifest=show_manifest,
                config_path=self.state.path,
            )


@register_setup_task(
    name="deploy",
    description="Build, render, and apply services defined in config (tasks via runtime args)",
    priority=60,
    dependencies=(),
)
def _factory(ctx: ConfigState, _deps) -> DeploySetup | None:
    # Always available if there are deployments defined
    if not ctx.config.deployments:
        return None
    return DeploySetup(state=ctx)

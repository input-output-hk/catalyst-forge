from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any, Optional

from pydantic import BaseModel, Field

from ..config import ConfigState
from ..utils import log, require_cmd
from .base import SetupTask
from .registry import register_setup_task


class HelmfileTaskConfig(BaseModel):
    """Configuration for running a Helmfile as a setup task."""

    enabled: bool = Field(default=True)
    path: str = Field(default="playground/helmfile/helmfile.yaml")
    environment: str | None = None
    working_dir: str | None = None
    timeout_sec: int = Field(default=600, ge=1)
    extra_args: list[str] = Field(default_factory=list)


@dataclass
class HelmfileSetup(SetupTask):
    cfg: HelmfileTaskConfig
    deps: Any

    def _normalize_list(self, value: Any) -> list[str]:
        if value is None:
            return []
        if isinstance(value, list):
            return [str(v) for v in value if str(v)]
        s = str(value)
        if not s:
            return []
        if "," in s:
            return [p for p in (x.strip() for x in s.split(",")) if p]
        return [s]

    def run(self) -> None:
        if not self.cfg.enabled:
            log("helmfile setup task disabled; skipping")
            return

        require_cmd("helmfile")

        # Ensure kubeconfig is resolved strictly via Deps
        _ = self.deps.k8s
        kubeconfig_path: Optional[Path] = getattr(self.deps, "kubeconfig_path", None)

        # Resolve helmfile path relative to repo root (config.cue lives at <repo>/playground/config.cue)
        repo_root = self.deps.state.path.parent.parent
        raw_cfg_path = Path(self.cfg.path).expanduser()
        cfg_path = raw_cfg_path if raw_cfg_path.is_absolute() else (repo_root / raw_cfg_path)
        cfg_path = cfg_path.resolve()

        # Working directory: explicit (anchored to repo root if relative) or helmfile's directory
        if self.cfg.working_dir:
            raw_workdir = Path(self.cfg.working_dir).expanduser()
            workdir = raw_workdir if raw_workdir.is_absolute() else (repo_root / raw_workdir)
            workdir = workdir.resolve()
        else:
            workdir = cfg_path.parent

        if not cfg_path.exists():
            from ..utils import err

            err(f"helmfile: file not found at {cfg_path}")
            raise SystemExit(1)

        # Runtime overrides
        runtime = getattr(self.deps.state, "runtime", None) or {}
        rt = runtime.get("helmfile", {}) if isinstance(runtime, dict) else {}

        action = str(rt.get("action", "apply")).strip().lower()
        if action not in {"apply", "sync"}:
            action = "apply"

        only = self._normalize_list(rt.get("only"))
        skip = self._normalize_list(rt.get("skip"))
        selectors = list(rt.get("selectors", []) if isinstance(rt.get("selectors"), list) else [])
        extra_rt_args = list(rt.get("args", []) if isinstance(rt.get("args"), list) else [])

        base_args: list[str] = [
            "helmfile",
            "-f",
            str(cfg_path),
        ]
        if self.cfg.environment:
            base_args += ["--environment", self.cfg.environment]

        env: dict[str, str] | None = None
        if kubeconfig_path is not None:
            env = {"KUBECONFIG": str(kubeconfig_path)}

        # Helper to execute one helmfile run with assembled selectors
        def _run_with_filters(name_selector: str | None) -> None:
            args = list(base_args)
            if name_selector:
                args += ["--selector", f"name={name_selector}"]
            for sk in skip:
                args += ["--selector", f"name!={sk}"]
            # Raw selectors from runtime
            for sel in selectors:
                args += ["--selector", str(sel)]

            args.append(action)
            if self.cfg.extra_args:
                args += list(self.cfg.extra_args)
            if extra_rt_args:
                args += list(extra_rt_args)

            self.deps.runner.run(args, cwd=workdir, env=env, timeout=self.cfg.timeout_sec)

        if only:
            for name in only:
                log(f"helmfile: running {action} for release '{name}'")
                _run_with_filters(name)
        else:
            log(f"helmfile: running {action} for all releases")
            _run_with_filters(None)


@register_setup_task(
    name="helmfile",
    description="Run Helmfile apply/sync with optional include/skip filters",
    priority=45,
    dependencies=("runner", "k8s"),
)
def _factory(ctx: ConfigState, deps) -> Optional[HelmfileSetup]:
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw = raw_tasks.get("helmfile") if isinstance(raw_tasks, dict) else None
    cfg = HelmfileTaskConfig() if raw is None else HelmfileTaskConfig.model_validate(raw)
    if not cfg.enabled:
        return None
    return HelmfileSetup(cfg=cfg, deps=deps)



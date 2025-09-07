"""Deploy helper that mirrors playground/scripts/deploy.sh in Python.

This module provides a clean, modular implementation of the deploy flow:
  1) Verify required tools and files exist
  2) Read `playground/deployments.cue` via the CUE CLI
  3) Generate a temporary Earthfile from the CUE config and build + push
  4) Generate module CUE via the Go CLI (mod dump)
  5) Write env override CUE from CUE config into the temp module directory
  6) Render the module template (mod template)
  7) Apply rendered manifests with kubectl

Functions are designed for readability and testability, avoiding a single
monolithic command implementation.
"""

from __future__ import annotations

import os
import json
import tempfile
from dataclasses import dataclass
from pathlib import Path

from .utils import require_cmd, get_repo_root
from .logging import get_logger
from .runner import CommandRunner
from .config import get_default_config_path


BASE_DIR = Path(__file__).resolve().parent.parent


@dataclass
class DeployPaths:
    """Resolved filesystem paths used by the deploy flow."""

    repo_root: Path
    playground_dir: Path
    earthly_config: Path
    cli_main_go: Path
    config_cue: Path
    kubeconfig: Path


def _resolve_paths(
    service_name: str, kubeconfig_opt: Path | None, config_path: Path | None
) -> DeployPaths:
    """Compute canonical paths used by the deploy flow.

    Args:
        service_name: Name of the service under `services/`.
        kubeconfig_opt: Optional explicit kubeconfig path.
        config_path: Optional explicit config.cue path.

    Returns:
        DeployPaths with absolute Paths for all required artifacts.
    """
    repo_root = get_repo_root(BASE_DIR)
    playground_dir = repo_root / "playground"
    earthly_config = (playground_dir / "config/earthly.yml").resolve()
    cli_main_go = repo_root / "cli/cmd/main.go"
    config_cue = (
        config_path.resolve() if config_path is not None else get_default_config_path(repo_root)
    )

    # Kubeconfig precedence: explicit option -> $KUBECONFIG -> default path
    kubeconfig_env = os.environ.get("KUBECONFIG")
    kubeconfig = (
        kubeconfig_opt
        if kubeconfig_opt is not None
        else Path(kubeconfig_env)
        if kubeconfig_env
        else playground_dir / "kubeconfig"
    )

    return DeployPaths(
        repo_root=repo_root,
        playground_dir=playground_dir,
        earthly_config=earthly_config,
        cli_main_go=cli_main_go,
        config_cue=config_cue,
        kubeconfig=kubeconfig,
    )


def _preflight(paths: DeployPaths) -> None:
    """Validate external tool requirements and input files."""
    logger = get_logger("console")

    for cmd in ("earthly", "go", "kubectl", "cue"):
        require_cmd(cmd)

    if not paths.earthly_config.is_file():
        logger.error(f"Earthly config not found at '{paths.earthly_config}'.")
        raise SystemExit(1)
    if not paths.cli_main_go.is_file():
        logger.error(f"CLI entrypoint not found at '{paths.cli_main_go}'.")
        raise SystemExit(1)
    if not paths.config_cue.is_file():
        logger.error(f"Configuration file not found: '{paths.config_cue}'.")
        raise SystemExit(1)
    if not paths.kubeconfig.is_file():
        logger.error(
            f"Kubeconfig not found at '{paths.kubeconfig}'. "
            "Run 'uv run python -m playground.cli.main k3d --yes' first."
        )
        raise SystemExit(1)


@dataclass
class DeploymentConfig:
    """Subset of deployment configuration needed for build and overrides."""

    registry: str
    project: str
    target: str
    image_name: str
    image_tag: str


def _read_deployments_config(paths: DeployPaths, service_name: str) -> DeploymentConfig:
    """Read and validate deployment config from CUE (via `cue export`).

    Args:
        paths: Resolved filesystem paths.
        service_name: Key under `deployments` in the CUE file.

    Returns:
        DeploymentConfig with required fields.
    """
    logger = get_logger("console")
    logger.info("Reading deployment configuration from CUE...")
    runner = CommandRunner()
    cp = runner.run(
        [
            "cue",
            "export",
            str(paths.config_cue),
        ],
        capture=True,
        cwd=paths.playground_dir,
    )
    try:
        data = json.loads(cp.stdout or "{}")
    except json.JSONDecodeError as exc:
        logger.error(f"Failed to parse CUE export JSON: {exc}")
        raise SystemExit(1)

    registry = data.get("registry_host")
    svc = (data.get("deployments") or {}).get(service_name) or {}
    project = svc.get("project")
    target = svc.get("target")
    image = svc.get("image") or {}
    image_name = image.get("name")
    image_tag = image.get("tag")

    missing: list[str] = []
    if not registry:
        missing.append("registry_host")
    if not project:
        missing.append(f"deployments.{service_name}.project")
    if not target:
        missing.append(f"deployments.{service_name}.target")
    if not image_name:
        missing.append(f"deployments.{service_name}.image.name")
    if not image_tag:
        missing.append(f"deployments.{service_name}.image.tag")
    if missing:
        logger.error("Missing required fields in deployments.cue: " + ", ".join(missing))
        raise SystemExit(1)

    return DeploymentConfig(
        registry=str(registry),
        project=str(project),
        target=str(target),
        image_name=str(image_name),
        image_tag=str(image_tag),
    )


def _write_temp_earthfile(
    *,
    tmpdir: Path,
    repo_root: Path,
    cfg: DeploymentConfig,
) -> Path:
    """Write an Earthfile into the temporary directory for a single `docker` target."""
    logger = get_logger("console")
    earthfile = tmpdir / "Earthfile"
    # Use absolute path to the service project to avoid relative path issues from tmpdir
    service_path = (repo_root / cfg.project).resolve()
    # Preserve indentation style similar to existing Earthfiles
    content = (
        "VERSION 0.8\n\n"
        "docker:\n"
        f"    FROM {service_path}+{cfg.target}\n\n"
        f"    SAVE IMAGE --push {cfg.registry}/{cfg.image_name}:{cfg.image_tag}\n"
    )
    earthfile.write_text(content)
    logger.info(f"Earthfile written to {earthfile}")
    return earthfile


def _earthly_build_push(paths: DeployPaths, tmpdir: Path) -> None:
    """Run Earthly build and push for the generated Earthfile target."""
    logger = get_logger("console")
    logger.info("Running Earthly build and push...")
    target_ref = f"{tmpdir}+docker"
    runner = CommandRunner()
    runner.run(
        [
            "earthly",
            "--config",
            str(paths.earthly_config),
            "--push",
            target_ref,
        ],
        cwd=tmpdir,
    )


def _generate_module(paths: DeployPaths, tmpdir: Path, service_project: str) -> Path:
    """Generate module CUE from the service directory using the Go CLI."""
    logger = get_logger("console")
    logger.info("Generating module CUE from service directory...")
    # Execute within the Go CLI directory to mirror relative behavior
    cmd = [
        "go",
        "run",
        "cmd/main.go",
        "mod",
        "dump",
        str((paths.repo_root / service_project).resolve()),
    ]
    # Run from the CLI repo dir so relative module paths mirror the shell script
    runner = CommandRunner()
    cp = runner.run(cmd, capture=True, env={**os.environ}, cwd=paths.repo_root / "cli")
    mod_path = tmpdir / "mod.cue"
    mod_path.write_text(cp.stdout or "")
    return mod_path


def _write_env_override_from_cue(paths: DeployPaths, service_name: str, tmpdir: Path) -> Path:
    """Write the environment override CUE into the temp module folder via `cue eval`."""
    logger = get_logger("console")
    logger.info("Writing env overrides from CUE to temporary module...")
    dst = tmpdir / "env.mod.cue"
    expr = f"deployments.{service_name}.overrides"
    runner = CommandRunner()
    cp = runner.run(
        [
            "cue",
            "eval",
            "-e",
            expr,
            str(paths.config_cue),
        ],
        capture=True,
        cwd=paths.playground_dir,
    )
    # Write verbatim CUE output
    dst.write_text(cp.stdout or "")
    return dst


def _render_template(paths: DeployPaths, tmpdir: Path, mod_path: Path) -> None:
    """Render the module template into YAML manifests via the Go CLI."""
    logger = get_logger("console")
    logger.info("Rendering module template...")
    cmd = [
        "go",
        "run",
        "cmd/main.go",
        "mod",
        "template",
        "-o",
        str(tmpdir),
        str(mod_path),
    ]
    runner = CommandRunner()
    runner.run(cmd, env={**os.environ}, cwd=paths.repo_root / "cli")


def _find_manifest_file(tmpdir: Path) -> Path | None:
    """Prefer `main.yaml` if present; otherwise return the first YAML file."""
    preferred = tmpdir / "main.yaml"
    if preferred.is_file():
        return preferred
    candidates = list(tmpdir.glob("*.y*ml"))
    return candidates[0] if candidates else None


def _apply_manifests(kubeconfig: Path, tmpdir: Path) -> None:
    """Apply all manifests in the temporary directory with kubectl."""
    env = {**os.environ, "KUBECONFIG": str(kubeconfig)}
    # Log current context, if possible
    runner = CommandRunner()
    try:
        cp = runner.run(["kubectl", "config", "current-context"], capture=True, env=env)
        context = (cp.stdout or "").strip() or "unknown"
    except Exception:
        context = "unknown"
    logger = get_logger("console")
    logger.info(f"Applying Kubernetes manifests from {tmpdir} (context: {context})...")
    runner.run(["kubectl", "apply", "-f", str(tmpdir)], env=env)


def deploy_service(
    service_name: str,
    kubeconfig_opt: Path | None = None,
    show_manifest: bool = True,
    config_path: Path | None = None,
) -> None:
    """Run the full deploy pipeline for a given service.

    Args:
        service_name: Name of the service under `services/` to build and deploy.
        kubeconfig_opt: Optional override path to the kubeconfig file.
        show_manifest: If True, prints the selected manifest to stdout.
    """
    paths = _resolve_paths(service_name, kubeconfig_opt, config_path)
    logger = get_logger("console")
    logger.info(f"Service: {service_name}")
    logger.info(f"Root dir: {paths.repo_root}")
    logger.info(f"Playground dir: {paths.playground_dir}")

    _preflight(paths)

    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)
        logger.info(f"Temp dir: {tmpdir}")

        # Read config from CUE and build + push via generated Earthfile
        cfg = _read_deployments_config(paths, service_name)
        _write_temp_earthfile(tmpdir=tmpdir, repo_root=paths.repo_root, cfg=cfg)
        _earthly_build_push(paths, tmpdir)

        # Dump module and write env overrides from CUE
        mod_path = _generate_module(paths, tmpdir, cfg.project)
        _write_env_override_from_cue(paths, service_name, tmpdir)
        _render_template(paths, tmpdir, mod_path)

        manifest = _find_manifest_file(tmpdir)
        if not manifest:
            logger.error(f"No Kubernetes manifest files (*.yaml|*.yml) found in {tmpdir}.")
            raise SystemExit(1)

        if show_manifest and manifest.is_file():
            logger.info("Manifest:")
            # Print manifest to stdout
            try:
                print(manifest.read_text())
            except Exception:
                pass

        _apply_manifests(paths.kubeconfig, tmpdir)

    logger.info("Done. Temporary files cleaned up.")

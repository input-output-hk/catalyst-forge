"""Centralized configuration loading for the Playground CLI.

This module defines Pydantic models for the structured parts of the
playground configuration and utilities to load/parse a CUE-based config
file. The configuration file is intended to live at
`<repo_root>/playground/config.cue` by default, but a legacy
`deployments.cue` is also supported as a fallback for compatibility.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Mapping, Tuple

from pydantic import BaseModel, Field

from cli.logging import get_logger
from cli.runner import CommandRunner


class Image(BaseModel):
    """Container image information for a deployable service."""

    name: str
    tag: str


class DeploymentConfig(BaseModel):
    """Deployment configuration for a single service.

    Notes:
        - The `overrides` field is intentionally untyped because its shape is
          defined in CUE and may vary per service. Subcommands that need to
          consume overrides for templating should call `load_overrides_cue` to
          retrieve the raw CUE snippet.
    """

    project: str
    target: str
    image: Image
    overrides: Any | None = None


class PlaygroundConfig(BaseModel):
    """Top-level playground configuration."""

    registry_host: str
    deployments: Dict[str, DeploymentConfig]

    # k3d cluster configuration
    k3d: Dict[str, Any] = Field(default_factory=dict)

    # DNS configuration
    dns: Dict[str, Any] = Field(default_factory=dict)

    # Typed external services configuration
    external: "ExternalConfig | None" = None

    # Unopinionated per-task configuration. Each setup task should read
    # its configuration from tasks.<task_name> and perform its own validation.
    tasks: Dict[str, Any] = Field(default_factory=dict)

    # Global dependency configuration (e.g., db, k8s). Concrete schemas are
    # defined and validated by the consumers that request these deps.
    deps: Dict[str, Any] = Field(default_factory=dict)


class HydraClientConfig(BaseModel):
    """Configuration for a single OAuth2/OIDC client in Hydra.

    Defaults mirror a public PKCE client for CLI usage.
    """

    client_id: str
    client_name: str | None = None
    scope: str = Field(default="openid offline")
    grant_types: list[str] = Field(default_factory=lambda: ["authorization_code", "refresh_token"])
    response_types: list[str] = Field(default_factory=lambda: ["code"])
    token_endpoint_auth_method: str = Field(default="none")
    redirect_uris: list[str]
    post_logout_redirect_uris: list[str] | None = None


class HydraExternalConfig(BaseModel):
    """Hydra-related external configuration."""

    clients: Mapping[str, HydraClientConfig] = Field(default_factory=dict)


class ExternalConfig(BaseModel):
    """External services configuration aggregation."""

    hydra: HydraExternalConfig | None = None


@dataclass
class ConfigState:
    """Resolved configuration state stored at CLI startup."""

    path: Path
    config: PlaygroundConfig
    raw: Dict[str, Any]
    # Runtime, task-scoped arguments injected at execution time (not from CUE)
    runtime: Dict[str, Any] | None = None


def get_default_config_path(repo_root: Path) -> Path:
    """Return the default config path."""

    return repo_root / "playground/config.cue"


def load_config(config_path: Path) -> Tuple[PlaygroundConfig, Dict[str, Any]]:
    """Load and validate the playground configuration via `cue export` and return both typed and raw data.

    Args:
        config_path: Absolute path to the CUE config file to export.

    Returns:
        A tuple: (typed `PlaygroundConfig`, raw JSON `dict`).
    """

    if not config_path.exists():
        logger = get_logger("console")
        logger.error(f"Config file not found: {config_path}")
        raise SystemExit(1)

    runner = CommandRunner()
    cp = runner.run(
        ["cue", "export", str(config_path)],
        capture=True,
        cwd=config_path.parent,
    )
    try:
        data = json.loads(cp.stdout or "{}")
    except json.JSONDecodeError as exc:
        logger.error(f"Failed to parse CUE export JSON from {config_path}: {exc}")
        raise SystemExit(1)

    try:
        typed = PlaygroundConfig.model_validate(data)
        return typed, data
    except Exception as exc:  # ValidationError, but keep dependency surface minimal here
        logger.error(f"Invalid playground configuration: {exc}")
        raise SystemExit(1)


def load_overrides_cue(config_path: Path, service_name: str) -> str:
    """Evaluate and return the raw CUE for a service's overrides subtree.

    Args:
        config_path: Absolute path to the CUE config file to evaluate against.
        service_name: Service key under `deployments`.

    Returns:
        The raw CUE text for `deployments.<service>.overrides`.
        If the field is absent or evaluation yields no output, returns an empty string.
    """

    expr = f"deployments.{service_name}.overrides"
    try:
        cp = CommandRunner().run(
            ["cue", "eval", "-e", expr, str(config_path)],
            capture=True,
            cwd=config_path.parent,
        )
        return (cp.stdout or "").strip()
    except Exception:
        # If overrides is undefined or evaluation fails, treat as empty snippet.
        return ""

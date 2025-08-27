from __future__ import annotations

import os
import signal
import subprocess
import time
from typing import Mapping as TypingMapping

import requests  # type: ignore[import-not-found,import-untyped]

from pydantic import BaseModel, Field

from ..config import ConfigState
from ..utils import err, log
from .base import SetupTask
from .registry import register_setup_task


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


class HydraTaskConfig(BaseModel):
    clients: dict[str, HydraClientConfig] = Field(default_factory=dict)


class HydraSetup(SetupTask):
    """Register/update Hydra OAuth2 clients based on typed config."""

    def __init__(self, clients: TypingMapping[str, HydraClientConfig], runner) -> None:
        self.clients = dict(clients)
        self.runner = runner

    def _port_forward(self) -> subprocess.Popen[bytes]:
        return subprocess.Popen(
            ["kubectl", "-n", "auth", "port-forward", "svc/hydra-admin", "8445:4445"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

    def run(self) -> None:
        # Ensure Hydra deployment is ready
        self.runner.run(
            ["kubectl", "-n", "auth", "rollout", "status", "deploy/hydra", "--timeout=180s"]
        )

        # Port-forward admin locally for API access
        pf = self._port_forward()
        try:
            for _ in range(30):
                try:
                    requests.get("http://127.0.0.1:8445/health/ready", timeout=1)
                    break
                except Exception:
                    time.sleep(1)

            url_base = "http://127.0.0.1:8445/admin/clients"
            headers = {"content-type": "application/json"}
            for key, spec in self.clients.items():
                client_id = spec.client_id or key
                resp = requests.post(url_base, json=spec.model_dump(), headers=headers)
                if resp.status_code >= 300:
                    resp = requests.put(
                        f"{url_base}/{client_id}", json=spec.model_dump(), headers=headers
                    )
                if resp.status_code >= 300:
                    err(
                        f"Failed to upsert Hydra client '{client_id}': {resp.status_code} {resp.text}"
                    )
                    raise SystemExit(1)
                log(f"Hydra client upserted: {client_id}")
        finally:
            try:
                os.kill(pf.pid, signal.SIGTERM)
            except Exception:
                pass


@register_setup_task(
    name="hydra",
    description="Register/update Hydra OAuth2 clients from tasks.hydra",
    priority=50,
    dependencies=("runner",),
)
def _factory(ctx: ConfigState, deps) -> HydraSetup | None:
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw_hydra = raw_tasks.get("hydra") if isinstance(raw_tasks, dict) else None
    if not raw_hydra:
        return None
    typed = HydraTaskConfig.model_validate(raw_hydra)
    if not typed.clients:
        return None
    return HydraSetup(typed.clients, deps.runner)

from __future__ import annotations

import os
import signal
import subprocess
import time
from typing import Mapping as TypingMapping
from datetime import datetime, timedelta, timezone
import tempfile
from pathlib import Path
from kubernetes.client import V1Secret, V1ObjectMeta  # type: ignore
from kubernetes.client.exceptions import ApiException  # type: ignore

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
    trusted_jwt_grant_issuers: list[dict] = Field(default_factory=list)


class HydraSetup(SetupTask):
    """Register/update Hydra OAuth2 clients based on typed config."""

    def __init__(
        self, clients: TypingMapping[str, HydraClientConfig], runner, trusted: list[dict], deps
    ) -> None:
        self.clients = dict(clients)
        self.runner = runner
        self.trusted = list(trusted)
        self.deps = deps

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

            # Ensure a persistent signing key secret exists for mock-oidc
            secret_ns = "oidc"
            secret_name = "mock-oidc-github-signing"
            core = self.deps.k8s["core"]

            try:
                core.read_namespaced_secret(name=secret_name, namespace=secret_ns)
                log(f"Signing key secret exists: {secret_ns}/{secret_name}")
            except ApiException as e:  # pragma: no cover
                if getattr(e, "status", None) != 404:
                    raise
                with tempfile.TemporaryDirectory() as td:
                    key_path = Path(td) / "signing_key.pem"
                    self.runner.run(["openssl", "genrsa", "-out", str(key_path), "2048"])
                    pem = key_path.read_text()
                sec = V1Secret(
                    metadata=V1ObjectMeta(name=secret_name),
                    type="Opaque",
                    string_data={"signing_key.pem": pem},
                )
                core.create_namespaced_secret(namespace=secret_ns, body=sec)
                log(f"Created signing key secret: {secret_ns}/{secret_name}")

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

            # Configure trusted JWT grant issuers (RFC 7523) for JWT-bearer assertions
            for entry in self.trusted:
                issuer = entry.get("issuer")
                jwks_uri = entry.get("jwks_uri") or entry.get("jwks_url")
                allow_any_subject = bool(entry.get("allow_any_subject", True))
                scope = entry.get("scope")  # optional
                subject = entry.get("subject")  # optional
                expires_at = entry.get("expires_at")  # optional RFC3339 string
                jwks_kid = entry.get("jwks_kid")  # optional: filter a specific key
                if not issuer or not jwks_uri:
                    err("trusted_jwt_grant_issuers entry missing 'issuer' or 'jwks_uri'")
                    raise SystemExit(1)
                # Fetch JWKS and create one trust record per key (or the selected kid)
                try:
                    sess = self.deps.requests_session
                    jwks_resp = sess.get(jwks_uri, timeout=10)
                    jwks = jwks_resp.json()
                    keys = jwks.get("keys", []) if isinstance(jwks, dict) else []
                except Exception as e:  # pragma: no cover
                    err(f"Failed to fetch JWKS from {jwks_uri}: {e}")
                    raise SystemExit(1)

                if not keys:
                    err(f"JWKS from {jwks_uri} contained no keys")
                    raise SystemExit(1)

                created = 0
                for jwk in keys:
                    kid = jwk.get("kid")
                    if jwks_kid and kid != jwks_kid:
                        continue

                    payload = {
                        "issuer": issuer,
                        "jwk": jwk,
                        "allow_any_subject": allow_any_subject,
                    }
                    if scope:
                        payload["scope"] = scope
                    if subject:
                        payload["subject"] = subject
                    if not expires_at:
                        one_year = datetime.now(timezone.utc) + timedelta(days=365)
                        expires_at = one_year.strftime("%Y-%m-%dT%H:%M:%SZ")
                    payload["expires_at"] = expires_at

                    resp = requests.post(
                        "http://127.0.0.1:8445/admin/trust/grants/jwt-bearer/issuers",
                        json=payload,
                        headers=headers,
                    )
                    # Accept 201 Created or 409 Conflict (already exists for this kid)
                    if resp.status_code not in (201, 409):
                        err(
                            f"Failed to trust issuer '{issuer}' (kid={kid}): {resp.status_code} {resp.text}"
                        )
                        raise SystemExit(1)
                    created += 1
                if created == 0 and jwks_kid:
                    err(f"No JWKS key with kid={jwks_kid} found at {jwks_uri}")
                    raise SystemExit(1)
                log(f"Trusted JWT grant issuer configured: {issuer} (keys={created})")

            # Verification: list trusted issuers from Hydra and log summaries
            try:
                resp = requests.get(
                    "http://127.0.0.1:8445/admin/trust/grants/jwt-bearer/issuers",
                    headers=headers,
                    timeout=10,
                )
                if resp.status_code == 200:
                    items = resp.json()
                    count = len(items) if isinstance(items, list) else 0
                    log(f"Hydra trusted JWT issuers: {count}")
                    if isinstance(items, list):
                        for it in items:
                            iss = it.get("issuer")
                            subj = it.get("subject")
                            anysub = it.get("allow_any_subject")
                            kid = (
                                (it.get("public_key") or {}).get("kid")
                                if isinstance(it.get("public_key"), dict)
                                else None
                            )
                            log(
                                f" - issuer={iss} subject={subj} allow_any_subject={anysub} kid={kid}"
                            )
                else:
                    err(f"Failed to list trusted issuers: {resp.status_code} {resp.text}")
            except Exception as e:
                err(f"List trusted issuers error: {e}")
        finally:
            try:
                os.kill(pf.pid, signal.SIGTERM)
            except Exception:
                pass


@register_setup_task(
    name="hydra",
    description="Register/update Hydra OAuth2 clients from tasks.hydra",
    priority=50,
    dependencies=("runner", "k8s"),
)
def _factory(ctx: ConfigState, deps) -> HydraSetup | None:
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw_hydra = raw_tasks.get("hydra") if isinstance(raw_tasks, dict) else None
    if not raw_hydra:
        return None
    typed = HydraTaskConfig.model_validate(raw_hydra)
    if not typed.clients and not typed.trusted_jwt_grant_issuers:
        return None
    return HydraSetup(typed.clients, deps.runner, typed.trusted_jwt_grant_issuers, deps)

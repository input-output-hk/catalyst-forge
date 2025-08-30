from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any, Optional
import time

from pydantic import BaseModel, Field, HttpUrl

from ..config import ConfigState
from ..utils import log, warn, get_repo_root
from ..runner import CommandRunner
from ..secrets.localstack import ensure_secret_json, read_secret_json
from .base import SetupTask
from .registry import register_setup_task


class GiteaTaskConfig(BaseModel):
    """Configuration for Gitea bootstrap task.

    Notes:
        - Defaults target the local playground host and are safe to override via
          `tasks.gitea` in playground/config.cue when we wire that up.
    """

    enabled: bool = Field(default=True)
    base_url: HttpUrl | str = Field(default="https://git.projectcatalyst.dev")
    readiness_timeout_sec: int = Field(default=180, ge=1)
    readiness_interval_sec: float = Field(default=2.0, ge=0.25)

    admin_username: str = Field(default="admin")
    admin_password: str = Field(default="admin_password")
    automation_username: str = Field(default="automation")
    automation_password: str = Field(default="automation_password")
    organization: str | None = None
    repository: str = Field(default="catalyst-forge")
    remote_name: str = Field(default="gitea")
    enable_push: bool = Field(default=True)
    secrets_prefix: str = Field(default="shared-services/git/gitea")
    token_scopes: list[str] = Field(default_factory=lambda: [
        "read:repository",
        "write:repository",
    ])


@dataclass
class GiteaSetup(SetupTask):
    cfg: GiteaTaskConfig
    deps: Any

    def _wait_ready(self) -> None:
        """Wait for Gitea HTTP API to report ready by polling /api/v1/version."""
        base = str(self.cfg.base_url).rstrip("/")
        url = f"{base}/api/v1/version"
        sess = self.deps.requests_session
        deadline = time.time() + float(self.cfg.readiness_timeout_sec)
        last_err: Exception | None = None
        log(f"Gitea: waiting for readiness at {url}")
        while time.time() < deadline:
            try:
                r = sess.get(url, timeout=5)
                if r.status_code == 200:
                    log("Gitea: ready")
                    return
                warn(f"Gitea: HTTP {r.status_code} from version endpoint; retrying...")
            except Exception as e:  # pragma: no cover - transient network
                last_err = e
            time.sleep(float(self.cfg.readiness_interval_sec))
        if last_err:
            raise RuntimeError(f"Gitea readiness check failed: {last_err}")
        raise RuntimeError("Gitea readiness check timed out")

    def run(self) -> None:
        if not self.cfg.enabled:
            log("gitea setup task disabled; skipping")
            return
        # 1) Health/Readiness gate
        self._wait_ready()

        # 2) Bootstrap users and repo
        base = str(self.cfg.base_url).rstrip("/")
        admin_user = self.cfg.admin_username
        admin_pass = self.cfg.admin_password
        auto_user = self.cfg.automation_username
        auto_pass = self.cfg.automation_password
        repo_name = self.cfg.repository
        owner = self.cfg.organization or auto_user

        # Ensure users exist (admin, automation) via gitea CLI inside the pod for idempotency
        self._ensure_user_cli(username=admin_user, password=admin_pass, admin=True)
        self._ensure_user_cli(username=auto_user, password=auto_pass, admin=False)

        # Create PAT for automation user using basic auth
        token = self._ensure_user_pat_user(username=auto_user, password=auto_pass, token_name="automation-ci")

        # Ensure repository exists under owner (org or user)
        self._ensure_repository(owner=owner, repo=repo_name, token=token)

        # Generate or reuse SSH keypair and add as deploy key (read-only)
        existing = read_secret_json(name=f"{self.cfg.secrets_prefix}/bootstrap") or {}
        priv_key = existing.get("ssh_private_key") if isinstance(existing, dict) else None
        pub_key = existing.get("ssh_public_key") if isinstance(existing, dict) else None
        if not priv_key or not pub_key:
            priv_key, pub_key = self._ensure_ssh_keypair()
        self._ensure_deploy_key(owner=owner, repo=repo_name, token=token, title="argocd-readonly", pubkey=pub_key)

        # Prepare known_hosts for git host
        known_hosts = self._ssh_known_hosts()

        # Store secrets in LocalStack
        from urllib.parse import urlparse
        host = urlparse(base).netloc or base.replace("https://", "").replace("http://", "").strip("/")
        repo_http = f"{base}/{owner}/{repo_name}.git"
        repo_ssh = f"git@{host}:{owner}/{repo_name}.git"
        ensure_secret_json(
            name=f"{self.cfg.secrets_prefix}/bootstrap",
            payload={
                "base_url": base,
                "automation_username": auto_user,
                "automation_pat": token,
                "repo_http_url": repo_http,
                "repo_ssh_url": repo_ssh,
                "ssh_private_key": priv_key,
                "ssh_public_key": pub_key,
                "known_hosts": known_hosts,
            },
        )

        # Optionally add remote and push
        if self.cfg.enable_push:
            self._ensure_git_remote_and_push(remote=self.cfg.remote_name, http_url=repo_http, username=auto_user, token=token)

        log("Gitea: bootstrap complete")

    # --- Helpers ---
    def _get_gitea_pod(self) -> str:
        # Find a ready pod for deployment gitea
        cp = CommandRunner().run(
            [
                "kubectl",
                "-n",
                "gitea",
                "get",
                "pods",
                "-l",
                "app.kubernetes.io/name=gitea",
                "-o",
                "jsonpath={.items[0].metadata.name}",
            ],
            capture=True,
        )
        name = (cp.stdout or "").strip()
        if not name:
            raise RuntimeError("Gitea pod not found")
        return name

    def _ensure_user_cli(self, *, username: str, password: str, admin: bool) -> None:
        pod = self._get_gitea_pod()
        args = [
            "kubectl",
            "-n",
            "gitea",
            "exec",
            pod,
            "--",
            "/usr/local/bin/gitea",
            "admin",
            "user",
            "create",
            "--username",
            username,
            "--password",
            password,
            "--email",
            f"{username}@local",
            "--must-change-password=false",
        ]
        if admin:
            args.append("--admin")
        # Try create; if fails (already exists), ignore
        try:
            CommandRunner().run(args, check=True, redact=[password])
        except Exception:
            warn(f"User '{username}' may already exist; continuing")

    def _ensure_user_pat_user(self, *, username: str, password: str, token_name: str) -> str:
        sess = self.deps.requests_session
        sess.auth = (username, password)
        base = str(self.cfg.base_url).rstrip("/")
        url = f"{base}/api/v1/users/{username}/tokens"
        payload = {"name": token_name}
        # Include scopes for Gitea versions that require them
        scopes = list(self.cfg.token_scopes or [])
        if scopes:
            payload["scopes"] = scopes
        r = sess.post(url, json=payload, timeout=10)
        if r.status_code in (200, 201):
            data = r.json()
            token = data.get("sha1") or data.get("token")
            if token:
                return str(token)
        # If server complains about scopes, retry once with repo RW scopes
        if r.status_code == 400 and "scope" in (r.text or "").lower():
            fallback = {"name": token_name, "scopes": ["read:repository", "write:repository"]}
            r2 = sess.post(url, json=fallback, timeout=10)
            if r2.status_code in (200, 201):
                data = r2.json()
                token = data.get("sha1") or data.get("token")
                if token:
                    return str(token)
        # If token name already exists, delete it and recreate for idempotence
        if r.status_code == 400 and "name has been used" in (r.text or "").lower():
            # List tokens to find id by name
            lr = sess.get(url, timeout=10)
            if lr.status_code == 200 and isinstance(lr.json(), list):
                items = lr.json()
                token_id = None
                for it in items:
                    if str(it.get("name")) == token_name:
                        token_id = it.get("id") or it.get("token_id")
                        break
                if token_id is not None:
                    dr = sess.delete(f"{url}/{token_id}", timeout=10)
                    if dr.status_code in (200, 204):
                        cr = sess.post(url, json=payload, timeout=10)
                        if cr.status_code in (200, 201):
                            data = cr.json()
                            token = data.get("sha1") or data.get("token")
                            if token:
                                return str(token)
        raise RuntimeError(f"Failed to create token via user API: {r.status_code} {r.text}")

    def _ensure_repository(self, *, owner: str, repo: str, token: str) -> None:
        base = str(self.cfg.base_url).rstrip("/")
        sess = self.deps.requests_session
        # Use token auth
        sess.headers.update({"Authorization": f"token {token}"})
        # Check existence
        r = sess.get(f"{base}/api/v1/repos/{owner}/{repo}", timeout=10)
        if r.status_code == 200:
            return
        # Create under user
        payload = {"name": repo, "private": False}
        r = sess.post(f"{base}/api/v1/user/repos", json=payload, timeout=10)
        if r.status_code not in (200, 201):
            raise RuntimeError(f"Failed to create repository: {r.status_code} {r.text}")

    def _ensure_ssh_keypair(self) -> tuple[str, str]:
        from tempfile import TemporaryDirectory

        with TemporaryDirectory() as td:
            priv = Path(td) / "id_ed25519"
            pub = Path(td) / "id_ed25519.pub"
            CommandRunner().run(["ssh-keygen", "-t", "ed25519", "-N", "", "-f", str(priv)])
            return priv.read_text(), pub.read_text()

    def _ssh_known_hosts(self) -> str:
        try:
            from urllib.parse import urlparse
            host = urlparse(str(self.cfg.base_url)).netloc
        except Exception:
            host = str(self.cfg.base_url)
        host = host.replace("https://", "").replace("http://", "").strip("/")
        try:
            cp = CommandRunner().run(["ssh-keyscan", "-t", "rsa,ecdsa,ed25519", host], capture=True, check=False)
            return (cp.stdout or "").strip()
        except Exception:
            return ""

    def _ensure_deploy_key(self, *, owner: str, repo: str, token: str, title: str, pubkey: str) -> None:
        base = str(self.cfg.base_url).rstrip("/")
        sess = self.deps.requests_session
        sess.headers.update({"Authorization": f"token {token}"})
        # Check if key exists by title
        try:
            r = sess.get(f"{base}/api/v1/repos/{owner}/{repo}/keys", timeout=10)
            if r.status_code == 200 and isinstance(r.json(), list):
                for k in r.json():
                    if str(k.get("title")) == title:
                        return
        except Exception:
            pass
        payload = {"title": title, "key": pubkey.strip(), "read_only": True}
        r = sess.post(f"{base}/api/v1/repos/{owner}/{repo}/keys", json=payload, timeout=10)
        if r.status_code not in (200, 201):
            warn(f"Failed to add deploy key: {r.status_code} {r.text}")

    def _ensure_git_remote_and_push(self, *, remote: str, http_url: str, username: str, token: str) -> None:
        repo_root = get_repo_root(Path(__file__).resolve())
        # Add remote if missing
        cp = CommandRunner().run(["git", "-C", str(repo_root), "remote"], capture=True)
        remotes = (cp.stdout or "").split()
        url_with_auth = http_url.replace("https://", f"https://{username}:{token}@")
        if remote not in remotes:
            CommandRunner().run(["git", "-C", str(repo_root), "remote", "add", remote, url_with_auth], redact=[token])
        else:
            # Update URL to ensure it contains auth
            CommandRunner().run(["git", "-C", str(repo_root), "remote", "set-url", remote, url_with_auth], redact=[token])
        # Determine branch
        cp = CommandRunner().run(["git", "-C", str(repo_root), "rev-parse", "--abbrev-ref", "HEAD"], capture=True)
        branch = (cp.stdout or "main").strip() or "main"
        # Configure per-URL CA bundle for this host so Git trusts mkcert root
        from urllib.parse import urlparse
        host = urlparse(http_url).scheme + "://" + (urlparse(http_url).netloc or "")
        try:
            ca_root = CommandRunner().run(["mkcert", "-CAROOT"], capture=True).stdout.strip()
            ca_pem = str(Path(ca_root) / "rootCA.pem")
            if ca_root:
                CommandRunner().run(["git", "-C", str(repo_root), "config", f"http.{host}.sslCAInfo", ca_pem])
        except Exception:
            pass
        # Push with upstream
        try:
            env = {}
            try:
                ca_root = CommandRunner().run(["mkcert", "-CAROOT"], capture=True).stdout.strip()
                ca_pem = str(Path(ca_root) / "rootCA.pem")
                env = {"GIT_SSL_CAINFO": ca_pem}
            except Exception:
                pass
            CommandRunner().run(["git", "-C", str(repo_root), "push", "-u", remote, branch], redact=[token], env=env)
        except Exception:
            warn("Git push failed (possibly already up to date)")


@register_setup_task(
    name="gitea",
    description="Bootstrap Gitea after Helm install (health, users, repo, secrets)",
    priority=52,  # run after helmfile foundational/app layer and before deploy
    dependencies=("runner",),
)
def _factory(ctx: ConfigState, deps) -> Optional[GiteaSetup]:
    # Read config from ctx.raw.tasks.gitea if present, else defaults
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw = raw_tasks.get("gitea") if isinstance(raw_tasks, dict) else None
    cfg = GiteaTaskConfig() if raw is None else GiteaTaskConfig.model_validate(raw)
    if not cfg.enabled:
        return None
    return GiteaSetup(cfg=cfg, deps=deps)



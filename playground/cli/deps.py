# ruff: noqa: E402
from __future__ import annotations

"""Dependency container for setup tasks.

Provides lazily-constructed shared dependencies: command runner, database root,
and Kubernetes clients. Tasks should declare dependencies in the registry and
the setup executor will construct only those requested.
"""

from dataclasses import dataclass
import os
from pathlib import Path
from typing import Any, Dict, Optional
import subprocess
import requests

from .config import ConfigState
from .db.base import DatabaseConfig, DatabaseRoot
from .utils import require_cmd
from .runner import CommandRunner
from .logging import get_logger


@dataclass
class Deps:
    state: ConfigState

    _runner: Optional[CommandRunner] = None
    _db: Optional[DatabaseRoot] = None
    _k8s_loaded: bool = False
    _k8s_clients: Dict[str, Any] | None = None
    _http_session: Optional[requests.Session] = None

    @property
    def runner(self) -> CommandRunner:
        if self._runner is None:
            self._runner = CommandRunner()
        return self._runner

    @property
    def db(self) -> DatabaseRoot:
        if self._db is None:
            # Read defaults from deps.db if present
            raw_deps = (self.state.raw or {}).get("deps", {})
            raw_db = raw_deps.get("db", {}) if isinstance(raw_deps, dict) else {}
            cfg = DatabaseConfig(
                host=raw_db.get("host", DatabaseConfig.host),
                port=raw_db.get("port", DatabaseConfig.port),
                user=raw_db.get("user", DatabaseConfig.user),
                password=raw_db.get("password", DatabaseConfig.password),
                admin_db=raw_db.get("admin_db", DatabaseConfig.admin_db),
            )
            self._db = DatabaseRoot(cfg)
        return self._db

    @property
    def k8s(self) -> Dict[str, Any]:
        if not self._k8s_loaded:
            require_cmd("kubectl")
            # Lazy import only when needed
            from kubernetes import client, config  # type: ignore

            raw_deps = (self.state.raw or {}).get("deps", {})
            raw_k8s = raw_deps.get("k8s", {}) if isinstance(raw_deps, dict) else {}

            # Resolution order:
            # 1) deps.k8s.kubeconfig (if provided)
            # 2) First existing file from $KUBECONFIG (PATH-like)
            # 3) <playground_dir>/kubeconfig (default)
            # Anchor default to the playground directory containing this CLI package.
            playground_dir = Path(__file__).resolve().parent.parent  # .../playground
            explicit = raw_k8s.get("kubeconfig")
            env_cfg = os.environ.get("KUBECONFIG", "")
            env_candidates = [p for p in env_cfg.split(os.pathsep) if p]

            kubeconfig_candidate: Path | None = None
            if explicit:
                kubeconfig_candidate = Path(str(explicit)).expanduser().resolve()
            else:
                for p in env_candidates:
                    path = Path(p).expanduser().resolve()
                    if path.exists() and path.is_file():
                        kubeconfig_candidate = path
                        break
                if kubeconfig_candidate is None:
                    kubeconfig_candidate = (playground_dir / "kubeconfig").resolve()

            # Validate and load
            logger = get_logger("console")

            if not kubeconfig_candidate.exists() or kubeconfig_candidate.stat().st_size == 0:
                logger.error(
                    "Kubeconfig not found or empty: "
                    f"{kubeconfig_candidate}. Run 'k3d' to create the cluster or set deps.k8s.kubeconfig."
                )
                raise SystemExit(1)

            try:
                config.load_kube_config(config_file=str(kubeconfig_candidate))
            except Exception:
                logger.error(
                    "Invalid kubeconfig. Ensure it points to a valid cluster. "
                    "Update deps.k8s.kubeconfig or export KUBECONFIG."
                )
                raise
            self._k8s_clients = {
                "core": client.CoreV1Api(),
                "apps": client.AppsV1Api(),
            }
            self._k8s_loaded = True
        return self._k8s_clients or {}

    @property
    def requests_session(self) -> requests.Session:
        """Return a requests Session configured to trust the local mkcert root CA.

        We assume the repository generates and stores the CA at `.certs/rootCA.pem`.
        This path is resolved relative to the repository root inferred from the
        known config path at `playground/config.cue`.
        """
        if self._http_session is None:
            # Derive repo root from the config path (…/playground/config.cue)
            cfg_path = self.state.path.resolve()
            repo_root = cfg_path.parent.parent
            ca_path = repo_root / ".certs/rootCA.pem"

            sess = requests.Session()
            if ca_path.exists():
                sess.verify = str(ca_path)
            else:
                # Fallback: try mkcert -CAROOT/rootCA.pem
                try:
                    out = subprocess.run(
                        ["mkcert", "-CAROOT"],
                        check=True,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                        text=True,
                    )
                    ca_dir = Path((out.stdout or "").strip())
                    alt = ca_dir / "rootCA.pem"
                    if alt.exists():
                        sess.verify = str(alt)
                except Exception:
                    pass
            self._http_session = sess
        return self._http_session

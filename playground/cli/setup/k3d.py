from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Optional
import json
import os
import time

import yaml

from pydantic import BaseModel, Field

from ..config import ConfigState
from ..utils import log, require_cmd, warn, get_repo_root
from ..runner import CommandRunner
from ..models import ClusterSummary
from .base import SetupTask
from .registry import register_setup_task


# Localized k3d helpers (inlined from previous k3d_ops to make task self-contained)

def cluster_exists(name: str) -> bool:
    runner = CommandRunner()
    try:
        cp = runner.run(["k3d", "cluster", "list", "-o", "json"], capture=True)
        data = json.loads(cp.stdout or "{}")
        clusters = data.get("clusters", [])
        return any(c.get("name") == name for c in clusters)
    except Exception:
        cp = runner.run(["k3d", "cluster", "list"], capture=True)
        return name in (cp.stdout or "")


def _build_k3d_create_args(
    name: str,
    servers: int,
    agents: int,
    http_port: int,
    https_port: int,
    api_port: int,
) -> list[str]:
    args = [
        "k3d",
        "cluster",
        "create",
        name,
        "--servers",
        str(servers),
        "--agents",
        str(agents),
        "--k3s-arg",
        "--disable=traefik@server:0",
        "-p",
        f"{http_port}:80@loadbalancer",
        "-p",
        f"{https_port}:443@loadbalancer",
        "-p",
        "8372:8372@loadbalancer",
        "-p",
        "5432:5432@loadbalancer",
        "--wait",
    ]
    if api_port != 0:
        args.extend(["--api-port", str(api_port)])
    return args


def create_cluster(
    name: str,
    servers: int,
    agents: int,
    http_port: int,
    https_port: int,
    api_port: int,
    extra_volumes: list[str] | None = None,
) -> None:
    args = _build_k3d_create_args(name, servers, agents, http_port, https_port, api_port)
    for vol in extra_volumes or []:
        args.extend(["--volume", vol])
    log(
        f"Creating k3d cluster '{name}' (servers={servers}, agents={agents}) "
        f"with host ports {http_port}/HTTP, {https_port}/HTTPS and 8372/tcp (buildkitd), 5432/tcp (postgres)..."
    )
    CommandRunner().run(args)


def delete_cluster(name: str) -> None:
    log(f"Deleting k3d cluster '{name}'...")
    CommandRunner().run(["k3d", "cluster", "delete", name], check=False)


def write_kubeconfig(name: str, out_path: Path, assume_yes: bool) -> None:
    tmp = CommandRunner().run(["k3d", "kubeconfig", "get", name], capture=True)
    new_text = str(tmp.stdout or "")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    if out_path.exists():
        try:
            current = out_path.read_text()
            if current == new_text:
                log(f"Kubeconfig unchanged at {out_path}")
                return
        except Exception:
            pass
    # Always write updated kubeconfig to avoid stale server/port causing readiness hangs
    out_path.write_text(new_text)
    if out_path.exists():
        log(f"Kubeconfig updated at {out_path}")
    else:
        log(f"Kubeconfig written to {out_path}")


def wait_for_nodes_ready(kubeconfig: Path, timeout_s: int = 120) -> None:
    """Poll Kubernetes API for node readiness using the Python client.

    Args:
        kubeconfig: Path to the kubeconfig file to load.
        timeout_s: Maximum number of seconds to wait.
    """
    log(f"Waiting for nodes to be Ready using kubeconfig: {kubeconfig}")

    try:
        from kubernetes import client as k8s_client, config as k8s_config  # type: ignore
    except Exception as _e:  # pragma: no cover - import-time environment issue
        from ..utils import err  # local import to avoid top-level cycle

        err("Failed to import kubernetes client; ensure 'kubernetes' package is installed")
        raise

    try:
        # Load kubeconfig directly; do not mutate or normalize
        k8s_config.load_kube_config(config_file=str(kubeconfig))
        try:
            contexts, active = k8s_config.list_kube_config_contexts()
            active_name = (active or {}).get("name") if isinstance(active, dict) else None
            if active_name:
                log(f"Active kube context: {active_name}")
                try:
                    # Log active cluster server URL for visibility only
                    raw_cfg = yaml.safe_load(kubeconfig.read_text())
                    active_cluster_name = (active or {}).get("context", {}).get("cluster")
                    cluster_entries = raw_cfg.get("clusters", []) if isinstance(raw_cfg, dict) else []
                    for c in cluster_entries:
                        if c.get("name") == active_cluster_name:
                            server_url = ((c or {}).get("cluster") or {}).get("server")
                            if server_url:
                                log(f"API server from kubeconfig: {server_url}")
                            break
                except Exception:
                    pass
        except Exception:
            pass
    except Exception:
        from ..utils import err  # local import

        err(f"Unable to load kubeconfig at {kubeconfig}")
        raise

    api = k8s_client.CoreV1Api()
    start = time.time()
    first_error_logged = False
    last_progress_log = 0.0
    last_error_log = 0.0
    while time.time() - start < timeout_s:
        try:
            nodes = api.list_node().items
            if not nodes:
                time.sleep(2)
                continue

            all_ready = True
            for n in nodes:
                conditions = n.status.conditions or []
                ready = any(
                    getattr(c, "type", None) == "Ready" and getattr(c, "status", None) == "True"
                    for c in conditions
                )
                name = getattr(getattr(n, "metadata", None), "name", "?")
                log(f"node={name} ready={ready}")
                if not ready:
                    all_ready = False
                    break

            if all_ready:
                # Emit a brief summary
                names = ", ".join(getattr(n.metadata, "name", "?") for n in nodes)
                log(f"All nodes Ready: {names}")
                return
            # Periodic progress log (every ~5s)
            now = time.time()
            if now - last_progress_log > 5:
                pending = ", ".join(
                    getattr(n.metadata, "name", "?")
                    for n in nodes
                    if not any(
                        getattr(c, "type", None) == "Ready" and getattr(c, "status", None) == "True"
                        for c in (n.status.conditions or [])
                    )
                )
                if pending:
                    log(f"Waiting for Ready nodes: {pending}")
                last_progress_log = now
        except Exception as e:
            # Transient errors while API server settles
            now = time.time()
            if not first_error_logged or (now - last_error_log > 10):
                try:
                    from ..utils import warn as _warn

                    _warn(f"K8s API not ready yet: {e}")
                except Exception:
                    pass
                first_error_logged = True
                last_error_log = now
        time.sleep(2)

    from ..utils import err  # local import to avoid top-level cycle

    err("Nodes did not become Ready within the timeout")
    raise SystemExit(1)


def get_k8s_version(kubeconfig: Path) -> str:
    """Return the Kubernetes server version

    Args:
        kubeconfig: Path to kubeconfig for connecting to the cluster.

    Returns:
        A compact version string; empty on failure.
    """
    try:
        from kubernetes import client as k8s_client, config as k8s_config  # type: ignore

        k8s_config.load_kube_config(config_file=str(kubeconfig))
        v = k8s_client.VersionApi().get_code()
        parts: list[str] = []
        if getattr(v, "git_version", None):
            parts.append(str(v.git_version))
        elif getattr(v, "major", None) and getattr(v, "minor", None):
            parts.append(f"v{v.major}.{v.minor}")
        if getattr(v, "platform", None):
            parts.append(f"({v.platform})")
        return " ".join(parts) if parts else ""
    except Exception:
        return ""


def emit_cluster_json(
    path: Path,
    name: str,
    servers: int,
    agents: int,
    http_port: int,
    https_port: int,
    kubeconfig: Path,
) -> None:
    summary = ClusterSummary(
        name=name,
        type="k3d",
        servers=servers,
        agents=agents,
        host_ip="127.0.0.1",
        http_port=http_port,
        https_port=https_port,
        kubeconfig=str(kubeconfig),
        kubernetes_version=get_k8s_version(kubeconfig),
    )
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(summary.model_dump_json(by_alias=True, indent=2))
    log(f"Cluster summary written to {path}")


def get_mkcert_caroot() -> Path:
    cp = CommandRunner().run(["mkcert", "-CAROOT"], capture=True)
    return Path(str(cp.stdout or "").strip())


def write_registries_yaml(tmpdir: Path, registry_host: str, ca_path: Path) -> Path:
    data = {
        "mirrors": {registry_host: {"endpoint": [f"https://{registry_host}"]}},
        "configs": {registry_host: {"tls": {"ca_file": str(ca_path)}}},
    }
    path = tmpdir / "registries.yaml"
    path.write_text(yaml.safe_dump(data, sort_keys=False))
    return path


class K3dTaskConfig(BaseModel):
    """Configuration for k3d cluster bootstrap as a setup task.

    Fields mirror the previous `k3d` subcommand flags and provide sensible
    defaults for local development.
    """

    enabled: bool = Field(default=True)
    name: str = Field(default="forge")
    servers: int = Field(default=1, ge=1)
    agents: int = Field(default=0, ge=0)
    http_port: int = Field(default=80, ge=1, le=65535)
    https_port: int = Field(default=443, ge=1, le=65535)
    api_port: int = Field(default=0, ge=0, le=65535)
    kubeconfig_out: str = Field(default="playground/kubeconfig")
    output_json: str = Field(default="playground/cluster.json")
    force_recreate: bool = Field(default=False)
    assume_yes: bool = Field(default=False)
    registry_host: str = Field(default="registry.projectcatalyst.dev")


@dataclass
class K3dSetup(SetupTask):
    cfg: K3dTaskConfig

    def run(self) -> None:
        if not self.cfg.enabled:
            log("k3d setup task disabled; skipping")
            return

        # Preflight required CLIs
        for cmd in ("docker", "k3d", "mkcert"):
            require_cmd(cmd)

        # Resolve playground directory from this file's location and repo root
        playground_dir = Path(__file__).resolve().parent.parent  # .../playground
        repo_root = get_repo_root(playground_dir)

        # Prepare mkcert CA and k3s registries.yaml with trusted CA
        caroot = get_mkcert_caroot()
        ca_file = caroot / "rootCA.pem"
        tmpdir = (playground_dir / ".certs").resolve()
        tmpdir.mkdir(parents=True, exist_ok=True)
        registries_yaml = write_registries_yaml(
            tmpdir=tmpdir,
            registry_host=self.cfg.registry_host,
            ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt"),
        ).resolve()

        volume_mounts = [
            f"{ca_file}:/etc/ssl/certs/mkcert-rootCA.crt@server:*;agent:*",
            f"{registries_yaml}:/etc/rancher/k3s/registries.yaml@server:*;agent:*",
        ]

        # Create or reuse cluster
        if cluster_exists(self.cfg.name):
            log(f"Cluster '{self.cfg.name}' already exists.")
            if self.cfg.force_recreate:
                delete_cluster(self.cfg.name)
                create_cluster(
                    self.cfg.name,
                    self.cfg.servers,
                    self.cfg.agents,
                    self.cfg.http_port,
                    self.cfg.https_port,
                    self.cfg.api_port,
                    extra_volumes=volume_mounts,
                )
            else:
                log("Reusing existing cluster.")
        else:
            create_cluster(
                self.cfg.name,
                self.cfg.servers,
                self.cfg.agents,
                self.cfg.http_port,
                self.cfg.https_port,
                self.cfg.api_port,
                extra_volumes=volume_mounts,
            )

        # Kubeconfig and readiness (anchor relative paths to repo root)
        kubeconfig_out = Path(self.cfg.kubeconfig_out).expanduser()
        if not kubeconfig_out.is_absolute():
            kubeconfig_out = (repo_root / kubeconfig_out).resolve()
        write_kubeconfig(self.cfg.name, kubeconfig_out, assume_yes=self.cfg.assume_yes)
        wait_for_nodes_ready(kubeconfig_out)

        # Emit summary JSON for automation
        output_json_path = Path(self.cfg.output_json).expanduser()
        if not output_json_path.is_absolute():
            output_json_path = (repo_root / output_json_path).resolve()
        emit_cluster_json(
            path=output_json_path,
            name=self.cfg.name,
            servers=self.cfg.servers,
            agents=self.cfg.agents,
            http_port=self.cfg.http_port,
            https_port=self.cfg.https_port,
            kubeconfig=kubeconfig_out,
        )


@register_setup_task(
    name="k3d",
    description="Create or reuse a local k3d cluster and write kubeconfig",
    priority=20,
    dependencies=(),
)
def _factory(ctx: ConfigState, _deps) -> Optional[K3dSetup]:
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw_cfg = raw_tasks.get("k3d") if isinstance(raw_tasks, dict) else None
    cfg = K3dTaskConfig() if raw_cfg is None else K3dTaskConfig.model_validate(raw_cfg)
    if not cfg.enabled:
        return None
    return K3dSetup(cfg=cfg)

"""Operations for managing k3d clusters within the SetupV2 k3d task.

Functions in this module wrap k3d and kubectl to create, delete, and inspect
local clusters, as well as write configuration artifacts consumed by the k3d task.
All functions are designed to be idempotent where reasonable.
"""

from __future__ import annotations

import json
import time
from pathlib import Path

import yaml

from .....runner import CommandRunner
from .....models import ClusterSummary
from .....logging import get_logger


def cluster_exists(name: str) -> bool:
    """Return True if a k3d cluster with the given name exists.

    Args:
        name: The k3d cluster name to query.

    Returns:
        True if the cluster exists, else False.
    """
    runner = CommandRunner()
    try:
        cp = runner.run(["k3d", "cluster", "list", "-o", "json"], capture=True)
        data = json.loads(cp.stdout or "{}")
        clusters = data.get("clusters", [])
        return any(c.get("name") == name for c in clusters)
    except Exception:
        cp = runner.run(["k3d", "cluster", "list"], capture=True)
        return name in (cp.stdout or "")


def build_k3d_create_args(
    name: str,
    servers: int,
    agents: int,
    http_port: int,
    https_port: int,
    api_port: int,
) -> list[str]:
    """Build the argument vector for `k3d cluster create`.

    Args:
        name: Cluster name.
        servers: Number of server nodes.
        agents: Number of agent nodes.
        http_port: Host HTTP port mapped to load balancer 80.
        https_port: Host HTTPS port mapped to load balancer 443.
        api_port: Optional host API port mapping (0 disables mapping).

    Returns:
        A list of CLI arguments suitable for subprocess execution.
    """
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
    """Create a k3d cluster with ports and optional extra volume mounts.

    Args:
        name: Cluster name.
        servers: Number of server nodes.
        agents: Number of agent nodes.
        http_port: Host HTTP port mapped to 80.
        https_port: Host HTTPS port mapped to 443.
        api_port: Optional host API port for kube-apiserver.
        extra_volumes: Extra `--volume` specs to pass to k3d.
    """
    args = build_k3d_create_args(name, servers, agents, http_port, https_port, api_port)
    for vol in extra_volumes or []:
        args.extend(["--volume", vol])

    logger = get_logger("console")
    logger.info(
        f"Creating k3d cluster '{name}' (servers={servers}, agents={agents}) "
        f"with host ports {http_port}/HTTP, {https_port}/HTTPS and 8372/tcp (buildkitd), 5432/tcp (postgres)..."
    )
    CommandRunner().run(args)


def delete_cluster(name: str) -> None:
    """Delete a k3d cluster if it exists.

    Args:
        name: Cluster name.
    """
    logger = get_logger("console")
    logger.info(f"Deleting k3d cluster '{name}'...")
    CommandRunner().run(["k3d", "cluster", "delete", name], check=False)


def write_kubeconfig(name: str, out_path: Path, assume_yes: bool = True) -> None:
    """Write kubeconfig for the cluster to a path, optionally confirming overwrite.

    Args:
        name: Cluster name.
        out_path: Destination path for the kubeconfig file.
        assume_yes: Overwrite without prompting if True.
    """
    tmp = CommandRunner().run(["k3d", "kubeconfig", "get", name], capture=True)
    new_text = str(tmp.stdout or "")
    out_path.parent.mkdir(parents=True, exist_ok=True)

    # Check if content has changed
    if out_path.exists():
        try:
            current = out_path.read_text()
            if current == new_text:
                logger = get_logger("console")
                logger.info(f"Kubeconfig unchanged at {out_path}")
                return
        except Exception:
            pass

    # Always write updated kubeconfig to avoid stale server/port causing readiness hangs
    out_path.write_text(new_text)


def wait_for_nodes_ready(kubeconfig: Path, timeout_s: int = 120) -> None:
    """Poll Kubernetes API for node readiness using the Python client.

    Args:
        kubeconfig: Path to the kubeconfig file to load.
        timeout_s: Maximum number of seconds to wait.
    """
    logger = get_logger("console")
    logger.info(f"Waiting for nodes to be Ready using kubeconfig: {kubeconfig}")

    try:
        from kubernetes import client as k8s_client, config as k8s_config  # type: ignore
    except Exception as _e:  # pragma: no cover - import-time environment issue
        logger.error("Failed to import kubernetes client; ensure 'kubernetes' package is installed")
        raise

    try:
        # Load kubeconfig directly; do not mutate or normalize
        k8s_config.load_kube_config(config_file=str(kubeconfig))
        try:
            contexts, active = k8s_config.list_kube_config_contexts()
            active_name = (active or {}).get("name") if isinstance(active, dict) else None
            if active_name:
                logger.info(f"Active kube context: {active_name}")
                try:
                    # Log active cluster server URL for visibility only
                    raw_cfg = yaml.safe_load(kubeconfig.read_text())
                    active_cluster_name = (active or {}).get("context", {}).get("cluster")
                    cluster_entries = (
                        raw_cfg.get("clusters", []) if isinstance(raw_cfg, dict) else []
                    )
                    for c in cluster_entries:
                        if c.get("name") == active_cluster_name:
                            server_url = ((c or {}).get("cluster") or {}).get("server")
                            if server_url:
                                logger.info(f"API server from kubeconfig: {server_url}")
                            break
                except Exception:
                    pass
        except Exception:
            pass
    except Exception:
        logger.error(f"Unable to load kubeconfig at {kubeconfig}")
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
                logger.info(f"node={name} ready={ready}")
                if not ready:
                    all_ready = False
                    break

            if all_ready:
                # Emit a brief summary
                names = ", ".join(getattr(n.metadata, "name", "?") for n in nodes)
                logger.info(f"All nodes Ready: {names}")
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
                    logger.info(f"Waiting for Ready nodes: {pending}")
                last_progress_log = now
        except Exception as e:
            # Transient errors while API server settles
            now = time.time()
            if not first_error_logged or (now - last_error_log > 10):
                logger.warning(f"K8s API not ready yet: {e}")
                first_error_logged = True
                last_error_log = now
        time.sleep(2)

    logger.error("Nodes did not become Ready within the timeout")
    raise SystemExit(1)


def get_k8s_version(kubeconfig: Path) -> str:
    """Return the Kubernetes server version.

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
    """Write a JSON summary of the cluster to `path`.

    Args:
        path: Destination file path for the cluster summary JSON.
        name: Cluster name.
        servers: Number of server nodes.
        agents: Number of agent nodes.
        http_port: Host HTTP port.
        https_port: Host HTTPS port.
        kubeconfig: Path to kubeconfig file.
    """
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


def write_registries_yaml(tmpdir: Path, registry_host: str, ca_path: Path) -> Path:
    """Write a registries.yaml that configures TLS with CA trust.

    Args:
        tmpdir: Temporary directory for the file.
        registry_host: The registry hostname to configure.
        ca_path: Path to the trusted CA certificate file.

    Returns:
        Absolute path to the written `registries.yaml` file.
    """
    data = {
        "mirrors": {
            registry_host: {
                "endpoint": [f"https://{registry_host}"],
            }
        },
        "configs": {
            registry_host: {
                "tls": {
                    "ca_file": str(ca_path),
                }
            }
        },
    }
    path = tmpdir / "registries.yaml"
    path.write_text(yaml.safe_dump(data, sort_keys=False))
    return path

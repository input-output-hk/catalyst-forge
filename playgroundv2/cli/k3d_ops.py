"""Operations for managing k3d clusters used by the Playground v2 CLI.

Functions in this module wrap k3d and kubectl to create, delete, and inspect
local clusters, as well as write configuration artifacts consumed by other
tools. All functions are designed to be idempotent where reasonable.
"""

from __future__ import annotations

import json
import os
import time
from pathlib import Path

import yaml

from .models import ClusterSummary
from .utils import err, log, run, warn


def cluster_exists(name: str) -> bool:
    """Return True if a k3d cluster with the given name exists.

    Args:
        name: The k3d cluster name to query.

    Returns:
        True if the cluster exists, else False.
    """
    try:
        cp = run(["k3d", "cluster", "list", "-o", "json"], capture=True)
        data = json.loads(cp.stdout or "{}")
        clusters = data.get("clusters", [])
        return any(c.get("name") == name for c in clusters)
    except Exception:
        cp = run(["k3d", "cluster", "list"], capture=True)
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
    log(
        f"Creating k3d cluster '{name}' (servers={servers}, agents={agents}) "
        f"with host ports {http_port}/HTTP, {https_port}/HTTPS and 8372/tcp (buildkitd), 5432/tcp (postgres)..."
    )
    run(args)


def delete_cluster(name: str) -> None:
    """Delete a k3d cluster if it exists.

    Args:
        name: Cluster name.
    """
    log(f"Deleting k3d cluster '{name}'...")
    run(["k3d", "cluster", "delete", name], check=False)


def write_kubeconfig(name: str, out_path: Path, assume_yes: bool) -> None:
    """Write kubeconfig for the cluster to a path, optionally confirming overwrite.

    Args:
        name: Cluster name.
        out_path: Destination path for the kubeconfig file.
        assume_yes: Overwrite without prompting if True.
    """
    tmp = run(["k3d", "kubeconfig", "get", name], capture=True)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    if out_path.exists() and not assume_yes:
        warn(f"Kubeconfig already exists at {out_path}; overwrite with --yes to replace.")
        return
    out_path.write_text(str(tmp.stdout or ""))
    log(f"Kubeconfig written to {out_path}")


def wait_for_nodes_ready(kubeconfig: Path, timeout_s: int = 120) -> None:
    """Poll until all nodes report Ready within a timeout.

    Args:
        kubeconfig: Path to the kubeconfig to use for kubectl.
        timeout_s: Timeout in seconds before failing.
    """
    env = {**os.environ, "KUBECONFIG": str(kubeconfig)}
    log("Waiting for nodes to be Ready...")
    start = time.time()
    while time.time() - start < timeout_s:
        try:
            cp = run(["kubectl", "get", "nodes", "-o", "json"], capture=True, env=env)
            data = json.loads(cp.stdout or "{}")
            items = data.get("items", [])
            if not items:
                time.sleep(2)
                continue
            all_ready = True
            for node in items:
                conditions = node.get("status", {}).get("conditions", [])
                ready = any(
                    c.get("type") == "Ready" and c.get("status") == "True" for c in conditions
                )
                if not ready:
                    all_ready = False
                    break
            if all_ready:
                run(["kubectl", "get", "nodes", "-o", "wide"], env=env)
                return
        except Exception:
            pass
        time.sleep(2)
    err("Nodes did not become Ready within the timeout")
    raise SystemExit(1)


def get_k8s_version(kubeconfig: Path) -> str:
    """Return a compact `kubectl version --short` string.

    Args:
        kubeconfig: Path to the kubeconfig to use for kubectl.

    Returns:
        String summary of client/server versions or empty string on failure.
    """
    env = {**os.environ, "KUBECONFIG": str(kubeconfig)}
    try:
        cp = run(["kubectl", "version", "--short"], capture=True, env=env)
        return " ".join(str(cp.stdout or "").split())
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
    path.write_text(summary.model_dump_json(by_alias=True, indent=2))
    log(f"Cluster summary written to {path}")


def get_mkcert_caroot() -> Path:
    """Return the mkcert CA root directory."""
    cp = run(["mkcert", "-CAROOT"], capture=True)
    return Path(str(cp.stdout or "").strip())


## intentionally no k3d registry helpers; use Helmfile for registry TLS


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

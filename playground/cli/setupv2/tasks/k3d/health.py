"""
Health check for k3d cluster task.

Verifies that the kubeconfig exists and that all Kubernetes nodes report Ready.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any, Dict, IO, Mapping


def health_k3d(ctx: Mapping[str, Any], cfg: Dict[str, Any], log: IO) -> None:
    """
    Validate that the k3d cluster is actually usable.

    Checks:
    - Kubeconfig file exists and is non-empty
    - Kubernetes API reachable and nodes are Ready
    """
    kubeconfig_path = Path(str(ctx.get("k3d.kubeconfig", ""))).expanduser()

    if not kubeconfig_path.exists() or kubeconfig_path.stat().st_size == 0:
        raise RuntimeError(f"Kubeconfig missing or empty: {kubeconfig_path}")

    try:
        from kubernetes import client as k8s_client, config as k8s_config  # type: ignore
    except Exception as e:  # pragma: no cover - import-time environment issue
        raise RuntimeError(f"Failed to import kubernetes client: {e}")

    # Attempt to connect and validate node readiness
    try:
        k8s_config.load_kube_config(config_file=str(kubeconfig_path))
        api = k8s_client.CoreV1Api()
        nodes = api.list_node().items
        if not nodes:
            raise RuntimeError("No Kubernetes nodes found")

        not_ready = []
        for n in nodes:
            name = getattr(getattr(n, "metadata", None), "name", "?")
            conditions = getattr(getattr(n, "status", None), "conditions", []) or []
            ready = any(
                getattr(c, "type", None) == "Ready" and getattr(c, "status", None) == "True"
                for c in conditions
            )
            if not ready:
                not_ready.append(name)

        if not_ready:
            raise RuntimeError(f"Nodes not Ready: {', '.join(not_ready)}")

        log.write("K3d health: nodes Ready and API reachable\n")
    except Exception as e:
        raise RuntimeError(f"K3d health check failed: {e}")

"""Pydantic models used by the Playground v2 CLI."""

from __future__ import annotations

from pydantic import BaseModel, Field, ConfigDict


class ClusterSummary(BaseModel):
    """Structured summary of the created/reused cluster.

    Attributes:
        cluster_name: Name of the k3d cluster.
        cluster_type: Static string "k3d" for identification.
        servers: Number of server nodes.
        agents: Number of agent nodes.
        host_ip: Host IP for HTTP/HTTPS (127.0.0.1 when using k3d port mapping).
        http_port: Host HTTP port mapped to load balancer port 80.
        https_port: Host HTTPS port mapped to load balancer port 443.
        kubeconfig: Absolute path to the kubeconfig file written by the CLI.
        kubernetes_version: Output of `kubectl version --short` for visibility.
    """

    cluster_name: str = Field(..., alias="name")
    cluster_type: str = Field("k3d", alias="type")
    servers: int
    agents: int
    host_ip: str
    http_port: int
    https_port: int
    kubeconfig: str
    kubernetes_version: str

    # Pydantic v2 configuration
    model_config = ConfigDict(populate_by_name=True)

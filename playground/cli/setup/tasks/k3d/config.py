"""
Configuration models for k3d task.
"""

from pydantic import BaseModel


class K3dConfig(BaseModel):
    """Configuration for k3d Kubernetes cluster creation and management.

    This model defines all the parameters needed to create and configure
    a local Kubernetes cluster using k3d for the Catalyst Forge playground.
    """

    cluster_name: str = "forge"
    """Name of the k3d cluster to create or manage.

    This will be the identifier used by k3d commands and kubectl contexts.
    Default: "forge" (matches the playground environment naming)
    """

    servers: int = 1
    """Number of server nodes to create in the cluster.

    Server nodes run the Kubernetes control plane components.
    For development environments, 1 server is typically sufficient.
    Default: 1
    """

    agents: int = 0
    """Number of agent (worker) nodes to create in the cluster.

    Agent nodes run the application workloads. For playground development,
    running everything on server nodes is often sufficient.
    Default: 0 (run workloads on server nodes)
    """

    http_port: int = 80
    """Host port to map for HTTP traffic (port 80 inside containers).

    This allows external access to services running on port 80 in the cluster.
    Used for HTTP-based services like web applications.
    Default: 80 (standard HTTP port)
    """

    https_port: int = 443
    """Host port to map for HTTPS traffic (port 443 inside containers).

    This allows external access to TLS-encrypted services in the cluster.
    Used for secure web applications and APIs.
    Default: 443 (standard HTTPS port)
    """

    api_port: int = 0
    """Host port to map for Kubernetes API server access.

    The Kubernetes API server port inside the cluster. When set to 0,
    k3d will auto-assign an available host port.
    Default: 0 (auto-assign)
    """

    kubeconfig_out: str = "playground/kubeconfig"
    """Path where the cluster's kubeconfig file will be written.

    This file contains the connection details and credentials needed to
    access the cluster with kubectl and other Kubernetes tools.
    Default: "playground/kubeconfig"
    """

    output_json: str = "playground/cluster.json"
    """Path where cluster metadata will be written as JSON.

    Contains information about the created cluster including node details,
    ports, and connection information for use by other tools.
    Default: "playground/cluster.json"
    """

    force_recreate: bool = False
    """Whether to force recreation of an existing cluster with the same name.

    If True, any existing cluster with the same name will be deleted
    before creating a new one. If False, reuse existing cluster.
    Default: False (reuse existing)
    """

    assume_yes: bool = True
    """Automatically confirm prompts without user interaction.

    When True, skips interactive prompts and assumes "yes" for all
    confirmation questions. Essential for automated/CI environments.
    Default: True
    """

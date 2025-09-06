"""
Configuration models for registry task.
"""

from pydantic import BaseModel


class RegistryConfig(BaseModel):
    """Configuration for Docker Registry deployment.

    This model defines all the parameters needed to deploy and configure
    Docker Registry for container image storage in the playground.
    """

    namespace: str = "registry"
    """Kubernetes namespace where Docker Registry will be deployed.

    All Docker Registry components will be installed in this namespace.
    Default: "registry"
    """

    chart: str = "twuni/docker-registry"
    """Helm chart to use for Docker Registry deployment.

    Specifies the Helm chart repository and name for Docker Registry.
    Default: "twuni/docker-registry"
    """

    username: str = "admin"
    """Username for Docker Registry authentication.

    This username will be used for basic authentication to the registry.
    Default: "admin"
    """

    password: str = "registry-password-123"
    """Password for Docker Registry authentication.

    This password will be used for basic authentication to the registry.
    Default: "registry-password-123"
    """

    service_type: str = "ClusterIP"
    """Kubernetes service type for Docker Registry.

    Determines how the Docker Registry service is exposed.
    Options: ClusterIP, LoadBalancer, NodePort
    Default: "ClusterIP"
    """

    nodeport: int = 30500
    """NodePort for external access to Docker Registry.

    When using NodePort service type, this port will be used for external access.
    Default: 30500
    """

    persistence: dict = {"enabled": True, "size": "20Gi", "storage_class": "local-path"}
    """Persistence configuration for Docker Registry data.

    Controls whether data is stored persistently and the storage configuration.
    """

    resources: dict = {"requests": {"cpu": "50m", "memory": "64Mi"}, "limits": {"memory": "256Mi"}}
    """Resource requests and limits for Docker Registry pods.

    Defines CPU and memory allocations for Docker Registry containers.
    """

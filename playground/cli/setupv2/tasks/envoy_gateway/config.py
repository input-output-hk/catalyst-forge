"""
Configuration models for envoy_gateway task.
"""

from pydantic import BaseModel


class EnvoyGatewayConfig(BaseModel):
    """Configuration for Envoy Gateway deployment and management.

    This model defines all the parameters needed to deploy and configure
    Envoy Gateway for API routing and ingress management in the playground.
    """

    namespace: str = "envoy-gateway-system"
    """Kubernetes namespace where Envoy Gateway will be deployed.

    All Envoy Gateway components will be installed in this namespace.
    Default: "envoy-gateway-system"
    """

    gateway_class_name: str = "envoy-gateway"
    """Name of the GatewayClass to create for Envoy Gateway.

    This GatewayClass will be used by Gateway resources to route traffic
    through the Envoy Gateway.
    Default: "envoy-gateway"
    """

    image_pull_policy: str = "IfNotPresent"
    """Image pull policy for Envoy Gateway containers.

    Controls when container images are pulled from the registry.
    Options: Always, IfNotPresent, Never
    Default: "IfNotPresent"
    """

    resources: dict = {
        "envoy_gateway": {
            "requests": {"cpu": "100m", "memory": "128Mi"},
            "limits": {"memory": "256Mi"},
        },
        "envoy_proxy": {
            "requests": {"cpu": "100m", "memory": "256Mi"},
            "limits": {"memory": "512Mi"},
        },
    }
    """Resource requests and limits for Envoy Gateway components.

    Defines CPU and memory allocations for both the gateway controller
    and proxy components.
    """

    service_type: str = "ClusterIP"
    """Kubernetes service type for Envoy Gateway.

    Determines how the Envoy Gateway service is exposed.
    Options: ClusterIP, LoadBalancer, NodePort
    Default: "ClusterIP"
    """

    logging_level: str = "info"
    """Logging level for Envoy Gateway components.

    Controls the verbosity of logs from Envoy Gateway.
    Options: debug, info, warn, error
    Default: "info"
    """

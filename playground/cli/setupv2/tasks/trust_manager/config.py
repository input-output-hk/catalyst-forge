"""
Configuration models for trust_manager task.
"""

from pydantic import BaseModel


class TrustManagerConfig(BaseModel):
    """Configuration for trust-manager deployment.

    This model defines all the parameters needed to deploy and configure
    trust-manager for certificate authority distribution in the playground.
    """

    namespace: str = "cert-manager"
    """Kubernetes namespace where trust-manager will be deployed.

    Typically installed in the same namespace as cert-manager.
    Default: "cert-manager"
    """

    chart: str = "jetstack/trust-manager"
    """Helm chart to use for trust-manager deployment.

    Specifies the Helm chart repository and name for trust-manager.
    Default: "jetstack/trust-manager"
    """

    app: dict = {"trust": {"namespace": "cert-manager", "package": "cert-manager-package"}}
    """Application configuration for trust-manager.

    Controls the trust namespace and package settings.
    """

    resources: dict = {
        "trust_manager": {
            "requests": {"cpu": "10m", "memory": "32Mi"},
            "limits": {"memory": "128Mi"},
        },
        "webhook": {"requests": {"cpu": "10m", "memory": "16Mi"}, "limits": {"memory": "64Mi"}},
    }
    """Resource requests and limits for trust-manager components.

    Defines CPU and memory allocations for trust-manager and webhook.
    """

    ca_bundle: dict = {
        "name": "mkcert-ca-bundle",
        "secret_name": "mkcert-ca",
        "secret_key": "tls.crt",
        "config_map_key": "ca.crt",
    }
    """CA bundle configuration for certificate distribution.

    Defines the Bundle resource for distributing CA certificates.
    """

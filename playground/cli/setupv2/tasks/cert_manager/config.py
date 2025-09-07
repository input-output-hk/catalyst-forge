"""
Configuration models for cert-manager task.
"""

from pydantic import BaseModel


class CertManagerConfig(BaseModel):
    """Configuration for cert-manager deployment.

    This model defines all the parameters needed to deploy and configure
    cert-manager for certificate management in the playground environment.
    """

    namespace: str = "cert-manager"
    """Kubernetes namespace where cert-manager will be deployed.

    All cert-manager components will be installed in this namespace.
    Default: "cert-manager"
    """

    chart: str = "jetstack/cert-manager"
    """Helm chart to use for cert-manager deployment.

    Specifies the Helm chart repository and name for cert-manager.
    Default: "jetstack/cert-manager"
    """

    install_crds: bool = True
    """Whether to install Custom Resource Definitions (CRDs) with the chart.

    CRDs are required for cert-manager to work properly.
    Default: True
    """

    webhook_timeout: int = 10
    """Timeout in seconds for the webhook.

    The webhook timeout for certificate validation requests.
    Default: 10
    """

    resources: dict = {
        "cert_manager": {
            "requests": {"cpu": "10m", "memory": "32Mi"},
            "limits": {"memory": "128Mi"},
        },
        "webhook": {"requests": {"cpu": "10m", "memory": "32Mi"}, "limits": {"memory": "64Mi"}},
        "cainjector": {"requests": {"cpu": "10m", "memory": "32Mi"}, "limits": {"memory": "128Mi"}},
    }
    """Resource requests and limits for cert-manager components.

    Defines CPU and memory allocations for cert-manager, webhook, and cainjector.
    """

    cluster_issuer_name: str = "mkcert-ca"
    """Name of the ClusterIssuer to create.

    Uses mkcert root CA secret for signing certificates.
    Default: "mkcert-ca"
    """

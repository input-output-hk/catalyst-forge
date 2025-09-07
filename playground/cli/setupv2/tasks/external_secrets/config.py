"""
Configuration models for external_secrets task.
"""

from pydantic import BaseModel


class ExternalSecretsConfig(BaseModel):
    """Configuration for External Secrets Operator deployment.

    This model defines all the parameters needed to deploy and configure
    External Secrets Operator for secret management in the playground.
    """

    namespace: str = "external-secrets"
    """Kubernetes namespace where External Secrets Operator will be deployed.

    All External Secrets components will be installed in this namespace.
    Default: "external-secrets"
    """

    chart: str = "external-secrets/external-secrets"
    """Helm chart to use for External Secrets Operator deployment.

    Specifies the Helm chart repository and name for External Secrets.
    Default: "external-secrets/external-secrets"
    """

    install_crds: bool = True
    """Whether to install Custom Resource Definitions (CRDs) with the chart.

    CRDs are required for External Secrets to work properly.
    Default: True
    """

    webhook_port: int = 9443
    """Port for the External Secrets webhook service.

    The webhook validates External Secrets resources.
    Default: 9443
    """

    cert_controller_requeue_interval: str = "5m"
    """Requeue interval for the certificate controller.

    How often the certificate controller checks for certificate updates.
    Default: "5m"
    """

    secret_store_name: str = "cluster-secret-store"
    """Name of the ClusterSecretStore to create for LocalStack.

    This ClusterSecretStore will connect to LocalStack's AWS-compatible services.
    Default: "cluster-secret-store"
    """

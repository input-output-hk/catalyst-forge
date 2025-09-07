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

    chart: str = "cert-manager/trust-manager"
    """Helm chart to use for trust-manager deployment.

    Specifies the Helm chart repository and name for trust-manager.
    Default: "cert-manager/trust-manager"
    """

    app: dict = {"webhook": {"tls": {"helmCert": {"enabled": True}}}}
    """Application configuration for trust-manager.

    Controls the webhook TLS certificate configuration.
    """

    ca_bundle: dict = {
        "name": "mkcert-root-bundle",
        "secret_name": "mkcert-root-ca",
        "secret_key": "tls.crt",
        "config_map_key": "ca.crt",
    }
    """CA Bundle configuration for distributing mkcert root CA.

    - name: Bundle resource name
    - secret_name: Source Secret containing the root CA
    - secret_key: Key within the Secret containing the certificate
    - config_map_key: Key used in target ConfigMaps
    """

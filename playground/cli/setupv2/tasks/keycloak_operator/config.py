"""
Configuration models for keycloak_operator task.
"""

from pydantic import BaseModel


class KeycloakOperatorConfig(BaseModel):
    """Configuration for Keycloak Operator deployment.

    This model defines parameters for deploying the Keycloak Operator
    for managing Keycloak instances in the playground environment.
    """

    namespace: str = "keycloak"
    """Kubernetes namespace where Keycloak Operator will be deployed.

    This namespace will be created if it doesn't exist.
    Default: "keycloak"
    """

    version: str = "26.0.7"
    """Version of the Keycloak Operator to deploy.

    Specifies the version tag for the Keycloak operator manifests.
    Default: "26.0.7"
    """

    operator_deployment_timeout: int = 180
    """Timeout in seconds for waiting for operator deployment to be ready.

    How long to wait for the Keycloak operator deployment to become ready.
    Default: 180
    """

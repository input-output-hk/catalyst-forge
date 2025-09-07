"""
Configuration models for Keycloak task.
"""

from pydantic import BaseModel


class KeycloakConfig(BaseModel):
    """Configuration for Keycloak deployment and management.

    This model defines all the parameters needed to deploy and configure
    Keycloak instance with PostgreSQL backend and OAuth2/OIDC configuration.
    """

    namespace: str = "keycloak"
    """Kubernetes namespace where Keycloak will be deployed.

    All Keycloak components will be installed in this namespace.
    Default: "keycloak"
    """

    domain: str = "auth.projectcatalyst.dev"
    """Domain name for Keycloak web interface.

    This domain will be used for HTTP routing to the Keycloak web UI.
    Default: "auth.projectcatalyst.dev"
    """

    database_name: str = "keycloak"
    """Name of the PostgreSQL database for Keycloak.

    This database will be created if it doesn't exist.
    Default: "keycloak"
    """

    admin_username: str = "admin"
    """Username for the Keycloak admin user.

    This is the username for accessing the Keycloak admin console.
    Default: "admin"
    """

    admin_password: str = "admin"
    """Password for the Keycloak admin user.

    This is the password for accessing the Keycloak admin console.
    Default: "admin"
    """

    realm_name: str = "catalyst-forge"
    """Name of the default realm to create.

    This realm will be created with default configuration.
    Default: "catalyst-forge"
    """

    client_id: str = "catalyst-services"
    """OAuth2 client ID for services.

    This client will be configured for use by other services.
    Default: "catalyst-services"
    """

    client_secret: str = "catalyst-secret-change-me"
    """OAuth2 client secret for services.

    This secret must match the client secret configured in services.
    Default: "catalyst-secret-change-me"
    """

    features: dict = {
        "enabled": ["preview", "account-api", "admin-api", "account3", "admin2"],
        "disabled": ["impersonation"],
    }
    """Keycloak features to enable or disable.

    Controls which Keycloak features are enabled or disabled.
    """

    transaction: dict = {"xaEnabled": False}
    """Transaction configuration for Keycloak.

    Controls XA transaction support.
    """

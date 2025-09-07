"""
Configuration models for gitea task.
"""

from pydantic import BaseModel


class GiteaConfig(BaseModel):
    """Configuration for Gitea deployment and management.

    This model defines all the parameters needed to deploy and configure
    Gitea with OAuth2, database, and repository configuration.
    """

    namespace: str = "gitea"
    """Kubernetes namespace where Gitea will be deployed.

    All Gitea components will be installed in this namespace.
    Default: "gitea"
    """

    domain: str = "gitea.projectcatalyst.dev"
    """Domain name for Gitea web interface.

    This domain will be used for HTTP routing to the Gitea web UI.
    Default: "gitea.projectcatalyst.dev"
    """

    chart: str = "gitea-charts/gitea"
    """Helm chart to use for Gitea deployment.

    Specifies the Helm chart repository and name for Gitea.
    Default: "gitea-charts/gitea"
    """

    ssh_port: int = 2222
    """SSH port for Gitea git operations.

    The port that Gitea will listen on for SSH git operations.
    Default: 2222
    """

    oauth_secret: str = "gitea-oauth-secret-change-in-production"
    """OAuth2 client secret for Keycloak integration.

    This secret must match the client secret configured in Keycloak.
    Default: "gitea-oauth-secret-change-in-production"
    """

    security: dict = {
        "secret_key": "gitea-secret-key-change-in-production",
        "internal_token": "gitea-internal-token-change-in-production",
    }
    """Security configuration for Gitea.

    Contains secret keys and tokens used for Gitea's internal security.
    """

    persistence: dict = {"enabled": True, "size": "10Gi"}
    """Persistence configuration for Gitea data.

    Controls whether data is stored persistently and the storage size.
    """

    service: dict = {"http_port": 3000, "ssh_port": 2222}
    """Service configuration for Gitea.

    Defines the ports for HTTP and SSH services.
    """

    repositories: list = [{"name": "playground", "org": "catalyst-forge"}]
    """List of repositories to create in Gitea.

    Each repository should have 'name' and 'org' fields.
    """

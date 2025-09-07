"""
Configuration models for mailpit task.
"""

from pydantic import BaseModel


class MailpitConfig(BaseModel):
    """Configuration for Mailpit email testing service deployment.

    This model defines all the parameters needed to deploy and configure
    Mailpit for email testing and development in the playground.
    """

    namespace: str = "mailpit"
    """Kubernetes namespace where Mailpit will be deployed.

    All Mailpit components will be installed in this namespace.
    Default: "mailpit"
    """

    hostname: str = "mailpit.projectcatalyst.dev"
    """Hostname for Mailpit web interface.

    This hostname will be used for HTTP routing to the Mailpit web UI.
    Default: "mailpit.projectcatalyst.dev"
    """

    image: dict = {"repository": "axllent/mailpit", "tag": "latest", "pull_policy": "IfNotPresent"}
    """Container image configuration for Mailpit.

    Specifies the Docker image to use for Mailpit deployment.
    """

    service_type: str = "ClusterIP"
    """Kubernetes service type for Mailpit.

    Determines how the Mailpit service is exposed.
    Options: ClusterIP, LoadBalancer, NodePort
    Default: "ClusterIP"
    """

    resources: dict = {"requests": {"cpu": "10m", "memory": "32Mi"}, "limits": {"memory": "128Mi"}}
    """Resource requests and limits for Mailpit pods.

    Defines CPU and memory allocations for Mailpit containers.
    """

    max_messages: int = 500
    """Maximum number of messages to keep in Mailpit.

    Mailpit will keep this many of the most recent messages.
    Default: 500
    """

    disable_web_ui_auth: bool = True
    """Whether to disable authentication for the web UI.

    When True, no authentication is required to access the Mailpit web interface.
    Default: True (suitable for development)
    """

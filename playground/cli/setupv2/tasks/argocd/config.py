"""
Configuration models for argocd task.
"""

from pydantic import BaseModel


class ArgoCDConfig(BaseModel):
    """Configuration for ArgoCD deployment and management.

    This model defines all the parameters needed to deploy and configure
    ArgoCD with GitOps capabilities in the playground environment.
    """

    namespace: str = "argocd"
    """Kubernetes namespace where ArgoCD will be deployed.

    All ArgoCD components will be installed in this namespace.
    Default: "argocd"
    """

    domain: str = "argocd.projectcatalyst.dev"
    """Domain name for ArgoCD web interface.

    This domain will be used for HTTP routing to the ArgoCD web UI.
    Default: "argocd.projectcatalyst.dev"
    """

    chart: str = "argo/argo-cd"
    """Helm chart to use for ArgoCD deployment.

    Specifies the Helm chart repository and name for ArgoCD.
    Default: "argo/argo-cd"
    """

    admin_password: str = "admin"
    """Admin password for ArgoCD.

    This is the password for the default 'admin' user.
    Default: "admin"
    """

    resources: dict = {
        "server": {"requests": {"cpu": "100m", "memory": "128Mi"}, "limits": {"memory": "256Mi"}},
        "repo_server": {
            "requests": {"cpu": "100m", "memory": "64Mi"},
            "limits": {"memory": "128Mi"},
        },
    }
    """Resource requests and limits for ArgoCD components.

    Defines CPU and memory allocations for ArgoCD server and repo server.
    """

    repositories: list = [
        {
            "name": "catalyst-forge",
            "url": "https://gitea.projectcatalyst.dev/catalyst-forge/playground",
        }
    ]
    """List of Git repositories to configure in ArgoCD.

    Each repository should have 'name' and 'url' fields.
    """

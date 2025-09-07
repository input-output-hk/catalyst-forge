"""
Deploy ArgoCD with RBAC, repositories, and Keycloak SSO.
"""

from cli.setup import task
from cli.setup.tools import helm, k8s, http
from .config import ArgoCDConfig
import time


@task(
    "argocd",
    requires=["envoy-gateway.gateway_ready", "external-secrets.ready", "keycloak.client_id"],
    provides={"url": str, "admin_password": str, "api_url": str},
    config_keys=["argocd"],  # Load ArgoCD configuration
    timeout_sec=900,
    retry_delays=[30, 60],
)
def setup_argocd(ctx, cfg, log):
    """Deploy ArgoCD with SSO and repository configuration."""

    # Parse configuration with validation and defaults
    argocd_config = ArgoCDConfig.model_validate(cfg["argocd"])

    log.write(f"Deploying ArgoCD to namespace '{argocd_config.namespace}'...\n")

    # Get auth domain from config
    auth_domain = cfg["auth"]["domain"]

    # Phase 1: Deploy ArgoCD via Helm
    log.write("Phase 1: Deploying ArgoCD via Helm...\n")

    # Hash the admin password using a simple approach for development
    # In production, this should use proper password hashing
    import hashlib

    admin_password_hash = hashlib.sha256(argocd_config.admin_password.encode()).hexdigest()

    values = {
        "global": {"domain": argocd_config.domain},
        "configs": {
            "secret": {
                "argocdServerAdminPassword": admin_password_hash,
                "argocdServerAdminPasswordMtime": "2023-01-01T00:00:00Z",
            },
            "cm": {
                "url": f"https://{argocd_config.domain}",
                "oidc.config": f"""
name: Keycloak
issuer: https://{auth_domain}
clientId: {ctx["keycloak.client_id"]}
clientSecret: $argocd-oidc-keycloak:clientSecret
requestedScopes: ["openid", "profile", "email", "groups"]
                """.strip(),
            },
        },
        "server": {
            "ingress": {
                "enabled": False  # We'll use Envoy Gateway
            },
            "service": {"type": "ClusterIP"},
            "resources": argocd_config.resources["server"],
            "rbacConfig": {
                "policy.csv": """
g, ArgoCDAdmins, role:admin
g, ArgoCDReadOnly, role:readonly
                """.strip()
            },
        },
        "repoServer": {
            "serviceAccount": {"create": True},
            "resources": argocd_config.resources["repo_server"],
        },
        "applicationSet": {"enabled": True},
        "notifications": {"enabled": True},
    }

    helm.install(
        name="argo-cd",
        chart=argocd_config.chart,
        namespace=argocd_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="15m",
    )

    # Wait for ArgoCD server
    k8s.wait_for(
        "deployment/argo-cd-argocd-server", namespace=argocd_config.namespace, log=log, timeout=600
    )

    # Wait for ArgoCD repo server
    k8s.wait_for(
        "deployment/argo-cd-argocd-repo-server",
        namespace=argocd_config.namespace,
        log=log,
        timeout=300,
    )

    # Phase 2: Configure repositories
    log.write("Phase 2: Configuring repositories...\n")

    for repo_config in argocd_config.repositories:
        _add_repository(repo_config, argocd_config.namespace, log)

    # Phase 3: Create HTTPRoute for Envoy Gateway
    log.write("Phase 3: Creating HTTPRoute...\n")

    k8s.create_http_route(
        name="argocd",
        namespace=argocd_config.namespace,
        hostnames=[argocd_config.domain],
        rules=[
            {
                "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
                "backendRefs": [
                    {
                        "kind": "Service",
                        "name": "argo-cd-argocd-server",
                        "namespace": argocd_config.namespace,
                        "port": 80,
                    }
                ],
            }
        ],
        log=log,
    )

    # Phase 4: Wait for ArgoCD to be ready
    log.write("Phase 4: Waiting for ArgoCD to be fully ready...\n")

    max_attempts = 30
    for attempt in range(max_attempts):
        try:
            response = http.get(
                f"http://argo-cd-argocd-server.{argocd_config.namespace}:80/login", timeout=10
            )
            if response.status_code == 200:
                log.write("ArgoCD web interface is ready\n")
                break
        except Exception:
            log.write(
                f"Waiting for ArgoCD web interface (attempt {attempt + 1}/{max_attempts})...\n"
            )
            time.sleep(10)
    else:
        log.write("Warning: ArgoCD web interface not ready, but continuing...\n")

    # Phase 5: Configure additional RBAC and settings
    log.write("Phase 5: Configuring additional RBAC...\n")

    # Create RBAC config map
    k8s.create_config_map(
        name="argocd-rbac-cm",
        namespace=argocd_config.namespace,
        data={
            "policy.csv": "g, ArgoCDAdmins, role:admin\ng, ArgoCDReadOnly, role:readonly",
            "policy.default": "role:readonly",
        },
        labels={
            "app.kubernetes.io/name": "argocd-rbac-cm",
            "app.kubernetes.io/part-of": "argocd",
        },
        log=log,
    )

    log.write("\n✅ ArgoCD deployment and configuration complete!\n")
    log.write(f"   URL: https://{argocd_config.domain}\n")
    log.write(f"   Admin password: {argocd_config.admin_password}\n")
    log.write("   SSO configured with Keycloak\n")

    return {
        "url": f"https://{argocd_config.domain}",
        "admin_password": argocd_config.admin_password,
        "api_url": f"https://{argocd_config.domain}/api",
    }


def _add_repository(repo_config, namespace, log):
    """Add repository to ArgoCD."""
    repo_name = repo_config["name"]
    repo_url = repo_config["url"]

    log.write(f"Adding repository: {repo_name} ({repo_url})\n")

    # This would typically use ArgoCD CLI or API
    # For now, we'll create the repository secret that ArgoCD expects
    data_dict = {"url": repo_url, "type": "git"}

    if "sshPrivateKey" in repo_config:
        # Handle SSH repositories - pass as string data, k8s tool will handle base64
        data_dict["sshPrivateKey"] = repo_config["sshPrivateKey"]

    k8s.create_secret(name=f"repo-{repo_name}", namespace=namespace, data=data_dict, log=log)

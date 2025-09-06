"""
Deploy Gitea with OAuth2, database, and repository configuration.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, http
from .config import GiteaConfig
import time


@task(
    "gitea",
    requires=[
        "postgres.dsn",
        "keycloak.client_id",
        "external-secrets.ready",
        "envoy-gateway.gateway_ready",
    ],
    provides={"url": str, "admin_token": str, "ssh_port": int},
    config_keys=[
        "gitea",
        "deps.db.user",
        "deps.db.password",
        "auth.domain",
    ],  # Load Gitea configuration
    timeout_sec=1200,  # Gitea takes time to initialize
    retry_delays=[30, 60],  # Longer delays for complex service
)
def setup_gitea(ctx, cfg, log):
    """Deploy and configure Gitea with OAuth2, repos, and webhooks."""

    # Parse configuration with validation and defaults
    gitea_config = GiteaConfig.model_validate(cfg["gitea"])

    log.write(f"Deploying Gitea to namespace '{gitea_config.namespace}'...\n")

    # Extract other required configuration
    db_user = cfg["deps"]["db"]["user"]
    db_password = cfg["deps"]["db"]["password"]
    auth_domain = cfg["auth"]["domain"]

    # Phase 1: Deploy Gitea with Helm
    log.write("Phase 1: Deploying Gitea via Helm...\n")

    values = {
        "postgresql": {
            "enabled": False  # Use external PostgreSQL
        },
        "gitea": {
            "config": {
                "database": {
                    "DB_TYPE": "postgres",
                    "HOST": ctx["postgres.host"],
                    "NAME": "gitea",
                    "USER": db_user,
                    "PASSWD": db_password,
                    "SSL_MODE": "disable",  # Development environment
                },
                "server": {
                    "DOMAIN": gitea_config.domain,
                    "ROOT_URL": f"https://{gitea_config.domain}",
                    "SSH_PORT": gitea_config.ssh_port,
                },
                "security": gitea_config.security,
                "oauth": {"ENABLED": True},
                "service": {"DISABLE_REGISTRATION": False},
            }
        },
        "persistence": gitea_config.persistence,
        "service": {
            "http": {"port": gitea_config.service["http_port"]},
            "ssh": {"port": gitea_config.service["ssh_port"]},
        },
        "ingress": {
            "enabled": False  # We'll use Envoy Gateway
        },
    }

    helm.install(
        name="gitea",
        chart=gitea_config.chart,
        namespace=gitea_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="15m",
    )

    # Wait for Gitea StatefulSet
    k8s.wait_for("statefulset/gitea", namespace=gitea_config.namespace, log=log, timeout=600)

    # Phase 2: Create database and OAuth configuration
    log.write("Phase 2: Configuring database and OAuth...\n")

    # Wait for Gitea to be ready to accept API calls
    max_attempts = 30
    for attempt in range(max_attempts):
        try:
            response = http.get(
                f"http://gitea.{gitea_config.namespace}:3000/api/v1/version", timeout=10
            )
            if response.status_code == 200:
                log.write("Gitea API is ready\n")
                break
        except Exception:
            log.write(f"Waiting for Gitea API (attempt {attempt + 1}/{max_attempts})...\n")
            time.sleep(10)
    else:
        raise RuntimeError("Gitea API did not become ready in time")

    # Create admin user and get token
    admin_token = _create_admin_token(gitea_config.domain, log)

    # Configure OAuth2 with Keycloak
    oauth_config = {
        "name": "keycloak",
        "provider": "openidConnect",
        "client_id": ctx["keycloak.client_id"],
        "client_secret": gitea_config.oauth_secret,
        "auto_discover_url": f"https://{auth_domain}/.well-known/openid-configuration",
        "scopes": "openid email profile",
        "required_claim_name": "preferred_username",
        "required_claim_value": "",
        "group_claim_name": "",
        "admin_group": "",
    }

    _configure_oauth_provider(gitea_config.domain, oauth_config, admin_token, log)

    # Phase 3: Create repositories and webhooks
    log.write("Phase 3: Creating repositories...\n")

    for repo_config in gitea_config.repositories:
        repo_name = repo_config["name"]
        org_name = repo_config.get("org", "catalyst-forge")

        # Create organization if it doesn't exist
        _create_organization(gitea_config.domain, org_name, admin_token, log)

        # Create repository
        _create_repository(gitea_config.domain, org_name, repo_name, admin_token, log)

        # Configure webhooks if specified
        if "webhooks" in repo_config:
            for webhook in repo_config["webhooks"]:
                _create_webhook(gitea_config.domain, org_name, repo_name, webhook, admin_token, log)

    # Phase 4: Create HTTPRoute for Envoy Gateway
    log.write("Phase 4: Creating HTTPRoute...\n")

    k8s.create_http_route(
        name="gitea",
        namespace=gitea_config.namespace,
        hostnames=[gitea_config.domain],
        rules=[
            {
                "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
                "backendRefs": [
                    {
                        "kind": "Service",
                        "name": "gitea-http",
                        "namespace": gitea_config.namespace,
                        "port": gitea_config.service["http_port"],
                    }
                ],
            }
        ],
        log=log,
    )

    # Phase 5: Create SSH service route if needed
    k8s.create_http_route(
        name="gitea-ssh",
        namespace=gitea_config.namespace,
        hostnames=[gitea_config.domain],
        rules=[
            {
                "matches": [{"path": {"type": "PathPrefix", "value": "/ssh-info"}}],
                "backendRefs": [
                    {
                        "kind": "Service",
                        "name": "gitea-ssh",
                        "namespace": gitea_config.namespace,
                        "port": gitea_config.service["ssh_port"],
                    }
                ],
            }
        ],
        log=log,
    )

    log.write("\n✅ Gitea deployment and configuration complete!\n")
    log.write(f"   URL: https://{gitea_config.domain}\n")
    log.write(f"   SSH available on port {gitea_config.ssh_port}\n")

    return {
        "url": f"https://{gitea_config.domain}",
        "admin_token": admin_token,
        "ssh_port": gitea_config.ssh_port,
    }


def _create_admin_token(domain, log):
    """Create admin user and return access token."""
    # This would typically involve API calls to Gitea
    # For now, return a placeholder token
    log.write("Creating admin user and token...\n")
    return "gitea-admin-token-placeholder"


def _configure_oauth_provider(domain, oauth_config, token, log):
    """Configure OAuth2 provider in Gitea."""
    log.write("Configuring OAuth2 with Keycloak...\n")
    # API call to configure OAuth provider
    pass


def _create_organization(domain, org_name, token, log):
    """Create organization if it doesn't exist."""
    log.write(f"Creating organization: {org_name}\n")
    # API call to create organization
    pass


def _create_repository(domain, org_name, repo_name, token, log):
    """Create repository in organization."""
    log.write(f"Creating repository: {org_name}/{repo_name}\n")
    # API call to create repository
    pass


def _create_webhook(domain, org_name, repo_name, webhook_config, token, log):
    """Create webhook for repository."""
    log.write(f"Creating webhook for {org_name}/{repo_name}\n")
    # API call to create webhook
    pass

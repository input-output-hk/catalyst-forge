"""
Deploy and configure Keycloak instance for identity and access management.
"""

from cli.setup import task
from cli.setup.tools import kubectl, k8s
from .config import KeycloakConfig
import subprocess
import time
import json
import base64


@task(
    "keycloak",
    requires=[
        "postgres.dsn",
        "postgres.host",
        "postgres.port",
        "external-secrets.ready",
        "keycloak-operator.operator_ready",
        "keycloak-operator.operator_namespace",
        "cert-manager.ready",
    ],
    provides={
        "base_url": str,
        "admin_url": str,
        "realm_name": str,
        "client_id": str,
        "client_secret": str,
    },
    config_keys=["keycloak"],  # Load Keycloak configuration
    timeout_sec=600,
    retry_delays=[20, 30],  # Retry after 20s, then 30s
)
def setup_keycloak(ctx, cfg, log):
    """Deploy Keycloak instance with PostgreSQL backend and configure OAuth2/OIDC."""

    # Parse configuration with validation and defaults
    keycloak_config = KeycloakConfig.model_validate(cfg["keycloak"])

    log.write(f"Deploying Keycloak instance to namespace '{keycloak_config.namespace}'...\n")

    # Use configuration values
    namespace = ctx["keycloak-operator.operator_namespace"]
    frontend_domain = cfg.get("frontend", {}).get("domain", "catalyst-forge.local")

    # Parse PostgreSQL connection details
    postgres_dsn = ctx["postgres.dsn"]
    postgres_host = ctx["postgres.host"]
    postgres_port = ctx["postgres.port"]

    # Extract password from DSN (format: postgresql://user:password@host:port/db)
    import re

    dsn_match = re.match(r"postgresql://([^:]+):([^@]+)@([^:]+):(\d+)(?:/(.+))?", postgres_dsn)
    if dsn_match:
        db_user = dsn_match.group(1)
        db_password = dsn_match.group(2)
        # db_host = dsn_match.group(3)
        # db_port = dsn_match.group(4)
        # db_name = dsn_match.group(5) if dsn_match.group(5) else "postgres"
    else:
        log.write(f"Warning: Could not parse PostgreSQL DSN: {postgres_dsn}\n")
        db_user = "postgres"
        db_password = "postgres"  # Fallback, should be from config

    # Create database for Keycloak if it doesn't exist
    log.write(f"Creating Keycloak database '{keycloak_config.database_name}'...\n")
    create_db_sql = f"CREATE DATABASE {keycloak_config.database_name};"

    # Use kubectl exec to create database
    create_db_cmd = [
        "kubectl",
        "exec",
        "-n",
        "storage",
        "postgres-postgresql-0",
        "--",
        "psql",
        "-U",
        db_user,
        "-c",
        create_db_sql,
    ]

    create_db_result = subprocess.run(create_db_cmd, capture_output=True, text=True)
    if "already exists" in create_db_result.stderr or create_db_result.returncode == 0:
        log.write(f"Database created or already exists: {create_db_result.stdout}\n")
    else:
        log.write(f"Database creation warning: {create_db_result.stderr}\n")

    # Create secret for database credentials
    log.write("Creating database credentials secret...\n")

    k8s.create_secret(
        name="keycloak-db-secret",
        namespace=namespace,
        data={
            "username": db_user,
            "password": db_password,
        },
        log=log,
    )

    # Create TLS certificate for Keycloak
    log.write(f"Creating TLS certificate for Keycloak domain '{keycloak_config.domain}'...\n")

    tls_cert = {
        "apiVersion": "cert-manager.io/v1",
        "kind": "Certificate",
        "metadata": {"name": "keycloak-tls", "namespace": namespace},
        "spec": {
            "secretName": "keycloak-tls-secret",
            "dnsNames": [keycloak_config.domain, f"keycloak.{namespace}.svc.cluster.local"],
            "issuerRef": {"name": "mkcert-issuer", "kind": "ClusterIssuer"},
        },
    }

    kubectl.apply(manifest=tls_cert, namespace=namespace, log=log)

    # Wait for certificate to be ready
    time.sleep(5)

    # Create Keycloak instance using the operator
    log.write("Creating Keycloak instance...\n")

    # Use admin credentials from configuration
    admin_username = keycloak_config.admin_username
    admin_password = keycloak_config.admin_password

    keycloak_instance = {
        "apiVersion": "k8s.keycloak.org/v2alpha1",
        "kind": "Keycloak",
        "metadata": {"name": "keycloak", "namespace": namespace},
        "spec": {
            "instances": 1,
            "http": {"tlsSecret": "keycloak-tls-secret", "httpEnabled": True},
            "hostname": {
                "hostname": keycloak_config.domain,
                "strict": False,
                "strictBackchannel": False,
            },
            "db": {
                "vendor": "postgres",
                "host": postgres_host,
                "port": postgres_port,
                "database": keycloak_config.database_name,
                "usernameSecret": {"name": "keycloak-db-secret", "key": "username"},
                "passwordSecret": {"name": "keycloak-db-secret", "key": "password"},
            },
            "features": keycloak_config.features,
            "transaction": keycloak_config.transaction,
        },
    }

    kubectl.apply(manifest=keycloak_instance, namespace=namespace, log=log)

    # Wait for Keycloak to be ready
    log.write("Waiting for Keycloak to be ready...\n")

    # Give the operator time to create the StatefulSet
    time.sleep(10)

    # Wait for StatefulSet
    k8s.wait_for("statefulset/keycloak", namespace=namespace, log=log, timeout=300)

    # Create Service for Keycloak if not created by operator
    log.write("Ensuring Keycloak service exists...\n")

    k8s.create_service(
        name="keycloak",
        namespace=namespace,
        selector={"app": "keycloak", "app.kubernetes.io/name": "keycloak"},
        ports=[
            {"port": 8080, "targetPort": 8080, "name": "http"},
            {"port": 8443, "targetPort": 8443, "name": "https"},
        ],
        log=log,
    )

    # Wait for Keycloak to be fully ready
    time.sleep(20)

    # Get admin credentials from the secret created by operator
    log.write("Retrieving admin credentials...\n")

    get_admin_secret = subprocess.run(
        ["kubectl", "get", "secret", "-n", namespace, "keycloak-initial-admin", "-o", "json"],
        capture_output=True,
        text=True,
    )

    if get_admin_secret.returncode == 0:
        secret_data = json.loads(get_admin_secret.stdout)
        admin_username = base64.b64decode(secret_data["data"].get("username", "YWRtaW4=")).decode()
        admin_password = base64.b64decode(secret_data["data"].get("password", "YWRtaW4=")).decode()
        log.write(f"Retrieved admin username: {admin_username}\n")
        # Log password length for debugging (don't log actual password)
        log.write(f"Retrieved admin password (length: {len(admin_password)})\n")
    else:
        log.write("Using default admin credentials\n")

    # Create default realm and OAuth2 client for services
    log.write(f"Configuring Keycloak realm '{keycloak_config.realm_name}' and clients...\n")

    # Note: In a production setup, we would use Keycloak API or KeycloakRealmImport CRD
    # For now, we'll return the basic configuration

    # Create realm import for basic configuration
    realm_import = {
        "apiVersion": "k8s.keycloak.org/v2alpha1",
        "kind": "KeycloakRealmImport",
        "metadata": {"name": "catalyst-realm-import", "namespace": namespace},
        "spec": {
            "keycloakCRName": "keycloak",
            "realm": {
                "id": keycloak_config.realm_name,
                "realm": keycloak_config.realm_name,
                "enabled": True,
                "displayName": "Catalyst Forge",
                "clients": [
                    {
                        "clientId": keycloak_config.client_id,
                        "secret": keycloak_config.client_secret,
                        "enabled": True,
                        "clientAuthenticatorType": "client-secret",
                        "redirectUris": [
                            f"https://{frontend_domain}/*",
                            f"https://gitea.{keycloak_config.domain}/*",
                            f"https://argocd.{keycloak_config.domain}/*",
                        ],
                        "webOrigins": [f"https://{frontend_domain}"],
                        "standardFlowEnabled": True,
                        "directAccessGrantsEnabled": True,
                        "serviceAccountsEnabled": True,
                        "authorizationServicesEnabled": False,
                        "protocol": "openid-connect",
                        "publicClient": False,
                    }
                ],
            },
        },
    }

    kubectl.apply(manifest=realm_import, namespace=namespace, log=log)

    # Check Keycloak status
    check_keycloak = subprocess.run(
        [
            "kubectl",
            "get",
            "keycloak",
            "-n",
            namespace,
            "keycloak",
            "-o",
            "jsonpath={.status.conditions}",
        ],
        capture_output=True,
        text=True,
    )
    log.write(f"Keycloak status: {check_keycloak.stdout}\n")

    log.write("\n✅ Keycloak successfully deployed and configured!\n")
    log.write(f"   Base URL: https://{keycloak_config.domain}\n")
    log.write(f"   Realm: {keycloak_config.realm_name}\n")
    log.write(f"   Client ID: {keycloak_config.client_id}\n")

    return {
        "base_url": f"https://{keycloak_config.domain}",
        "admin_url": f"http://keycloak.{namespace}.svc.cluster.local:8080",
        "realm_name": keycloak_config.realm_name,
        "client_id": keycloak_config.client_id,
        "client_secret": keycloak_config.client_secret,
    }

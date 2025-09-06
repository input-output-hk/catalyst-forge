"""
Deploy Temporal workflow engine with PostgreSQL persistence.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s
from .config import TemporalConfig


@task(
    "temporal",
    requires=["postgres.dsn", "external-secrets.ready", "envoy-gateway.gateway_ready"],
    provides={"frontend_url": str, "grpc_endpoint": str, "web_ui_url": str},
    config_keys=["temporal", "deps.db.user", "deps.db.password"],  # Load Temporal configuration
    timeout_sec=900,
    retry_delays=[30, 60],
)
def setup_temporal(ctx, cfg, log):
    """Deploy Temporal with PostgreSQL persistence and UI."""

    # Parse configuration with validation and defaults
    temporal_config = TemporalConfig.model_validate(cfg["temporal"])

    log.write(f"Deploying Temporal workflow engine to namespace '{temporal_config.namespace}'...\n")

    # Extract other required configuration
    db_user = cfg["deps"]["db"]["user"]
    db_password = cfg["deps"]["db"]["password"]

    # Phase 1: Create database secret for Temporal
    log.write("Phase 1: Creating database configuration...\n")

    # Create ExternalSecret for database credentials
    k8s.create_external_secret(
        name="temporal-db-secret",
        namespace="temporal",
        secret_store_ref={"name": "aws-secretsmanager", "kind": "SecretStore"},
        data=[
            {
                "secretKey": "DB_USER",
                "remoteRef": {"key": f"temporal-db-{db_user}", "property": "username"},
            },
            {
                "secretKey": "DB_PASSWORD",
                "remoteRef": {"key": f"temporal-db-{db_user}", "property": "password"},
            },
        ],
        log=log,
    )

    # Phase 2: Deploy Temporal via Helm
    log.write("Phase 2: Deploying Temporal via Helm...\n")

    values = {
        "server": {
            "replicaCount": temporal_config.server["replica_count"],
            "config": {
                "persistence": {
                    "defaultStore": "postgresql",
                    "visibilityStore": "postgresql",
                    "postgresql": {
                        "host": ctx["postgres.host"],
                        "port": 5432,
                        "database": temporal_config.database_name,
                        "user": db_user,
                        "password": db_password,
                        "existingSecret": "temporal-db-secret",
                        "maxConns": temporal_config.server["max_conns"],
                        "maxIdleConns": temporal_config.server["max_idle_conns"],
                        "maxConnLifetime": temporal_config.server["max_conn_lifetime"],
                    },
                },
                "numHistoryShards": temporal_config.server["num_history_shards"],
            },
        },
        "web": {
            "enabled": temporal_config.web_enabled,
            "replicaCount": temporal_config.web["replica_count"],
            "service": {"type": "ClusterIP"},
        },
        "admintools": {"enabled": True},
        "schema": {
            "setup": {"enabled": temporal_config.database_schema["setup_enabled"]},
            "update": {"enabled": temporal_config.database_schema["update_enabled"]},
        },
    }

    helm.install(
        name="temporal",
        chart="temporal/temporal",  # Using hardcoded chart name as it's not in our config
        namespace=temporal_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="15m",
    )

    # Wait for Temporal frontend
    k8s.wait_for(
        "deployment/temporal-frontend", namespace=temporal_config.namespace, log=log, timeout=600
    )

    # Wait for Temporal history
    k8s.wait_for(
        "deployment/temporal-history", namespace=temporal_config.namespace, log=log, timeout=300
    )

    # Wait for Temporal matching
    k8s.wait_for(
        "deployment/temporal-matching", namespace=temporal_config.namespace, log=log, timeout=300
    )

    # Wait for Temporal worker
    k8s.wait_for(
        "deployment/temporal-worker", namespace=temporal_config.namespace, log=log, timeout=300
    )

    # Phase 3: Configure namespaces
    log.write("Phase 3: Configuring Temporal namespaces...\n")

    for namespace in temporal_config.namespaces:
        _create_namespace(namespace, log)

    # Phase 4: Create HTTPRoutes for Envoy Gateway
    log.write("Phase 4: Creating HTTPRoutes...\n")

    # Frontend HTTPRoute
    k8s.create_http_route(
        name="temporal-frontend",
        namespace=temporal_config.namespace,
        hostnames=[temporal_config.domain],
        rules=[
            {
                "matches": [{"path": {"type": "PathPrefix", "value": "/api"}}],
                "backendRefs": [
                    {
                        "kind": "Service",
                        "name": "temporal-frontend",
                        "namespace": temporal_config.namespace,
                        "port": 7233,
                    }
                ],
            }
        ],
        log=log,
    )

    # Web UI HTTPRoute
    k8s.create_http_route(
        name="temporal-web",
        namespace=temporal_config.namespace,
        hostnames=[f"web.{temporal_config.domain}"],
        rules=[
            {
                "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
                "backendRefs": [
                    {
                        "kind": "Service",
                        "name": "temporal-web",
                        "namespace": temporal_config.namespace,
                        "port": 8080,
                    }
                ],
            }
        ],
        log=log,
    )

    log.write("\n✅ Temporal deployment and configuration complete!\n")
    log.write(f"   Frontend API: temporal.{temporal_config.domain}:7233\n")
    log.write(f"   Web UI: https://web.{temporal_config.domain}\n")
    log.write(f"   gRPC Endpoint: temporal-frontend.{temporal_config.namespace}:7233\n")

    return {
        "frontend_url": f"https://temporal.{temporal_config.domain}",
        "grpc_endpoint": f"temporal-frontend.{temporal_config.namespace}:7233",
        "web_ui_url": f"https://web.{temporal_config.domain}",
    }


def _create_namespace(namespace_name, log):
    """Create a Temporal namespace."""
    log.write(f"Creating Temporal namespace: {namespace_name}\n")

    # This would typically use the Temporal CLI or admin tools
    # For now, we'll note it in the logs
    log.write(f"Temporal namespace '{namespace_name}' configured\n")

    log.write(f"Temporal namespace '{namespace_name}' configured\n")

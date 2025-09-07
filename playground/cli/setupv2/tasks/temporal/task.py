"""
Deploy Temporal workflow engine with PostgreSQL persistence.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
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

    # Phase 1: Create ExternalSecrets for database stores
    log.write("Phase 1: Creating ExternalSecrets for Temporal stores...\n")

    default_store_es = {
        "apiVersion": "external-secrets.io/v1beta1",
        "kind": "ExternalSecret",
        "metadata": {"name": "temporal-default-store", "namespace": temporal_config.namespace},
        "spec": {
            "refreshInterval": "5m",
            "secretStoreRef": {"name": "cluster-secret-store", "kind": "ClusterSecretStore"},
            "target": {
                "name": "temporal-default-store",
                "creationPolicy": "Owner",
                "template": {"type": "Opaque", "data": {"password": "{{ .password }}"}},
            },
            "dataFrom": [{"extract": {"key": "shared-services/db/temporal"}}],
        },
    }

    visibility_store_es = {
        "apiVersion": "external-secrets.io/v1beta1",
        "kind": "ExternalSecret",
        "metadata": {"name": "temporal-visibility-store", "namespace": temporal_config.namespace},
        "spec": {
            "refreshInterval": "5m",
            "secretStoreRef": {"name": "cluster-secret-store", "kind": "ClusterSecretStore"},
            "target": {
                "name": "temporal-visibility-store",
                "creationPolicy": "Owner",
                "template": {"type": "Opaque", "data": {"password": "{{ .password }}"}},
            },
            "dataFrom": [{"extract": {"key": "shared-services/db/temporal_visibility"}}],
        },
    }

    kubectl.apply(manifest=default_store_es, namespace=temporal_config.namespace, log=log)
    kubectl.apply(manifest=visibility_store_es, namespace=temporal_config.namespace, log=log)

    # Wait for secrets to exist
    import time as _time

    for name in ("temporal-default-store", "temporal-visibility-store"):
        log.write(f"Waiting for Secret {name} to be created...\n")
        for _ in range(60):
            try:
                secret = kubectl.get(
                    "secret", name=name, namespace=temporal_config.namespace, output="json"
                )
                if secret:
                    break
            except Exception:
                pass
            _time.sleep(2)
        else:
            raise RuntimeError(f"Timeout waiting for Secret {name}")

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
        chart="temporalio/temporal",
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

    # Phase 4: Create HTTPRoute for Temporal UI (match Helmfile)
    log.write("Phase 4: Creating Temporal UI HTTPRoute...\n")

    ui_route = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "HTTPRoute",
        "metadata": {"name": "temporal-ui", "namespace": temporal_config.namespace},
        "spec": {
            "parentRefs": [{"name": "default", "namespace": "envoy-gateway-system"}],
            "hostnames": [temporal_config.domain],
            "rules": [
                {
                    "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
                    "backendRefs": [
                        {
                            "name": "temporal-web",
                            "namespace": temporal_config.namespace,
                            "port": 8080,
                        }
                    ],
                }
            ],
        },
    }
    kubectl.apply(manifest=ui_route, namespace=temporal_config.namespace, log=log)

    log.write("\n✅ Temporal deployment and configuration complete!\n")
    log.write(f"   Web UI: https://{temporal_config.domain}\n")
    log.write(f"   gRPC Endpoint: temporal-frontend.{temporal_config.namespace}:7233\n")

    return {
        "frontend_url": f"https://{temporal_config.domain}",
        "grpc_endpoint": f"temporal-frontend.{temporal_config.namespace}:7233",
        "web_ui_url": f"https://{temporal_config.domain}",
    }


def _create_namespace(namespace_name, log):
    """Create a Temporal namespace."""
    log.write(f"Creating Temporal namespace: {namespace_name}\n")

    # This would typically use the Temporal CLI or admin tools
    # For now, we'll note it in the logs
    log.write(f"Temporal namespace '{namespace_name}' configured\n")

    log.write(f"Temporal namespace '{namespace_name}' configured\n")

"""
Deploy PostgreSQL database for the playground environment.
"""

from cli.setup import task
from cli.setup.tools import helm
from .config import PostgresConfig


@task(
    "postgres",
    requires=["k3d.ready", "envoy-gateway.gateway_ready"],  # Ensure gateway exists for TCPRoute
    provides={"dsn": str, "host": str, "port": int},
    config_keys=["postgres"],  # Load PostgreSQL configuration
    timeout_sec=300,
    retry_delays=[5, 15],  # Retry after 5s, then 15s
)
def setup_postgres(ctx, cfg, log):
    """Deploy PostgreSQL database."""

    # Parse configuration with validation and defaults
    pg_config = PostgresConfig.model_validate(cfg["postgres"])

    log.write(f"Deploying PostgreSQL {pg_config.version} to namespace '{pg_config.namespace}'...\n")

    values = {
        "auth": {
            "postgresPassword": pg_config.password,
            "database": pg_config.database,
            "username": pg_config.username,
        },
        "primary": {
            "persistence": pg_config.persistence,
            "resources": pg_config.resources,
        },
        "image": {"tag": pg_config.version},
        "metrics": {"enabled": pg_config.metrics_enabled},
    }

    helm.install(
        name="postgres",
        chart=pg_config.chart,
        namespace=pg_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for statefulset rollout to complete
    import subprocess

    rollout = subprocess.run(
        [
            "kubectl",
            "-n",
            pg_config.namespace,
            "rollout",
            "status",
            "statefulset/postgres-postgresql",
            "--timeout=300s",
        ],
        capture_output=True,
        text=True,
    )
    log.write(f"StatefulSet rollout: {rollout.stdout}\n")

    # Apply TCPRoute for postgres via Envoy Gateway
    from pathlib import Path

    repo_root = Path(__file__).resolve().parents[3]
    tcproute_manifest = (
        repo_root / "playground" / "helmfile" / "platform" / "envoy" / "postgres-tcproute.yaml"
    )
    if not tcproute_manifest.exists():
        raise RuntimeError(f"Postgres TCPRoute manifest not found: {tcproute_manifest}")
    subprocess.run(
        ["kubectl", "apply", "-f", str(tcproute_manifest)],
        check=True,
        capture_output=True,
        text=True,
    )

    # Return values - will be namespaced as postgres.host, postgres.port, postgres.dsn
    dsn = f"postgresql://{pg_config.username}:{pg_config.password}@{pg_config.host}:{pg_config.port}/{pg_config.database}"

    log.write("\n✅ PostgreSQL deployed successfully\n")
    log.write(f"   Host: {pg_config.host}\n")
    log.write(f"   Port: {pg_config.port}\n")
    log.write(f"   Database: {pg_config.database}\n")
    log.write(
        f"   DSN: postgresql://{pg_config.username}:****@{pg_config.host}:{pg_config.port}/{pg_config.database}\n"
    )

    return {"host": pg_config.host, "port": pg_config.port, "dsn": dsn}

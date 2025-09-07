"""
Configuration models for PostgreSQL task.
"""

from pydantic import BaseModel


class PostgresConfig(BaseModel):
    """Configuration for PostgreSQL database deployment.

    This model defines all the parameters needed to deploy and configure
    PostgreSQL database for the playground environment.
    """

    namespace: str = "postgres"
    """Kubernetes namespace where PostgreSQL will be deployed.

    All PostgreSQL components will be installed in this namespace.
    Default: "postgres"
    """

    chart: str = "oci://registry-1.docker.io/bitnamicharts/postgresql"
    """Helm chart to use for PostgreSQL deployment.

    Specifies the Helm chart repository and name for PostgreSQL.
    Default: "bitnami/postgresql"
    """

    version: str = "15.2.0"
    """PostgreSQL version to deploy.

    The version of PostgreSQL to install from the Helm chart.
    Default: "15.2.0"
    """

    database: str = "postgres"
    """Name of the default database to create.

    This database will be created during PostgreSQL initialization.
    Default: "postgres"
    """

    username: str = "postgres"
    """Username for the PostgreSQL admin user.

    This user will have superuser privileges on the database.
    Default: "postgres"
    """

    password: str = "postgres"
    """Password for the PostgreSQL admin user.

    This password will be used for database authentication.
    Default: "postgres"
    """

    host: str = "postgres-postgresql.postgres.svc.cluster.local"
    """Fully qualified domain name for PostgreSQL service.

    This is the DNS name that other services will use to connect to PostgreSQL.
    Default: "postgres-postgresql.postgres.svc.cluster.local"
    """

    port: int = 5432
    """Port number for PostgreSQL connections.

    The TCP port that PostgreSQL will listen on.
    Default: 5432
    """

    persistence: dict = {"enabled": True, "size": "20Gi"}
    """Persistence configuration for PostgreSQL data.

    Controls whether data is stored persistently and the storage size.
    """

    resources: dict = {
        "requests": {"memory": "256Mi", "cpu": "250m"},
        "limits": {"memory": "512Mi"},
    }
    """Resource requests and limits for PostgreSQL pods.

    Defines CPU and memory allocations for PostgreSQL containers.
    """

    metrics_enabled: bool = False
    """Whether to enable PostgreSQL metrics collection.

    When enabled, Prometheus metrics will be exposed by PostgreSQL.
    Default: False
    """

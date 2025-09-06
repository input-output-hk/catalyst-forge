"""
Configuration models for temporal task.
"""

from typing import Any, Dict, List
from pydantic import BaseModel


class TemporalConfig(BaseModel):
    """Configuration for Temporal workflow engine deployment.

    This model defines all the parameters needed to deploy and configure
    Temporal workflow engine with PostgreSQL persistence.
    """

    namespace: str = "temporal"
    """Kubernetes namespace where Temporal will be deployed.

    All Temporal components will be installed in this namespace.
    Default: "temporal"
    """

    domain: str = "temporal.projectcatalyst.dev"
    """Domain name for Temporal web interface.

    This domain will be used for HTTP routing to the Temporal web UI.
    Default: "temporal.projectcatalyst.dev"
    """

    database_name: str = "temporal"
    """Name of the PostgreSQL database for Temporal.

    This database will be used for Temporal's persistence layer.
    Default: "temporal"
    """

    namespaces: List[str] = ["default"]
    """List of Temporal namespaces to create.

    These namespaces will be created in Temporal for workflow isolation.
    Default: ["default"]
    """

    web_enabled: bool = True
    """Whether to enable the Temporal web UI.

    The web UI provides a graphical interface for monitoring workflows.
    Default: True
    """

    database_schema: Dict[str, Any] = {"setup_enabled": True, "update_enabled": True}
    """Database schema management configuration for Temporal.

    Controls automatic schema setup and updates for the database.
    """

    server: Dict[str, Any] = {
        "replica_count": 1,
        "num_history_shards": 1,
        "max_conns": 20,
        "max_idle_conns": 2,
        "max_conn_lifetime": "1h",
    }
    """Server configuration for Temporal.

    Controls replication, sharding, and database connection settings.
    """

    web: Dict[str, Any] = {"replica_count": 1}
    """Web UI configuration for Temporal.

    Controls the replication settings for the web UI component.
    """

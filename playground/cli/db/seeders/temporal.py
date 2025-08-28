from __future__ import annotations

from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING

from ..base import DatabaseRoot, Seeder
from ..registry import register_seeder

if TYPE_CHECKING:  # avoid runtime cycles
    from ...deps import Deps
    from ...config import ConfigState
from ...secrets.localstack import ensure_secret_json


@dataclass
class TemporalConfig:
    database_default: str = "temporal"
    database_visibility: str = "temporal_visibility"
    username: str = "temporal"
    password: str = "temporal_password"
    pg_namespace: str = "postgres"
    pg_release: str = "postgres"
    host_override: Optional[str] = None


class TemporalSeeder(Seeder):
    def __init__(self, config: TemporalConfig | None = None) -> None:
        self.config = config or TemporalConfig()

    def run(self, root: DatabaseRoot) -> None:
        root.ensure_role(self.config.username, self.config.password)
        root.ensure_database(self.config.database_default, self.config.username)
        root.ensure_database(self.config.database_visibility, self.config.username)

        host = (
            self.config.host_override
            if self.config.host_override
            else f"{self.config.pg_release}-postgresql.{self.config.pg_namespace}.svc.cluster.local"
        )

        # Seed LocalStack secrets to be consumed via ESO → K8s Secrets used by the Temporal chart
        ensure_secret_json(
            name="shared-services/db/temporal",
            payload={
                "host": host,
                "port": "5432",
                "username": self.config.username,
                "password": self.config.password,
            },
        )
        ensure_secret_json(
            name="shared-services/db/temporal_visibility",
            payload={
                "host": host,
                "port": "5432",
                "username": self.config.username,
                "password": self.config.password,
            },
        )


@register_seeder(
    name="temporal",
    description="Seed Postgres role/db and LocalStack secrets for Temporal",
    priority=40,
    dependencies=(),
)
def _factory(ctx: "ConfigState", deps: "Deps") -> Optional[TemporalSeeder]:
    # No config gating yet; always include with defaults.
    return TemporalSeeder()

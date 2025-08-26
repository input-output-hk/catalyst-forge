from __future__ import annotations

from dataclasses import dataclass
from typing import Optional

from ..base import DatabaseRoot, Seeder
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

from __future__ import annotations

from dataclasses import dataclass

from ..base import DatabaseRoot, Seeder
from ...secrets.localstack import ensure_secret_json


@dataclass
class KratosConfig:
    database: str = "kratos"
    username: str = "kratos"
    password: str = "kratos_password"


class KratosSeeder(Seeder):
    def __init__(self, config: KratosConfig | None = None) -> None:
        self.config = config or KratosConfig()

    def run(self, root: DatabaseRoot) -> None:
        root.ensure_role(self.config.username, self.config.password)
        root.ensure_database(self.config.database, self.config.username)
        # Seed LocalStack secret for DSN components consumed by ESO
        ensure_secret_json(
            name="shared-services/db/kratos",
            payload={
                "host": "postgres-postgresql.postgres.svc.cluster.local",
                "port": "5432",
                "username": self.config.username,
                "password": self.config.password,
            },
        )
        # Seed OIDC client credentials for Kratos (google)
        ensure_secret_json(
            name="shared-services/auth/kratos/oidc/google",
            payload={
                "client_id": "kratos-mock-client",
                "client_secret": "kratos-mock-secret",
            },
        )

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
class ForgeConfig:
    database: str = "forge"
    username: str = "forge"
    password: str = "forge_password"
    pg_namespace: str = "postgres"
    pg_release: str = "postgres"
    host_override: Optional[str] = None


class ForgeSeeder(Seeder):
    def __init__(self, config: ForgeConfig | None = None) -> None:
        self.config = config or ForgeConfig()

    def run(self, root: DatabaseRoot) -> None:
        root.ensure_role(self.config.username, self.config.password)
        root.ensure_database(self.config.database, self.config.username)

        # Seed LocalStack secrets expected by Forge via ExternalSecret
        host = (
            self.config.host_override
            if self.config.host_override
            else f"{self.config.pg_release}-postgresql.{self.config.pg_namespace}.svc.cluster.local"
        )
        ensure_secret_json(
            name="shared-services/db/foundry",
            payload={
                "host": host,
                "port": "5432",
                "username": self.config.username,
                "password": self.config.password,
            },
        )
        ensure_secret_json(
            name="shared-services/db/root_account",
            payload={
                "username": "postgres",
                "password": "postgres",
            },
        )


@register_seeder(
    name="forge",
    description="Seed Postgres role/db and LocalStack secrets for Forge",
    priority=30,
    dependencies=(),
)
def _factory(ctx: "ConfigState", deps: "Deps") -> Optional[ForgeSeeder]:
    # No config gating yet; always include with defaults.
    return ForgeSeeder()

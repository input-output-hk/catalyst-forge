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
class GiteaConfig:
    database: str = "gitea"
    username: str = "gitea"
    password: str = "gitea_password"


class GiteaSeeder(Seeder):
    def __init__(self, config: GiteaConfig | None = None) -> None:
        self.config = config or GiteaConfig()

    def run(self, root: DatabaseRoot) -> None:
        root.ensure_role(self.config.username, self.config.password)
        root.ensure_database(self.config.database, self.config.username)
        # Seed LocalStack secret for DSN components consumed by ESO
        ensure_secret_json(
            name="shared-services/db/gitea",
            payload={
                "host": "postgres-postgresql.postgres.svc.cluster.local",
                "port": "5432",
                "username": self.config.username,
                "password": self.config.password,
                "database": self.config.database,
            },
        )


@register_seeder(
    name="gitea",
    description="Seed Postgres role/db and LocalStack secrets for Gitea",
    priority=25,
    dependencies=(),
)
def _factory(ctx: "ConfigState", deps: "Deps") -> Optional[GiteaSeeder]:
    # Always include with defaults for playground usage.
    return GiteaSeeder()

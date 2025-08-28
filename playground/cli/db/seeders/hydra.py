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
class HydraConfig:
    database: str = "hydra"
    username: str = "hydra"
    password: str = "hydra_password"


class HydraSeeder(Seeder):
    def __init__(self, config: HydraConfig | None = None) -> None:
        self.config = config or HydraConfig()

    def run(self, root: DatabaseRoot) -> None:
        root.ensure_role(self.config.username, self.config.password)
        root.ensure_database(self.config.database, self.config.username)
        # Seed LocalStack secret for DSN components consumed by ESO
        ensure_secret_json(
            name="shared-services/db/hydra",
            payload={
                "host": "postgres-postgresql.postgres.svc.cluster.local",
                "port": "5432",
                "username": self.config.username,
                "password": self.config.password,
            },
        )


@register_seeder(
    name="hydra",
    description="Seed Postgres role/db and LocalStack secrets for Ory Hydra",
    priority=20,
    dependencies=(),
)
def _factory(ctx: "ConfigState", deps: "Deps") -> Optional[HydraSeeder]:
    # No config gating yet; always include with defaults.
    return HydraSeeder()

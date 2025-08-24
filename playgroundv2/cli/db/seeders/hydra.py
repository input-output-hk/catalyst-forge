from __future__ import annotations

from dataclasses import dataclass

from ..base import DatabaseRoot, Seeder


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

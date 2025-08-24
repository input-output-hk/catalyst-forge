from __future__ import annotations

from dataclasses import dataclass

from ..base import DatabaseRoot, Seeder


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

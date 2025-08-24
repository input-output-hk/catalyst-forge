from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol

import psycopg
from psycopg import sql


class Seeder(Protocol):
    """Protocol for pluggable seeders."""

    def run(self, root: "DatabaseRoot") -> None:  # pragma: no cover - simple protocol
        ...


@dataclass
class DatabaseConfig:
    host: str = "postgres-postgresql.postgres.svc.cluster.local"
    port: int = 5432
    user: str = "postgres"
    password: str = "postgres"
    admin_db: str = "postgres"


class DatabaseRoot:
    """Manage root Postgres connection and helpers for idempotent operations."""

    def __init__(self, config: DatabaseConfig | None = None) -> None:
        self.config = config or DatabaseConfig()

    def _connect(self, dbname: str | None = None) -> psycopg.Connection:
        target_db = dbname or self.config.admin_db
        dsn = (
            f"postgres://{self.config.user}:{self.config.password}"
            f"@{self.config.host}:{self.config.port}/{target_db}?sslmode=disable"
        )
        return psycopg.connect(dsn)

    def ensure_role(self, username: str, password: str) -> None:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute("SELECT 1 FROM pg_roles WHERE rolname = %s", (username,))
            exists = cur.fetchone() is not None
            if not exists:
                cur.execute(
                    sql.SQL("CREATE ROLE {} LOGIN PASSWORD {};").format(
                        sql.Identifier(username), sql.Literal(password)
                    )
                )

    def ensure_database(self, dbname: str, owner: str) -> None:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute("SELECT 1 FROM pg_database WHERE datname = %s", (dbname,))
            exists = cur.fetchone() is not None
            if exists:
                cur.execute(
                    sql.SQL("ALTER DATABASE {} OWNER TO {};").format(
                        sql.Identifier(dbname), sql.Identifier(owner)
                    )
                )
                return
        # CREATE DATABASE must run outside a transaction
        conn = self._connect()
        try:
            conn.autocommit = True
            with conn.cursor() as cur:
                cur.execute(
                    sql.SQL("CREATE DATABASE {} OWNER {};").format(
                        sql.Identifier(dbname), sql.Identifier(owner)
                    )
                )
        finally:
            conn.close()

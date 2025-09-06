from __future__ import annotations

from dataclasses import dataclass
from typing import Optional, Any

from ..config import ConfigState
from ..utils import log

from ..db.registry import registry as seeder_registry
from .base import SetupTask
from .registry import register_setup_task


@dataclass
class MigrateSetup(SetupTask):
    state: ConfigState
    deps: Any

    def run(self) -> None:
        # Import seeders to populate registry
        import cli.db.seeders as seeders_pkg
        import importlib
        import pkgutil
        import subprocess
        import time

        for mod in pkgutil.iter_modules(seeders_pkg.__path__, seeders_pkg.__name__ + "."):
            importlib.import_module(mod.name)

        metas = list(seeder_registry.all())
        metas.sort(key=lambda m: (m.priority, m.name))

        # Optional runtime filtering: tasks can read their scoped args from state.runtime
        runtime = getattr(self.state, "runtime", None) or {}
        mig_rt = runtime.get("migrate", {}) if isinstance(runtime, dict) else {}
        only = mig_rt.get("only") if isinstance(mig_rt, dict) else None
        skip = mig_rt.get("skip") if isinstance(mig_rt, dict) else None
        if isinstance(only, str):
            only_set = {only}
        elif isinstance(only, list):
            only_set = {str(x) for x in only}
        else:
            only_set = set()
        if isinstance(skip, str):
            skip_set = {skip}
        elif isinstance(skip, list):
            skip_set = {str(x) for x in skip}
        else:
            skip_set = set()
        if only_set:
            metas = [m for m in metas if m.name in only_set]
        if skip_set:
            metas = [m for m in metas if m.name not in skip_set]

        root = self.deps.db

        # If DB is not reachable, attempt a temporary kubectl port-forward to Postgres
        def _can_connect(r) -> bool:
            try:
                conn = r._connect()  # type: ignore[attr-defined]
                conn.close()
                return True
            except Exception:
                return False

        pf: subprocess.Popen[bytes] | None = None
        try:
            if not _can_connect(root):
                raw_deps = (self.state.raw or {}).get("deps", {})
                raw_db = raw_deps.get("db", {}) if isinstance(raw_deps, dict) else {}
                ns = raw_db.get("pg_namespace", "postgres")
                release = raw_db.get("pg_release", "postgres")
                svc = f"{release}-postgresql"
                local_port = int(raw_db.get("local_forward_port", 65432))

                pf = subprocess.Popen(
                    [
                        "kubectl",
                        "-n",
                        ns,
                        "port-forward",
                        f"svc/{svc}",
                        f"{local_port}:5432",
                    ],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )

                # Build a localhost DatabaseRoot for the forwarded port
                from ..db.base import DatabaseConfig, DatabaseRoot  # local import to avoid cycles

                alt_root = DatabaseRoot(
                    DatabaseConfig(
                        host="127.0.0.1",
                        port=local_port,
                        user=root.config.user,
                        password=root.config.password,
                        admin_db=root.config.admin_db,
                    )
                )

                # Wait briefly for the port-forward to become ready
                for _ in range(20):
                    if _can_connect(alt_root):
                        root = alt_root
                        log(
                            f"Using localhost:{local_port} via kubectl port-forward to svc/{svc} in ns/{ns}"
                        )
                        break
                    time.sleep(0.25)

            ran_any = False
            for meta in metas:
                seeder = meta.factory(self.state, self.deps)
                if seeder is None:
                    log(f"[INFO] Skipping seeder '{meta.name}' (not applicable)")
                    continue
                log(
                    f"[INFO] Running seeder '{meta.name}' (prio={meta.priority}) — {meta.description}"
                )
                seeder.run(root)
                ran_any = True

            if not ran_any:
                log("[INFO] No seeders to run")
            else:
                log("Database migration completed.")
        finally:
            if pf is not None:
                try:
                    pf.terminate()
                except Exception:
                    pass


@register_setup_task(
    name="migrate",
    description="Initialize PostgreSQL roles/databases and secrets via seeders",
    priority=35,
    dependencies=("db",),
)
def _factory(ctx: ConfigState, deps) -> Optional[MigrateSetup]:
    return MigrateSetup(state=ctx, deps=deps)

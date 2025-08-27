from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Optional, Any

import yaml
from pydantic import BaseModel, Field

from ..config import ConfigState
from ..utils import get_client_cert_paths, get_repo_root, log, require_cmd
from ..k3d_ops import get_mkcert_caroot
from .base import SetupTask
from .registry import register_setup_task


class GenerateTaskConfig(BaseModel):
    """Configuration for generate task: TLS certs and Earthly config."""

    client_name: str = Field(default="earthly-client")
    buildkit_host: str = Field(default="tcp://buildkit.projectcatalyst.dev:8372")


@dataclass
class GenerateSetup(SetupTask):
    cfg: GenerateTaskConfig
    runner: Any

    def run(self) -> None:
        require_cmd("mkcert")

        # Resolve repo and playground paths
        playground_dir = Path(__file__).resolve().parent.parent  # .../playground
        repo_root = get_repo_root(playground_dir)
        cert_dir = repo_root / "playground/.certs"
        cert_dir.mkdir(parents=True, exist_ok=True)

        # Generate client TLS certs if missing
        client = get_client_cert_paths(cert_dir, self.cfg.client_name)
        caroot = get_mkcert_caroot()
        ca_src = caroot / "rootCA.pem"

        if not client["cert"].exists() or not client["key"].exists():
            log(
                f"Generating mkcert client cert at {client['cert']} and {client['key']} for {self.cfg.client_name}"
            )
            self.runner.run(
                [
                    "mkcert",
                    "-client",
                    "-cert-file",
                    str(client["cert"].resolve()),
                    "-key-file",
                    str(client["key"].resolve()),
                    self.cfg.client_name,
                ]
            )
        if ca_src.exists() and not client["ca"].exists():
            client["ca"].write_bytes(ca_src.read_bytes())

        log(
            "Client TLS ready for Earthly CLI (mtls):\n"
            f"- cert: {client['cert']}\n"
            f"- key:  {client['key']}\n"
            f"- ca:   {client['ca']}\n"
            "Configure your Earthly client to use these paths."
        )

        # Write Earthly configuration with absolute TLS paths
        earthly_cfg_path = repo_root / "playground/config/earthly.yml"
        earthly_cfg_path.parent.mkdir(parents=True, exist_ok=True)
        cfg = {
            "global": {
                "buildkit_host": self.cfg.buildkit_host,
                "tlsca": str(client["ca"].resolve()),
                "tlscert": str(client["cert"].resolve()),
                "tlskey": str(client["key"].resolve()),
            }
        }
        earthly_cfg_path.write_text(yaml.safe_dump(cfg, sort_keys=False))
        log(f"Earthly config written to {earthly_cfg_path}")


@register_setup_task(
    name="generate",
    description="Generate local TLS certs and Earthly config (tasks.generate)",
    priority=30,
    dependencies=("runner",),
)
def _factory(ctx: ConfigState, deps) -> Optional[GenerateSetup]:
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw_gen = raw_tasks.get("generate") if isinstance(raw_tasks, dict) else None
    cfg = GenerateTaskConfig() if raw_gen is None else GenerateTaskConfig.model_validate(raw_gen)
    return GenerateSetup(cfg=cfg, runner=deps.runner)

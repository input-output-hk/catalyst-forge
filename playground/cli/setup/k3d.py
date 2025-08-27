from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Optional

from pydantic import BaseModel, Field

from ..config import ConfigState
from ..utils import log, require_cmd
from ..k3d_ops import (
    cluster_exists,
    create_cluster,
    delete_cluster,
    emit_cluster_json,
    get_mkcert_caroot,
    wait_for_nodes_ready,
    write_kubeconfig,
    write_registries_yaml,
)
from .base import SetupTask
from .registry import register_setup_task


class K3dTaskConfig(BaseModel):
    """Configuration for k3d cluster bootstrap as a setup task.

    Fields mirror the previous `k3d` subcommand flags and provide sensible
    defaults for local development.
    """

    enabled: bool = Field(default=True)
    name: str = Field(default="forge")
    servers: int = Field(default=1, ge=1)
    agents: int = Field(default=0, ge=0)
    http_port: int = Field(default=80, ge=1, le=65535)
    https_port: int = Field(default=443, ge=1, le=65535)
    api_port: int = Field(default=0, ge=0, le=65535)
    kubeconfig_out: str = Field(default="playground/kubeconfig")
    output_json: str = Field(default="playground/cluster.json")
    force_recreate: bool = Field(default=False)
    assume_yes: bool = Field(default=False)
    registry_host: str = Field(default="registry.projectcatalyst.dev")


@dataclass
class K3dSetup(SetupTask):
    cfg: K3dTaskConfig

    def run(self) -> None:
        if not self.cfg.enabled:
            log("k3d setup task disabled; skipping")
            return

        # Preflight required CLIs
        for cmd in ("docker", "k3d", "kubectl", "mkcert"):
            require_cmd(cmd)

        # Resolve playground directory from this file's location
        playground_dir = Path(__file__).resolve().parent.parent  # .../playground

        # Prepare mkcert CA and k3s registries.yaml with trusted CA
        caroot = get_mkcert_caroot()
        ca_file = caroot / "rootCA.pem"
        tmpdir = (playground_dir / ".certs").resolve()
        tmpdir.mkdir(parents=True, exist_ok=True)
        registries_yaml = write_registries_yaml(
            tmpdir=tmpdir,
            registry_host=self.cfg.registry_host,
            ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt"),
        ).resolve()

        volume_mounts = [
            f"{ca_file}:/etc/ssl/certs/mkcert-rootCA.crt@server:*;agent:*",
            f"{registries_yaml}:/etc/rancher/k3s/registries.yaml@server:*;agent:*",
        ]

        # Create or reuse cluster
        if cluster_exists(self.cfg.name):
            log(f"Cluster '{self.cfg.name}' already exists.")
            if self.cfg.force_recreate:
                delete_cluster(self.cfg.name)
                create_cluster(
                    self.cfg.name,
                    self.cfg.servers,
                    self.cfg.agents,
                    self.cfg.http_port,
                    self.cfg.https_port,
                    self.cfg.api_port,
                    extra_volumes=volume_mounts,
                )
            else:
                log("Reusing existing cluster.")
        else:
            create_cluster(
                self.cfg.name,
                self.cfg.servers,
                self.cfg.agents,
                self.cfg.http_port,
                self.cfg.https_port,
                self.cfg.api_port,
                extra_volumes=volume_mounts,
            )

        # Kubeconfig and readiness
        kubeconfig_out = Path(self.cfg.kubeconfig_out)
        write_kubeconfig(self.cfg.name, kubeconfig_out, assume_yes=self.cfg.assume_yes)
        wait_for_nodes_ready(kubeconfig_out)

        # Emit summary JSON for automation
        emit_cluster_json(
            path=Path(self.cfg.output_json),
            name=self.cfg.name,
            servers=self.cfg.servers,
            agents=self.cfg.agents,
            http_port=self.cfg.http_port,
            https_port=self.cfg.https_port,
            kubeconfig=kubeconfig_out,
        )


@register_setup_task(
    name="k3d",
    description="Create or reuse a local k3d cluster and write kubeconfig",
    priority=20,
    dependencies=(),
)
def _factory(ctx: ConfigState, _deps) -> Optional[K3dSetup]:
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw_cfg = raw_tasks.get("k3d") if isinstance(raw_tasks, dict) else None
    cfg = K3dTaskConfig() if raw_cfg is None else K3dTaskConfig.model_validate(raw_cfg)
    if not cfg.enabled:
        return None
    return K3dSetup(cfg=cfg)

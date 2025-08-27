from __future__ import annotations

from dataclasses import dataclass
from typing import Optional, Any

from pydantic import BaseModel, Field

from ..config import ConfigState
from ..utils import log
from .base import SetupTask
from .registry import register_setup_task


class DnsTaskConfig(BaseModel):
    """Configuration for DNS wildcard pinning.

    By default this task is enabled and targets the base domain
    `projectcatalyst.dev`, inserting a wildcard template into the CoreDNS
    Corefile that maps all subdomains to the Envoy Service ClusterIP.
    """

    domain: str = Field(default="projectcatalyst.dev")
    envoy_namespace: str = Field(default="envoy-gateway-system")
    coredns_namespace: str = Field(default="kube-system")
    coredns_configmap: str = Field(default="coredns")


@dataclass
class DnsSetup(SetupTask):
    cfg: DnsTaskConfig
    deps: Any

    def run(self) -> None:
        # Discover Envoy Service ClusterIP
        core = self.deps.k8s["core"]
        apps = self.deps.k8s["apps"]

        svcs = core.list_namespaced_service(
            self.cfg.envoy_namespace, label_selector="app.kubernetes.io/name=envoy"
        ).items
        if not svcs:
            raise RuntimeError(f"Envoy Service not found in namespace {self.cfg.envoy_namespace}")
        envoy_ip = svcs[0].spec.cluster_ip

        # Fetch CoreDNS Corefile
        cm = core.read_namespaced_config_map(
            name=self.cfg.coredns_configmap, namespace=self.cfg.coredns_namespace
        )
        corefile: str = cm.data.get("Corefile", "") if cm and cm.data else ""
        if not corefile:
            raise RuntimeError("CoreDNS Corefile not found")

        begin = f"# BEGIN catalyst-forge wildcard {self.cfg.domain}"
        end = f"# END catalyst-forge wildcard {self.cfg.domain}"

        # Remove existing managed block if present (anywhere in file)
        if begin in corefile and end in corefile:
            pre, _, rest = corefile.partition(begin)
            _, _, post = rest.partition(end)
            corefile = pre.rstrip() + "\n" + post.lstrip()

        # Build managed block
        escaped_domain = self.cfg.domain.replace(".", "\\.")
        managed_block = (
            f"{begin}\n"
            f"    template IN A {self.cfg.domain} {{\n"
            f'        match "^([a-z0-9-]+\\.)*{escaped_domain}\\.$"\n'
            f'        answer "{{{{ .Name }}}} 300 IN A {envoy_ip}"\n'
            f"        fallthrough\n"
            f"    }}\n"
            f"{end}\n"
        )

        # Insert inside the main .:53 server block before its closing brace
        lines = corefile.splitlines()
        start_idx = -1
        brace_depth = 0
        in_main = False
        for i, ln in enumerate(lines):
            stripped = ln.strip()
            if start_idx < 0 and stripped.startswith(".:53") and stripped.endswith("{"):
                start_idx = i
                brace_depth = 1
                in_main = True
                continue
            if in_main:
                brace_depth += ln.count("{")
                brace_depth -= ln.count("}")
                if brace_depth == 0:
                    insert_at = i
                    if insert_at > 0 and lines[insert_at - 1].strip() != "":
                        managed_block_to_insert = "\n" + managed_block
                    else:
                        managed_block_to_insert = managed_block
                    new_lines = (
                        lines[:insert_at]
                        + [managed_block_to_insert.rstrip("\n")]
                        + lines[insert_at:]
                    )
                    cm.data["Corefile"] = "\n".join(new_lines) + "\n"
                    core.patch_namespaced_config_map(
                        name=self.cfg.coredns_configmap,
                        namespace=self.cfg.coredns_namespace,
                        body=cm,
                    )

                    # Restart CoreDNS to apply changes
                    from datetime import datetime

                    ts = datetime.utcnow().isoformat() + "Z"
                    apps.patch_namespaced_deployment(
                        name="coredns",
                        namespace=self.cfg.coredns_namespace,
                        body={
                            "spec": {
                                "template": {
                                    "metadata": {
                                        "annotations": {"kubectl.kubernetes.io/restartedAt": ts}
                                    }
                                }
                            }
                        },
                    )
                    log(
                        "CoreDNS wildcard template applied inside main server block and restart triggered"
                    )
                    return

        raise RuntimeError("Could not find main .:53 server block in CoreDNS Corefile")


@register_setup_task(
    name="dns",
    description="Pin CoreDNS wildcard to Envoy ClusterIP (tasks.dns)",
    priority=40,
    dependencies=("k8s",),
)
def _factory(ctx: ConfigState, deps) -> Optional[DnsSetup]:
    # Use defaults if no explicit config is provided, so DNS setup runs by default.
    raw_tasks = (ctx.raw or {}).get("tasks", {})
    raw_dns = raw_tasks.get("dns") if isinstance(raw_tasks, dict) else None
    if raw_dns is None:
        cfg = DnsTaskConfig()
    else:
        cfg = DnsTaskConfig.model_validate(raw_dns)
    return DnsSetup(cfg=cfg, deps=deps)

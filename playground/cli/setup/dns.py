from __future__ import annotations

from dataclasses import dataclass
from typing import Optional, Any, Tuple

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

    def _render_managed_block(self, domain: str, envoy_ip: str) -> str:
        escaped_domain = domain.replace(".", "\\.")
        begin = f"# BEGIN catalyst-forge wildcard {domain}"
        end = f"# END catalyst-forge wildcard {domain}"
        return (
            f"{begin}\n"
            f"    template IN A {domain} {{\n"
            f'        match "^([a-z0-9-]+\\.)*{escaped_domain}\\.$"\n'
            f'        answer "{{{{ .Name }}}} 300 IN A {envoy_ip}"\n'
            f"        fallthrough\n"
            f"    }}\n"
            f"{end}\n"
        )

    def _find_main_server_block(self, lines: list[str]) -> Tuple[int, int]:
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
                    return start_idx, i
        return -1, -1

    def _merge_corefile(
        self, corefile: str, managed_block: str, begin: str, end: str
    ) -> tuple[str, bool, bool]:
        original = corefile if corefile.endswith("\n") else corefile + "\n"

        # Remove existing managed block if present
        if begin in corefile and end in corefile:
            pre, _, rest = corefile.partition(begin)
            _, _, post = rest.partition(end)
            corefile = pre.rstrip() + "\n" + post.lstrip()

        lines = corefile.splitlines()
        start_idx, close_idx = self._find_main_server_block(lines)
        if start_idx < 0 or close_idx < 0:
            raise RuntimeError("Could not find main .:53 server block in CoreDNS Corefile")

        # Detect reload plugin inside main server block
        has_reload_plugin = any("reload" in (ln.strip()) for ln in lines[start_idx : close_idx + 1])

        insert_at = close_idx
        block_to_insert = managed_block.rstrip("\n")
        if insert_at > 0 and lines[insert_at - 1].strip() != "":
            block_to_insert = "\n" + block_to_insert

        new_lines = lines[:insert_at] + [block_to_insert] + lines[insert_at:]
        new_corefile = "\n".join(new_lines) + "\n"

        return new_corefile, new_corefile != original, has_reload_plugin

    def run(self) -> None:
        # Discover Envoy Service ClusterIP
        core = self.deps.k8s["core"]
        apps = self.deps.k8s["apps"]

        svcs = core.list_namespaced_service(
            self.cfg.envoy_namespace, label_selector="app.kubernetes.io/name=envoy"
        ).items
        if not svcs:
            raise RuntimeError(f"Envoy Service not found in namespace {self.cfg.envoy_namespace}")
        # Prefer a service with a valid ClusterIP
        envoy_ip = None
        for s in svcs:
            ip = getattr(getattr(s, "spec", None), "cluster_ip", None)
            if isinstance(ip, str) and ip:
                envoy_ip = ip
                break
        if not envoy_ip:
            raise RuntimeError("Envoy Service has no ClusterIP; cannot pin wildcard DNS")

        # Fetch CoreDNS Corefile
        cm = core.read_namespaced_config_map(
            name=self.cfg.coredns_configmap, namespace=self.cfg.coredns_namespace
        )
        corefile: str = cm.data.get("Corefile", "") if cm and cm.data else ""
        if not corefile:
            raise RuntimeError("CoreDNS Corefile not found")

        begin = f"# BEGIN catalyst-forge wildcard {self.cfg.domain}"
        end = f"# END catalyst-forge wildcard {self.cfg.domain}"
        managed_block = self._render_managed_block(self.cfg.domain, envoy_ip)
        new_corefile, changed, has_reload = self._merge_corefile(
            corefile, managed_block, begin, end
        )

        if not changed:
            log("DNS: CoreDNS Corefile already up-to-date; no changes")
            return

        # Apply changes
        cm.data["Corefile"] = new_corefile
        core.patch_namespaced_config_map(
            name=self.cfg.coredns_configmap,
            namespace=self.cfg.coredns_namespace,
            body=cm,
        )

        if has_reload:
            log("DNS: Corefile updated; reload plugin detected, skipping CoreDNS restart")
            return

        # Restart CoreDNS to apply changes (only when reload plugin not present)
        from datetime import datetime

        ts = datetime.utcnow().isoformat() + "Z"
        apps.patch_namespaced_deployment(
            name="coredns",
            namespace=self.cfg.coredns_namespace,
            body={
                "spec": {
                    "template": {
                        "metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": ts}}
                    }
                }
            },
        )
        log("DNS: Corefile updated; CoreDNS restart triggered")


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

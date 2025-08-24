#!/usr/bin/env python3
"""Playground v2 CLI for local k3d cluster lifecycle.

This CLI provides commands to create or reuse a local k3d Kubernetes cluster
configured for the Catalyst Forge playground. It disables the default k3s
ingress controller (Traefik), maps host ports 80/443 to the k3d load balancer
for Envoy Gateway, waits for nodes to be Ready, writes a kubeconfig file, and
emits a cluster.json summary for automation.

Usage examples:
    uv run python playgroundv2/cli/main.py k3d --name forge --servers 1 \
        --agents 0 --kubeconfig-out playgroundv2/kubeconfig \
        --output-json playgroundv2/cluster.json --yes
"""

from __future__ import annotations

from pathlib import Path

import typer
import yaml

from .k3d_ops import (
    cluster_exists,
    create_cluster,
    delete_cluster,
    emit_cluster_json,
    get_mkcert_caroot,
    wait_for_nodes_ready,
    write_kubeconfig,
    write_registries_yaml,
)
from .models import ClusterSummary  # re-export for tests/importers
from .utils import get_client_cert_paths, get_repo_root, log, require_cmd, run
from .db import DatabaseRoot, KratosSeeder, HydraSeeder, ForgeSeeder, TemporalSeeder
from .db.base import DatabaseConfig

# Resolve the playground root (one level up from this file's directory)
BASE_DIR = Path(__file__).resolve().parent.parent

app = typer.Typer(help="CLI helpers for the Playground v2 local environment")

# Re-export public API for downstream imports/tests
__all__ = ["ClusterSummary", "k3d_up", "app"]
dns_app = typer.Typer(help="DNS helpers (CoreDNS pinning)")
app.add_typer(dns_app, name="dns")


@dns_app.command("pin-registry")
def dns_pin_registry(
    host: list[str] = typer.Option(
        [
            "registry.projectcatalyst.dev",
            "auth-mock.projectcatalyst.dev",
        ],
        "--host",
        help="Hostname(s) to pin to the Envoy Service ClusterIP (repeatable)",
    ),
) -> None:
    """Pin one or more hostnames to Envoy Service ClusterIP in CoreDNS NodeHosts (idempotent)."""
    from kubernetes import client, config

    # Load kubeconfig
    config.load_kube_config(config_file=str((BASE_DIR / "kubeconfig").resolve()))
    v1 = client.CoreV1Api()

    # Discover Envoy Service ClusterIP
    svcs = v1.list_namespaced_service(
        "envoy-gateway-system", label_selector="app.kubernetes.io/name=envoy"
    ).items
    if not svcs:
        raise RuntimeError("Envoy Service not found in envoy-gateway-system")
    envoy_ip = svcs[0].spec.cluster_ip

    # Fetch CoreDNS ConfigMap (Corefile + NodeHosts)
    cm = v1.read_namespaced_config_map("coredns", "kube-system")
    corefile = cm.data.get("Corefile", "")
    node_hosts = cm.data.get("NodeHosts", "")
    if not corefile:
        raise RuntimeError("CoreDNS Corefile not found")

    # Ensure mappings exist in NodeHosts (file-mode for hosts plugin)
    node_lines = node_hosts.splitlines() if node_hosts else []
    existing = {ln.strip() for ln in node_lines}
    for h in host:
        mapping_line = f"{envoy_ip} {h}"
        if mapping_line not in existing:
            node_lines.append(mapping_line)
            existing.add(mapping_line)
    cm.data["NodeHosts"] = "\n".join(node_lines) + "\n"

    # Remove any stray inline mapping lines from Corefile hosts block (avoid parse errors)
    if any(h in corefile for h in host):
        pruned: list[str] = []
        for ln in corefile.splitlines():
            # Drop inline IP mappings for any of the target hosts
            if any(h in ln for h in host) and ln.strip().split()[0].replace(".", "").isdigit():
                continue
            pruned.append(ln)
        cm.data["Corefile"] = "\n".join(pruned)

    v1.patch_namespaced_config_map("coredns", "kube-system", cm)
    # Restart CoreDNS (force new pod template annotation with timestamp)
    from datetime import datetime

    ts = datetime.utcnow().isoformat() + "Z"
    apps = client.AppsV1Api()
    apps.patch_namespaced_deployment(
        name="coredns",
        namespace="kube-system",
        body={
            "spec": {
                "template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": ts}}}
            }
        },
    )
    log("CoreDNS updated and restart triggered")


@dns_app.command("pin-wildcard")
def dns_pin_wildcard(
    domain: str = typer.Option(
        "projectcatalyst.dev", "--domain", help="Base domain to wildcard to Envoy"
    ),
) -> None:
    """Patch CoreDNS Corefile with a wildcard template that points all subdomains to Envoy.

    Idempotent: updates or inserts a managed block delimited by BEGIN/END markers, placed inside the main .:53 server block.
    """
    from kubernetes import client, config

    # Load kubeconfig
    config.load_kube_config(config_file=str((BASE_DIR / "kubeconfig").resolve()))
    v1 = client.CoreV1Api()

    # Discover Envoy Service ClusterIP
    svcs = v1.list_namespaced_service(
        "envoy-gateway-system", label_selector="app.kubernetes.io/name=envoy"
    ).items
    if not svcs:
        raise RuntimeError("Envoy Service not found in envoy-gateway-system")
    envoy_ip = svcs[0].spec.cluster_ip

    # Fetch CoreDNS ConfigMap (Corefile)
    cm = v1.read_namespaced_config_map("coredns", "kube-system")
    corefile = cm.data.get("Corefile", "")
    if not corefile:
        raise RuntimeError("CoreDNS Corefile not found")

    begin = f"# BEGIN catalyst-forge wildcard {domain}"
    end = f"# END catalyst-forge wildcard {domain}"

    # Remove any existing managed block anywhere
    if begin in corefile and end in corefile:
        pre, _, rest = corefile.partition(begin)
        _, _, post = rest.partition(end)
        corefile = pre.rstrip() + "\n" + post.lstrip()

    # Build managed block (proper indent inside main server block)
    escaped_domain = domain.replace(".", "\\.")
    managed_block = (
        f"{begin}\n"
        f"    template IN A {domain} {{\n"
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
        if start_idx < 0 and ln.strip().startswith(".:53") and ln.strip().endswith("{"):
            start_idx = i
            brace_depth = 1
            in_main = True
            continue
        if in_main:
            # Count braces naively
            brace_depth += ln.count("{")
            brace_depth -= ln.count("}")
            if brace_depth == 0:
                # Insert block just before this closing brace
                insert_at = i
                # Ensure a blank line before managed block
                if insert_at > 0 and lines[insert_at - 1].strip() != "":
                    managed_block = "\n" + managed_block
                new_lines = lines[:insert_at] + [managed_block.rstrip("\n")] + lines[insert_at:]
                cm.data["Corefile"] = "\n".join(new_lines) + "\n"
                v1.patch_namespaced_config_map("coredns", "kube-system", cm)
                # Restart CoreDNS to apply changes
                from datetime import datetime

                ts = datetime.utcnow().isoformat() + "Z"
                apps = client.AppsV1Api()
                apps.patch_namespaced_deployment(
                    name="coredns",
                    namespace="kube-system",
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


def _generate_assets(
    *,
    base_dir: Path,
    _registry_host: str,
    client_name: str,
    buildkit_host: str,
) -> None:
    """Generate local TLS client certs and Earthly configuration.

    Args:
        base_dir: The base directory of the playground (playgroundv2 root).
        registry_host: Hostname for the in-cluster Docker registry.
        client_name: Common Name for the mkcert client certificate.
        buildkit_host: BuildKit TCP endpoint to write into Earthly config.
    """
    # Ensure prerequisites
    require_cmd("mkcert")

    repo_root = get_repo_root(base_dir)
    cert_dir = repo_root / "playgroundv2/.certs"
    cert_dir.mkdir(parents=True, exist_ok=True)

    # 1) Generate client TLS cert for Earthly CLI and copy CA (idempotent)
    client = get_client_cert_paths(repo_root / "playgroundv2/.certs", client_name)
    caroot = get_mkcert_caroot()
    ca_src = caroot / "rootCA.pem"

    if not client["cert"].exists() or not client["key"].exists():
        log(
            f"Generating mkcert client cert at {client['cert']} and {client['key']} for {client_name}"
        )
        run(
            [
                "mkcert",
                "-client",
                "-cert-file",
                str(client["cert"].resolve()),
                "-key-file",
                str(client["key"].resolve()),
                client_name,
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

    # 2) Write Earthly configuration with absolute TLS paths
    earthly_cfg_path = repo_root / "playgroundv2/config/earthly.yml"
    earthly_cfg_path.parent.mkdir(parents=True, exist_ok=True)
    cfg = {
        "global": {
            "buildkit_host": buildkit_host,
            "tlsca": str(client["ca"].resolve()),
            "tlscert": str(client["cert"].resolve()),
            "tlskey": str(client["key"].resolve()),
        }
    }
    earthly_cfg_path.write_text(yaml.safe_dump(cfg, sort_keys=False))
    log(f"Earthly config written to {earthly_cfg_path}")


@app.command("generate")
def generate(
    client_name: str = typer.Option(
        "earthly-client", help="Client certificate common name to generate"
    ),
    buildkit_host: str = typer.Option(
        "tcp://buildkit.projectcatalyst.dev:8372",
        help="BuildKit TCP endpoint to write into Earthly config",
    ),
) -> None:
    """Generate local TLS certs and Earthly config (idempotent)."""
    _generate_assets(
        base_dir=BASE_DIR,
        client_name=client_name,
        buildkit_host=buildkit_host,
    )


@app.command("k3d")
def k3d_up(
    name: str = typer.Option("forge", help="Cluster name"),
    servers: int = typer.Option(1, min=1, help="Number of server nodes"),
    agents: int = typer.Option(0, min=0, help="Number of agent nodes"),
    http_port: int = typer.Option(80, min=1, max=65535, help="Host HTTP port => LB:80"),
    https_port: int = typer.Option(443, min=1, max=65535, help="Host HTTPS port => LB:443"),
    api_port: int = typer.Option(
        0, min=0, max=65535, help="Host API port for kube-apiserver (0=disabled)"
    ),
    kubeconfig_out: Path = typer.Option((BASE_DIR / "kubeconfig"), help="Path to write kubeconfig"),
    output_json: Path = typer.Option(
        (BASE_DIR / "cluster.json"), help="Path to write cluster summary JSON"
    ),
    force_recreate: bool = typer.Option(False, help="Delete and recreate cluster if it exists"),
    yes: bool = typer.Option(False, "--yes", help="Assume 'yes' for prompts and overwrites"),
    registry_host: str = typer.Option(
        "registry.projectcatalyst.dev", help="Registry host DNS name to trust via mkcert CA"
    ),
    registry_name: str = typer.Option("reg", help="Name for the k3d registry instance"),
    generate_client_cert: bool = typer.Option(
        False, help="[Deprecated] Use the 'generate' command instead"
    ),
) -> None:
    """Create or reuse a k3d cluster configured for Envoy Gateway ingress."""

    # Preflight
    for cmd in ("docker", "k3d", "kubectl", "mkcert"):
        require_cmd(cmd)

    caroot = get_mkcert_caroot()
    ca_file = caroot / "rootCA.pem"
    tmpdir = BASE_DIR / ".certs"
    tmpdir.mkdir(parents=True, exist_ok=True)
    registries_yaml = write_registries_yaml(
        tmpdir=tmpdir, registry_host=registry_host, ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt")
    ).resolve()

    volume_mounts = [
        f"{ca_file}:/etc/ssl/certs/mkcert-rootCA.crt@server:*;agent:*",
        f"{registries_yaml}:/etc/rancher/k3s/registries.yaml@server:*;agent:*",
    ]

    # Reuse or create cluster
    if cluster_exists(name):
        log(f"Cluster '{name}' already exists.")
        if force_recreate:
            delete_cluster(name)
            create_cluster(
                name, servers, agents, http_port, https_port, api_port, extra_volumes=volume_mounts
            )
        else:
            log("Reusing existing cluster.")
    else:
        create_cluster(
            name, servers, agents, http_port, https_port, api_port, extra_volumes=volume_mounts
        )

    # Kubeconfig and readiness
    write_kubeconfig(name, kubeconfig_out, assume_yes=yes)
    wait_for_nodes_ready(kubeconfig_out)

    # Summary
    emit_cluster_json(
        path=output_json,
        name=name,
        servers=servers,
        agents=agents,
        http_port=http_port,
        https_port=https_port,
        kubeconfig=kubeconfig_out,
    )

    log(
        "\nCluster is ready.\n\n"
        f"- Name: {name}\n"
        f"- Kubeconfig: {kubeconfig_out}\n"
        f"- Ingress (host): http://127.0.0.1:{http_port} and https://127.0.0.1:{https_port}\n\n"
        "Next: install Envoy Gateway, ESO, LocalStack, Postgres, Mailpit via Helmfile."
    )

    # Deprecated behavior retained for compatibility
    if generate_client_cert:
        log("[Deprecated] --generate-client-cert is deprecated; use 'generate' command instead.")
        _generate_assets(
            base_dir=BASE_DIR,
            registry_host=registry_host,
            client_name="earthly-client",
            buildkit_host="tcp://buildkit.projectcatalyst.dev:8372",
        )


@app.command("migrate")
def migrate(
    pg_host: str = typer.Option("127.0.0.1", help="PostgreSQL host"),
    pg_port: int = typer.Option(5432, help="PostgreSQL port"),
    pg_user: str = typer.Option("postgres", help="PostgreSQL admin user"),
    pg_password: str = typer.Option("postgres", help="PostgreSQL admin password"),
) -> None:
    """Initialize PostgreSQL for dependent apps (Kratos, Hydra).

    Idempotent: safe to re-run. Extensible via additional seeders.
    """
    root = DatabaseRoot(
        DatabaseConfig(
            host=pg_host, port=pg_port, user=pg_user, password=pg_password, admin_db="postgres"
        )
    )
    # Run seeders
    for seeder in (KratosSeeder(), HydraSeeder(), ForgeSeeder(), TemporalSeeder()):
        seeder.run(root)
    log("Database migration completed.")


@app.command("down")
def k3d_down(name: str = typer.Option("forge", help="Cluster name")) -> None:
    """Destroy the k3d cluster and detach resources."""
    if cluster_exists(name):
        delete_cluster(name)
        log(f"Cluster '{name}' deleted.")
    else:
        log(f"Cluster '{name}' not found; nothing to do.")


if __name__ == "__main__":
    app()

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

# Resolve the playground root (one level up from this file's directory)
BASE_DIR = Path(__file__).resolve().parent.parent

app = typer.Typer(help="CLI helpers for the Playground v2 local environment")

# Re-export public API for downstream imports/tests
__all__ = ["ClusterSummary", "k3d_up", "app"]
dns_app = typer.Typer(help="DNS helpers (CoreDNS pinning)")
app.add_typer(dns_app, name="dns")


@dns_app.command("pin-registry")
def dns_pin_registry(host: str = typer.Option("registry.projectcatalyst.dev", "--host")) -> None:
    """Pin a hostname to Envoy Service ClusterIP in CoreDNS NodeHosts (idempotent)."""
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

    # Ensure mapping exists in NodeHosts (file-mode for hosts plugin)
    mapping_line = f"{envoy_ip} {host}"
    node_lines = node_hosts.splitlines() if node_hosts else []
    if not any(ln.strip().endswith(f" {host}") for ln in node_lines):
        node_lines.append(mapping_line)
        cm.data["NodeHosts"] = "\n".join(node_lines) + "\n"

    # Remove any stray inline mapping lines from Corefile hosts block (avoid parse errors)
    if host in corefile:
        pruned: list[str] = []
        for ln in corefile.splitlines():
            if host in ln and ln.strip().split()[0].replace(".", "").isdigit():
                # drop inline mapping in Corefile
                continue
            pruned.append(ln)
        cm.data["Corefile"] = "\n".join(pruned)

    v1.patch_namespaced_config_map("coredns", "kube-system", cm)
    # Restart CoreDNS
    apps = client.AppsV1Api()
    apps.patch_namespaced_deployment(
        name="coredns",
        namespace="kube-system",
        body={
            "spec": {"template": {"metadata": {"annotations": {"restartedAt": str(Path.cwd())}}}}
        },
    )
    log("CoreDNS updated and restart triggered")


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
        True, help="Generate local client certs for Earthly CLI under playgroundv2/.certs"
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

    # Auto-generate cert/key with mkcert under playgroundv2/.certs if missing
    cert_dir = BASE_DIR / ".certs"
    cert_file = cert_dir / f"{registry_host}.pem"
    key_file = cert_dir / f"{registry_host}-key.pem"
    if not cert_file.exists() or not key_file.exists():
        log(f"Generating mkcert certs for {registry_host} at {cert_file} and {key_file}")
        import subprocess as _sub

        cert_dir.mkdir(parents=True, exist_ok=True)
        _sub.run(
            [
                "mkcert",
                "-cert-file",
                str(cert_file.resolve()),
                "-key-file",
                str(key_file.resolve()),
                registry_host,
            ],
            check=True,
        )

    log(
        "\nCluster is ready.\n\n"
        f"- Name: {name}\n"
        f"- Kubeconfig: {kubeconfig_out}\n"
        f"- Ingress (host): http://127.0.0.1:{http_port} and https://127.0.0.1:{https_port}\n\n"
        "Next: install Envoy Gateway, ESO, LocalStack, Postgres, Mailpit via Helmfile."
    )

    # Optionally generate client cert for Earthly CLI mTLS
    if generate_client_cert:
        repo_root = get_repo_root(BASE_DIR)
        client = get_client_cert_paths(repo_root / "playgroundv2/.certs", "earthly-client")
        # Ensure mkcert CAROOT is copied to local .certs for consistent pathing in config
        caroot = get_mkcert_caroot()
        ca_src = caroot / "rootCA.pem"
        client["cert"].parent.mkdir(parents=True, exist_ok=True)
        # Idempotent: only generate if files are missing
        if not client["cert"].exists() or not client["key"].exists():
            log(
                f"Generating mkcert client cert at {client['cert']} and {client['key']} for Earthly CLI"
            )
            run(
                [
                    "mkcert",
                    "-client",
                    "-cert-file",
                    str(client["cert"].resolve()),
                    "-key-file",
                    str(client["key"].resolve()),
                    "earthly-client",
                ]
            )
        # Copy CAROOT rootCA.pem for consistent local CA path if missing
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
        earthly_cfg_path = repo_root / "playgroundv2/config/earthly.yml"
        earthly_cfg_path.parent.mkdir(parents=True, exist_ok=True)
        cfg = {
            "global": {
                "buildkit_host": "tcp://buildkit.projectcatalyst.dev:8372",
                "tlsca": str(client["ca"].resolve()),
                "tlscert": str(client["cert"].resolve()),
                "tlskey": str(client["key"].resolve()),
            }
        }
        earthly_cfg_path.write_text(yaml.safe_dump(cfg, sort_keys=False))
        log(f"Earthly config written to {earthly_cfg_path}")


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

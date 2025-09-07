"""
Create or manage k3d cluster for the playground environment.

This task must run before all other infrastructure tasks as it provides
the Kubernetes cluster itself.
"""

from cli.setupv2 import task
from pathlib import Path
import subprocess
import json
import os

# Import k3d operations from the existing k3d module
from cli.k3d import (
    cluster_exists,
    create_cluster,
    delete_cluster,
    write_kubeconfig,
    wait_for_nodes_ready,
    emit_cluster_json,
    get_mkcert_caroot,
    write_registries_yaml,
)
from cli.utils import require_cmd, get_repo_root
from .config import K3dConfig


@task(
    "k3d",
    provides={
        "cluster_name": str,
        "kubeconfig": str,
        "http_port": int,
        "https_port": int,
        "api_port": int,
        "ready": bool,
    },
    config_keys=["k3d", "registry_host"],
    timeout_sec=600,
    retry_delays=[10, 30],  # k3d cluster creation can be slow
)
def setup_k3d(ctx, cfg, log):
    """Create or reuse k3d cluster and configure kubeconfig."""

    # Parse configuration with validation and defaults
    k3d_config = K3dConfig.model_validate(cfg["k3d"])
    registry_host = cfg["registry_host"]

    log.write(f"Setting up k3d cluster '{k3d_config.cluster_name}'...\n")

    # Preflight checks
    for cmd in ("docker", "k3d", "mkcert"):
        try:
            require_cmd(cmd)
            log.write(f"✓ {cmd} is available\n")
        except Exception as e:
            log.write(f"✗ {cmd} is missing: {e}\n")
            raise RuntimeError(f"Required command '{cmd}' not found")

    # Resolve paths
    playground_dir = Path(__file__).resolve().parent.parent.parent  # .../playground
    repo_root = get_repo_root(playground_dir)

    # Prepare mkcert CA and k3s registries.yaml with trusted CA
    log.write("Preparing mkcert CA certificates...\n")
    caroot = get_mkcert_caroot()
    ca_file = caroot / "rootCA.pem"

    if not ca_file.exists():
        log.write(f"Warning: mkcert CA file not found at {ca_file}\n")
        log.write("Running 'mkcert -install' to create CA...\n")
        result = subprocess.run(["mkcert", "-install"], capture_output=True, text=True)
        log.write(f"{result.stdout}\n")
        if result.returncode != 0:
            log.write(f"Error: {result.stderr}\n")
            raise RuntimeError("Failed to install mkcert CA")

    tmpdir = (playground_dir / ".certs").resolve()
    tmpdir.mkdir(parents=True, exist_ok=True)

    log.write(f"Writing registries configuration for {registry_host}...\n")
    registries_yaml = write_registries_yaml(
        tmpdir=tmpdir,
        registry_host=registry_host,
        ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt"),
    ).resolve()

    # Volume mounts for k3d
    volume_mounts = [
        f"{ca_file}:/etc/ssl/certs/mkcert-rootCA.crt@server:*;agent:*",
        f"{registries_yaml}:/etc/rancher/k3s/registries.yaml@server:*;agent:*",
    ]

    # Create or reuse cluster
    if cluster_exists(k3d_config.cluster_name):
        log.write(f"Cluster '{k3d_config.cluster_name}' already exists.\n")
        if k3d_config.force_recreate:
            log.write("Force recreate enabled, deleting existing cluster...\n")
            delete_cluster(k3d_config.cluster_name)
            log.write(f"Creating new cluster '{k3d_config.cluster_name}'...\n")
            create_cluster(
                k3d_config.cluster_name,
                k3d_config.servers,
                k3d_config.agents,
                k3d_config.http_port,
                k3d_config.https_port,
                k3d_config.api_port,
                extra_volumes=volume_mounts,
            )
            log.write(f"Cluster '{k3d_config.cluster_name}' created successfully.\n")
        else:
            log.write("Reusing existing cluster.\n")
    else:
        log.write(f"Creating new cluster '{k3d_config.cluster_name}'...\n")
        create_cluster(
            k3d_config.cluster_name,
            k3d_config.servers,
            k3d_config.agents,
            k3d_config.http_port,
            k3d_config.https_port,
            k3d_config.api_port,
            extra_volumes=volume_mounts,
        )
        log.write(f"Cluster '{k3d_config.cluster_name}' created successfully.\n")

    # Write kubeconfig
    kubeconfig_path = Path(k3d_config.kubeconfig_out).expanduser()
    if not kubeconfig_path.is_absolute():
        kubeconfig_path = (repo_root / kubeconfig_path).resolve()

    log.write(f"Writing kubeconfig to {kubeconfig_path}...\n")
    write_kubeconfig(k3d_config.cluster_name, kubeconfig_path, assume_yes=k3d_config.assume_yes)

    # Set KUBECONFIG environment variable for subsequent tasks
    os.environ["KUBECONFIG"] = str(kubeconfig_path)
    log.write(f"Set KUBECONFIG={kubeconfig_path}\n")

    # Wait for nodes to be ready
    log.write("Waiting for cluster nodes to be ready...\n")
    wait_for_nodes_ready(kubeconfig_path)
    log.write("All nodes are ready.\n")

    # Emit cluster summary JSON
    output_json_path = Path(k3d_config.output_json).expanduser()
    if not output_json_path.is_absolute():
        output_json_path = (repo_root / output_json_path).resolve()

    log.write(f"Writing cluster info to {output_json_path}...\n")
    emit_cluster_json(
        path=output_json_path,
        name=k3d_config.cluster_name,
        servers=k3d_config.servers,
        agents=k3d_config.agents,
        http_port=k3d_config.http_port,
        https_port=k3d_config.https_port,
        kubeconfig=kubeconfig_path,
    )

    # Get actual API port if it was auto-assigned
    api_port = k3d_config.api_port
    if api_port == 0:
        # Parse the cluster.json to get the actual API port
        with open(output_json_path, "r") as f:
            json.load(f)  # Load but don't store - API port parsing not yet implemented
            # The API port might be in the kubeconfig or cluster info
            # For now, we'll keep it as 0 to indicate auto-assigned
            api_port = 0

    log.write(f"\n✅ k3d cluster '{k3d_config.cluster_name}' is ready!\n")
    log.write(f"   Kubeconfig: {kubeconfig_path}\n")
    log.write(f"   HTTP Port: {k3d_config.http_port}\n")
    log.write(f"   HTTPS Port: {k3d_config.https_port}\n")
    log.write(f"   API Port: {api_port if api_port > 0 else 'auto-assigned'}\n")

    return {
        "cluster_name": k3d_config.cluster_name,
        "kubeconfig": str(kubeconfig_path),
        "http_port": k3d_config.http_port,
        "https_port": k3d_config.https_port,
        "api_port": api_port,
        "ready": True,
    }

"""
Deploy Docker Registry for container image storage.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
from .config import RegistryConfig


@task(
    "registry",
    requires=["k3d.ready"],  # Ensure cluster exists first
    provides={"host": str, "url": str, "username": str, "password": str},
    config_keys=["registry"],  # Load registry configuration
    timeout_sec=300,
    retry_delays=[5, 15],  # Retry after 5s, then 15s
)
def setup_registry(ctx, cfg, log):
    """Deploy Docker Registry for container images."""

    # Parse configuration with validation and defaults
    reg_config = RegistryConfig.model_validate(cfg["registry"])

    log.write(f"Deploying Docker Registry to namespace '{reg_config.namespace}'...\n")

    # Use credentials from configuration
    username = reg_config.username
    password = reg_config.password

    # Generate htpasswd for basic auth
    # Using a simple approach - in production use proper htpasswd generation
    import subprocess
    import tempfile
    import os

    # Create htpasswd file
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as f:
        htpasswd_file = f.name

    # Generate htpasswd entry using Docker registry htpasswd format
    # Note: Using openssl to generate the password hash
    result = subprocess.run(
        ["openssl", "passwd", "-apr1", password], capture_output=True, text=True
    )

    if result.returncode != 0:
        log.write(f"Warning: Could not generate htpasswd: {result.stderr}\n")
        # Fallback to no auth for development
        htpasswd_content = ""
    else:
        password_hash = result.stdout.strip()
        htpasswd_content = f"{username}:{password_hash}"

    with open(htpasswd_file, "w") as f:
        f.write(htpasswd_content)

    # Read the htpasswd content for the secret
    with open(htpasswd_file, "r") as f:
        htpasswd_data = f.read()

    # Clean up temp file
    os.unlink(htpasswd_file)

    # Add Docker Registry Helm repo
    add_repo = subprocess.run(
        ["helm", "repo", "add", "twuni", "https://helm.twun.io"], capture_output=True, text=True
    )
    log.write(f"Added Docker Registry repo: {add_repo.stdout}\n")

    update_repo = subprocess.run(["helm", "repo", "update"], capture_output=True, text=True)
    log.write(f"Updated Helm repos: {update_repo.stdout}\n")

    # Registry configuration using values from config
    values: dict = {
        "replicaCount": 1,
        "persistence": reg_config.persistence,
        "service": {"type": reg_config.service_type, "port": 5000},
        "ingress": {
            "enabled": False  # We'll handle ingress separately if needed
        },
        "resources": reg_config.resources,
        "secrets": {"htpasswd": htpasswd_data if htpasswd_data else None},
        "configData": {
            "version": "0.1",
            "log": {"fields": {"service": "registry"}},
            "storage": {
                "cache": {"blobdescriptor": "inmemory"},
                "filesystem": {"rootdirectory": "/var/lib/registry"},
                "delete": {"enabled": True},
            },
            "http": {"addr": ":5000", "headers": {"X-Content-Type-Options": ["nosniff"]}},
            "health": {"storagedriver": {"enabled": True, "interval": "10s", "threshold": 3}},
        },
    }

    # If we have auth, configure it
    if htpasswd_data:
        if "configData" not in values:
            values["configData"] = {}
        values["configData"]["auth"] = {
            "htpasswd": {"realm": "Registry Realm", "path": "/auth/htpasswd"}
        }

    helm.install(
        name="docker-registry",
        chart=reg_config.chart,
        namespace=reg_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for Registry to be ready
    log.write("Waiting for Docker Registry to be ready...\n")
    k8s.wait_for(
        "deployment/docker-registry",
        namespace=reg_config.namespace,
        log=log,
        condition="condition=available",
        timeout=300,
    )

    # Create a NodePort service for external access (for k3d)
    # This allows pushing images from outside the cluster
    nodeport_service = {
        "apiVersion": "v1",
        "kind": "Service",
        "metadata": {"name": "docker-registry-nodeport", "namespace": reg_config.namespace},
        "spec": {
            "type": "NodePort",
            "ports": [
                {
                    "port": 5000,
                    "targetPort": 5000,
                    "nodePort": reg_config.nodeport,
                    "protocol": "TCP",
                }
            ],
            "selector": {"app": "docker-registry"},
        },
    }

    log.write("Creating NodePort service for external access...\n")
    kubectl.apply(manifest=nodeport_service, namespace=reg_config.namespace, log=log)

    # Registry details
    internal_host = f"docker-registry.{reg_config.namespace}.svc.cluster.local:5000"
    external_url = f"http://localhost:{reg_config.nodeport}"  # For k3d access

    log.write("\n✅ Docker Registry deployed successfully\n")
    log.write(f"   Internal Host: {internal_host}\n")
    log.write(f"   External URL (k3d): {external_url}\n")
    log.write(f"   Username: {username}\n")
    log.write(f"   Password: {'(configured)' if htpasswd_data else '(no auth)'}\n")

    # Instructions for usage
    log.write("\nUsage:\n")
    log.write(f"  Internal: docker pull {internal_host}/image:tag\n")
    log.write(f"  External: docker push localhost:{reg_config.nodeport}/image:tag\n")

    return {
        "host": internal_host,
        "url": external_url,
        "username": username if htpasswd_data else "",
        "password": password if htpasswd_data else "",
    }

"""
Deploy cert-manager for certificate management in the cluster.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
from .config import CertManagerConfig  # Import from sibling config module
import time


@task(
    "cert-manager",
    requires=["k3d.ready"],  # Ensure cluster exists first
    provides={"ready": bool, "webhook_ready": bool},
    config_keys=["cert_manager"],  # Load cert-manager configuration
    timeout_sec=600,
    retry_delays=[10, 30],  # Retry after 10s, then 30s - cert-manager can be slow
)
def setup_cert_manager(ctx, cfg, log):
    """Deploy cert-manager for certificate management."""

    # Parse configuration with validation and defaults
    cm_config = CertManagerConfig.model_validate(cfg["cert_manager"])

    log.write(f"Deploying cert-manager to namespace '{cm_config.namespace}'...\n")

    # First, add the Jetstack Helm repository if not already added
    # Note: This is idempotent
    import subprocess

    add_repo = subprocess.run(
        ["helm", "repo", "add", "jetstack", "https://charts.jetstack.io"],
        capture_output=True,
        text=True,
    )
    log.write(f"Added Jetstack repo: {add_repo.stdout}\n")

    update_repo = subprocess.run(["helm", "repo", "update"], capture_output=True, text=True)
    log.write(f"Updated Helm repos: {update_repo.stdout}\n")

    values = {
        "installCRDs": cm_config.install_crds,  # Install CRDs with the chart
        "prometheus": {
            "enabled": False  # Disable Prometheus metrics for now
        },
        "webhook": {
            "timeoutSeconds": cm_config.webhook_timeout,
            "resources": cm_config.resources["webhook"],
        },
        "resources": cm_config.resources["cert_manager"],
        "cainjector": {"resources": cm_config.resources["cainjector"]},
    }

    helm.install(
        name="cert-manager",
        chart=cm_config.chart,
        namespace=cm_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="10m",
    )

    # Wait for cert-manager webhook to be ready
    # This is critical as other resources may depend on the webhook
    log.write("Waiting for cert-manager webhook to be ready...\n")
    k8s.wait_for(
        "deployment/cert-manager-webhook", namespace=cm_config.namespace, log=log, timeout=300
    )

    # Wait for cert-manager itself to be ready
    k8s.wait_for("deployment/cert-manager", namespace=cm_config.namespace, log=log, timeout=300)

    # Wait for cainjector to be ready
    k8s.wait_for(
        "deployment/cert-manager-cainjector", namespace=cm_config.namespace, log=log, timeout=300
    )

    # Give the webhook a moment to fully initialize
    # This helps prevent race conditions with ClusterIssuer creation
    time.sleep(5)

    # Create a self-signed ClusterIssuer for basic certificate needs
    cluster_issuer = {
        "apiVersion": "cert-manager.io/v1",
        "kind": "ClusterIssuer",
        "metadata": {"name": cm_config.cluster_issuer_name},
        "spec": {"selfSigned": {}},
    }

    log.write(f"Creating self-signed ClusterIssuer '{cm_config.cluster_issuer_name}'...\n")
    kubectl.apply(manifest=cluster_issuer, namespace=cm_config.namespace, log=log)

    log.write("\n✅ cert-manager deployed successfully\n")
    log.write("   Webhook is ready for certificate requests\n")
    log.write(f"   Self-signed ClusterIssuer '{cm_config.cluster_issuer_name}' created\n")

    return {"ready": True, "webhook_ready": True}

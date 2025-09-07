"""
Deploy trust-manager for certificate authority distribution.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
from .config import TrustManagerConfig


@task(
    "trust-manager",
    requires=["cert-manager.ready"],  # cert-manager must be ready first
    provides={"ready": bool, "ca_bundle_ready": bool},
    config_keys=["trust_manager"],  # Load trust-manager configuration
    timeout_sec=600,
    retry_delays=[10, 30],  # trust-manager can take time to initialize
)
def setup_trust_manager(ctx, cfg, log):
    """Deploy trust-manager for CA certificate distribution."""

    # Parse configuration with validation and defaults
    tm_config = TrustManagerConfig.model_validate(cfg["trust_manager"])

    log.write(f"Deploying trust-manager to namespace '{tm_config.namespace}'...\n")

    # Add the Jetstack Helm repository (should already be added by cert-manager)
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
        "app": tm_config.app,
    }

    helm.install(
        name="trust-manager",
        chart=tm_config.chart,
        namespace=tm_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for trust-manager deployment
    k8s.wait_for("deployment/trust-manager", namespace=tm_config.namespace, log=log, timeout=300)

    # Wait for trust-manager webhook
    k8s.wait_for(
        "deployment/trust-manager-webhook", namespace=tm_config.namespace, log=log, timeout=300
    )

    # Create a Bundle for mkcert CA certificates (common in development)
    ca_bundle = {
        "apiVersion": "trust.cert-manager.io/v1alpha1",
        "kind": "Bundle",
        "metadata": {"name": tm_config.ca_bundle["name"], "namespace": tm_config.namespace},
        "spec": {
            "sources": [
                {
                    "secret": {
                        "name": tm_config.ca_bundle["secret_name"],
                        "key": tm_config.ca_bundle["secret_key"],
                    }
                }
            ],
            "target": {
                "configMap": {"key": tm_config.ca_bundle["config_map_key"]},
                "namespaceSelector": {},
            },
        },
    }

    log.write(f"Creating {tm_config.ca_bundle['name']} CA Bundle for certificate distribution...\n")
    kubectl.apply(manifest=ca_bundle, namespace=tm_config.namespace, log=log)

    log.write("\n✅ trust-manager deployed successfully\n")
    log.write(
        f"   CA certificate bundle '{tm_config.ca_bundle['name']}' created for automatic distribution\n"
    )

    return {"ready": True, "ca_bundle_ready": True}

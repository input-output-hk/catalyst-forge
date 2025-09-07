"""
Deploy Earthly remote buildkit instance for distributed builds.
"""

from cli.setup import task
from cli.setup.tools import k8s
from pathlib import Path
import subprocess
from .config import EarthlyConfig


@task(
    "earthly",
    requires=["registry.ready", "envoy-gateway.gateway_ready"],  # Need registry and envoy gateway
    provides={"buildkit_host": str, "ready": bool},
    config_keys=["earthly"],  # Load earthly configuration from config.cue
    timeout_sec=600,
    retry_delays=[10, 30],  # Earthly buildkit can take time to initialize
)
def setup_earthly(ctx, cfg, log):
    """Deploy Earthly remote buildkit instance."""

    # Parse configuration with validation and defaults
    earthly_config = EarthlyConfig.model_validate(cfg.get("earthly", {}))

    log.write("Deploying Earthly remote buildkit instance...\n")

    # Get repo root
    current_file = Path(__file__).resolve()
    playground_dir = current_file.parent.parent.parent  # .../playground
    repo_root = playground_dir.parent.parent  # .../catalyst-forge

    manifests_dir = repo_root / earthly_config.manifests_dir

    if not manifests_dir.exists():
        raise RuntimeError(f"Earthly manifests directory not found: {manifests_dir}")

    # Apply PVC first
    log.write("Creating buildkit PVC...\n")
    pvc_manifest = manifests_dir / "buildkitd-pvc.yaml"
    if not pvc_manifest.exists():
        raise RuntimeError(f"Buildkit PVC manifest not found: {pvc_manifest}")

    subprocess.run(
        ["kubectl", "apply", "-f", str(pvc_manifest), "-n", earthly_config.namespace],
        check=True,
        capture_output=True,
        text=True,
    )

    # Apply deployment
    log.write("Deploying buildkit deployment...\n")
    deployment_manifest = manifests_dir / "buildkitd-deployment.yaml"
    if not deployment_manifest.exists():
        raise RuntimeError(f"Buildkit deployment manifest not found: {deployment_manifest}")

    subprocess.run(
        ["kubectl", "apply", "-f", str(deployment_manifest), "-n", earthly_config.namespace],
        check=True,
        capture_output=True,
        text=True,
    )

    # Apply service
    log.write("Creating buildkit service...\n")
    service_manifest = manifests_dir / "buildkitd-service.yaml"
    if not service_manifest.exists():
        raise RuntimeError(f"Buildkit service manifest not found: {service_manifest}")

    subprocess.run(
        ["kubectl", "apply", "-f", str(service_manifest), "-n", earthly_config.namespace],
        check=True,
        capture_output=True,
        text=True,
    )

    # Wait for buildkit deployment to be ready
    log.write("Waiting for buildkit deployment to be ready...\n")
    k8s.wait_for(
        "deployment/earthly-buildkitd",
        namespace=earthly_config.namespace,
        log=log,
        timeout=300,
    )

    # Apply TCPRoute for external access via Envoy Gateway
    log.write("Configuring TCPRoute for external buildkit access...\n")
    tcproute_manifest = repo_root / "platform" / "envoy" / "buildkitd-tcproute.yaml"
    if not tcproute_manifest.exists():
        raise RuntimeError(f"Buildkit TCPRoute manifest not found: {tcproute_manifest}")

    subprocess.run(
        ["kubectl", "apply", "-f", str(tcproute_manifest), "-n", "envoy-gateway-system"],
        check=True,
        capture_output=True,
        text=True,
    )

    log.write("\n✅ Earthly buildkit instance deployed successfully\n")
    log.write(f"   Buildkit Host: {earthly_config.buildkit_host}\n")
    log.write("   Ready to accept remote Earthly builds\n")

    return {
        "buildkit_host": earthly_config.buildkit_host,
        "ready": True,
    }

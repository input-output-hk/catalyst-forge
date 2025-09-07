"""
Deploy Envoy Gateway for API routing and ingress management.
"""

from cli.setup import task
from cli.setup.tools import helm, k8s, kubectl
from .config import EnvoyGatewayConfig


@task(
    "envoy-gateway",
    requires=["cert-manager.ready"],  # cert-manager for TLS certificates
    provides={"gateway_ready": bool, "gateway_class": str, "gateway_namespace": str},
    config_keys=["envoy_gateway"],  # Load Envoy Gateway configuration
    timeout_sec=900,
    retry_delays=[30, 60],  # Envoy Gateway can take time to initialize
)
def setup_envoy_gateway(ctx, cfg, log):
    """Deploy Envoy Gateway for API routing and ingress."""

    # Parse configuration with validation and defaults
    eg_config = EnvoyGatewayConfig.model_validate(cfg["envoy_gateway"])

    log.write(f"Deploying Envoy Gateway to namespace '{eg_config.namespace}'...\n")

    # Add the Envoy Gateway Helm repository
    import subprocess

    update_repo = subprocess.run(["helm", "repo", "update"], capture_output=True, text=True)
    log.write(f"Updated Helm repos: {update_repo.stdout}\n")

    values = {
        "deployment": {
            "envoyGateway": {
                "imagePullPolicy": eg_config.image_pull_policy,
                "resources": eg_config.resources["envoy_gateway"],
            },
            "envoyProxy": {
                "imagePullPolicy": eg_config.image_pull_policy,
                "resources": eg_config.resources["envoy_proxy"],
            },
        },
        "config": {
            "envoyGateway": {
                "gateway": {"controllerName": "gateway.envoyproxy.io/gatewayclass-controller"},
                "logging": {"level": {"default": eg_config.logging_level}},
            }
        },
        "service": {
            "type": eg_config.service_type,
            "ports": [
                {"name": "http", "port": 80, "targetPort": 8080},
                {"name": "https", "port": 443, "targetPort": 8443},
            ],
        },
    }

    helm.install(
        name="envoy-gateway",
        chart="oci://docker.io/envoyproxy/gateway-helm",
        namespace=eg_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="10m",
    )

    # Wait for Envoy Gateway controller and proxy (release name matches Helmfile)
    k8s.wait_for("deployment/envoy-gateway", namespace=eg_config.namespace, log=log, timeout=300)
    k8s.wait_for("deployment/envoy-proxy", namespace=eg_config.namespace, log=log, timeout=300)

    # Create a GatewayClass for the gateway (no parametersRef, match Helmfile)
    gateway_class = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "GatewayClass",
        "metadata": {"name": eg_config.gateway_class_name},
        "spec": {
            "controllerName": "gateway.envoyproxy.io/gatewayclass-controller",
        },
    }

    log.write(f"Creating GatewayClass '{eg_config.gateway_class_name}'...\n")
    kubectl.apply(manifest=gateway_class, namespace=eg_config.namespace, log=log)

    # Apply the default Gateway matching platform manifest
    gateway_manifest = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "Gateway",
        "metadata": {"name": "default", "namespace": eg_config.namespace},
        "spec": {
            "gatewayClassName": eg_config.gateway_class_name,
            "listeners": [
                {
                    "name": "http",
                    "protocol": "HTTP",
                    "port": 80,
                    "allowedRoutes": {"namespaces": {"from": "All"}},
                },
                {
                    "name": "https",
                    "protocol": "HTTPS",
                    "port": 443,
                    "allowedRoutes": {"namespaces": {"from": "All"}},
                    "tls": {
                        "mode": "Terminate",
                        "certificateRefs": [
                            {"kind": "Secret", "name": "wildcard-projectcatalyst-tls"}
                        ],
                    },
                    "hostname": "*.projectcatalyst.dev",
                },
                {
                    "name": "buildkit-tcp",
                    "protocol": "TCP",
                    "port": 8372,
                    "allowedRoutes": {"namespaces": {"from": "All"}},
                },
                {
                    "name": "postgres-tcp",
                    "protocol": "TCP",
                    "port": 5432,
                    "allowedRoutes": {"namespaces": {"from": "All"}},
                },
            ],
        },
    }

    log.write("Applying default Gateway resource...\n")
    kubectl.apply(manifest=gateway_manifest, namespace=eg_config.namespace, log=log)

    log.write(f"\n✅ Envoy Gateway deployed successfully to '{eg_config.namespace}'\n")
    log.write(f"   GatewayClass: '{eg_config.gateway_class_name}'\n")
    log.write("   Default Gateway applied with HTTP/HTTPS/TCP listeners\n")

    return {
        "gateway_ready": True,
        "gateway_class": eg_config.gateway_class_name,
        "gateway_namespace": eg_config.namespace,
    }

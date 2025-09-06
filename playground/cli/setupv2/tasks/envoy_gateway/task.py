"""
Deploy Envoy Gateway for API routing and ingress management.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
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

    add_repo = subprocess.run(
        ["helm", "repo", "add", "eg", "https://gateway.envoyproxy.io"],
        capture_output=True,
        text=True,
    )
    log.write(f"Added Envoy Gateway repo: {add_repo.stdout}\n")

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
        name="eg",
        chart="eg/gateway-helm",
        namespace=eg_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="10m",
    )

    # Wait for Envoy Gateway controller
    k8s.wait_for(
        "deployment/gateway-helm-envoy-gateway", namespace=eg_config.namespace, log=log, timeout=300
    )

    # Wait for Envoy Proxy deployment
    k8s.wait_for(
        "deployment/gateway-helm-envoy-proxy", namespace=eg_config.namespace, log=log, timeout=300
    )

    # Create a GatewayClass for the gateway
    gateway_class = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "GatewayClass",
        "metadata": {"name": eg_config.gateway_class_name},
        "spec": {
            "controllerName": "gateway.envoyproxy.io/gatewayclass-controller",
            "parametersRef": {
                "group": "gateway.envoyproxy.io",
                "kind": "EnvoyProxy",
                "name": "default-envoy-proxy",
                "namespace": eg_config.namespace,
            },
        },
    }

    log.write(f"Creating GatewayClass '{eg_config.gateway_class_name}'...\n")
    kubectl.apply(manifest=gateway_class, namespace=eg_config.namespace, log=log)

    # Create an EnvoyProxy configuration
    envoy_proxy = {
        "apiVersion": "gateway.envoyproxy.io/v1alpha1",
        "kind": "EnvoyProxy",
        "metadata": {"name": "default-envoy-proxy", "namespace": eg_config.namespace},
        "spec": {
            "telemetry": {
                "accessLog": {
                    "settings": [
                        {
                            "format": {
                                "type": "Text",
                                "text": '[%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"\n',
                            },
                            "sinks": [{"type": "File", "file": {"path": "/dev/stdout"}}],
                        }
                    ]
                }
            },
            "provider": {
                "type": "Kubernetes",
                "kubernetes": {
                    "watchMode": "Endpoints",
                    "watchNamespace": "",  # Watch all namespaces
                },
            },
        },
    }

    log.write("Creating EnvoyProxy configuration...\n")
    kubectl.apply(manifest=envoy_proxy, namespace=eg_config.namespace, log=log)

    log.write(f"\n✅ Envoy Gateway deployed successfully to '{eg_config.namespace}'\n")
    log.write(f"   GatewayClass: '{eg_config.gateway_class_name}'\n")
    log.write("   Ready to accept Gateway and HTTPRoute resources\n")

    return {
        "gateway_ready": True,
        "gateway_class": eg_config.gateway_class_name,
        "gateway_namespace": eg_config.namespace,
    }

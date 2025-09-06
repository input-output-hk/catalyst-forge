"""
Deploy Mailpit for email testing and development.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
from .config import MailpitConfig


@task(
    "mailpit",
    requires=["envoy-gateway.gateway_ready"],  # Need gateway for routing
    provides={"smtp_host": str, "smtp_port": int, "web_url": str},
    config_keys=["mailpit"],  # Load Mailpit configuration
    timeout_sec=600,
    retry_delays=[10, 30],  # Standard retry delays
)
def setup_mailpit(ctx, cfg, log):
    """Deploy Mailpit email testing service."""

    # Parse configuration with validation and defaults
    mp_config = MailpitConfig.model_validate(cfg["mailpit"])

    log.write(f"Deploying Mailpit to namespace '{mp_config.namespace}'...\n")

    values = {
        "image": {
            "repository": mp_config.image["repository"],
            "tag": mp_config.image["tag"],
            "pullPolicy": mp_config.image["pull_policy"],
        },
        "service": {
            "type": mp_config.service_type,
            "ports": {
                "http": {"port": 8025, "targetPort": 8025},
                "smtp": {"port": 1025, "targetPort": 1025},
            },
        },
        "ingress": {
            "enabled": False  # We'll use Envoy Gateway instead
        },
        "resources": mp_config.resources,
        "persistence": {
            "enabled": False  # No persistence needed for testing
        },
        "mailpit": {
            "uiBindAddr": "0.0.0.0:8025",
            "smtpBindAddr": "0.0.0.0:1025",
            "maxMessages": mp_config.max_messages,
            "disableWebUIAuth": mp_config.disable_web_ui_auth,
        },
    }

    helm.install(
        name="mailpit",
        chart="mailpit",  # Using hardcoded chart name as it's not in our config
        namespace=mp_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for Mailpit deployment
    k8s.wait_for("deployment/mailpit", namespace=mp_config.namespace, log=log, timeout=300)

    # Create an HTTPRoute for the Mailpit web UI
    http_route = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "HTTPRoute",
        "metadata": {"name": "mailpit-web", "namespace": mp_config.namespace},
        "spec": {
            "parentRefs": [
                {
                    "name": "default-gateway",  # Reference the default gateway
                    "namespace": "envoy-gateway-system",
                }
            ],
            "hostnames": [mp_config.hostname],
            "rules": [
                {
                    "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
                    "backendRefs": [
                        {
                            "kind": "Service",
                            "name": "mailpit",
                            "namespace": mp_config.namespace,
                            "port": 8025,
                        }
                    ],
                }
            ],
        },
    }

    log.write("Creating HTTPRoute for Mailpit web UI...\n")
    kubectl.apply(manifest=http_route, namespace=mp_config.namespace, log=log)

    # Create a default Gateway if it doesn't exist
    gateway = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "Gateway",
        "metadata": {"name": "default-gateway", "namespace": "envoy-gateway-system"},
        "spec": {
            "gatewayClassName": "envoy-gateway",
            "listeners": [
                {
                    "name": "http",
                    "hostname": "*.projectcatalyst.dev",
                    "port": 80,
                    "protocol": "HTTP",
                }
            ],
        },
    }

    log.write("Ensuring default Gateway exists...\n")
    kubectl.apply(manifest=gateway, namespace="envoy-gateway-system", log=log)

    log.write("\n✅ Mailpit deployed successfully\n")
    log.write(f"   SMTP server available at mailpit.{mp_config.namespace}:1025\n")
    log.write(f"   Web UI available at https://{mp_config.hostname}\n")

    return {
        "smtp_host": f"mailpit.{mp_config.namespace}",
        "smtp_port": 1025,
        "web_url": f"https://{mp_config.hostname}",
    }

"""
Deploy Mailpit for email testing and development.
"""

from cli.setup import task
from cli.setup.tools import helm, k8s, kubectl
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
            "http": {"name": "http"},
            "smtp": {"name": "smtp"},
        },
        "ingress": {"enabled": False},
        "resources": mp_config.resources,
        "persistence": {"enabled": False},
        "mailpit": {
            "uiBindAddr": "0.0.0.0:8025",
            "smtpBindAddr": "0.0.0.0:1025",
            "maxMessages": mp_config.max_messages,
            "disableWebUIAuth": mp_config.disable_web_ui_auth,
        },
    }

    helm.install(
        name="mailpit",
        chart="jouve/mailpit",
        namespace=mp_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for Mailpit deployment
    k8s.wait_for("deployment/mailpit", namespace=mp_config.namespace, log=log, timeout=300)

    # Create HTTPRoute for Mailpit web UI without reading from filesystem
    http_route = {
        "apiVersion": "gateway.networking.k8s.io/v1",
        "kind": "HTTPRoute",
        "metadata": {"name": "mailpit", "namespace": mp_config.namespace},
        "spec": {
            "parentRefs": [{"name": "default", "namespace": "envoy-gateway-system"}],
            "hostnames": [mp_config.hostname],
            "rules": [
                {
                    "matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
                    "backendRefs": [{"name": "http", "port": 80}],
                }
            ],
        },
    }

    log.write("Creating HTTPRoute for Mailpit web UI...\n")
    kubectl.apply(manifest=http_route, namespace=mp_config.namespace, log=log)

    log.write("\n✅ Mailpit deployed successfully\n")
    log.write(f"   SMTP server available at mailpit.{mp_config.namespace}:1025\n")
    log.write(f"   Web UI available at https://{mp_config.hostname}\n")

    return {
        "smtp_host": f"mailpit.{mp_config.namespace}",
        "smtp_port": 1025,
        "web_url": f"https://{mp_config.hostname}",
    }

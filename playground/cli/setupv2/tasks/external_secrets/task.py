"""
Deploy External Secrets Operator for secret management.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, kubectl, k8s
from .config import ExternalSecretsConfig
import time
import subprocess


@task(
    "external-secrets",
    requires=["k3d.ready", "localstack.endpoint"],  # Needs cluster and AWS endpoint
    provides={"ready": bool, "cluster_secret_store_ready": bool},
    config_keys=["external_secrets"],  # Load External Secrets configuration
    timeout_sec=300,
    retry_delays=[10, 20],  # Retry after 10s, then 20s
)
def setup_external_secrets(ctx, cfg, log):
    """Deploy External Secrets Operator and configure ClusterSecretStore."""

    # Parse configuration with validation and defaults
    es_config = ExternalSecretsConfig.model_validate(cfg["external_secrets"])

    log.write(f"Deploying External Secrets Operator to namespace '{es_config.namespace}'...\n")

    # Add External Secrets Helm repo
    add_repo = subprocess.run(
        ["helm", "repo", "add", "external-secrets", "https://charts.external-secrets.io"],
        capture_output=True,
        text=True,
    )
    log.write(f"Added External Secrets repo: {add_repo.stdout}\n")

    update_repo = subprocess.run(["helm", "repo", "update"], capture_output=True, text=True)
    log.write(f"Updated Helm repos: {update_repo.stdout}\n")

    # Deploy External Secrets Operator
    values = {
        "installCRDs": es_config.install_crds,
        "webhook": {"port": es_config.webhook_port},
        "certController": {"requeueInterval": es_config.cert_controller_requeue_interval},
    }

    helm.install(
        name="external-secrets",
        chart=es_config.chart,
        namespace=es_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for operator to be ready
    log.write("Waiting for External Secrets Operator components to be ready...\n")
    k8s.wait_for(
        "deployment/external-secrets-webhook", namespace=es_config.namespace, log=log, timeout=180
    )
    k8s.wait_for("deployment/external-secrets", namespace=es_config.namespace, log=log, timeout=180)
    k8s.wait_for(
        "deployment/external-secrets-cert-controller",
        namespace=es_config.namespace,
        log=log,
        timeout=180,
    )

    # Create ClusterSecretStore for LocalStack AWS Secrets Manager
    log.write(f"Creating ClusterSecretStore '{es_config.secret_store_name}' for LocalStack...\n")

    # Get LocalStack endpoint from context
    localstack_endpoint = ctx["localstack.endpoint"]
    aws_region = ctx["localstack.region"]
    aws_access_key = ctx["localstack.aws_access_key"]
    aws_secret_key = ctx["localstack.aws_secret_key"]

    # Create secret with AWS credentials for External Secrets to use
    k8s.create_secret(
        name="localstack-credentials",
        namespace=es_config.namespace,
        data={
            "access-key": aws_access_key,
            "secret-key": aws_secret_key,
        },
        log=log,
    )

    # Create ClusterSecretStore
    cluster_secret_store = {
        "apiVersion": "external-secrets.io/v1beta1",
        "kind": "ClusterSecretStore",
        "metadata": {"name": es_config.secret_store_name},
        "spec": {
            "provider": {
                "aws": {
                    "service": "SecretsManager",
                    "region": aws_region,
                    "auth": {
                        "secretRef": {
                            "accessKeyIDSecretRef": {
                                "name": "localstack-credentials",
                                "namespace": es_config.namespace,
                                "key": "access-key",
                            },
                            "secretAccessKeySecretRef": {
                                "name": "localstack-credentials",
                                "namespace": es_config.namespace,
                                "key": "secret-key",
                            },
                        }
                    },
                    "endpoint": {"url": localstack_endpoint},
                }
            }
        },
    }

    kubectl.apply(manifest=cluster_secret_store, namespace=es_config.namespace, log=log)

    # Give it a moment to initialize
    time.sleep(5)

    # Check ClusterSecretStore status
    check_store = subprocess.run(
        [
            "kubectl",
            "get",
            "clustersecretstore",
            es_config.secret_store_name,
            "-o",
            "jsonpath={.status.conditions[0].type}",
        ],
        capture_output=True,
        text=True,
    )
    log.write(f"ClusterSecretStore status: {check_store.stdout}\n")

    log.write("\n✅ External Secrets Operator successfully deployed!\n")
    log.write(f"   Namespace: {es_config.namespace}\n")
    log.write(f"   ClusterSecretStore: {es_config.secret_store_name}\n")

    return {"ready": True, "cluster_secret_store_ready": True}

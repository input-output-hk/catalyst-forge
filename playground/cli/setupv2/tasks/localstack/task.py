"""
Deploy LocalStack for AWS service simulation.
"""

from cli.setupv2 import task
from cli.setupv2.tools import helm, k8s, kubectl
from .config import LocalStackConfig
import time


@task(
    "localstack",
    requires=["k3d.ready"],  # Ensure cluster exists first
    provides={"endpoint": str, "region": str, "aws_access_key": str, "aws_secret_key": str},
    config_keys=["localstack"],  # Load LocalStack configuration
    timeout_sec=300,
    retry_delays=[5, 15],  # Retry after 5s, then 15s
)
def setup_localstack(ctx, cfg, log):
    """Deploy LocalStack for AWS service simulation."""

    # Parse configuration with validation and defaults
    ls_config = LocalStackConfig.model_validate(cfg["localstack"])

    log.write(f"Deploying LocalStack to namespace '{ls_config.namespace}'...\n")

    # Add LocalStack Helm repo
    import subprocess

    add_repo = subprocess.run(
        ["helm", "repo", "add", "localstack", "https://localstack.github.io/helm-charts"],
        capture_output=True,
        text=True,
    )
    log.write(f"Added LocalStack repo: {add_repo.stdout}\n")

    update_repo = subprocess.run(["helm", "repo", "update"], capture_output=True, text=True)
    log.write(f"Updated Helm repos: {update_repo.stdout}\n")

    # LocalStack configuration using values from config
    values = {
        "startServices": ls_config.services,
        "service": {"type": ls_config.service_type, "edgeService": {"targetPort": 4566}},
        "persistence": ls_config.persistence,
        "resources": ls_config.resources,
        "debug": ls_config.debug,
        "extraEnvVars": [
            {
                "name": "LOCALSTACK_HOST",
                "value": f"localstack.{ls_config.namespace}.svc.cluster.local",
            },
            {"name": "DEFAULT_REGION", "value": ls_config.region},
            {"name": "AWS_ACCESS_KEY_ID", "value": ls_config.aws_access_key},
            {"name": "AWS_SECRET_ACCESS_KEY", "value": ls_config.aws_secret_key},
            {"name": "PERSISTENCE", "value": "1"},
        ],
    }

    helm.install(
        name="localstack",
        chart=ls_config.chart,
        namespace=ls_config.namespace,
        values=values,
        log=log,
        wait=True,
        timeout="5m",
    )

    # Wait for LocalStack to be ready
    log.write("Waiting for LocalStack to be ready...\n")
    k8s.wait_for(
        "deployment/localstack",
        namespace=ls_config.namespace,
        log=log,
        condition="condition=available",
        timeout=300,
    )

    # Give LocalStack a moment to fully initialize its services
    log.write("Waiting for LocalStack services to initialize...\n")
    time.sleep(10)

    # Check if LocalStack is responding
    # We'll create a job to test connectivity
    test_job = {
        "apiVersion": "batch/v1",
        "kind": "Job",
        "metadata": {"name": "localstack-test", "namespace": ls_config.namespace},
        "spec": {
            "backoffLimit": 3,
            "template": {
                "spec": {
                    "restartPolicy": "Never",
                    "containers": [
                        {
                            "name": "test",
                            "image": "curlimages/curl:latest",
                            "command": [
                                "sh",
                                "-c",
                                "curl -f http://localstack:4566/_localstack/health || exit 1",
                            ],
                        }
                    ],
                }
            },
        },
    }

    log.write("Testing LocalStack connectivity...\n")
    kubectl.apply(manifest=test_job, namespace=ls_config.namespace, log=log)

    # Wait for the test job to complete
    k8s.wait_for(
        "job/localstack-test",
        namespace=ls_config.namespace,
        log=log,
        condition="condition=complete",
        timeout=60,
    )

    # Clean up test job
    cleanup = subprocess.run(
        ["kubectl", "delete", "job", "localstack-test", "-n", ls_config.namespace],
        capture_output=True,
        text=True,
    )
    log.write(f"Cleaned up test job: {cleanup.stdout}\n")

    # LocalStack endpoint details
    endpoint = f"http://localstack.{ls_config.namespace}.svc.cluster.local:4566"

    log.write("\n✅ LocalStack deployed successfully\n")
    log.write(f"   Endpoint: {endpoint}\n")
    log.write(f"   Region: {ls_config.region}\n")
    log.write(f"   AWS Access Key: {ls_config.aws_access_key}\n")
    log.write(f"   Services: {ls_config.services}\n")

    return {
        "endpoint": endpoint,
        "region": ls_config.region,
        "aws_access_key": ls_config.aws_access_key,
        "aws_secret_key": ls_config.aws_secret_key,
    }

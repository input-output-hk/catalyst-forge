"""
Deploy Keycloak Operator for managing Keycloak instances.
"""

from cli.setupv2 import task
from cli.setupv2.tools import k8s
from .config import KeycloakOperatorConfig
import subprocess
import time


@task(
    "keycloak-operator",
    requires=["k3d.ready", "cert-manager.ready"],  # Needs cluster and cert-manager
    provides={"operator_ready": bool, "operator_namespace": str},
    config_keys=["keycloak_operator"],  # Load Keycloak operator configuration
    timeout_sec=300,
    retry_delays=[10, 20],  # Retry after 10s, then 20s
)
def setup_keycloak_operator(ctx, cfg, log):
    """Deploy Keycloak Operator using manifest installation."""

    # Parse configuration with validation and defaults
    kc_config = KeycloakOperatorConfig.model_validate(cfg.get("keycloak_operator", {}))

    log.write("Deploying Keycloak Operator...\n")

    # Create namespace for Keycloak operator
    namespace = kc_config.namespace
    create_ns = subprocess.run(
        ["kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "yaml"],
        capture_output=True,
        text=True,
    )

    apply_ns = subprocess.run(
        ["kubectl", "apply", "-f", "-"], input=create_ns.stdout, capture_output=True, text=True
    )
    log.write(f"Created namespace: {apply_ns.stdout}\n")

    # Deploy Keycloak Operator CRDs
    log.write("Installing Keycloak CRDs...\n")

    # Using the configured version of Keycloak operator
    keycloak_version = kc_config.version
    operator_manifest_url = f"https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/{keycloak_version}/kubernetes/keycloaks.k8s.keycloak.org-v1.yml"
    realm_import_crd_url = f"https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/{keycloak_version}/kubernetes/keycloakrealmimports.k8s.keycloak.org-v1.yml"
    operator_deployment_url = f"https://raw.githubusercontent.com/keycloak/keycloak-k8s-resources/{keycloak_version}/kubernetes/kubernetes.yml"

    # Apply Keycloak CRD
    apply_keycloak_crd = subprocess.run(
        ["kubectl", "apply", "-f", operator_manifest_url], capture_output=True, text=True
    )
    log.write(f"Applied Keycloak CRD: {apply_keycloak_crd.stdout}\n")
    if apply_keycloak_crd.returncode != 0:
        log.write(f"Error applying Keycloak CRD: {apply_keycloak_crd.stderr}\n")

    # Apply RealmImport CRD
    apply_realm_crd = subprocess.run(
        ["kubectl", "apply", "-f", realm_import_crd_url], capture_output=True, text=True
    )
    log.write(f"Applied RealmImport CRD: {apply_realm_crd.stdout}\n")
    if apply_realm_crd.returncode != 0:
        log.write(f"Error applying RealmImport CRD: {apply_realm_crd.stderr}\n")

    # Deploy the operator itself
    log.write("Deploying Keycloak Operator...\n")
    apply_operator = subprocess.run(
        ["kubectl", "apply", "-n", namespace, "-f", operator_deployment_url],
        capture_output=True,
        text=True,
    )
    log.write(f"Applied Keycloak Operator: {apply_operator.stdout}\n")
    if apply_operator.returncode != 0:
        log.write(f"Error applying operator: {apply_operator.stderr}\n")
        raise RuntimeError(f"Failed to deploy Keycloak operator: {apply_operator.stderr}")

    # Wait for operator deployment to be ready
    log.write("Waiting for Keycloak Operator to be ready...\n")

    # Give it a moment for the deployment to be created
    time.sleep(5)

    # Check if deployment exists
    check_deployment = subprocess.run(
        [
            "kubectl",
            "get",
            "deployment",
            "-n",
            namespace,
            "-l",
            "app.kubernetes.io/name=keycloak-operator",
        ],
        capture_output=True,
        text=True,
    )
    log.write(f"Operator deployments: {check_deployment.stdout}\n")

    # Wait for the operator deployment
    k8s.wait_for(
        "deployment/keycloak-operator",
        namespace=namespace,
        log=log,
        timeout=kc_config.operator_deployment_timeout,
    )

    # Verify CRDs are installed
    check_crds = subprocess.run(["kubectl", "get", "crd"], capture_output=True, text=True)
    log.write(f"Available CRDs:\n{check_crds.stdout}\n")

    # Verify operator is running
    check_pods = subprocess.run(
        ["kubectl", "get", "pods", "-n", namespace], capture_output=True, text=True
    )
    log.write(f"Operator pods:\n{check_pods.stdout}\n")

    log.write("Keycloak Operator successfully deployed!\n")

    return {"operator_ready": True, "operator_namespace": namespace}

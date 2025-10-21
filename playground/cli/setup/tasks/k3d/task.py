"""
Create or manage k3d cluster for the playground environment.

This task must run before all other infrastructure tasks as it provides
the Kubernetes cluster itself.
"""

from cli.setup.task_base import BaseTask
from pathlib import Path
import subprocess
import os
from typing import Dict, List

# Import k3d operations from the local ops module
from .ops import (
    cluster_exists,
    create_cluster,
    delete_cluster,
    write_kubeconfig,
    emit_cluster_json,
    write_registries_yaml,
)
from cli.setup.tools.utils import get_mkcert_caroot
from cli.utils import require_cmd, get_repo_root
from .config import K3dConfig


class K3dTask(BaseTask):
    """K3d cluster setup task."""

    @classmethod
    def provides(cls) -> Dict[str, type]:
        """Define what this task provides."""
        return {
            "cluster_name": str,
            "kubeconfig": str,
            "http_port": int,
            "https_port": int,
            "api_port": int,
            "ready": bool,
        }

    @classmethod
    def requires(cls) -> List[str]:
        """K3d has no dependencies on other tasks."""
        return []

    @classmethod
    def config_keys(cls) -> List[str]:
        """Config keys needed by this task."""
        return ["k3d", "registry_host"]

    def setup(self):
        """Register all subtasks and health checks."""
        # Parse config once
        self.k3d_config = K3dConfig.model_validate(self.cfg["k3d"])
        self.registry_host = self.cfg["registry_host"]

        # Setup paths
        playground_dir = Path(__file__).resolve().parent.parent.parent.parent
        self.repo_root = get_repo_root(playground_dir)

        # Register subtasks in execution order
        self.add_subtask("preflight", "Checking prerequisites", 5, self.check_prerequisites)
        self.add_subtask("certificates", "Preparing certificates", 10, self.prepare_certificates)
        self.add_subtask("cluster_plan", "Planning cluster actions", 5, self.plan_cluster)
        self.add_subtask("cluster_create", "Creating k3d cluster", 45, self.create_cluster)
        self.add_subtask("cluster_post", "Post-create configuration", 10, self.post_create)
        self.add_subtask("kubeconfig", "Writing kubeconfig", 5, self.write_kubeconfig)
        self.add_subtask("output", "Writing cluster info", 5, self.write_output)

        # Register health checks (run automatically after subtasks)
        self.add_healthcheck(
            "cluster_exists", "Verifying cluster exists", self.verify_cluster_exists
        )
        self.add_healthcheck(
            "api_responsive", "Checking API connectivity", self.test_api_connection, retries=3
        )
        self.add_healthcheck(
            "nodes_ready",
            "Waiting for nodes",
            self.wait_for_nodes,
            retries=5,
            delays=[5, 10, 15, 20, 30],
        )
        self.add_healthcheck("system_pods", "Verifying system pods", self.verify_system_pods)

    def health_weight_ratio(self) -> float:
        """Make health checks represent a larger portion for k3d.

        Cluster readiness tends to be dominated by readiness waits, so reflect
        that by giving health checks equal weight to subtasks (1.0 * subtask).
        This yields total = subtasks + health = 2x subtasks.
        """
        return 1.0

    def check_prerequisites(self):
        """Subtask: Check required commands are available."""
        for cmd in ("docker", "k3d", "mkcert"):
            try:
                require_cmd(cmd)
                self.log.write(f"✓ {cmd} is available\n")
            except Exception as e:
                self.log.write(f"✗ {cmd} is missing: {e}\n")
                raise RuntimeError(f"Required command '{cmd}' not found")

    def prepare_certificates(self):
        """Subtask: Prepare mkcert CA certificates."""
        self.log.write("Preparing mkcert CA certificates...\n")

        caroot = get_mkcert_caroot()
        self.ca_file = caroot / "rootCA.pem"

        if not self.ca_file.exists():
            self.log.write(f"Warning: mkcert CA file not found at {self.ca_file}\n")
            self.log.write("Running 'mkcert -install' to create CA...\n")
            result = subprocess.run(["mkcert", "-install"], capture_output=True, text=True)
            self.log.write(f"{result.stdout}\n")
            if result.returncode != 0:
                self.log.write(f"Error: {result.stderr}\n")
                raise RuntimeError("Failed to install mkcert CA")

        # Prepare registries.yaml
        tmpdir = (self.repo_root.parent / "playground" / ".certs").resolve()
        tmpdir.mkdir(parents=True, exist_ok=True)

        self.log.write(f"Writing registries configuration for {self.registry_host}...\n")
        self.registries_yaml = write_registries_yaml(
            tmpdir=tmpdir,
            registry_host=self.registry_host,
            ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt"),
        ).resolve()

    def plan_cluster(self):
        """Subtask: Plan cluster action (recreate or reuse)."""
        if cluster_exists(self.k3d_config.cluster_name):
            self.log.write(f"Cluster '{self.k3d_config.cluster_name}' already exists.\n")
            if self.k3d_config.force_recreate:
                self.log.write("Force recreate enabled, will delete and re-create.\n")
                self._plan_action = "recreate"
            else:
                self.log.write("Reusing existing cluster.\n")
                self._plan_action = "reuse"
        else:
            self.log.write("Cluster does not exist; will create new cluster.\n")
            self._plan_action = "create"

    def create_cluster(self):
        """Subtask: Create the k3d cluster."""
        volume_mounts = [
            f"{self.ca_file}:/etc/ssl/certs/mkcert-rootCA.crt@server:*;agent:*",
            f"{self.registries_yaml}:/etc/rancher/k3s/registries.yaml@server:*;agent:*",
        ]
        action = getattr(self, "_plan_action", "create")
        if action == "reuse":
            self.log.write("Reusing existing cluster (no create needed).\n")
            return

        if action == "recreate":
            self.log.write("Deleting existing cluster...\n")
            delete_cluster(self.k3d_config.cluster_name, log=self.log)

        self.log.write(f"Creating cluster '{self.k3d_config.cluster_name}'...\n")
        create_cluster(
            self.k3d_config.cluster_name,
            self.k3d_config.servers,
            self.k3d_config.agents,
            self.k3d_config.http_port,
            self.k3d_config.https_port,
            self.k3d_config.api_port,
            extra_volumes=volume_mounts,
            log=self.log,
        )

    def write_kubeconfig(self):
        """Subtask: Write kubeconfig file."""
        self.kubeconfig_path = Path(self.k3d_config.kubeconfig_out).expanduser()
        if not self.kubeconfig_path.is_absolute():
            self.kubeconfig_path = (self.repo_root / self.kubeconfig_path).resolve()

        self.log.write(f"Writing kubeconfig to {self.kubeconfig_path}...\n")
        write_kubeconfig(
            self.k3d_config.cluster_name,
            self.kubeconfig_path,
            assume_yes=self.k3d_config.assume_yes,
        )

        # Set environment variable
        os.environ["KUBECONFIG"] = str(self.kubeconfig_path)
        self.log.write(f"Set KUBECONFIG={self.kubeconfig_path}\n")

        # Store in provides
        self._provides["kubeconfig"] = str(self.kubeconfig_path)

    def post_create(self):
        """Subtask: Post-create steps and quick checks/logs."""
        # Placeholder for any fast post-creation configuration or logs
        # Keeps a user-visible step between create and kubeconfig
        self.log.write("Cluster created. Performing post-create steps...\n")

    def write_output(self):
        """Subtask: Write cluster summary."""
        output_json_path = Path(self.k3d_config.output_json).expanduser()
        if not output_json_path.is_absolute():
            output_json_path = (self.repo_root / output_json_path).resolve()

        self.log.write(f"Writing cluster info to {output_json_path}...\n")
        emit_cluster_json(
            path=output_json_path,
            name=self.k3d_config.cluster_name,
            servers=self.k3d_config.servers,
            agents=self.k3d_config.agents,
            http_port=self.k3d_config.http_port,
            https_port=self.k3d_config.https_port,
            kubeconfig=self.kubeconfig_path,
        )

        # Get actual API port if it was auto-assigned
        api_port = self.k3d_config.api_port
        if api_port == 0:
            # For now, keep as 0 to indicate auto-assigned
            api_port = 0

        # Populate all provides values
        self._provides.update(
            {
                "cluster_name": self.k3d_config.cluster_name,
                "http_port": self.k3d_config.http_port,
                "https_port": self.k3d_config.https_port,
                "api_port": api_port,
                "ready": True,
            }
        )

        self.log.write(f"\n✅ k3d cluster '{self.k3d_config.cluster_name}' is ready!\n")

    # Health check methods (called automatically with retry logic)
    def verify_cluster_exists(self):
        """Health check: Verify cluster exists."""
        if not cluster_exists(self.k3d_config.cluster_name):
            raise RuntimeError(f"Cluster '{self.k3d_config.cluster_name}' not found")

    def test_api_connection(self):
        """Health check: Test kubernetes API connectivity."""
        from kubernetes import client as k8s_client, config as k8s_config

        try:
            k8s_config.load_kube_config(config_file=str(self.kubeconfig_path))
            api = k8s_client.CoreV1Api()
            api.list_namespace(limit=1)
        except Exception as e:
            raise RuntimeError(f"API connection failed: {e}")

    def wait_for_nodes(self):
        """Health check: Wait for all nodes to be ready."""
        from kubernetes import client as k8s_client, config as k8s_config

        k8s_config.load_kube_config(config_file=str(self.kubeconfig_path))
        api = k8s_client.CoreV1Api()

        nodes = api.list_node().items
        if not nodes:
            raise RuntimeError("No nodes found in cluster")

        not_ready = []
        for node in nodes:
            conditions = node.status.conditions or []
            is_ready = any(c.type == "Ready" and c.status == "True" for c in conditions)
            if not is_ready:
                not_ready.append(node.metadata.name)

        if not_ready:
            raise RuntimeError(f"Nodes not ready: {', '.join(not_ready)}")

    def verify_system_pods(self):
        """Health check: Verify system pods are running."""
        from kubernetes import client as k8s_client, config as k8s_config

        k8s_config.load_kube_config(config_file=str(self.kubeconfig_path))
        api = k8s_client.CoreV1Api()

        # Check kube-system pods
        pods = api.list_namespaced_pod("kube-system").items
        not_running = [pod.metadata.name for pod in pods if pod.status.phase != "Running"]

        if not_running:
            # Provide a more informative error message
            if len(not_running) > 3:
                # Show first 3 pods and indicate how many more
                msg = f"{len(not_running)} pods not ready: {', '.join(not_running[:3])}... (+{len(not_running)-3} more)"
            else:
                msg = f"{len(not_running)} pods not ready: {', '.join(not_running)}"
            raise RuntimeError(msg)

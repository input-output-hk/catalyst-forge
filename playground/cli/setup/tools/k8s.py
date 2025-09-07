"""
Kubernetes Python API operations helper.
"""

import os
from pathlib import Path
from typing import Any, Dict, IO, List, Optional
from kubernetes import client, config  # type: ignore
from kubernetes.client.exceptions import ApiException  # type: ignore


class K8s:
    """Helper for Kubernetes Python API operations."""

    def __init__(self):
        """Initialize Kubernetes client."""
        self._clients: Optional[Dict[str, Any]] = None
        self._loaded = False

    def _ensure_clients(self):
        """Lazy load Kubernetes clients."""
        if self._loaded:
            return

        # Load kube config following the same logic as deps.py
        playground_dir = Path(__file__).resolve().parent.parent.parent  # .../playground
        env_cfg = os.environ.get("KUBECONFIG", "")
        env_candidates = [p for p in env_cfg.split(os.pathsep) if p]

        kubeconfig_candidate: Optional[Path] = None
        for p in env_candidates:
            path = Path(p).expanduser().resolve()
            if path.exists() and path.is_file():
                kubeconfig_candidate = path
                break
        if kubeconfig_candidate is None:
            kubeconfig_candidate = (playground_dir / "kubeconfig").resolve()

        # Validate and load
        if not kubeconfig_candidate.exists() or kubeconfig_candidate.stat().st_size == 0:
            raise RuntimeError(
                f"Kubeconfig not found or empty: {kubeconfig_candidate}. "
                "Run 'k3d' to create the cluster or set KUBECONFIG."
            )

        try:
            config.load_kube_config(config_file=str(kubeconfig_candidate))
        except Exception as e:
            raise RuntimeError(
                f"Invalid kubeconfig. Ensure it points to a valid cluster. "
                f"Update KUBECONFIG or check {kubeconfig_candidate}. Error: {e}"
            )

        self._clients = {
            "core": client.CoreV1Api(),
            "apps": client.AppsV1Api(),
            "batch": client.BatchV1Api(),
            "rbac": client.RbacAuthorizationV1Api(),
            "networking": client.NetworkingV1Api(),
            "custom_objects": client.CustomObjectsApi(),
        }
        self._loaded = True

    @property
    def clients(self) -> Dict[str, Any]:
        """Get Kubernetes API clients."""
        self._ensure_clients()
        return self._clients or {}

    def get(
        self,
        kind: str,
        name: str,
        namespace: str = "default",
        log: Optional[IO] = None,
    ) -> Any:
        """
        Get a Kubernetes resource.

        Args:
            kind: Resource kind (e.g., "Secret", "ConfigMap")
            name: Resource name
            namespace: Kubernetes namespace
            log: Log file handle (optional)

        Returns:
            Resource object
        """
        self._ensure_clients()

        if log:
            log.write(f"Getting {kind} {name} in namespace {namespace}\n")
            log.flush()

        try:
            if kind == "Secret":
                return self.clients["core"].read_namespaced_secret(name, namespace)
            elif kind == "ConfigMap":
                return self.clients["core"].read_namespaced_config_map(name, namespace)
            elif kind == "Service":
                return self.clients["core"].read_namespaced_service(name, namespace)
            elif kind == "Deployment":
                return self.clients["apps"].read_namespaced_deployment(name, namespace)
            elif kind == "ServiceAccount":
                return self.clients["core"].read_namespaced_service_account(name, namespace)
            elif kind == "Pod":
                return self.clients["core"].read_namespaced_pod(name, namespace)
            else:
                raise NotImplementedError(f"Get not implemented for {kind}")
        except ApiException as e:
            if e.status == 404:
                return None
            raise

    def delete(
        self,
        kind: str,
        name: str,
        namespace: str = "default",
        log: Optional[IO] = None,
    ) -> Any:
        """
        Delete a Kubernetes resource.

        Args:
            kind: Resource kind (e.g., "Secret", "ConfigMap")
            name: Resource name
            namespace: Kubernetes namespace
            log: Log file handle (optional)

        Returns:
            API response object
        """
        self._ensure_clients()

        if log:
            log.write(f"Deleting {kind} {name} in namespace {namespace}\n")
            log.flush()

        try:
            if kind == "Secret":
                return self.clients["core"].delete_namespaced_secret(name, namespace)
            elif kind == "ConfigMap":
                return self.clients["core"].delete_namespaced_config_map(name, namespace)
            elif kind == "Service":
                return self.clients["core"].delete_namespaced_service(name, namespace)
            elif kind == "Deployment":
                return self.clients["apps"].delete_namespaced_deployment(name, namespace)
            elif kind == "ServiceAccount":
                return self.clients["core"].delete_namespaced_service_account(name, namespace)
            else:
                raise NotImplementedError(f"Delete not implemented for {kind}")
        except ApiException as e:
            if e.status == 404:
                if log:
                    log.write(f"Resource {kind}/{name} not found, skipping delete\n")
                return None
            raise

    def wait_for_condition(
        self,
        kind: str,
        name: str,
        namespace: str = "default",
        condition: str = "Ready",
        timeout: int = 300,
        log: Optional[IO] = None,
    ) -> bool:
        """
        Wait for a resource to reach a specific condition.

        Args:
            kind: Resource kind (e.g., "Deployment", "Pod")
            name: Resource name
            namespace: Kubernetes namespace
            condition: Condition to wait for
            timeout: Timeout in seconds
            log: Log file handle (optional)

        Returns:
            True if condition met, False if timeout
        """
        import time

        self._ensure_clients()

        if log:
            log.write(f"Waiting for {kind} {name} to be {condition}...\n")
            log.flush()

        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                resource = self.get(kind, name, namespace)
                if resource:
                    # Check status based on resource type
                    if kind == "Deployment":
                        ready_replicas = getattr(resource.status, "ready_replicas", 0) or 0
                        replicas = getattr(resource.status, "replicas", 1) or 1
                        if ready_replicas >= replicas:
                            if log:
                                log.write(f"{kind} {name} is ready\n")
                            return True
                    elif kind == "Pod":
                        pod_status = getattr(resource.status, "phase", "")
                        if pod_status == "Running":
                            if log:
                                log.write(f"{kind} {name} is ready\n")
                            return True

                time.sleep(5)

            except Exception as e:
                if log:
                    log.write(f"Error checking status: {e}\n")
                time.sleep(5)

        if log:
            log.write(f"Timeout waiting for {kind} {name} to be {condition}\n")
        return False

    def create_http_route(
        self,
        name: str,
        namespace: str,
        hostnames: List[str],
        rules: List[Dict[str, Any]],
        parent_refs: Optional[List[Dict[str, Any]]] = None,
        log: Optional[IO] = None,
    ) -> Any:
        """
        Create an HTTPRoute resource for Gateway API routing.

        Args:
            name: HTTPRoute name
            namespace: Kubernetes namespace
            hostnames: List of hostnames for the route
            rules: List of routing rules
            parent_refs: Optional parent Gateway references
            log: Log file handle (optional)

        Returns:
            API response object
        """
        if parent_refs is None:
            parent_refs = [{"name": "default-gateway", "namespace": "envoy-gateway-system"}]

        manifest = {
            "apiVersion": "gateway.networking.k8s.io/v1",
            "kind": "HTTPRoute",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "parentRefs": parent_refs,
                "hostnames": hostnames,
                "rules": rules,
            },
        }

        from .kubectl import kubectl

        if log is not None:
            return kubectl.apply(manifest, namespace, log)
        else:
            # If no log provided, apply without logging
            import tempfile
            import yaml
            import os

            with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
                yaml.dump(manifest, f, default_flow_style=False)
                manifest_file = f.name
            try:
                import subprocess

                cmd = ["kubectl", "apply", "-n", namespace, "-f", manifest_file]
                result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                return result.stdout
            finally:
                if os.path.exists(manifest_file):
                    os.unlink(manifest_file)

    def create_service(
        self,
        name: str,
        namespace: str,
        selector: Dict[str, str],
        ports: List[Dict[str, Any]],
        service_type: str = "ClusterIP",
        log: Optional[IO] = None,
    ) -> Any:
        """
        Create a Service resource.

        Args:
            name: Service name
            namespace: Kubernetes namespace
            selector: Label selector for pods
            ports: List of port mappings
            service_type: Service type (ClusterIP, NodePort, LoadBalancer)
            log: Log file handle (optional)

        Returns:
            API response object
        """
        manifest = {
            "apiVersion": "v1",
            "kind": "Service",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "type": service_type,
                "selector": selector,
                "ports": ports,
            },
        }

        from .kubectl import kubectl

        if log is not None:
            return kubectl.apply(manifest, namespace, log)
        else:
            # If no log provided, apply without logging
            import tempfile
            import yaml
            import os

            with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
                yaml.dump(manifest, f, default_flow_style=False)
                manifest_file = f.name
            try:
                import subprocess

                cmd = ["kubectl", "apply", "-n", namespace, "-f", manifest_file]
                result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                return result.stdout
            finally:
                if os.path.exists(manifest_file):
                    os.unlink(manifest_file)

    def create_secret(
        self,
        name: str,
        namespace: str,
        data: Dict[str, str],
        secret_type: str = "Opaque",
        log: Optional[IO] = None,
    ) -> Any:
        """
        Create a Secret resource.

        Args:
            name: Secret name
            namespace: Kubernetes namespace
            data: Dictionary of key-value data
            secret_type: Secret type (Opaque, TLS, etc.)
            log: Log file handle (optional)

        Returns:
            API response object
        """
        # Convert string values to base64 for Kubernetes
        string_data = {}
        for key, value in data.items():
            if isinstance(value, str):
                string_data[key] = value

        manifest = {
            "apiVersion": "v1",
            "kind": "Secret",
            "metadata": {"name": name, "namespace": namespace},
            "type": secret_type,
        }

        if string_data:
            manifest["stringData"] = string_data
        else:
            # If no string data, assume data is already base64 encoded
            manifest["data"] = data

        from .kubectl import kubectl

        if log is not None:
            return kubectl.apply(manifest, namespace, log)
        else:
            # If no log provided, apply without logging
            import tempfile
            import yaml
            import os

            with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
                yaml.dump(manifest, f, default_flow_style=False)
                manifest_file = f.name
            try:
                import subprocess

                cmd = ["kubectl", "apply", "-n", namespace, "-f", manifest_file]
                result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                return result.stdout
            finally:
                if os.path.exists(manifest_file):
                    os.unlink(manifest_file)

    def create_external_secret(
        self,
        name: str,
        namespace: str,
        secret_store_ref: Dict[str, str],
        data: List[Dict[str, Any]],
        target_name: Optional[str] = None,
        creation_policy: str = "Owner",
        log: Optional[IO] = None,
    ) -> Any:
        """
        Create an ExternalSecret resource for fetching secrets from external providers.

        Args:
            name: ExternalSecret name
            namespace: Kubernetes namespace
            secret_store_ref: Reference to the SecretStore (name and kind)
            data: List of data mappings with secretKey and remoteRef
            target_name: Name of the target Secret (defaults to same as ExternalSecret)
            creation_policy: Secret creation policy (Owner, Merge, None)
            log: Log file handle (optional)

        Returns:
            API response object
        """
        if target_name is None:
            target_name = name

        manifest = {
            "apiVersion": "external-secrets.io/v1beta1",
            "kind": "ExternalSecret",
            "metadata": {"name": name, "namespace": namespace},
            "spec": {
                "secretStoreRef": secret_store_ref,
                "target": {"name": target_name, "creationPolicy": creation_policy},
                "data": data,
            },
        }

        from .kubectl import kubectl

        if log is not None:
            return kubectl.apply(manifest, namespace, log)
        else:
            # If no log provided, apply without logging
            import tempfile
            import yaml
            import os

            with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
                yaml.dump(manifest, f, default_flow_style=False)
                manifest_file = f.name
            try:
                import subprocess

                cmd = ["kubectl", "apply", "-n", namespace, "-f", manifest_file]
                result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                return result.stdout
            finally:
                if os.path.exists(manifest_file):
                    os.unlink(manifest_file)

    def create_config_map(
        self,
        name: str,
        namespace: str,
        data: Optional[Dict[str, str]] = None,
        binary_data: Optional[Dict[str, bytes]] = None,
        labels: Optional[Dict[str, str]] = None,
        log: Optional[IO] = None,
    ) -> Any:
        """
        Create a ConfigMap resource for storing configuration data.

        Args:
            name: ConfigMap name
            namespace: Kubernetes namespace
            data: Dictionary of string key-value pairs
            binary_data: Dictionary of binary key-value pairs
            labels: Optional labels for the ConfigMap
            log: Log file handle (optional)

        Returns:
            API response object
        """
        metadata: Dict[str, Any] = {"name": name, "namespace": namespace}
        if labels:
            metadata["labels"] = labels

        manifest = {
            "apiVersion": "v1",
            "kind": "ConfigMap",
            "metadata": metadata,
        }

        if data:
            manifest["data"] = data
        if binary_data:
            manifest["binaryData"] = binary_data

        from .kubectl import kubectl

        if log is not None:
            return kubectl.apply(manifest, namespace, log)
        else:
            # If no log provided, apply without logging
            import tempfile
            import yaml
            import os

            with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
                yaml.dump(manifest, f, default_flow_style=False)
                manifest_file = f.name
            try:
                import subprocess

                cmd = ["kubectl", "apply", "-n", namespace, "-f", manifest_file]
                result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                return result.stdout
            finally:
                if os.path.exists(manifest_file):
                    os.unlink(manifest_file)

    def wait_for(
        self,
        resource: str,
        namespace: str,
        log: Optional[IO] = None,
        condition: str = "condition=available",
        timeout: int = 300,
    ) -> bool:
        """
        Wait for a Kubernetes resource to reach a specific condition.

        Args:
            resource: Resource type/name (e.g., "deployment/app", "statefulset/db")
            namespace: Kubernetes namespace
            log: Log file handle (optional)
            condition: Condition to wait for (default: "condition=available")
            timeout: Timeout in seconds (default: 300)

        Returns:
            True if condition met within timeout, False otherwise
        """
        import time

        self._ensure_clients()

        # Parse resource type and name (e.g., "deployment/app" -> "deployment", "app")
        if "/" in resource:
            resource_type, resource_name = resource.split("/", 1)
        else:
            raise ValueError("Resource must be in format 'type/name' (e.g., 'deployment/app')")

        if log:
            log.write(f"Waiting for {resource} to be {condition}...\n")
            log.flush()

        start_time = time.time()

        # Map resource types to API methods
        api_mappings = {
            "deployment": (
                self.clients["apps"].read_namespaced_deployment,
                "ready_replicas",
                "replicas",
            ),
            "statefulset": (
                self.clients["apps"].read_namespaced_stateful_set,
                "ready_replicas",
                "replicas",
            ),
            "pod": (self.clients["core"].read_namespaced_pod, "phase", None),
            "job": (self.clients["batch"].read_namespaced_job, "status", None),
        }

        if resource_type not in api_mappings:
            raise ValueError(f"Unsupported resource type: {resource_type}")

        read_func, ready_field, total_field = api_mappings[resource_type]

        while time.time() - start_time < timeout:
            try:
                resource_obj = read_func(resource_name, namespace)

                if resource_type == "pod":
                    # Check pod phase
                    pod_status = getattr(resource_obj.status, "phase", "")
                    if pod_status == "Running":
                        if log:
                            log.write(f"{resource} is ready\n")
                        return True

                elif resource_type in ["deployment", "statefulset"]:
                    # Check replica counts
                    status = resource_obj.status
                    ready_count = getattr(status, ready_field, 0) or 0

                    if total_field:
                        total_count = getattr(status, total_field, 1) or 1
                        if ready_count >= total_count:
                            if log:
                                log.write(f"{resource} is ready\n")
                            return True
                    else:
                        # For resources without total count, just check if ready > 0
                        if ready_count > 0:
                            if log:
                                log.write(f"{resource} is ready\n")
                            return True

                elif resource_type == "job":
                    # Check job completion
                    job_status = getattr(resource_obj.status, "conditions", [])
                    for condition in job_status:
                        if getattr(condition, "type", "") == "Complete":
                            if log:
                                log.write(f"{resource} is ready\n")
                            return True

            except Exception as e:
                if log:
                    log.write(f"Error checking status: {e}\n")

            time.sleep(5)

        if log:
            log.write(f"Timeout waiting for {resource} to be ready\n")
        return False


# Singleton instance
k8s = K8s()

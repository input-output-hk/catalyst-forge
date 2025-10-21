"""
Configure CoreDNS wildcard DNS for the playground environment.

This task pins a wildcard domain to the Envoy gateway ClusterIP,
allowing all subdomains to resolve to the gateway for ingress routing.
"""

from cli.setup.task_base import BaseTask
from .config import DNSConfig
from cli.setup.tools.kubectl import kubectl
import json
import subprocess
from datetime import datetime, timezone
from typing import Dict, List


class DNSTask(BaseTask):
    """CoreDNS wildcard DNS configuration task."""

    @classmethod
    def provides(cls) -> Dict[str, type]:
        """Define what this task provides."""
        return {
            "configured": bool,
            "domain": str,
            "envoy_ip": str,
        }

    @classmethod
    def requires(cls) -> List[str]:
        """Define what context keys this task requires."""
        return ["k3d.ready", "envoy-gateway.gateway_ready", "envoy-gateway.gateway_namespace"]

    @classmethod
    def config_keys(cls) -> List[str]:
        """Define what config keys this task needs."""
        return ["dns"]

    def setup(self):
        """Register all subtasks and health checks."""
        # Parse configuration once
        self.dns_config = DNSConfig.model_validate(self.cfg["dns"])

        # Get envoy namespace from context (required dependency)
        self.envoy_namespace = self.ctx["envoy-gateway.gateway_namespace"]

        # Register subtasks in execution order
        self.add_subtask("preflight", "Checking prerequisites", 5, self.check_prerequisites)
        self.add_subtask("find_envoy", "Finding Envoy service", 15, self.find_envoy_service)
        self.add_subtask("get_coredns", "Fetching CoreDNS ConfigMap", 10, self.get_coredns_config)
        self.add_subtask("update_corefile", "Updating CoreDNS Corefile", 30, self.update_corefile)
        self.add_subtask("apply_changes", "Applying DNS configuration", 30, self.apply_changes)
        self.add_subtask("output", "Writing configuration summary", 5, self.write_output)

        # Register health checks (run automatically after subtasks)
        self.add_healthcheck(
            "coredns_updated", "Verifying CoreDNS configuration", self.verify_coredns_config
        )
        self.add_healthcheck(
            "dns_resolution", "Testing DNS resolution", self.test_dns_resolution, retries=3
        )

    def check_prerequisites(self):
        """Subtask: Check required commands and context."""
        # Verify kubectl is available
        try:
            subprocess.run(["kubectl", "version", "--client"], capture_output=True, check=True)
            self.log.write("✓ kubectl is available\n")
        except subprocess.CalledProcessError:
            raise RuntimeError("kubectl command not found or not working")

        # Verify we have required context
        if not self.envoy_namespace:
            raise RuntimeError("Envoy gateway namespace not found in context")

        self.log.write(f"✓ Prerequisites check complete (envoy namespace: {self.envoy_namespace})\n")

    def find_envoy_service(self):
        """Subtask: Find Envoy service ClusterIP."""
        self.log.write(f"Finding Envoy service in {self.envoy_namespace}...\n")

        try:
            # Use kubectl helper to get services
            result = kubectl.get(
                "svc",
                namespace=self.envoy_namespace,
                output="json"
            )

            # Filter services by label
            envoy_services = []
            for svc in result.get("items", []):
                labels = svc.get("metadata", {}).get("labels", {})
                if labels.get("app.kubernetes.io/name") == "envoy":
                    envoy_services.append(svc)

            if not envoy_services:
                raise RuntimeError(f"No Envoy services found in namespace {self.envoy_namespace}")

            # Find Envoy ClusterIP
            self.envoy_ip = None
            for svc in envoy_services:
                ip = svc.get("spec", {}).get("clusterIP")
                if ip and ip != "None":
                    self.envoy_ip = ip
                    self.log.write(f"Found Envoy ClusterIP: {ip}\n")
                    break

            if not self.envoy_ip:
                raise RuntimeError("Envoy service has no ClusterIP assigned yet")

        except Exception as e:
            self.log.write(f"Failed to get Envoy services: {e}\n")
            raise RuntimeError(f"Could not find Envoy services in namespace {self.envoy_namespace}")

    def get_coredns_config(self):
        """Subtask: Fetch current CoreDNS ConfigMap."""
        self.log.write(f"Fetching CoreDNS ConfigMap from {self.dns_config.coredns_namespace}...\n")

        try:
            self.coredns_cm = kubectl.get(
                "configmap",
                self.dns_config.coredns_configmap,
                namespace=self.dns_config.coredns_namespace,
                output="json"
            )
        except Exception as e:
            self.log.write(f"Failed to get CoreDNS ConfigMap: {e}\n")
            raise RuntimeError("CoreDNS ConfigMap not found")

        self.original_corefile = self.coredns_cm.get("data", {}).get("Corefile", "")
        if not self.original_corefile:
            raise RuntimeError("CoreDNS Corefile not found in ConfigMap")

        self.log.write("✓ CoreDNS ConfigMap retrieved successfully\n")

    def update_corefile(self):
        """Subtask: Update CoreDNS Corefile with wildcard DNS template."""
        self.log.write(f"Updating Corefile with wildcard DNS for *.{self.dns_config.domain}...\n")

        # Create managed block for wildcard DNS
        escaped_domain = self.dns_config.domain.replace(".", "\\.")
        begin_marker = f"# BEGIN catalyst-forge wildcard {self.dns_config.domain}"
        end_marker = f"# END catalyst-forge wildcard {self.dns_config.domain}"

        managed_block = f"""{begin_marker}
    template IN A {self.dns_config.domain} {{
        match "^([a-z0-9-]+\\.)*{escaped_domain}\\.$"
        answer "{{{{ .Name }}}} 300 IN A {self.envoy_ip}"
        fallthrough
    }}
{end_marker}"""

        corefile = self.original_corefile

        # Check if already configured - remove existing block if present
        if begin_marker in corefile and end_marker in corefile:
            self.log.write("DNS wildcard already configured, updating IP if needed...\n")
            # Remove existing block
            pre, _, rest = corefile.partition(begin_marker)
            _, _, post = rest.partition(end_marker)
            corefile = pre.rstrip() + "\n" + post.lstrip()

        # Find main server block (.:53)
        lines = corefile.splitlines()
        insert_idx = -1
        for i, line in enumerate(lines):
            if line.strip().startswith(".:53") and "{" in line:
                # Find the closing brace for this block
                brace_count = 1
                for j in range(i + 1, len(lines)):
                    brace_count += lines[j].count("{")
                    brace_count -= lines[j].count("}")
                    if brace_count == 0:
                        insert_idx = j
                        break
                break

        if insert_idx < 0:
            raise RuntimeError("Could not find main .:53 server block in CoreDNS Corefile")

        # Insert managed block before closing brace
        lines.insert(insert_idx, managed_block)
        self.updated_corefile = "\n".join(lines)

        # Update ConfigMap data
        self.coredns_cm["data"]["Corefile"] = self.updated_corefile

        self.log.write("✓ Corefile updated with wildcard DNS configuration\n")

    def apply_changes(self):
        """Subtask: Apply CoreDNS configuration changes."""
        self.log.write("Applying DNS configuration changes...\n")

        try:
            kubectl.apply(self.coredns_cm, self.dns_config.coredns_namespace, self.log)
            self.log.write("✓ ConfigMap updated successfully\n")
        except Exception as e:
            self.log.write(f"Failed to apply ConfigMap: {e}\n")
            raise RuntimeError("Failed to apply CoreDNS configuration")

        # Check if reload plugin is present
        self.has_reload = "reload" in self.updated_corefile

        if self.has_reload:
            self.log.write("✓ CoreDNS has reload plugin, changes will be applied automatically\n")
        else:
            # Restart CoreDNS to apply changes
            self.log.write("Restarting CoreDNS to apply changes...\n")
            timestamp = datetime.now(timezone.utc).isoformat()

            try:
                result = subprocess.run(
                    [
                        "kubectl",
                        "patch",
                        "deployment",
                        "coredns",
                        "-n",
                        self.dns_config.coredns_namespace,
                        "--patch",
                        json.dumps(
                            {
                                "spec": {
                                    "template": {
                                        "metadata": {
                                            "annotations": {"kubectl.kubernetes.io/restartedAt": timestamp}
                                        }
                                    }
                                }
                            }
                        ),
                    ],
                    capture_output=True,
                    text=True,
                    check=True,
                )
                self.log.write(f"✓ CoreDNS restarted: {result.stdout}\n")
            except subprocess.CalledProcessError as e:
                self.log.write(f"Failed to restart CoreDNS: {e}\n")
                raise RuntimeError("Failed to restart CoreDNS deployment")

    def write_output(self):
        """Subtask: Write configuration summary."""
        self.log.write("\n✅ DNS wildcard configured:\n")
        self.log.write(f"   Domain: *.{self.dns_config.domain}\n")
        self.log.write(f"   Envoy IP: {self.envoy_ip}\n")

        # Populate provides
        self._provides.update({
            "configured": True,
            "domain": self.dns_config.domain,
            "envoy_ip": self.envoy_ip,
        })

    # Health check methods (called automatically with retry logic)

    def verify_coredns_config(self):
        """Health check: Verify CoreDNS configuration is applied."""
        try:
            cm = kubectl.get(
                "configmap",
                self.dns_config.coredns_configmap,
                namespace=self.dns_config.coredns_namespace,
                output="json"
            )
            current_corefile = cm.get("data", {}).get("Corefile", "")
            expected_marker = f"# BEGIN catalyst-forge wildcard {self.dns_config.domain}"
            if expected_marker not in current_corefile:
                raise RuntimeError("DNS wildcard configuration not found in CoreDNS")
        except Exception as e:
            raise RuntimeError(f"Cannot verify CoreDNS configuration: {e}")

    def test_dns_resolution(self):
        """Health check: Test DNS resolution from within the k3d cluster."""
        test_domain = f"test.{self.dns_config.domain}"

        # Test DNS resolution from within the cluster using nslookup
        try:
            # Find a running pod in the cluster to use for DNS testing
            result = subprocess.run(
                [
                    "kubectl",
                    "get",
                    "pods",
                    "-n",
                    "kube-system",
                    "-l",
                    "k8s-app=kube-dns",
                    "-o",
                    "jsonpath={.items[0].metadata.name}",
                ],
                capture_output=True,
                text=True,
                check=True,
            )
            coredns_pod = result.stdout.strip()

            if not coredns_pod:
                raise RuntimeError("No CoreDNS pod found for DNS testing")

            # Use kubectl exec to run nslookup from within the cluster
            result = subprocess.run(
                [
                    "kubectl",
                    "exec",
                    "-n",
                    "kube-system",
                    coredns_pod,
                    "--",
                    "nslookup",
                    test_domain,
                ],
                capture_output=True,
                text=True,
                timeout=10,
            )

            # Check if the DNS query succeeded and returned the expected IP
            if result.returncode == 0:
                output = result.stdout.lower()
                if f"name: {test_domain}" in output and self.envoy_ip in output:
                    self.log.write(f"✓ DNS resolution successful: {test_domain} -> {self.envoy_ip}\n")
                    return
                else:
                    raise RuntimeError(f"DNS query succeeded but didn't return expected IP. Output: {result.stdout}")
            else:
                raise RuntimeError(f"DNS query failed. stderr: {result.stderr}")

        except subprocess.TimeoutExpired:
            raise RuntimeError(f"DNS query timed out for {test_domain}")
        except subprocess.CalledProcessError as e:
            raise RuntimeError(f"Failed to execute DNS test: {e}")
        except Exception as e:
            raise RuntimeError(f"DNS resolution test failed: {e}")

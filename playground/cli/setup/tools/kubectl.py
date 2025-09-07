"""
Kubectl operations helper.
"""

import subprocess
import yaml
import json
import tempfile
import os
from typing import Dict, Any, IO, Optional


class Kubectl:
    """Helper for kubectl operations."""

    def apply(self, manifest: Dict[str, Any], namespace: str, log: IO):
        """
        Apply a Kubernetes manifest.

        Args:
            manifest: Kubernetes manifest dictionary
            namespace: Kubernetes namespace
            log: Log file handle
        """
        with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
            yaml.dump(manifest, f, default_flow_style=False)
            manifest_file = f.name

        try:
            cmd = ["kubectl", "apply", "-n", namespace, "-f", manifest_file]

            log.write(f"$ {' '.join(cmd)}\n")
            log.write(
                f"--- Manifest ---\n{yaml.dump(manifest, default_flow_style=False)}\n--- End Manifest ---\n\n"
            )
            log.flush()

            proc = subprocess.Popen(
                cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True
            )

            if proc.stdout:
                for line in proc.stdout:
                    log.write(line)
                    log.flush()

            proc.wait()
            if proc.returncode != 0:
                raise RuntimeError("kubectl apply failed")
        finally:
            if os.path.exists(manifest_file):
                os.unlink(manifest_file)

    def get(
        self,
        resource: str,
        name: Optional[str] = None,
        namespace: str = "default",
        output: str = "json",
    ) -> Any:
        """
        Get a Kubernetes resource.

        Args:
            resource: Resource type (e.g., "secret", "configmap")
            name: Resource name (optional)
            namespace: Kubernetes namespace
            output: Output format (json, yaml, etc.)

        Returns:
            Parsed resource data
        """
        cmd = ["kubectl", "get", resource]
        if name:
            cmd.append(name)
        cmd.extend(["-n", namespace, "-o", output])

        result = subprocess.run(cmd, capture_output=True, text=True, check=True)

        if output == "json":
            return json.loads(result.stdout)
        elif output == "yaml":
            return yaml.safe_load(result.stdout)
        else:
            return result.stdout


# Singleton instance
kubectl = Kubectl()

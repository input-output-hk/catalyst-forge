"""
Helm chart operations helper.
"""

import subprocess
import yaml
from typing import Dict, Any, IO
import tempfile
import os


class Helm:
    """Helper for Helm chart operations."""

    def install(
        self,
        name: str,
        chart: str,
        namespace: str,
        values: Dict[str, Any],
        log: IO,
        wait: bool = True,
        timeout: str = "5m",
    ):
        """
        Install a Helm chart with automatic logging.

        Args:
            name: Release name
            chart: Chart reference (e.g., "bitnami/postgresql")
            namespace: Kubernetes namespace
            values: Values dictionary to pass to Helm
            log: Log file handle
            wait: Whether to wait for deployment
            timeout: Timeout duration (default: 5m)
        """
        with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
            yaml.dump(values, f, default_flow_style=False)
            values_file = f.name

        try:
            cmd = [
                "helm",
                "upgrade",
                "--install",
                name,
                chart,
                "-n",
                namespace,
                "--create-namespace",
                "-f",
                values_file,
            ]
            if wait:
                cmd.extend(["--wait", "--timeout", timeout])

            log.write(f"$ {' '.join(cmd)}\n")
            log.write(f"Values file: {values_file}\n")
            log.write(
                f"--- Values ---\n{yaml.dump(values, default_flow_style=False)}\n--- End Values ---\n\n"
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
                raise RuntimeError(f"Helm install failed for {name}")
        finally:
            if os.path.exists(values_file):
                os.unlink(values_file)


# Singleton instance
helm = Helm()

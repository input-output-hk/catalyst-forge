"""
Configure CoreDNS wildcard DNS for the playground environment.

This task pins a wildcard domain to the Envoy gateway ClusterIP,
allowing all subdomains to resolve to the gateway for ingress routing.
"""

from cli.setupv2 import task
from .config import DNSConfig
import json
import subprocess


@task(
    "dns",
    requires=["k3d.ready", "envoy-gateway.gateway_ready"],  # Need cluster and envoy gateway
    provides={"configured": bool, "domain": str, "envoy_ip": str},
    config_keys=["dns"],  # Load DNS configuration from config.cue
    timeout_sec=120,
    retry_delays=[5, 10],
)
def setup_dns(ctx, cfg, log):
    """Configure CoreDNS wildcard DNS pinning to Envoy gateway."""

    # Parse configuration with validation and defaults
    dns_config = DNSConfig.model_validate(cfg["dns"])

    # Get envoy namespace from context (required dependency)
    envoy_namespace = ctx["envoy-gateway.gateway_namespace"]

    log.write(f"Configuring DNS wildcard for *.{dns_config.domain}...\n")

    # Get Envoy Service ClusterIP
    log.write(f"Finding Envoy service in {envoy_namespace}...\n")

    # Use kubectl to get services
    try:
        result = subprocess.run(
            [
                "kubectl",
                "get",
                "svc",
                "-n",
                envoy_namespace,
                "-l",
                "app.kubernetes.io/name=envoy",
                "-o",
                "json",
            ],
            capture_output=True,
            text=True,
            check=True,
        )
        services = json.loads(result.stdout)
    except subprocess.CalledProcessError as e:
        log.write(f"Failed to get Envoy services: {e}\n")
        log.write(f"stderr: {e.stderr}\n")
        # Envoy may not be installed yet, but we can still configure DNS
        log.write("Envoy gateway not found yet, skipping DNS configuration\n")
        return {"configured": False, "domain": dns_config.domain, "envoy_ip": ""}

    # Find Envoy ClusterIP
    envoy_ip = None
    for svc in services.get("items", []):
        ip = svc.get("spec", {}).get("clusterIP")
        if ip and ip != "None":
            envoy_ip = ip
            log.write(f"Found Envoy ClusterIP: {ip}\n")
            break

    if not envoy_ip:
        log.write("Envoy service has no ClusterIP yet, skipping DNS configuration\n")
        return {"configured": False, "domain": dns_config.domain, "envoy_ip": ""}

    # Get CoreDNS ConfigMap
    log.write(f"Fetching CoreDNS ConfigMap from {dns_config.coredns_namespace}...\n")

    try:
        result = subprocess.run(
            [
                "kubectl",
                "get",
                "configmap",
                dns_config.coredns_configmap,
                "-n",
                dns_config.coredns_namespace,
                "-o",
                "json",
            ],
            capture_output=True,
            text=True,
            check=True,
        )
        cm = json.loads(result.stdout)
    except subprocess.CalledProcessError as e:
        log.write(f"Failed to get CoreDNS ConfigMap: {e}\n")
        raise RuntimeError("CoreDNS ConfigMap not found")

    corefile = cm.get("data", {}).get("Corefile", "")
    if not corefile:
        raise RuntimeError("CoreDNS Corefile not found in ConfigMap")

    # Create managed block for wildcard DNS
    escaped_domain = dns_config.domain.replace(".", "\\.")
    begin_marker = f"# BEGIN catalyst-forge wildcard {dns_config.domain}"
    end_marker = f"# END catalyst-forge wildcard {dns_config.domain}"

    managed_block = f"""{begin_marker}
    template IN A {dns_config.domain} {{
        match "^([a-z0-9-]+\\.)*{escaped_domain}\\.$"
        answer "{{{{ .Name }}}} 300 IN A {envoy_ip}"
        fallthrough
    }}
{end_marker}"""

    # Check if already configured
    if begin_marker in corefile and end_marker in corefile:
        log.write("DNS wildcard already configured, updating IP if needed...\n")
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
    new_corefile = "\n".join(lines)

    # Update ConfigMap
    cm["data"]["Corefile"] = new_corefile

    log.write("Updating CoreDNS ConfigMap...\n")

    # Write updated ConfigMap to temp file and apply
    import tempfile

    with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
        json.dump(cm, f)
        temp_file = f.name

    try:
        result = subprocess.run(
            ["kubectl", "apply", "-f", temp_file], capture_output=True, text=True, check=True
        )
        log.write(f"ConfigMap updated: {result.stdout}\n")
    finally:
        import os

        os.unlink(temp_file)

    # Check if reload plugin is present
    has_reload = "reload" in new_corefile

    if has_reload:
        log.write("CoreDNS has reload plugin, changes will be applied automatically\n")
    else:
        # Restart CoreDNS to apply changes
        log.write("Restarting CoreDNS to apply changes...\n")
        from datetime import datetime

        timestamp = datetime.utcnow().isoformat() + "Z"

        result = subprocess.run(
            [
                "kubectl",
                "patch",
                "deployment",
                "coredns",
                "-n",
                dns_config.coredns_namespace,
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
        log.write(f"CoreDNS restarted: {result.stdout}\n")

    log.write("\n✅ DNS wildcard configured:\n")
    log.write(f"   Domain: *.{dns_config.domain}\n")
    log.write(f"   Envoy IP: {envoy_ip}\n")

    return {"configured": True, "domain": dns_config.domain, "envoy_ip": envoy_ip}

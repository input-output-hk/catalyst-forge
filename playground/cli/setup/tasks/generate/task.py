"""
Generate TLS certificates and Earthly configuration.

This task creates client certificates for mTLS authentication with buildkitd
and generates the Earthly configuration file with proper TLS settings.
"""

from cli.setup import task
from pathlib import Path
import subprocess
import yaml
from .config import GenerateConfig


@task(
    "generate",
    requires=["k3d.ready"],  # Need cluster context
    provides={
        "client_cert": str,
        "client_key": str,
        "ca_cert": str,
        "earthly_config": str,
        "ready": bool,
    },
    config_keys=["generate"],  # Load generate configuration from config.cue
    timeout_sec=60,
    retry_delays=[5],
)
def setup_generate(ctx, cfg, log):
    """Generate TLS certificates and Earthly configuration."""

    # Parse configuration with validation and defaults
    generate_config = GenerateConfig.model_validate(cfg["generate"])

    log.write("Generating TLS certificates and Earthly config...\n")

    # Check mkcert is available
    try:
        result = subprocess.run(["mkcert", "-version"], capture_output=True, text=True, check=True)
        log.write(f"mkcert available: {result.stdout.strip()}\n")
    except (subprocess.CalledProcessError, FileNotFoundError):
        raise RuntimeError("mkcert is required but not found. Please install mkcert.")

    # Get mkcert CA root
    result = subprocess.run(["mkcert", "-CAROOT"], capture_output=True, text=True, check=True)
    caroot = Path(result.stdout.strip())
    ca_src = caroot / "rootCA.pem"

    if not ca_src.exists():
        log.write("mkcert CA not found, installing...\n")
        subprocess.run(["mkcert", "-install"], capture_output=True, text=True, check=True)

    # Resolve paths
    from cli.utils import get_repo_root

    # Get repo root starting from this file's location
    current_file = Path(__file__).resolve()
    playground_dir = current_file.parent.parent.parent  # .../playground
    repo_root = get_repo_root(playground_dir)

    # Resolve certificate directory path
    cert_dir = Path(generate_config.cert_dir).expanduser()
    if not cert_dir.is_absolute():
        cert_dir = (repo_root / cert_dir).resolve()
    cert_dir.mkdir(parents=True, exist_ok=True)

    # Certificate paths
    client_cert = cert_dir / f"{generate_config.client_name}.pem"
    client_key = cert_dir / f"{generate_config.client_name}-key.pem"
    ca_cert = cert_dir / "rootCA.pem"

    # Generate client certificate if missing
    if not client_cert.exists() or not client_key.exists():
        log.write(f"Generating client certificate for {generate_config.client_name}...\n")

        result = subprocess.run(
            [
                "mkcert",
                "-client",
                "-cert-file",
                str(client_cert),
                "-key-file",
                str(client_key),
                generate_config.client_name,
            ],
            capture_output=True,
            text=True,
            check=True,
        )

        if result.returncode != 0:
            log.write(f"Error generating certificate: {result.stderr}\n")
            raise RuntimeError("Failed to generate client certificate")

        log.write("Generated client certificate:\n")
        log.write(f"  Cert: {client_cert}\n")
        log.write(f"  Key:  {client_key}\n")
    else:
        log.write("Client certificates already exist:\n")
        log.write(f"  Cert: {client_cert}\n")
        log.write(f"  Key:  {client_key}\n")

    # Copy CA certificate
    if ca_src.exists() and not ca_cert.exists():
        log.write(f"Copying CA certificate to {ca_cert}...\n")
        ca_cert.write_bytes(ca_src.read_bytes())
    elif ca_cert.exists():
        log.write(f"CA certificate already exists: {ca_cert}\n")

    # Generate Earthly configuration
    earthly_config_dir = Path(generate_config.earthly_config_dir).expanduser()
    if not earthly_config_dir.is_absolute():
        earthly_config_dir = (repo_root / earthly_config_dir).resolve()
    earthly_config_dir.mkdir(parents=True, exist_ok=True)
    earthly_config_path = earthly_config_dir / "earthly.yml"

    log.write("Generating Earthly configuration...\n")

    earthly_config = {
        "global": {
            "buildkit_host": generate_config.buildkit_host,
            "tlsca": str(ca_cert.resolve()),
            "tlscert": str(client_cert.resolve()),
            "tlskey": str(client_key.resolve()),
        }
    }

    earthly_config_path.write_text(yaml.safe_dump(earthly_config, sort_keys=False))
    log.write(f"Earthly config written to: {earthly_config_path}\n")

    # Optionally update the user's ~/.earthly/config.yml
    if generate_config.update_home_config:
        home_earthly = Path.home() / ".earthly"
        home_earthly.mkdir(parents=True, exist_ok=True)
        home_config = home_earthly / "config.yml"

        # Create or update home config
        if not home_config.exists() or home_config.read_text() != earthly_config_path.read_text():
            log.write("Updating ~/.earthly/config.yml...\n")
            home_config.write_text(earthly_config_path.read_text())

    log.write("\n✅ TLS and Earthly configuration ready:\n")
    log.write(f"   Client cert: {client_cert}\n")
    log.write(f"   Client key:  {client_key}\n")
    log.write(f"   CA cert:     {ca_cert}\n")
    log.write(f"   Earthly config: {earthly_config_path}\n")
    log.write(f"   Buildkit host: {generate_config.buildkit_host}\n")
    log.write("\nYou can now use Earthly with mTLS authentication.\n")

    return {
        "client_cert": str(client_cert),
        "client_key": str(client_key),
        "ca_cert": str(ca_cert),
        "earthly_config": str(earthly_config_path),
        "ready": True,
    }

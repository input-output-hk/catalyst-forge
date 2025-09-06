"""
Configuration models for generate task.
"""

from pydantic import BaseModel


class GenerateConfig(BaseModel):
    """Configuration for TLS certificate generation and Earthly setup.

    This model defines parameters for generating client certificates for mTLS
    authentication with buildkitd and creating Earthly configuration files.
    """

    client_name: str = "earthly-client"
    """Name for the client certificate and key files.

    This will be used as the base name for generated certificate files
    (e.g., earthly-client.pem, earthly-client-key.pem).
    Default: "earthly-client"
    """

    buildkit_host: str = "tcp://buildkit.projectcatalyst.dev:8372"
    """Buildkit daemon host URL for Earthly configuration.

    The TCP endpoint where the buildkit daemon is running, used for
    Earthly's remote build execution with TLS authentication.
    Default: "tcp://buildkit.projectcatalyst.dev:8372"
    """

    cert_dir: str = "playground/.certs"
    """Directory path for storing generated certificates.

    Relative to the repository root where client certificates and CA
    certificates will be stored. Will be created if it doesn't exist.
    Default: "playground/.certs"
    """

    earthly_config_dir: str = "playground/config"
    """Directory path for storing Earthly configuration.

    Relative to the repository root where earthly.yml will be written.
    Will be created if it doesn't exist.
    Default: "playground/config"
    """

    update_home_config: bool = False
    """Whether to update the user's ~/.earthly/config.yml.

    When True, copies the generated Earthly config to the user's home
    directory for system-wide Earthly usage. When False, only creates
    the config in the project directory.
    Default: False
    """

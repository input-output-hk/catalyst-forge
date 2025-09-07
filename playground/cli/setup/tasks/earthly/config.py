"""
Configuration models for earthly task.
"""

from pydantic import BaseModel


class EarthlyConfig(BaseModel):
    """Configuration for Earthly buildkit deployment."""

    buildkit_host: str = "tcp://buildkit.projectcatalyst.dev:8372"
    """Buildkit daemon host URL for Earthly remote builds."""

    manifests_dir: str = "platform/earthly"
    """Directory containing Earthly buildkit manifests."""

    namespace: str = "registry"
    """Kubernetes namespace for buildkit deployment."""

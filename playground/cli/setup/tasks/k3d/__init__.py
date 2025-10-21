"""
k3d task package.

This package contains the k3d Kubernetes cluster deployment task for the playground environment.
"""

from .task import K3dTask

__all__ = ["K3dTask"]

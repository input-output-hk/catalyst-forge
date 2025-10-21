"""
DNS task package.

This package contains the CoreDNS wildcard DNS configuration task for the playground environment.
"""

from .task import DNSTask

__all__ = ["DNSTask"]

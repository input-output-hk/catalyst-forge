"""Database migration package for playground CLI.

Provides a modular seeding framework with a root connection manager and
pluggable seeders for different applications (e.g., Kratos, Hydra).
"""

from .base import DatabaseRoot

__all__ = [
    "DatabaseRoot",
]

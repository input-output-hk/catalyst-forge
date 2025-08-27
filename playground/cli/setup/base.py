from __future__ import annotations

from typing import Protocol


class SetupTask(Protocol):
    """Protocol for pluggable setup tasks.

    Implementations should be idempotent and safe to re-run.
    """

    def run(self) -> None:  # pragma: no cover - simple protocol
        ...

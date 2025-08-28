from __future__ import annotations

from dataclasses import dataclass
from typing import Callable, Dict, Iterable, Optional, Tuple, TYPE_CHECKING

from .base import Seeder
from ..config import ConfigState

if TYPE_CHECKING:  # avoid runtime import cycle
    from ..deps import Deps


SeederFactory = Callable[[ConfigState, "Deps"], Optional[Seeder]]


@dataclass(frozen=True)
class SeederMeta:
    name: str
    description: str
    priority: int
    dependencies: Tuple[str, ...]
    factory: SeederFactory


class _SeederRegistry:
    def __init__(self) -> None:
        self._by_name: Dict[str, SeederMeta] = {}

    def register(
        self,
        name: str,
        description: str,
        priority: int,
        dependencies: Tuple[str, ...],
        factory: SeederFactory,
    ) -> None:
        if name in self._by_name:
            raise ValueError(f"seeder '{name}' already registered")
        self._by_name[name] = SeederMeta(
            name=name,
            description=description,
            priority=priority,
            dependencies=dependencies,
            factory=factory,
        )

    def get(self, name: str) -> Optional[SeederMeta]:
        return self._by_name.get(name)

    def all(self) -> Iterable[SeederMeta]:
        return self._by_name.values()


registry = _SeederRegistry()


def register_seeder(
    name: str, description: str, priority: int = 100, dependencies: Tuple[str, ...] = ()
):
    def _decorator(factory: SeederFactory) -> SeederFactory:
        registry.register(
            name=name,
            description=description,
            priority=priority,
            dependencies=dependencies,
            factory=factory,
        )
        return factory

    return _decorator

from __future__ import annotations

from dataclasses import dataclass
from typing import Callable, Dict, Iterable, Optional, Protocol, Tuple, TYPE_CHECKING

from ..config import ConfigState

if TYPE_CHECKING:  # avoid runtime import cycle
    from ..deps import Deps


class SetupTask(Protocol):
    def run(self) -> None:  # pragma: no cover - simple protocol
        ...


SetupTaskFactory = Callable[[ConfigState, "Deps"], Optional[SetupTask]]


@dataclass(frozen=True)
class SetupTaskMeta:
    name: str
    description: str
    priority: int
    dependencies: Tuple[str, ...]
    factory: SetupTaskFactory


class _SetupRegistry:
    def __init__(self) -> None:
        self._by_name: Dict[str, SetupTaskMeta] = {}

    def register(
        self,
        name: str,
        description: str,
        priority: int,
        dependencies: Tuple[str, ...],
        factory: SetupTaskFactory,
    ) -> None:
        if name in self._by_name:
            raise ValueError(f"setup task '{name}' already registered")
        self._by_name[name] = SetupTaskMeta(
            name=name,
            description=description,
            priority=priority,
            dependencies=dependencies,
            factory=factory,
        )

    def get(self, name: str) -> Optional[SetupTaskMeta]:
        return self._by_name.get(name)

    def all(self) -> Iterable[SetupTaskMeta]:
        return self._by_name.values()


registry = _SetupRegistry()


def register_setup_task(
    name: str, description: str, priority: int = 100, dependencies: Tuple[str, ...] = ()
):
    def _decorator(factory: SetupTaskFactory) -> SetupTaskFactory:
        registry.register(
            name=name,
            description=description,
            priority=priority,
            dependencies=dependencies,
            factory=factory,
        )
        return factory

    return _decorator

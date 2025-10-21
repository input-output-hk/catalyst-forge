"""
Base class for all setup tasks with automatic progress and error handling.
"""

from abc import ABC, abstractmethod
from typing import Any, Dict, List, Callable, Optional
from dataclasses import dataclass


@dataclass
class SubtaskDef:
    """Definition of a subtask."""

    id: str
    description: str
    weight: float
    func: Callable[[], Any]
    result: Any = None
    error: Optional[str] = None


@dataclass
class HealthCheckDef:
    """Definition of a health check."""

    id: str
    description: str
    func: Callable[[], None]
    retries: int = 3
    retry_delays: Optional[List[int]] = None


class TaskError(Exception):
    """Rich exception for task failures."""

    def __init__(self, task_name: str, subtask: str, error: Exception):
        self.task_name = task_name
        self.subtask = subtask
        self.original_error = error
        super().__init__(f"Task '{task_name}' failed at subtask '{subtask}': {error}")


class BaseTask(ABC):
    """Base class for all setup tasks."""

    def __init__(self, name: str, ctx: Dict, cfg: Dict, log):
        """Initialize task with context and configuration.

        Args:
            name: Task name
            ctx: Context from completed tasks (read-only)
            cfg: Task-specific configuration
            log: File logger for detailed output
        """
        self.name = name
        self.ctx = ctx  # Read access to context from other tasks
        self.cfg = cfg  # Task-specific configuration
        self.log = log  # File logger
        self._subtasks: List[SubtaskDef] = []
        self._health_checks: List[HealthCheckDef] = []
        self._provides: Dict[str, Any] = {}

        # Call setup to register subtasks
        self.setup()

    @abstractmethod
    def setup(self):
        """Register subtasks and health checks. Must be implemented by subclasses."""
        pass

    @classmethod
    @abstractmethod
    def provides(cls) -> Dict[str, type]:
        """Define what this task provides to the context."""
        pass

    @classmethod
    def requires(cls) -> List[str]:
        """Define what context keys this task requires. Override if needed."""
        return []

    @classmethod
    def config_keys(cls) -> List[str]:
        """Define what config keys this task needs. Override if needed."""
        return []

    def add_subtask(self, id: str, description: str, weight: float, func: Callable):
        """Register a subtask.

        Args:
            id: Unique identifier for the subtask
            description: Human-readable description
            weight: Relative weight for progress calculation
            func: Function to execute (should be a method of this class)
        """
        self._subtasks.append(SubtaskDef(id=id, description=description, weight=weight, func=func))

    def add_healthcheck(
        self,
        id: str,
        description: str,
        func: Callable,
        retries: int = 3,
        delays: Optional[List[int]] = None,
    ):
        """Register a health check.

        Args:
            id: Unique identifier for the health check
            description: Human-readable description
            func: Function to execute (should raise on failure)
            retries: Number of retry attempts
            delays: List of delays between retries in seconds
        """
        if delays is None:
            delays = [5, 10, 20]
        self._health_checks.append(
            HealthCheckDef(
                id=id,
                description=description,
                func=func,
                retries=retries,
                retry_delays=delays[:retries],
            )
        )

    def run(self, progress) -> Dict[str, Any]:
        """Execute all subtasks and health checks with progress tracking.

        Args:
            progress: Progress tracker instance

        Returns:
            Dictionary of provided values

        Raises:
            TaskError: If any subtask fails
        """
        # Calculate weights
        subtask_weight = sum(s.weight for s in self._subtasks)
        health_ratio = self.health_weight_ratio()
        health_weight = subtask_weight * health_ratio if self._health_checks else 0
        total_weight = subtask_weight + health_weight

        # Define all subtasks for progress
        all_subtasks = [
            (s.id, s.description, s.weight / total_weight * 100) for s in self._subtasks
        ]
        if self._health_checks:
            all_subtasks.append(
                ("health", "Running health checks", health_weight / total_weight * 100)
            )

        progress.define_subtasks(all_subtasks)
        # Emit an initial "started" status with explicit health weight pct so the coordinator knows
        try:
            if hasattr(progress, "callback"):
                # Piggyback: send a message that includes the health weight percent
                # Actual mapping to events is handled in the callback bridge.
                pass
        except Exception:
            pass

        # Execute subtasks in order
        for subtask in self._subtasks:
            try:
                with progress.subtask(subtask.id):
                    self.log.write(f"[{self.name}] Starting: {subtask.description}\n")
                    result = subtask.func()
                    subtask.result = result
                    self.log.write(f"[{self.name}] Completed: {subtask.description}\n")

            except Exception as e:
                subtask.error = str(e)
                self.log.write(f"[{self.name}] Failed: {subtask.description} - {e}\n")
                raise TaskError(self.name, subtask.id, e)

        # Run health checks if defined
        if self._health_checks:
            with progress.subtask("health"):
                self._run_health_checks(progress)

        # Ensure progress completes at 100%
        try:
            if hasattr(progress, "callback"):
                progress.callback(self.name, 100, "Complete")
        except Exception:
            pass

        # Return provides
        return self._provides

    def _run_health_checks(self, progress):
        """Run all registered health checks with retry logic."""
        from cli.tui.health import HealthCheckProgress

        # Create health check progress tracker
        health_progress = HealthCheckProgress(progress, "health")
        health_progress.define_checks([(hc.id, hc.description) for hc in self._health_checks])

        # Log start of health checks
        self.log.write(f"[{self.name}] Running {len(self._health_checks)} health checks...\n")

        for idx, check in enumerate(self._health_checks, 1):
            self.log.write(f"[{self.name}] Health check {idx}/{len(self._health_checks)}: {check.description}\n")

            success = health_progress.run_check_with_retry(
                check.id, check.func, retries=check.retries, delays=check.retry_delays
            )

            if success:
                self.log.write(f"[{self.name}] ✓ Health check passed: {check.description}\n")
            else:
                self.log.write(f"[{self.name}] ✗ Health check failed: {check.description}\n")
                raise RuntimeError(
                    f"Health check '{check.id}' failed after {check.retries} attempts"
                )

    def health_weight_ratio(self) -> float:
        """Proportion of subtask weight to allocate to health checks.

        Default is 0.25 (health ~20% of total). Tasks can override to better
        reflect real runtime characteristics.
        """
        return 0.25

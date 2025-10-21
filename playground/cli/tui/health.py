"""
Structured health check progress tracking.
"""

import time
from typing import List, Tuple, Callable, Optional
from dataclasses import dataclass


@dataclass
class HealthCheck:
    """Represents a single health check."""

    id: str
    description: str
    status: str = "pending"  # pending, running, passed, failed
    attempt: int = 0
    error: Optional[str] = None


class HealthCheckProgress:
    """Progress tracker for structured health checks."""

    def __init__(self, parent_progress, subtask_id: str):
        """
        Initialize health check progress.

        Args:
            parent_progress: Parent TaskProgress instance
            subtask_id: ID of the subtask this health check belongs to
        """
        self.parent_progress = parent_progress
        self.subtask_id = subtask_id
        self.checks: List[HealthCheck] = []
        self.current_check: Optional[str] = None

    def define_checks(self, checks: List[Tuple[str, str]]):
        """
        Define health checks to run.

        Args:
            checks: List of (id, description) pairs
        """
        self.checks = [HealthCheck(id=cid, description=desc) for cid, desc in checks]
        self._update_display()

    def run_check_with_retry(
        self,
        check_id: str,
        check_fn: Callable[[], None],
        retries: int = 3,
        delays: Optional[List[int]] = None,
    ) -> bool:
        """
        Run a check with automatic retry and progress updates.

        Args:
            check_id: ID of the check to run
            check_fn: Function to execute (raises exception on failure)
            retries: Number of retry attempts
            delays: Delays between retries

        Returns:
            True if check passed, False if failed after retries
        """
        if delays is None:
            delays = [5, 10, 20]

        check = self._get_check(check_id)
        if not check:
            return False

        for attempt in range(retries):
            check.attempt = attempt + 1
            check.status = "running"
            self.current_check = check_id
            self._update_display()

            try:
                check_fn()
                check.status = "passed"
                self.current_check = None
                self._update_display()
                return True

            except Exception as e:
                check.error = str(e)
                check.status = "failed"

                if attempt < retries - 1:
                    # Truncate long error messages more intelligently
                    error_msg = str(e)
                    if len(error_msg) > 80:
                        # Try to keep the most important part of the error
                        if ":" in error_msg:
                            # Keep the part after the last colon (usually the specific error)
                            parts = error_msg.rsplit(":", 1)
                            if len(parts) > 1 and len(parts[1]) < 60:
                                error_msg = "..." + parts[1].strip()
                            else:
                                error_msg = error_msg[:77] + "..."
                        else:
                            error_msg = error_msg[:77] + "..."

                    self._update_display(f"Health retry in {delays[attempt]}s ({check_id}): {error_msg}")
                    time.sleep(delays[attempt])
                else:
                    self.current_check = None
                    self._update_display()
                    return False

        return False

    def _get_check(self, check_id: str) -> Optional[HealthCheck]:
        """Get a check by ID."""
        for check in self.checks:
            if check.id == check_id:
                return check
        return None

    def _update_display(self, message: Optional[str] = None):
        """Update the parent progress with health check status."""
        # Always calculate these values for progress calculation
        passed = sum(1 for c in self.checks if c.status == "passed")
        total = len(self.checks)

        if message:
            status_msg = message
        else:
            if self.current_check:
                check = self._get_check(self.current_check)
                if check:
                    status_msg = f"Health: {check.description}"
                    if check.attempt > 1:
                        status_msg += f" (attempt {check.attempt})"
            else:
                status_msg = f"Health checks: {passed}/{total} passed"

        # Update parent progress message with proper health check progress calculation
        if self.parent_progress and hasattr(self.parent_progress, "callback"):
            # Calculate base progress from completed subtasks
            base_progress = self.parent_progress.completed_weight

            # Find the health subtask weight
            health_weight = 0
            for sid, desc, weight in self.parent_progress.subtasks:
                if sid == self.subtask_id:
                    health_weight = weight / 100 * self.parent_progress.total_weight
                    break

            # Calculate health check completion percentage
            if total > 0:
                health_completion_pct = passed / total
            else:
                health_completion_pct = 1.0

            # Add health progress to base progress
            total_progress = base_progress + (health_weight * health_completion_pct)
            current_pct = int((total_progress / self.parent_progress.total_weight) * 100)

            self.parent_progress.callback(self.parent_progress.task_name, current_pct, status_msg)

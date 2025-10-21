"""
Task progress tracking with subtask support.
"""

import threading
from typing import List, Tuple, Callable, Optional


class SubtaskContext:
    """Context manager for subtask execution."""

    def __init__(self, progress: "TaskProgress", subtask_id: str):
        self.progress = progress
        self.subtask_id = subtask_id
        self.started = False

    def __enter__(self):
        self.progress.start_subtask(self.subtask_id)
        self.started = True
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        if exc_type is None:
            self.progress.complete_subtask(self.subtask_id)
        else:
            self.progress.fail_subtask(self.subtask_id, str(exc_val))
        # Don't suppress exceptions
        return False


class TaskProgress:
    """Progress reporter based on subtask completion."""

    def __init__(self, task_name: str, callback: Callable[[str, int, str], None]):
        """
        Initialize progress tracker.

        Args:
            task_name: Name of the task
            callback: Function to call with (task_name, percentage, message)
        """
        self.task_name = task_name
        self.callback = callback
        self.subtasks: List[Tuple[str, str, float]] = []
        self.subtask_status: dict = {}
        self.total_weight = 0.0
        self.completed_weight = 0.0
        self.current_subtask: Optional[str] = None
        self._lock = threading.Lock()

    def define_subtasks(self, subtasks: List[Tuple[str, str, float]]):
        """
        Define subtasks for this task.

        Args:
            subtasks: List of (id, description, weight) tuples
        """
        with self._lock:
            self.subtasks = subtasks
            self.total_weight = sum(weight for _, _, weight in subtasks)
            for sid, _, _ in subtasks:
                self.subtask_status[sid] = "pending"
            self._update_progress()

    def subtask(self, subtask_id: str) -> SubtaskContext:
        """Create a context manager for a subtask."""
        return SubtaskContext(self, subtask_id)

    def start_subtask(self, subtask_id: str):
        """Mark a subtask as in-progress."""
        with self._lock:
            self.subtask_status[subtask_id] = "running"
            self.current_subtask = subtask_id
            # Add a small portion of the weight when starting to show activity
            for sid, _, weight in self.subtasks:
                if sid == subtask_id:
                    self.completed_weight += weight * 0.1
                    break
            self._update_progress()

    def complete_subtask(self, subtask_id: str):
        """Mark a subtask as complete."""
        with self._lock:
            self.subtask_status[subtask_id] = "completed"

            # Find and add remaining weight (we already added half when starting)
            for sid, _, weight in self.subtasks:
                if sid == subtask_id:
                    # Add the remaining 90% of the subtask weight
                    self.completed_weight += weight * 0.9
                    break

            if self.current_subtask == subtask_id:
                self.current_subtask = None
            self._update_progress()

    def fail_subtask(self, subtask_id: str, error: str):
        """Mark a subtask as failed."""
        with self._lock:
            self.subtask_status[subtask_id] = "failed"
            if self.current_subtask == subtask_id:
                self.current_subtask = None
            # Include subtask identifier in error for clearer context
            self._update_progress(error=f"{subtask_id}: {error}")

    def _update_progress(self, error: Optional[str] = None):
        """Calculate and report progress."""
        if self.total_weight == 0:
            percentage = 0
        else:
            percentage = int((self.completed_weight / self.total_weight) * 100)

        # Build message
        if error:
            message = f"Failed: {error}"
        elif self.current_subtask:
            # Find description for current subtask
            for sid, desc, _ in self.subtasks:
                if sid == self.current_subtask:
                    message = desc
                    break
            else:
                message = f"Running {self.current_subtask}"
        else:
            completed = sum(1 for s in self.subtask_status.values() if s == "completed")
            total = len(self.subtasks)
            if total > 0:
                message = f"{completed}/{total} subtasks complete"
            else:
                message = "Initializing..."

        # Avoid emitting duplicate consecutive messages to reduce flicker
        try:
            if getattr(self, "_last_message", None) == (percentage, message):
                return
            self._last_message = (percentage, message)
        except Exception:
            pass

        self.callback(self.task_name, percentage, message)

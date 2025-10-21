"""
Test task implementation for verifying the new class-based TUI system.
"""

from cli.setup.task_base import BaseTask
from typing import Dict, List
import time


class TestTask(BaseTask):
    """A simple test task to verify the new class-based architecture."""

    @classmethod
    def provides(cls) -> Dict[str, type]:
        """Define what this task provides."""
        return {
            "test_result": str,
            "test_status": bool,
        }

    @classmethod
    def requires(cls) -> List[str]:
        """Test task has no dependencies."""
        return []

    @classmethod
    def config_keys(cls) -> List[str]:
        """Config keys needed by this task."""
        return ["test_task"]

    def setup(self):
        """Register subtasks and health checks."""
        self.test_config = self.cfg.get("test_task", {})

        # Register subtasks
        self.add_subtask("init", "Initializing test", 20, self.initialize)
        self.add_subtask("process", "Processing test data", 50, self.process)
        self.add_subtask("finalize", "Finalizing test", 20, self.finalize)

        # Register health checks
        self.add_healthcheck("verify_result", "Verifying test result", self.verify_result)

    def initialize(self):
        """Subtask: Initialize the test."""
        self.log.write("Test task initialization...\n")
        time.sleep(0.5)  # Simulate work
        self.log.write("Test task initialized successfully\n")

    def process(self):
        """Subtask: Process test data."""
        self.log.write("Processing test data...\n")
        time.sleep(1.0)  # Simulate work
        self.test_data = "test_processed_data"
        self.log.write(f"Processed: {self.test_data}\n")

    def finalize(self):
        """Subtask: Finalize the test."""
        self.log.write("Finalizing test...\n")
        time.sleep(0.5)  # Simulate work

        # Populate provides
        self._provides.update(
            {
                "test_result": self.test_data,
                "test_status": True,
            }
        )

        self.log.write("Test task completed successfully\n")

    def verify_result(self):
        """Health check: Verify the result."""
        if not self._provides.get("test_result"):
            raise RuntimeError("Test result is missing")
        if not self._provides.get("test_status"):
            raise RuntimeError("Test status is not set")

        self.log.write("Test result verification passed\n")

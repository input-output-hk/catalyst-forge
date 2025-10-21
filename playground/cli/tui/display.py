"""
Rich-based TUI display for task execution.
"""

from rich.console import Console
from rich.progress import Progress, SpinnerColumn, BarColumn, TextColumn, TimeElapsedColumn
from rich.table import Column
from typing import Callable, Optional
from typing import Dict, Any
import threading
import time


class TUIDisplay:
    """Manages the Rich TUI display for task execution."""

    def __init__(self):
        self.console = Console()
        self.progress = Progress(
            SpinnerColumn(),
            TextColumn(
                "[progress.description]{task.description}",
                justify="left",
                table_column=Column(width=70),
            ),
            BarColumn(),
            TextColumn("{task.percentage:>3.0f}%"),
            TimeElapsedColumn(),
            console=self.console,
            expand=True,
        )
        self.task_ids: Dict[str, Any] = {}
        self.task_states: Dict[str, str] = {}
        self.task_messages: Dict[str, str] = {}
        self._lock = threading.RLock()
        self._last_update_ns: Dict[str, int] = {}
        self._min_update_interval_ns = int(1e8)  # ~100ms throttle
        self.health_messages: Dict[str, str] = {}
        self._last_payload: Dict[str, tuple[int, str]] = {}
        self._poller: Optional[threading.Thread] = None
        self._stop = threading.Event()

    def add_task(self, task_name: str) -> None:
        """Add a new task to the display."""
        with self._lock:
            task_id = self.progress.add_task(
                f"[cyan]{task_name}[/cyan] - Waiting...", total=100, completed=0, start=False
            )
            self.task_ids[task_name] = task_id
            self.task_states[task_name] = "pending"
            self.task_messages[task_name] = "Waiting..."

    def update_task(self, task_name: str, percentage: int, message: str) -> None:
        """Update task progress.

        Args:
            task_name: Name of the task
            percentage: Progress percentage (0-100, or -1 for failure)
            message: Status message to display
        """
        now_ns = time.time_ns()
        with self._lock:
            if task_name not in self.task_ids:
                return

            # Dedupe and throttle
            prev_payload = self._last_payload.get(task_name)
            if prev_payload and prev_payload == (percentage, message):
                return
            same_message = prev_payload is not None and prev_payload[1] == message

            # Throttle only when the message hasn't changed (avoid hiding fast state transitions)
            last = self._last_update_ns.get(task_name, 0)
            if (
                same_message
                and now_ns - last < self._min_update_interval_ns
                and percentage not in (0, 100, -1)
            ):
                return
            self._last_update_ns[task_name] = now_ns

            task_id = self.task_ids[task_name]

            # Drop duplicate payloads for this task (percentage+message)
            self._last_payload[task_name] = (percentage, message)

            # Truncate message to prevent progress bar shrinking
            # Leave room for task name and status indicators
            max_msg_len = 60
            if len(message) > max_msg_len:
                # Try to intelligently truncate
                if ":" in message and message.count(":") == 1:
                    # Keep the most informative part after colon
                    parts = message.split(":", 1)
                    if len(parts[1]) <= max_msg_len:
                        truncated_msg = parts[1].strip()
                    else:
                        truncated_msg = parts[1][:max_msg_len-3].strip() + "..."
                else:
                    truncated_msg = message[:max_msg_len-3] + "..."
            else:
                truncated_msg = message

            # On first real update, start the progress task
            if not self.progress.tasks[self.task_ids[task_name]].started:
                self.progress.start_task(task_id)

            # Update state based on percentage
            if percentage >= 100:
                self.task_states[task_name] = "completed"
                description = f"[green]✓[/green] [cyan]{task_name}[/cyan] - Complete"
            elif percentage < 0:
                self.task_states[task_name] = "failed"
                description = f"[red]✗[/red] [cyan]{task_name}[/cyan] - {truncated_msg}"
            else:
                self.task_states[task_name] = "running"
                description = f"[cyan]{task_name}[/cyan] - {truncated_msg}"

            self.task_messages[task_name] = message
            if message.startswith("Health"):
                self.health_messages[task_name] = message

            self.progress.update(
                task_id, completed=max(0, min(100, percentage)), description=description
            )

    def print_health_panel(self, task_names: list[str]):
        """Print a simple health checks panel for provided tasks.

        This provides periodic visibility (between levels) without complex live layout.
        """
        lines = []
        for name in task_names:
            msg = self.health_messages.get(name)
            if msg:
                lines.append(f"[dim]{name}[/dim]: {msg}")
        if lines:
            self.console.print("\n[bold]Health Checks[/bold]")
            for line in lines:
                self.console.print(line)

    def print_header(self):
        """Print setup header."""
        self.console.print("\n[bold cyan]🚀 Playground Setup[/bold cyan]\n")

    def print_summary(self):
        """Print final summary."""
        completed = sum(1 for s in self.task_states.values() if s == "completed")
        failed = sum(1 for s in self.task_states.values() if s == "failed")

        if failed > 0:
            self.console.print(f"\n[red]✗ Setup failed: {failed} task(s) failed[/red]")
        else:
            self.console.print(f"\n[green]✓ All {completed} tasks completed successfully![/green]")

    def __enter__(self):
        """Start the progress display."""
        self.print_header()
        self.progress.__enter__()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Stop the progress display."""
        self.progress.__exit__(exc_type, exc_val, exc_tb)
        self.print_summary()

    def start_snapshot_poller(self, get_snapshots: Callable[[], Dict[str, Any]], interval: float = 0.1):
        """Start a background thread to poll snapshots and render updates."""
        if self._poller is not None:
            return

        def loop():
            while not self._stop.is_set():
                try:
                    snaps = get_snapshots()
                    # Add any new tasks outside of the update loop to avoid churn
                    for name in list(snaps.keys()):
                        with self._lock:
                            missing = name not in self.task_ids
                        if missing:
                            self.add_task(name)

                    with self._lock:
                        for name, snap in snaps.items():
                            pct = snap.percentage
                            msg = snap.message
                            self.update_task(name, pct, msg)
                    time.sleep(max(0.05, interval))
                except Exception:
                    time.sleep(interval)

        self._stop.clear()
        self._poller = threading.Thread(target=loop, daemon=True)
        self._poller.start()

    def stop_snapshot_poller(self):
        if self._poller is None:
            return
        self._stop.set()
        self._poller.join(timeout=1.0)
        self._poller = None

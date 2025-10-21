# Implementation Tasks for TUI Progress System

This document outlines the step-by-step implementation of the class-based TUI progress system described in `UI_PRD.md`.

## Phase 1: Core Infrastructure

### 1.1 Add Rich Dependency
**File:** `pyproject.toml`
```python
dependencies = [
    "typer>=0.12.3",
    "pydantic>=2.6.0",
    "pyyaml>=6.0.0",
    "kubernetes>=28.0.0",
    "psycopg[binary]>=3.1.0",
    "requests>=2.31.0",
    "networkx>=3.0",
    "rich>=13.8.0",  # ADD THIS
]
```

### 1.2 Create Base Task Class
**File:** `cli/setup/task_base.py`
```python
"""
Base class for all setup tasks with automatic progress and error handling.
"""
from abc import ABC, abstractmethod
from typing import Any, Dict, List, Callable, Optional
from dataclasses import dataclass
import time
import threading


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
    retry_delays: List[int] = None


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
        self._subtasks.append(SubtaskDef(
            id=id,
            description=description,
            weight=weight,
            func=func
        ))
    
    def add_healthcheck(self, id: str, description: str, func: Callable, 
                       retries: int = 3, delays: List[int] = None):
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
        self._health_checks.append(HealthCheckDef(
            id=id,
            description=description,
            func=func,
            retries=retries,
            retry_delays=delays[:retries]
        ))
    
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
        health_weight = subtask_weight * 0.25 if self._health_checks else 0
        total_weight = subtask_weight + health_weight
        
        # Define all subtasks for progress
        all_subtasks = [
            (s.id, s.description, s.weight / total_weight * 100)
            for s in self._subtasks
        ]
        if self._health_checks:
            all_subtasks.append(
                ("health", "Running health checks", health_weight / total_weight * 100)
            )
        
        progress.define_subtasks(all_subtasks)
        
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
        
        # Return provides
        return self._provides
    
    def _run_health_checks(self, progress):
        """Run all registered health checks with retry logic."""
        from cli.tui.health import HealthCheckProgress
        
        # Create health check progress tracker
        health_progress = HealthCheckProgress(progress, "health")
        health_progress.define_checks([
            (hc.id, hc.description) for hc in self._health_checks
        ])
        
        for check in self._health_checks:
            success = health_progress.run_check_with_retry(
                check.id,
                check.func,
                retries=check.retries,
                delays=check.retry_delays
            )
            if not success:
                raise RuntimeError(f"Health check '{check.id}' failed after {check.retries} attempts")
```

### 1.3 Create Progress Module
**File:** `cli/tui/progress.py`
```python
"""
Task progress tracking with subtask support.
"""
import threading
from typing import List, Tuple, Callable, Optional
from contextlib import contextmanager


class SubtaskContext:
    """Context manager for subtask execution."""
    
    def __init__(self, progress: 'TaskProgress', subtask_id: str):
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
            self._update_progress()
    
    def complete_subtask(self, subtask_id: str):
        """Mark a subtask as complete."""
        with self._lock:
            self.subtask_status[subtask_id] = "completed"
            
            # Find and add weight
            for sid, _, weight in self.subtasks:
                if sid == subtask_id:
                    self.completed_weight += weight
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
            self._update_progress(error=error)
    
    def _update_progress(self, error: str = None):
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
        
        self.callback(self.task_name, percentage, message)
```

### 1.4 Create Health Check Module
**File:** `cli/tui/health.py`
```python
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
        self.checks = [
            HealthCheck(id=cid, description=desc)
            for cid, desc in checks
        ]
        self._update_display()
    
    def run_check_with_retry(self, check_id: str, check_fn: Callable[[], None],
                           retries: int = 3, delays: List[int] = None) -> bool:
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
                    self._update_display(f"Retrying in {delays[attempt]}s: {e}")
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
    
    def _update_display(self, message: str = None):
        """Update the parent progress with health check status."""
        if message:
            status_msg = message
        else:
            passed = sum(1 for c in self.checks if c.status == "passed")
            total = len(self.checks)
            
            if self.current_check:
                check = self._get_check(self.current_check)
                if check:
                    status_msg = f"Health: {check.description}"
                    if check.attempt > 1:
                        status_msg += f" (attempt {check.attempt})"
            else:
                status_msg = f"Health checks: {passed}/{total} passed"
        
        # Update parent progress message
        if self.parent_progress and hasattr(self.parent_progress, 'callback'):
            current_pct = int((self.parent_progress.completed_weight / self.parent_progress.total_weight) * 100)
            self.parent_progress.callback(
                self.parent_progress.task_name,
                current_pct,
                status_msg
            )
```

### 1.5 Create TUI Display Module
**File:** `cli/tui/display.py`
```python
"""
Rich-based TUI display for task execution.
"""
from rich.console import Console
from rich.progress import Progress, SpinnerColumn, BarColumn, TextColumn, TimeElapsedColumn
from rich.panel import Panel
from rich.table import Table
from rich.layout import Layout
from typing import Dict, List
import threading


class TUIDisplay:
    """Manages the Rich TUI display for task execution."""
    
    def __init__(self):
        self.console = Console()
        self.progress = Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            BarColumn(),
            TextColumn("{task.percentage:>3.0f}%"),
            TimeElapsedColumn(),
            console=self.console,
            expand=True,
        )
        self.task_ids: Dict[str, int] = {}
        self.task_states: Dict[str, str] = {}
        self.task_messages: Dict[str, str] = {}
        self._lock = threading.Lock()
    
    def add_task(self, task_name: str) -> None:
        """Add a new task to the display."""
        with self._lock:
            task_id = self.progress.add_task(
                f"[cyan]{task_name}[/cyan] - Waiting...",
                total=100,
                completed=0
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
        with self._lock:
            if task_name not in self.task_ids:
                return
                
            task_id = self.task_ids[task_name]
            
            # Update state based on percentage
            if percentage >= 100:
                self.task_states[task_name] = "completed"
                description = f"[green]✓[/green] [cyan]{task_name}[/cyan] - Complete"
            elif percentage < 0:
                self.task_states[task_name] = "failed"
                description = f"[red]✗[/red] [cyan]{task_name}[/cyan] - {message}"
            else:
                self.task_states[task_name] = "running"
                description = f"[cyan]{task_name}[/cyan] - {message}"
            
            self.task_messages[task_name] = message
            
            self.progress.update(
                task_id,
                completed=max(0, min(100, percentage)),
                description=description
            )
    
    def print_header(self):
        """Print setup header."""
        self.console.print("\n[bold cyan]🚀 Playground Setup[/bold cyan]\n")
    
    def print_summary(self):
        """Print final summary."""
        completed = sum(1 for s in self.task_states.values() if s == "completed")
        failed = sum(1 for s in self.task_states.values() if s == "failed")
        total = len(self.task_states)
        
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
```

## Phase 2: Update Core Integration

### 2.1 Update Task Registration
**File:** `cli/setup/core.py` (modifications)
```python
from typing import Type, Dict, Any, List
from cli.setup.task_base import BaseTask, TaskError
from cli.tui.progress import TaskProgress

# Change from function registry to class registry
_task_classes: Dict[str, Dict] = {}

def register_task(name: str, task_class: Type[BaseTask]):
    """Register a task class.
    
    Args:
        name: Unique task name
        task_class: Class inheriting from BaseTask
    """
    _task_classes[name] = {
        "class": task_class,
        "requires": task_class.requires(),
        "config_keys": task_class.config_keys(),
        "provides": task_class.provides(),
    }

def load_tasks():
    """Auto-discover and register all task classes."""
    from cli.setup.tasks.k3d import K3dTask
    from cli.setup.tasks.cert_manager import CertManagerTask
    from cli.setup.tasks.registry import RegistryTask
    from cli.setup.tasks.postgres import PostgresTask
    # ... import all task classes
    
    register_task("k3d", K3dTask)
    register_task("cert_manager", CertManagerTask)
    register_task("registry", RegistryTask)
    register_task("postgres", PostgresTask)
    # ... register all tasks

def run_task_sync(name: str, cfg: Any, log_dir: Path, progress_callback):
    """Execute a task using its class."""
    task_info = _task_classes[name]
    
    # Filter config to only what task needs
    filtered_cfg = _filter_config(cfg, task_info["config_keys"])
    
    # Get current context (read-only for task)
    ctx = dict(_context)
    
    # Create file logger
    from cli.setup.logging import task_logging_context
    
    with task_logging_context(name, log_dir) as log:
        # Instantiate task
        task_class = task_info["class"]
        task = task_class(name, ctx, filtered_cfg, log)
        
        # Create progress wrapper
        progress = TaskProgress(name, progress_callback)
        
        # Run task (base class handles everything)
        try:
            result = task.run(progress)
            
            # Update global context with namespaced results
            namespaced_result = {f"{name}.{k}": v for k, v in result.items()}
            with _context_lock:
                _context.update(namespaced_result)
            
            return result
            
        except TaskError as e:
            # Rich error context available
            logger.error(f"Task '{e.task_name}' failed at subtask '{e.subtask}': {e.original_error}")
            raise
        except Exception as e:
            # Unexpected error
            logger.error(f"Task '{name}' failed unexpectedly: {e}")
            raise

def run_all(cfg: Any, only: Optional[List[str]] = None, dry_run: bool = False):
    """Run all tasks with TUI progress display."""
    from cli.tui.display import TUIDisplay
    
    logger = get_logger("console")
    
    # Load all task classes
    load_tasks()
    
    # Compute execution plan
    levels = compute_execution_levels(_task_classes, only)
    
    # Show execution plan
    total_tasks = sum(len(level) for level in levels)
    logger.info(f"Execution plan: {len(levels)} levels, {total_tasks} tasks")
    for i, level in enumerate(levels, 1):
        logger.info(f"  Level {i}: {', '.join(level)}")
    
    if dry_run:
        logger.info("Dry run complete (no tasks executed)")
        return
    
    # Setup directories
    playground_dir = Path(__file__).resolve().parent.parent.parent
    log_dir = playground_dir / "logs" / "latest"
    if log_dir.exists():
        shutil.rmtree(log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)
    
    # Execute with TUI
    with TUIDisplay() as display:
        # Add all tasks to display
        for level_tasks in levels:
            for task_name in level_tasks:
                display.add_task(task_name)
        
        # Execute levels
        for level_num, level_tasks in enumerate(levels, 1):
            with ThreadPoolExecutor(max_workers=len(level_tasks)) as executor:
                # Create progress callback for each task
                def make_callback(name):
                    return lambda n, p, m: display.update_task(n, p, m)
                
                futures = {}
                for name in level_tasks:
                    # Start task
                    display.update_task(name, 0, "Starting...")
                    
                    future = executor.submit(
                        run_task_sync,
                        name, cfg, log_dir,
                        make_callback(name)
                    )
                    futures[future] = name
                
                # Wait for completion
                failed = False
                for future in as_completed(futures):
                    name = futures[future]
                    try:
                        result = future.result()
                        display.update_task(name, 100, "Complete")
                    except TaskError as e:
                        display.update_task(name, -1, f"Failed at {e.subtask}")
                        failed = True
                        logger.error(f"Task failed: {e}")
                    except Exception as e:
                        display.update_task(name, -1, f"Failed: {e}")
                        failed = True
                        logger.error(f"Task failed: {e}")
                
                if failed:
                    remaining_levels = levels[level_num:]
                    if remaining_levels:
                        remaining_tasks = [task for level in remaining_levels for task in level]
                        logger.error(f"Stopping execution due to failures at level {level_num}")
                        logger.error(f"Skipping: {', '.join(remaining_tasks)}")
                    raise RuntimeError("One or more tasks failed")
    
    logger.info("All tasks completed successfully")
```

## Phase 3: Task Migration

### 3.1 K3d Task Migration
**File:** `cli/setup/tasks/k3d/__init__.py`
```python
"""K3d cluster setup task."""
from .task import K3dTask

__all__ = ["K3dTask"]
```

**File:** `cli/setup/tasks/k3d/task.py`
```python
"""K3d cluster setup task implementation."""
from cli.setup.task_base import BaseTask
from pathlib import Path
import subprocess
import os
from typing import Dict, Any, List

from .config import K3dConfig
from .ops import (
    cluster_exists,
    create_cluster,
    delete_cluster,
    write_kubeconfig,
    emit_cluster_json,
    write_registries_yaml,
)
from cli.setup.tools.utils import get_mkcert_caroot
from cli.utils import require_cmd, get_repo_root


class K3dTask(BaseTask):
    """K3d cluster setup task."""
    
    @classmethod
    def provides(cls) -> Dict[str, type]:
        """Define what this task provides."""
        return {
            "cluster_name": str,
            "kubeconfig": str,
            "http_port": int,
            "https_port": int,
            "api_port": int,
            "ready": bool,
        }
    
    @classmethod
    def requires(cls) -> List[str]:
        """K3d has no dependencies on other tasks."""
        return []
    
    @classmethod
    def config_keys(cls) -> List[str]:
        """Config keys needed by this task."""
        return ["k3d", "registry_host"]
    
    def setup(self):
        """Register all subtasks and health checks."""
        # Parse config once
        self.k3d_config = K3dConfig.model_validate(self.cfg["k3d"])
        self.registry_host = self.cfg["registry_host"]
        
        # Setup paths
        playground_dir = Path(__file__).resolve().parent.parent.parent.parent
        self.repo_root = get_repo_root(playground_dir)
        
        # Register subtasks in execution order
        self.add_subtask("preflight", "Checking prerequisites", 5, 
                        self.check_prerequisites)
        self.add_subtask("certificates", "Preparing certificates", 10,
                        self.prepare_certificates)
        self.add_subtask("cluster", "Creating k3d cluster", 60,
                        self.create_cluster)
        self.add_subtask("kubeconfig", "Writing kubeconfig", 5,
                        self.write_kubeconfig)
        self.add_subtask("output", "Writing cluster info", 5,
                        self.write_output)
        
        # Register health checks (run automatically after subtasks)
        self.add_healthcheck("cluster_exists", "Verifying cluster exists",
                           self.verify_cluster_exists)
        self.add_healthcheck("api_responsive", "Checking API connectivity",
                           self.test_api_connection, retries=3)
        self.add_healthcheck("nodes_ready", "Waiting for nodes",
                           self.wait_for_nodes, retries=5, delays=[5,10,15,20,30])
        self.add_healthcheck("system_pods", "Verifying system pods",
                           self.verify_system_pods)
    
    def check_prerequisites(self):
        """Subtask: Check required commands are available."""
        for cmd in ("docker", "k3d", "mkcert"):
            try:
                require_cmd(cmd)
                self.log.write(f"✓ {cmd} is available\n")
            except Exception as e:
                self.log.write(f"✗ {cmd} is missing: {e}\n")
                raise RuntimeError(f"Required command '{cmd}' not found")
    
    def prepare_certificates(self):
        """Subtask: Prepare mkcert CA certificates."""
        self.log.write("Preparing mkcert CA certificates...\n")
        
        caroot = get_mkcert_caroot()
        self.ca_file = caroot / "rootCA.pem"
        
        if not self.ca_file.exists():
            self.log.write(f"Warning: mkcert CA file not found at {self.ca_file}\n")
            self.log.write("Running 'mkcert -install' to create CA...\n")
            result = subprocess.run(["mkcert", "-install"], capture_output=True, text=True)
            self.log.write(f"{result.stdout}\n")
            if result.returncode != 0:
                self.log.write(f"Error: {result.stderr}\n")
                raise RuntimeError("Failed to install mkcert CA")
        
        # Prepare registries.yaml
        tmpdir = (self.repo_root.parent / "playground" / ".certs").resolve()
        tmpdir.mkdir(parents=True, exist_ok=True)
        
        self.log.write(f"Writing registries configuration for {self.registry_host}...\n")
        self.registries_yaml = write_registries_yaml(
            tmpdir=tmpdir,
            registry_host=self.registry_host,
            ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt"),
        ).resolve()
    
    def create_cluster(self):
        """Subtask: Create the k3d cluster."""
        volume_mounts = [
            f"{self.ca_file}:/etc/ssl/certs/mkcert-rootCA.crt@server:*;agent:*",
            f"{self.registries_yaml}:/etc/rancher/k3s/registries.yaml@server:*;agent:*",
        ]
        
        if cluster_exists(self.k3d_config.cluster_name):
            self.log.write(f"Cluster '{self.k3d_config.cluster_name}' already exists.\n")
            if self.k3d_config.force_recreate:
                self.log.write("Force recreate enabled, deleting existing cluster...\n")
                delete_cluster(self.k3d_config.cluster_name, log=self.log)
                self.log.write(f"Creating new cluster '{self.k3d_config.cluster_name}'...\n")
                create_cluster(
                    self.k3d_config.cluster_name,
                    self.k3d_config.servers,
                    self.k3d_config.agents,
                    self.k3d_config.http_port,
                    self.k3d_config.https_port,
                    self.k3d_config.api_port,
                    extra_volumes=volume_mounts,
                    log=self.log,
                )
            else:
                self.log.write("Reusing existing cluster.\n")
        else:
            self.log.write(f"Creating new cluster '{self.k3d_config.cluster_name}'...\n")
            create_cluster(
                self.k3d_config.cluster_name,
                self.k3d_config.servers,
                self.k3d_config.agents,
                self.k3d_config.http_port,
                self.k3d_config.https_port,
                self.k3d_config.api_port,
                extra_volumes=volume_mounts,
                log=self.log,
            )
    
    def write_kubeconfig(self):
        """Subtask: Write kubeconfig file."""
        self.kubeconfig_path = Path(self.k3d_config.kubeconfig_out).expanduser()
        if not self.kubeconfig_path.is_absolute():
            self.kubeconfig_path = (self.repo_root / self.kubeconfig_path).resolve()
        
        self.log.write(f"Writing kubeconfig to {self.kubeconfig_path}...\n")
        write_kubeconfig(
            self.k3d_config.cluster_name,
            self.kubeconfig_path,
            assume_yes=self.k3d_config.assume_yes
        )
        
        # Set environment variable
        os.environ["KUBECONFIG"] = str(self.kubeconfig_path)
        self.log.write(f"Set KUBECONFIG={self.kubeconfig_path}\n")
        
        # Store in provides
        self._provides["kubeconfig"] = str(self.kubeconfig_path)
    
    def write_output(self):
        """Subtask: Write cluster summary."""
        output_json_path = Path(self.k3d_config.output_json).expanduser()
        if not output_json_path.is_absolute():
            output_json_path = (self.repo_root / output_json_path).resolve()
        
        self.log.write(f"Writing cluster info to {output_json_path}...\n")
        emit_cluster_json(
            path=output_json_path,
            name=self.k3d_config.cluster_name,
            servers=self.k3d_config.servers,
            agents=self.k3d_config.agents,
            http_port=self.k3d_config.http_port,
            https_port=self.k3d_config.https_port,
            kubeconfig=self.kubeconfig_path,
        )
        
        # Get actual API port if it was auto-assigned
        api_port = self.k3d_config.api_port
        if api_port == 0:
            # For now, keep as 0 to indicate auto-assigned
            api_port = 0
        
        # Populate all provides values
        self._provides.update({
            "cluster_name": self.k3d_config.cluster_name,
            "http_port": self.k3d_config.http_port,
            "https_port": self.k3d_config.https_port,
            "api_port": api_port,
            "ready": True,
        })
        
        self.log.write(f"\n✅ k3d cluster '{self.k3d_config.cluster_name}' is ready!\n")
    
    # Health check methods (called automatically with retry logic)
    def verify_cluster_exists(self):
        """Health check: Verify cluster exists."""
        if not cluster_exists(self.k3d_config.cluster_name):
            raise RuntimeError(f"Cluster '{self.k3d_config.cluster_name}' not found")
    
    def test_api_connection(self):
        """Health check: Test kubernetes API connectivity."""
        from kubernetes import client as k8s_client, config as k8s_config
        
        try:
            k8s_config.load_kube_config(config_file=str(self.kubeconfig_path))
            api = k8s_client.CoreV1Api()
            api.list_namespace(limit=1)
        except Exception as e:
            raise RuntimeError(f"API connection failed: {e}")
    
    def wait_for_nodes(self):
        """Health check: Wait for all nodes to be ready."""
        from kubernetes import client as k8s_client, config as k8s_config
        
        k8s_config.load_kube_config(config_file=str(self.kubeconfig_path))
        api = k8s_client.CoreV1Api()
        
        nodes = api.list_node().items
        if not nodes:
            raise RuntimeError("No nodes found in cluster")
        
        not_ready = []
        for node in nodes:
            conditions = node.status.conditions or []
            is_ready = any(
                c.type == "Ready" and c.status == "True"
                for c in conditions
            )
            if not is_ready:
                not_ready.append(node.metadata.name)
        
        if not_ready:
            raise RuntimeError(f"Nodes not ready: {', '.join(not_ready)}")
    
    def verify_system_pods(self):
        """Health check: Verify system pods are running."""
        from kubernetes import client as k8s_client, config as k8s_config
        
        k8s_config.load_kube_config(config_file=str(self.kubeconfig_path))
        api = k8s_client.CoreV1Api()
        
        # Check kube-system pods
        pods = api.list_namespaced_pod("kube-system").items
        not_running = [
            pod.metadata.name for pod in pods
            if pod.status.phase != "Running"
        ]
        
        if not_running:
            raise RuntimeError(f"System pods not running: {', '.join(not_running[:5])}")
```

### 3.2 Task Migration Template
For other tasks, follow this pattern:

```python
"""Task description."""
from cli.setup.task_base import BaseTask
from typing import Dict, Any, List


class ExampleTask(BaseTask):
    """Example task implementation."""
    
    @classmethod
    def provides(cls) -> Dict[str, type]:
        """Define what this task provides."""
        return {
            "result": str,
            "ready": bool,
        }
    
    @classmethod
    def requires(cls) -> List[str]:
        """Define required context keys from other tasks."""
        return ["k3d.ready", "k3d.kubeconfig"]
    
    @classmethod
    def config_keys(cls) -> List[str]:
        """Define required config keys."""
        return ["example_config"]
    
    def setup(self):
        """Register subtasks and health checks."""
        # Parse config
        self.config = self.cfg["example_config"]
        
        # Register subtasks
        self.add_subtask("init", "Initializing", 20, self.initialize)
        self.add_subtask("process", "Processing", 60, self.process)
        self.add_subtask("cleanup", "Cleaning up", 20, self.cleanup)
        
        # Register health checks
        self.add_healthcheck("verify", "Verifying result", self.verify_result)
    
    def initialize(self):
        """Subtask: Initialize resources."""
        self.log.write("Initializing...\n")
        # Implementation
    
    def process(self):
        """Subtask: Main processing."""
        self.log.write("Processing...\n")
        # Implementation
        self._provides["result"] = "done"
    
    def cleanup(self):
        """Subtask: Clean up resources."""
        self.log.write("Cleaning up...\n")
        # Implementation
        self._provides["ready"] = True
    
    def verify_result(self):
        """Health check: Verify the result."""
        if not self._provides.get("result"):
            raise RuntimeError("No result produced")
```

## Phase 4: Testing

### 4.1 Unit Tests
**File:** `tests/test_task_base.py`
```python
import pytest
from cli.setup.task_base import BaseTask, TaskError
from cli.tui.progress import TaskProgress


class TestTask(BaseTask):
    """Test task for unit testing."""
    
    @classmethod
    def provides(cls):
        return {"test_result": str}
    
    def setup(self):
        self.add_subtask("step1", "First step", 50, self.step1)
        self.add_subtask("step2", "Second step", 50, self.step2)
        self.add_healthcheck("check", "Test check", self.check)
    
    def step1(self):
        self.log.write("Step 1\n")
        self._provides["test_result"] = "step1"
    
    def step2(self):
        self.log.write("Step 2\n")
        self._provides["test_result"] = "done"
    
    def check(self):
        if self._provides.get("test_result") != "done":
            raise RuntimeError("Not done")


def test_task_execution(tmp_path):
    """Test that tasks execute subtasks in order."""
    log_file = tmp_path / "test.log"
    
    with open(log_file, "w") as log:
        task = TestTask("test", {}, {}, log)
        
        progress_updates = []
        def callback(name, pct, msg):
            progress_updates.append((name, pct, msg))
        
        progress = TaskProgress("test", callback)
        result = task.run(progress)
    
    assert result["test_result"] == "done"
    assert len(progress_updates) > 0
    
    # Check that progress went from 0 to 100
    percentages = [pct for _, pct, _ in progress_updates]
    assert min(percentages) >= 0
    assert max(percentages) == 100


def test_task_error_handling(tmp_path):
    """Test that task errors include context."""
    
    class FailingTask(BaseTask):
        @classmethod
        def provides(cls):
            return {}
        
        def setup(self):
            self.add_subtask("fail", "Failing step", 100, self.fail_step)
        
        def fail_step(self):
            raise ValueError("Intentional failure")
    
    log_file = tmp_path / "test.log"
    
    with open(log_file, "w") as log:
        task = FailingTask("failing", {}, {}, log)
        progress = TaskProgress("failing", lambda *args: None)
        
        with pytest.raises(TaskError) as exc_info:
            task.run(progress)
        
        assert exc_info.value.task_name == "failing"
        assert exc_info.value.subtask == "fail"
        assert "Intentional failure" in str(exc_info.value.original_error)
```

## Phase 5: Rollout Plan

### Week 1: Core Infrastructure
- [ ] Add Rich dependency
- [ ] Implement BaseTask class
- [ ] Implement Progress and Health modules
- [ ] Implement TUI Display
- [ ] Update core integration points
- [ ] Write unit tests for core modules

### Week 2: Task Migration
- [ ] Migrate k3d task
- [ ] Migrate cert_manager task
- [ ] Migrate registry task
- [ ] Migrate postgres task
- [ ] Migrate remaining infrastructure tasks
- [ ] Migrate service tasks (keycloak, argocd, temporal)

### Week 3: Testing & Polish
- [ ] Integration testing with full task suite
- [ ] Performance benchmarking
- [ ] Terminal compatibility testing
- [ ] Documentation updates
- [ ] Bug fixes and polish
- [ ] Final review and merge

## Success Criteria

1. All tasks successfully migrated to class-based architecture
2. TUI displays progress clearly without interleaving
3. Health checks show retry attempts and status
4. Errors include task and subtask context
5. No performance regression vs current implementation
6. Works in standard terminals and CI/CD environments

## Notes

- Single PR approach for atomic change
- Feature flag `--tui` for initial testing
- Preserve all file logging functionality
- Graceful degradation for non-TTY environments
# PRD: TUI-Based Progress System for Playground CLI Setup

## Executive Summary

Replace the current multi-threaded console output streaming in the Playground CLI `setup` command with a structured Terminal User Interface (TUI) that provides clear, organized progress tracking for parallel task execution. This will be achieved through a class-based task architecture that provides automatic progress management and error handling.

## Problem Statement

The current `setup` command suffers from:
- **Interleaved output** from parallel tasks making logs unreadable
- **No progress visibility** - users can't tell how far along tasks are
- **Opaque operations** - long-running commands provide no feedback
- **Unclear health checks** - retries and validations happen invisibly
- **Poor failure diagnosis** - hard to identify which step failed

## Solution Overview

Implement a Rich-based TUI progress system with:
1. **Class-based task architecture** - tasks inherit from `BaseTask` which handles all progress complexity
2. **Structured subtasks** - tasks register their steps with the base class
3. **Integrated health checks** - health checks are registered and run automatically
4. **Parallel progress bars** - one bar per task, updated in real-time
5. **Rich error context** - exceptions include task and subtask information

## Design Decisions

### Class-Based Architecture
- All tasks inherit from a `BaseTask` base class
- Base class handles progress tracking, error handling, and execution flow
- Tasks only implement their core business logic
- No backward compatibility - all tasks updated in single PR

### Automatic Progress Management
- Tasks register subtasks and health checks in `setup()` method
- Base class executes subtasks in order and manages progress
- Health checks run automatically after subtasks complete
- Progress calculated automatically from subtask weights

### Structured Error Handling
- Custom exception types provide rich context about failures
- Exceptions include task name, subtask name, and original error
- Framework has full visibility into what failed and where

## Detailed Design

### 1. Base Task Class

```python
# cli/setup/task_base.py
from abc import ABC, abstractmethod
from typing import Any, Dict, List, Tuple, Callable, Optional
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
    
    @abstractmethod
    def provides(self) -> Dict[str, type]:
        """Define what this task provides to the context."""
        pass
    
    def add_subtask(self, id: str, description: str, weight: float, func: Callable):
        """Register a subtask."""
        self._subtasks.append(SubtaskDef(
            id=id,
            description=description,
            weight=weight,
            func=func
        ))
    
    def add_healthcheck(self, id: str, description: str, func: Callable, 
                       retries: int = 3, delays: List[int] = None):
        """Register a health check."""
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
        """Execute all subtasks and health checks with progress tracking."""
        
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
                    self.log.write(f"Starting: {subtask.description}\n")
                    result = subtask.func()
                    subtask.result = result
                    
            except Exception as e:
                subtask.error = str(e)
                self.log.write(f"Failed: {subtask.description} - {e}\n")
                raise TaskError(self.name, subtask.id, e)
        
        # Run health checks if defined
        if self._health_checks:
            with progress.subtask("health"):
                self._run_health_checks(progress)
        
        # Return provides
        return self._provides
    
    def _run_health_checks(self, progress):
        """Run all registered health checks with retry logic."""
        # Implementation handles retries and progress updates
        pass
```

### 2. Task Implementation Example

**Before (Current Function-Based Implementation):**
```python
@task("k3d", provides={"cluster_name": str, "ready": bool})
def setup_k3d(ctx, cfg, log):
    log.write("Setting up k3d cluster...\n")
    
    # Preflight checks
    for cmd in ("docker", "k3d", "mkcert"):
        require_cmd(cmd)
        log.write(f"✓ {cmd} is available\n")
    
    # Prepare certificates
    log.write("Preparing mkcert CA certificates...\n")
    prepare_certificates()
    
    # Create cluster
    log.write(f"Creating cluster '{name}'...\n")
    create_cluster(name, servers, agents, http_port, https_port)
    
    # Write kubeconfig
    log.write(f"Writing kubeconfig to {path}...\n")
    write_kubeconfig(name, path)
    
    log.write("✅ k3d cluster is ready!\n")
    return {"cluster_name": name, "ready": True}
```

**After (New Class-Based Implementation):**
```python
class K3dTask(BaseTask):
    """K3d cluster setup task."""
    
    def provides(self) -> Dict[str, type]:
        """Define what this task provides."""
        return {
            "cluster_name": str,
            "kubeconfig": str,
            "http_port": int,
            "https_port": int,
            "ready": bool,
        }
    
    def setup(self):
        """Register all subtasks and health checks."""
        # Parse config once
        self.k3d_config = K3dConfig.model_validate(self.cfg["k3d"])
        self.registry_host = self.cfg["registry_host"]
        
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
        self.add_healthcheck("coredns", "Verifying CoreDNS",
                           self.verify_coredns)
    
    def check_prerequisites(self):
        """Subtask: Check required commands are available."""
        for cmd in ("docker", "k3d", "mkcert"):
            require_cmd(cmd)
            self.log.write(f"✓ {cmd} is available\n")
    
    def prepare_certificates(self):
        """Subtask: Prepare mkcert CA certificates."""
        caroot = get_mkcert_caroot()
        self.ca_file = caroot / "rootCA.pem"
        
        if not self.ca_file.exists():
            subprocess.run(["mkcert", "-install"], check=True)
        
        self.registries_yaml = write_registries_yaml(
            tmpdir=self.get_tmpdir(),
            registry_host=self.registry_host,
            ca_path=Path("/etc/ssl/certs/mkcert-rootCA.crt"),
        )
    
    def create_cluster(self):
        """Subtask: Create the k3d cluster."""
        if cluster_exists(self.k3d_config.cluster_name):
            if self.k3d_config.force_recreate:
                delete_cluster(self.k3d_config.cluster_name)
            else:
                return  # Reuse existing
        
        create_cluster(
            self.k3d_config.cluster_name,
            self.k3d_config.servers,
            self.k3d_config.agents,
            # ... other params
        )
    
    def write_kubeconfig(self):
        """Subtask: Write kubeconfig file."""
        self.kubeconfig_path = Path(self.k3d_config.kubeconfig_out).expanduser()
        write_kubeconfig(self.k3d_config.cluster_name, self.kubeconfig_path)
        self._provides["kubeconfig"] = str(self.kubeconfig_path)
    
    def write_output(self):
        """Subtask: Write cluster summary."""
        # Write JSON output
        # Populate all provides values
        self._provides.update({
            "cluster_name": self.k3d_config.cluster_name,
            "http_port": self.k3d_config.http_port,
            "https_port": self.k3d_config.https_port,
            "ready": True,
        })
    
    # Health check methods (called automatically with retry logic)
    def verify_cluster_exists(self):
        if not cluster_exists(self.k3d_config.cluster_name):
            raise RuntimeError(f"Cluster not found")
    
    def test_api_connection(self):
        # Test kubernetes API connectivity
        pass
    
    def wait_for_nodes(self):
        # Wait for all nodes to be ready
        pass
    
    def verify_coredns(self):
        # Verify CoreDNS is running
        pass
```

### 3. Task Registration and Execution

```python
# cli/setup/core.py
from typing import Type, Dict
from cli.setup.task_base import BaseTask, TaskError

_task_registry: Dict[str, Dict] = {}

def register_task(
    name: str,
    task_class: Type[BaseTask],
    requires: List[str] = None,
    config_keys: List[str] = None,
):
    """Register a task class."""
    _task_registry[name] = {
        "class": task_class,
        "requires": requires or [],
        "config_keys": config_keys or [],
        "provides": task_class.provides(),
    }

def run_task_sync(name: str, cfg: Any, log_dir: Path, progress_callback):
    """Execute a task using its class."""
    task_info = _task_registry[name]
    
    # Filter config to only what task needs
    filtered_cfg = _filter_config(cfg, task_info["config_keys"])
    
    # Get current context
    ctx = dict(_context)
    
    # Create file logger
    with task_logging_context(name, log_dir) as log:
        # Instantiate task
        task = task_info["class"](name, ctx, filtered_cfg, log)
        
        # Create progress wrapper
        progress = TaskProgress(name, progress_callback)
        
        # Run task (base class handles everything)
        try:
            result = task.run(progress)
            
            # Update global context
            namespaced_result = {f"{name}.{k}": v for k, v in result.items()}
            with _context_lock:
                _context.update(namespaced_result)
            
            return result
            
        except TaskError as e:
            # Rich error context available
            logger.error(f"Task '{e.task_name}' failed at '{e.subtask}': {e.original_error}")
            raise
```

### 4. User Interface Examples

**Current Output (Interleaved Mess):**
```
Setting up k3d cluster...
Setting up cert-manager...
✓ docker is available
Installing cert-manager CRDs...
✓ k3d is available
Waiting for cert-manager webhook...
Creating cluster 'forge'...
Error: webhook not ready
Retrying...
Cluster created successfully
Webhook ready!
Writing kubeconfig...
```

**New TUI Display:**
```
╭─────────────────── Playground Setup ───────────────────╮
│                                                         │
│ Level 1/3                                               │
│                                                         │
│ k3d              ████████████████████░░░░  75%         │
│                  Creating k3d cluster                   │
│                                                         │
│ Level 2/3                                               │
│                                                         │
│ cert-manager     ██████░░░░░░░░░░░░░░░░░  25%         │
│                  ⟳ Installing CRDs                      │
│                                                         │
│ registry         ████████████████████████ 100% ✓       │
│                                                         │
│ postgres         ░░░░░░░░░░░░░░░░░░░░░░░   0%         │
│                  Waiting...                             │
│                                                         │
├─────────────────────────────────────────────────────────┤
│ Health Checks: k3d                                      │
│ ✓ Verifying cluster exists                             │
│ ✓ Checking API connectivity                            │
│ ⟳ Waiting for nodes (attempt 2/5)     agent-0          │
│ ○ Verifying CoreDNS                   pending          │
├─────────────────────────────────────────────────────────┤
│ Completed: 1 | Running: 3 | Pending: 8 | Failed: 0     │
╰─────────────────────────────────────────────────────────╯
```

## Implementation Benefits

### For Task Authors
1. **Minimal Boilerplate**: Inherit from BaseTask, implement two methods
2. **Clear Structure**: All task logic in one class with organized methods
3. **Automatic Progress**: No manual progress calculations or updates
4. **Built-in Retry Logic**: Health checks automatically retry with delays
5. **Rich Error Context**: Exceptions automatically include task/subtask info

### For Framework
1. **Full Visibility**: Framework knows all subtasks and health checks upfront
2. **Consistent Structure**: Every task follows the same pattern
3. **Better Testing**: Each subtask is a testable method
4. **Easy Extension**: Add features to BaseTask, all tasks inherit them
5. **Clean Separation**: Business logic separate from progress/error handling

### For Users
1. **Clear Progress**: See exactly what's happening in each task
2. **Health Check Visibility**: Know what's being validated and retry attempts
3. **Better Debugging**: Errors show exactly which subtask failed
4. **No Interleaving**: Clean, organized display
5. **Predictable Behavior**: All tasks behave consistently

## Technical Requirements

### Dependencies
- `rich >= 13.8.0` - For TUI components
- `networkx >= 3.0` (existing) - For dependency resolution
- Python 3.10+ (existing requirement)

### Architecture Changes
- All tasks converted from functions to classes
- New `BaseTask` abstract base class
- Task registry maps names to classes instead of functions
- Progress tracking integrated into base class
- Health checks integrated into base class

### Performance
- Progress updates max 10Hz to avoid overwhelming display
- Thread-safe progress updates via locks in base class
- No blocking operations in progress callbacks
- Minimal overhead from class instantiation

## Migration Strategy

1. **Single PR Approach**: All changes in one PR
2. **Task Migration Order**: Start with leaf tasks (no dependents)
3. **Testing Strategy**: Unit test each task class, integration test full setup
4. **Feature Flag**: Initially behind `--tui` flag for testing
5. **Documentation**: Update all task documentation with new structure

## Success Metrics

1. **Code Reduction**: ~30% less code per task due to base class
2. **Error Clarity**: 100% of errors include task and subtask context
3. **Progress Accuracy**: Progress bars accurately reflect work completed
4. **Test Coverage**: Each subtask method independently testable
5. **User Satisfaction**: Clear visibility into setup process

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Learning curve for class-based tasks | Provide clear examples and migration guide |
| Subtask weight estimation | Start with rough estimates, refine based on timing |
| Terminal compatibility | Test on multiple terminals, graceful degradation |
| Performance overhead | Benchmark before/after, optimize if needed |

## Example Task Migration Checklist

For each task:
- [ ] Create class inheriting from `BaseTask`
- [ ] Implement `provides()` method
- [ ] Implement `setup()` method to register subtasks/health checks
- [ ] Move task logic into subtask methods
- [ ] Move health checks into separate methods
- [ ] Update task registration to use class
- [ ] Test progress calculation
- [ ] Verify error handling

## Conclusion

The class-based architecture dramatically simplifies task implementation while providing rich progress tracking and error handling. Task authors focus on business logic while the base class handles all UI concerns. This creates a consistent, maintainable, and user-friendly system for parallel task execution with clear progress visibility.
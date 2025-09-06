# Setup V2 - Product Requirements Document

## Executive Summary

Setup V2 is a minimal, contract-based task runner for the Catalyst Forge playground environment. It replaces numeric priority ordering with explicit dependency contracts, enabling parallel execution while maintaining simplicity for platform engineers to experiment and extend.

## Problem Statement

The current setup system has fundamental issues:

1. **Sequential execution** - Tasks run one-by-one despite no actual dependencies
2. **Split responsibilities** - Helmfile deploys services, Python configures them, creating maintenance burden
3. **Brittle shell scripts** - 600+ line Helmfile with embedded bash/sed/awk that breaks frequently
4. **Poor debuggability** - When tasks fail, engineers have no visibility into what went wrong
5. **Hard to extend** - Platform engineers need to understand the entire system to add a service

## Goals and Non-Goals

### Goals
- **Enable parallel execution** where dependencies allow
- **Make dependencies explicit** through contracts (what tasks provide/require)
- **Maximize debuggability** with automatic logging and clear error messages
- **Minimize learning curve** for platform engineers adding services
- **Keep it hackable** - playground for experimentation, not production

### Non-Goals
- **Not a framework** - No base classes, no plugin architecture
- **Not production-grade** - No rollback, no state management, no distributed execution
- **Not a Helm replacement** - Still use Helm for deployments
- **Not a configuration manager** - No drift detection or reconciliation

## Design Principles

1. **Contracts over coupling** - Tasks declare what they provide/require, not WHO they depend on
2. **Functions over frameworks** - Tasks are simple Python functions, not classes
3. **Explicit over magic** - Data flow is visible and debuggable
4. **Convention over configuration** - Clear file structure without abstractions
5. **Observability by default** - Every action is logged with actionable errors

## Architecture

### File Structure
```
playground/cli/setupv2/
├── __init__.py        # Public API: @task decorator, run_all(), load_tasks()
├── core.py            # ~100 lines: task registry, runner using networkx + threading
├── tools.py           # ~200 lines: helm, kubectl, http helpers
├── tasks/
│   ├── __init__.py    # Empty - tasks auto-discovered by load_tasks()
│   ├── postgres.py    # Example: ~30 lines
│   ├── kratos.py      # Example: ~50 lines
│   └── ...            # Platform engineers add files here
├── values/            # Static YAML for large configurations
│   └── argocd-base.yaml
└── logs/              # Single latest run directory (replaced each time)
    └── latest/
        ├── postgres.log
        └── kratos.log
```

### Core Components

#### Task Decorator
```python
@task(name="postgres", provides=["dsn", "host", "port"])
def setup_postgres(ctx, cfg, log):
    """Simple function - no base class needed."""
    # ctx: read-only dict of values from completed tasks
    # cfg: user configuration from config.cue
    # log: file handle for debugging output
    # Note: provides are auto-namespaced as postgres.dsn, postgres.host, etc.
    return {
        "dsn": "postgresql://...",
        "host": "postgres.storage",
        "port": 5432
    }
```

#### Contract System
```python
@task("kratos", requires=["postgres.dsn"])
def setup_kratos(ctx, cfg, log):
    dsn = ctx["postgres.dsn"]  # Guaranteed to exist
    # ...
```

Tasks can:
- **Require** specific keys that must exist in context (namespaced)
- **Provide** keys that other tasks can use (auto-namespaced with task name)
- **Read** any available keys with `ctx.get("key", default)`

**Namespacing**: All provided keys are automatically prefixed with the task name to prevent collisions. For example, a task named "postgres" that provides "dsn" will actually provide "postgres.dsn" to the context.

#### Execution Model
```python
# Automatic parallelization based on dependencies using ThreadPoolExecutor
Level 1: [postgres, localstack, cert-manager]  # Run in parallel
Level 2: [kratos, external-secrets]            # Run after Level 1
Level 3: [hydra]                               # Run after Level 2
```

The system uses:
- **networkx** for dependency graph analysis and cycle detection
- **ThreadPoolExecutor** for parallel task execution (simpler than async for subprocess-heavy workloads)
- **Thread-safe context** with locks to prevent race conditions

## Implementation Details

### Core Runner (~100 lines with networkx)
```python
# setupv2/core.py
import networkx as nx
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from types import MappingProxyType
import shutil
from typing import Dict, Any, Callable, List, Optional

_tasks: Dict[str, Dict[str, Any]] = {}
_context: Dict[str, Any] = {}
# No lock needed - DAG ensures tasks reading values only run after providers complete

def task(
    name: str, 
    requires: Optional[List[str]] = None, 
    provides: Optional[Dict[str, type]] = None,  # Type hints for validation
    config_keys: Optional[List[str]] = None,      # Explicit config dependencies
    timeout_sec: int = 600,
    retry_delays: Optional[List[int]] = None      # Explicit retry delays
):
    """Register a task with automatic namespacing and validation."""
    def decorator(fn: Callable):
        # Auto-namespace provides with task name
        provides_dict = provides or {}
        namespaced_provides = {f"{name}.{k}": v for k, v in provides_dict.items()}
        
        _tasks[name] = {
            "fn": fn,
            "requires": requires or [],
            "provides": namespaced_provides,
            "config_keys": config_keys or [],
            "timeout": timeout_sec,
            "retry_delays": retry_delays or [10],  # Default single retry after 10s
            "status": "pending"
        }
        return fn
    return decorator

def _filter_config(cfg: Any, keys: List[str]) -> Dict[str, Any]:
    """Extract only requested keys from config, ensuring they exist."""
    filtered = {}
    for key_path in keys:
        parts = key_path.split(".")
        value = cfg
        for part in parts:
            if not hasattr(value, "get"):
                raise ValueError(f"Config key '{key_path}' not found (not a dict at '{part}')")
            value = value.get(part)
            if value is None:
                raise ValueError(f"Config key '{key_path}' not found")
        
        # Reconstruct nested dict
        current = filtered
        for part in parts[:-1]:
            if part not in current:
                current[part] = {}
            current = current[part]
        current[parts[-1]] = value
    
    return filtered if keys else cfg

def _validate_result(result: Dict[str, Any], expected: Dict[str, type]) -> None:
    """Validate task result matches expected types."""
    for key, expected_type in expected.items():
        if key not in result:
            raise ValueError(f"Task must provide '{key}'")
        if result[key] is None:
            raise ValueError(f"Task provided None for required key '{key}'")
        if not isinstance(result[key], expected_type):
            raise ValueError(
                f"Task provided wrong type for '{key}': "
                f"expected {expected_type.__name__}, got {type(result[key]).__name__}"
            )

def run_task_sync(name: str, cfg: Any, log_dir: Path) -> Dict[str, Any]:
    """Execute a single task with retries (synchronous)."""
    task_info = _tasks[name]
    
    # Validate requirements exist in context
    ctx = MappingProxyType(_context.copy())  # Read-only view, no lock needed due to DAG
    
    for req in task_info["requires"]:
        if req not in ctx:
            raise ValueError(f"Task '{name}' requires '{req}' but it's not available")
    
    # Filter config to only requested keys
    try:
        filtered_cfg = _filter_config(cfg, task_info["config_keys"])
    except ValueError as e:
        print(f"⚙️  {name}... ✗")
        print(f"  ❌ Config Error: {e}")
        task_info["status"] = "failed"
        raise
    
    # Setup logging
    log_file = log_dir / f"{name}.log"
    
    # Run with retries using explicit delays
    retry_delays = task_info["retry_delays"]
    last_error = None
    
    for attempt in range(len(retry_delays) + 1):
        if attempt > 0:
            print(f"⚙️  {name} (retry {attempt}/{len(retry_delays)})...", end="", flush=True)
        else:
            print(f"⚙️  {name}...", end="", flush=True)
        
        try:
            with open(log_file, "a") as log:
                if attempt > 0:
                    log.write(f"\n\n=== RETRY {attempt} ===\n\n")
                
                # Run task with immutable context view and filtered config
                result = task_info["fn"](ctx, filtered_cfg, log)
            
            # Validate result matches expected types
            if result:
                expected_types = {k.split(".")[-1]: v for k, v in task_info["provides"].items()}
                _validate_result(result, expected_types)
                
                # Namespace the results with task name
                namespaced_result = {f"{name}.{k}": v for k, v in result.items()}
                _context.update(namespaced_result)  # No lock needed due to DAG
            
            print(f" ✓")
            task_info["status"] = "completed"
            return result or {}
            
        except Exception as e:
            print(f" ✗")
            last_error = str(e)
        
        if attempt < len(retry_delays):
            import time
            time.sleep(retry_delays[attempt])
    
    # All retries failed
    print(f"  📁 Logs: {log_file}")
    print(f"  ❌ Error: {last_error}")
    task_info["status"] = "failed"
    raise RuntimeError(f"Task '{name}' failed after {len(retry_delays) + 1} attempts")

def run_all(cfg: Any, only: Optional[List[str]] = None, dry_run: bool = False):
    """Run all tasks respecting dependencies and parallelism."""
    levels = compute_execution_levels(_tasks, only)
    
    print(f"📋 Execution plan: {len(levels)} levels, {sum(len(l) for l in levels)} tasks")
    for i, level in enumerate(levels, 1):
        print(f"  Level {i}: {', '.join(level)}")
    print()
    
    if dry_run:
        print("✨ Dry run complete (no tasks executed)")
        return
    
    # Single log directory - remove previous
    log_dir = Path("logs/latest")
    if log_dir.exists():
        shutil.rmtree(log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)
    
    for level_num, level_tasks in enumerate(levels, 1):
        # Run all tasks at this level in parallel using threads
        with ThreadPoolExecutor(max_workers=len(level_tasks)) as executor:
            futures = {
                executor.submit(run_task_sync, name, cfg, log_dir): name 
                for name in level_tasks
            }
            
            failed = False
            for future in as_completed(futures):
                name = futures[future]
                try:
                    future.result()
                except Exception as e:
                    # Mark failure but let other tasks at this level complete
                    failed = True
                    print(f"  ⚠️  Task '{name}' failed, will stop after level {level_num} completes")
            
            if failed:
                print(f"\n❌ Stopping execution due to failures at level {level_num}")
                print(f"   Skipping levels: {', '.join(f'Level {i}' for i in range(level_num + 1, len(levels) + 1))}")
                raise RuntimeError("One or more tasks failed")

def compute_execution_levels(tasks: Dict[str, Dict], only: Optional[List[str]] = None) -> List[List[str]]:
    """
    Compute parallel execution levels using networkx for dependency resolution.
    Returns list of task groups that can run in parallel.
    """
    G = nx.DiGraph()
    
    # Filter tasks if 'only' specified
    active = set(only) if only else set(tasks.keys())
    
    # Build provides map
    provides_map = {}
    for name in active:
        for key in tasks[name]["provides"]:
            if key in provides_map:
                raise ValueError(f"Contract conflict: '{key}' provided by both '{provides_map[key]}' and '{name}'")
            provides_map[key] = name
    
    # Add nodes
    for name in active:
        G.add_node(name)
    
    # Add edges based on dependencies
    for name in active:
        for req in tasks[name]["requires"]:
            if req not in provides_map:
                raise ValueError(f"Task '{name}' requires '{req}' but no task provides it")
            provider = provides_map[req]
            if provider in active:
                G.add_edge(provider, name)
    
    # Check for cycles
    if not nx.is_directed_acyclic_graph(G):
        cycles = list(nx.simple_cycles(G))
        raise ValueError(f"Circular dependency detected: {cycles[0]}")
    
    # Get execution levels using topological generations
    return [list(gen) for gen in nx.topological_generations(G)]
```

### Tools Layer (~200 lines)
```python
# setupv2/tools.py
import subprocess
import yaml
from typing import Dict, Any, IO

class Helm:
    def install(self, name: str, chart: str, namespace: str, 
                values: Dict[str, Any], log: IO, wait: bool = True):
        """Install a Helm chart with automatic logging."""
        # Write values to temp file
        values_file = f"/tmp/{name}-values.yaml"
        with open(values_file, "w") as f:
            yaml.dump(values, f)
        
        cmd = [
            "helm", "upgrade", "--install", name, chart,
            "-n", namespace, "--create-namespace",
            "-f", values_file
        ]
        if wait:
            cmd.extend(["--wait", "--timeout", "5m"])
        
        log.write(f"$ {' '.join(cmd)}\n")
        log.flush()
        
        # Run with streaming output to log file
        proc = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True
        )
        
        for line in proc.stdout:
            log.write(line)
            log.flush()
        
        proc.wait()
        if proc.returncode != 0:
            raise RuntimeError(f"Helm install failed for {name}")

class Kubectl:
    def wait_for(self, resource: str, namespace: str, log: IO, timeout: int = 300):
        """Wait for a resource to be ready."""
        cmd = [
            "kubectl", "wait", "--for=condition=available",
            f"--timeout={timeout}s", "-n", namespace, resource
        ]
        
        log.write(f"$ {' '.join(cmd)}\n")
        log.flush()
        
        proc = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True
        )
        
        for line in proc.stdout:
            log.write(line)
            log.flush()
        
        proc.wait()
        if proc.returncode != 0:
            raise RuntimeError(f"Timeout waiting for {resource}")

# Singleton instances - no magic, just convenience
helm = Helm()
kubectl = Kubectl()

def load_tasks():
    """Import all task modules from tasks/ directory."""
    import importlib
    from pathlib import Path
    
    tasks_dir = Path(__file__).parent / "tasks"
    for task_file in tasks_dir.glob("*.py"):
        if task_file.name.startswith("_"):
            continue
        module_name = f"setupv2.tasks.{task_file.stem}"
        importlib.import_module(module_name)
```

### Example Tasks

#### Simple Infrastructure Task
```python
# setupv2/tasks/postgres.py
from setupv2 import task
from setupv2.tools import helm, kubectl

@task("postgres", 
      provides={"dsn": str, "host": str, "port": int},  # Type hints for validation
      config_keys=["deps.db.password"],  # Explicit config dependencies
      timeout_sec=300,
      retry_delays=[5, 15])  # Retry after 5s, then 15s
def setup_postgres(ctx, cfg, log):
    """Deploy PostgreSQL database."""
    
    # Config is pre-filtered to only requested keys
    password = cfg["deps"]["db"]["password"]  # Guaranteed to exist
    
    values = {
        "auth": {
            "postgresPassword": password,
            "database": "postgres"
        },
        "primary": {
            "persistence": {"size": "10Gi"}
        }
    }
    
    helm.install(
        name="postgres",
        chart="bitnami/postgresql",
        namespace="storage",
        values=values,
        log=log
    )
    
    kubectl.wait_for(
        "statefulset/postgres-postgresql",
        namespace="storage",
        log=log
    )
    
    # Return values - will be namespaced as postgres.host, postgres.port, postgres.dsn
    return {
        "host": "postgres-postgresql.storage",
        "port": 5432,
        "dsn": f"postgresql://postgres:{password}@postgres-postgresql.storage:5432"
    }
```

#### Task with Dependencies
```python
# setupv2/tasks/kratos.py
from setupv2 import task
from setupv2.tools import helm, kubectl
import yaml

@task("kratos", 
      requires=["postgres.dsn"],
      provides={"public_url": str, "admin_url": str},  # Type hints
      config_keys=["auth.domain", "frontend.domain"])  # Explicit deps
def setup_kratos(ctx, cfg, log):
    """Deploy Ory Kratos identity service."""
    
    # Config is pre-filtered and validated
    auth_domain = cfg["auth"]["domain"]
    frontend_domain = cfg["frontend"]["domain"]
    
    values = {
        "kratos": {
            "config": {
                "dsn": ctx["postgres.dsn"] + "/kratos",
                "serve": {
                    "public": {
                        "base_url": f"https://{auth_domain}/kratos/public"
                    }
                },
                "selfservice": {
                    "default_browser_return_url": f"https://{frontend_domain}/"
                }
            }
        }
    }
    
    # Save rendered values for debugging
    log.write(f"\n--- Rendered Values ---\n{yaml.dump(values)}\n---\n\n")
    
    helm.install(
        name="kratos",
        chart="ory/kratos",
        namespace="auth",
        values=values,
        log=log
    )
    
    kubectl.wait_for("deployment/kratos", namespace="auth", log=log)
    
    # Return values - will be namespaced as kratos.public_url, kratos.admin_url
    return {
        "public_url": f"https://{auth_domain}/kratos/public",
        "admin_url": "http://kratos-admin.auth:4434"
    }
```

#### Platform Engineer Adding a Service
```python
# setupv2/tasks/istio.py - Engineer just adds this file
from setupv2 import task
from setupv2.tools import helm, kubectl

@task("istio", 
      requires=["cert-manager.ready"],
      provides={"ready": bool, "gateway": str},  # Type hints
      config_keys=[],  # No config needed for this task
      timeout_sec=900,
      retry_delays=[30, 60])  # Retry after 30s, then 60s
def setup_istio(ctx, cfg, log):
    """Add Istio service mesh."""
    
    # No framework to learn - just write Python
    log.write("Installing Istio service mesh\n")
    
    # Install base
    helm.install(
        name="istio-base",
        chart="istio/base",
        namespace="istio-system",
        values={},
        log=log
    )
    
    # Install control plane
    helm.install(
        name="istiod",
        chart="istio/istiod",
        namespace="istio-system",
        values={"pilot": {"autoscaleEnabled": False}},
        log=log
    )
    
    # Return values - will be namespaced as istio.ready, istio.gateway
    return {
        "ready": True,
        "gateway": "istio-gateway.istio-system"
    }
```

## CLI Integration

```bash
# Run everything
uv run cli setupv2

# Dry run to see execution plan
uv run cli setupv2 --dry-run

# Run specific tasks
uv run cli setupv2 --only postgres --only kratos

# List available tasks and their contracts
uv run cli setupv2 --list
postgres: requires() → provides(postgres.dsn: str, postgres.host: str, postgres.port: int)
kratos: requires(postgres.dsn) → provides(kratos.public_url: str, kratos.admin_url: str)
istio: requires(cert-manager.ready) → provides(istio.ready: bool, istio.gateway: str)

# Debugging failed tasks - check the logs
ls -la logs/latest/
postgres.log
kratos.log
temporal.log

cat logs/latest/temporal.log
```

## Error Handling and Debugging

When tasks fail, engineers get actionable information:

```
⚙️  postgres... ✓
⚙️  kratos... ✗
  📁 Logs: logs/20240115_103045/kratos.log
  ❌ Error: Timeout waiting for deployment/kratos
```

Key features:
- Every subprocess command is logged with timestamps
- Rendered values are saved for inspection in log files
- Failed tasks can be retried with `--only <task>` after fixing issues
- Tasks are idempotent - safe to re-run
- Optional validation functions catch configuration issues early

## Migration Strategy

Since this is an alpha system with a single user, we'll build V2 in isolation:

1. **Week 1**: Implement core.py and tools.py
2. **Week 2**: Migrate base services (postgres, cert-manager, registry)
3. **Week 3**: Migrate auth stack (kratos, hydra, oathkeeper)
4. **Week 4**: Migrate remaining services
5. **Week 5**: Update `just up` to use setupv2, deprecate old system

During migration:
- Both systems coexist (`uv run cli setup` vs `uv run cli setupv2`)
- Tasks are migrated incrementally
- Validation by comparing cluster state between old and new

### Complex Task Migration Example

The current Gitea setup (15k lines) combines Helmfile deployment with Python configuration. Here's how it migrates:

```python
# setupv2/tasks/gitea.py
@task("gitea", 
      requires=["postgres.dsn", "hydra.client_id", "registry.host"],
      provides={"url": str, "admin_token": str},  # Type hints
      config_keys=["deps.db.user", "deps.db.password", "gitea.domain", 
                   "gitea.oauth_secret", "auth.domain", "gitea.repos"],
      timeout_sec=600)
def setup_gitea(ctx, cfg, log):
    """Deploy and configure Gitea with OAuth2, repos, and webhooks."""
    
    # Phase 1: Deploy via Helm (replaces Helmfile)
    values = {
        "postgresql": {"enabled": False},
        "gitea": {
            "config": {
                "database": {
                    "DB_TYPE": "postgres",
                    "HOST": ctx["postgres.host"],
                    "NAME": "gitea",
                    "USER": cfg["deps"]["db"]["user"],
                    "PASSWD": cfg["deps"]["db"]["password"],
                },
                "server": {
                    "DOMAIN": cfg["gitea"]["domain"],
                    "ROOT_URL": f"https://{cfg['gitea']['domain']}",
                },
            }
        }
    }
    
    helm.install("gitea", "gitea-charts/gitea", "gitea", values, log)
    kubectl.wait_for("statefulset/gitea", "gitea", log)
    
    # Phase 2: Configure OAuth2 (existing Python logic)
    admin_token = _create_admin_token(cfg, log)
    
    oauth_config = {
        "provider": "OpenIDConnect", 
        "client_id": ctx["hydra.client_id"],
        "client_secret": cfg["gitea"]["oauth_secret"],
        "auto_discover_url": f"https://{cfg['auth']['domain']}/hydra/.well-known/openid-configuration",
    }
    _configure_oauth_provider(oauth_config, admin_token, log)
    
    # Phase 3: Database operations (existing Python logic)
    _run_migrations(ctx["postgres.dsn"], log)
    
    # Phase 4: Create repos and webhooks (existing Python logic)
    for repo in cfg["gitea"]["repos"]:
        _create_repo(repo, admin_token, log)
        if repo.get("webhooks"):
            _configure_webhooks(repo, admin_token, log)
    
    return {
        "url": f"https://{cfg['gitea']['domain']}",
        "admin_token": admin_token,
    }
```

Key migration points:
- Helm deployment moves from Helmfile to Python
- Complex configuration logic stays in Python (unchanged)
- Database operations remain in Python tasks
- All phases run within a single task for atomicity

## Success Metrics

1. **Faster execution**: Parallel tasks reduce setup time from ~15min to ~5min
2. **Better debugging**: 100% of failures have actionable error messages
3. **Easier extension**: Adding a service requires only creating one file
4. **Cleaner codebase**: Remove 600+ lines of embedded bash from Helmfile
5. **Platform engineer satisfaction**: Time to add Istio drops from hours to minutes

## Open Questions (Resolved)

1. **Should we use async or threading?** 
   - Decision: **Threading** - Subprocess calls release the GIL, making threads simpler than async
   - Using ThreadPoolExecutor for parallel task execution

2. **Should tasks support cleanup/rollback?**
   - Decision: **No** - Focus on idempotency instead
   - Tasks should be safe to re-run; if cleanup needed, destroy and recreate cluster

3. **Should we track partial completion state?**
   - Decision: **No** - Too complex, rely on idempotent operations
   - Tasks use `helm upgrade --install` and similar idempotent commands

4. **Dependency resolution library?**
   - Decision: **networkx** - Battle-tested, minimal configuration, replaces ~80 lines with ~15
   - Provides cycle detection and topological sorting out of the box

5. **Optional features to include:**
   - ✅ **Retries and timeouts** via task parameters
   - ✅ **Dry-run mode** to show execution plan without running
   - ✅ **Automatic namespacing** for provided contract keys
   - ❌ **Optional dependencies** - Dropped to avoid partial state complexity
   - ❌ **Conditional execution** - Dropped for simplicity in v1
   - ❌ **Validation functions** - Dropped for simplicity in v1
   - ❌ **Progress reporting** - Keep it simple, logs are sufficient

## Appendix: Comparison with Current System

| Aspect | Current (V1) | Proposed (V2) |
|--------|--------------|---------------|
| Task Definition | Class with registry decorator | Simple function with @task |
| Dependencies | Numeric priorities (20-60) | Explicit contracts (provides/requires) |
| Execution | Sequential | Parallel using ThreadPoolExecutor |
| Debugging | No built-in logging | Automatic log files in logs/latest/ |
| Configuration | Split between Helmfile/Python | All in Python (explicit) |
| Framework Size | ~500 lines | ~150 lines core + ~200 lines tools |
| Dependency Resolution | Manual priority ordering | networkx for DAG analysis |
| Learning Curve | Must understand registry, deps, priorities | Just write a Python function |
| Adding a Service | Edit Helmfile + create Python task | Add one Python file |
| Error Recovery | Manual debugging | Configurable retry delays |
| Error Handling | Fail fast | Let level complete, then stop |
| Validation | None | Type hints + config key validation |
| Config Access | Full config object | Filtered to declared keys only |
| Dry Run | Not supported | --dry-run flag shows plan |
| Contract Namespacing | Manual | Automatic with task name prefix |
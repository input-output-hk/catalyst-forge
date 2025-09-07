# Playground CLI

The Playground CLI provides developer tooling to create and manage a local k3d-based environment, deploy services, and run environment setup tasks. It is designed for composability, safety, and ease of extension.

## Quick start

Prerequisites:
- Docker
- k3d
- kubectl
- mkcert
- cue
- uv (Python package/runtime manager)

Run commands with uv:
```bash
uv run python playground/cli/main.py --help
uv run python playground/cli/main.py setup --list
uv run python playground/cli/main.py setup --only k3d
uv run python playground/cli/main.py setup --only hydra
uv run python playground/cli/main.py setup --only deploy
uv run python playground/cli/main.py up
```
### `up` command

The `up` command performs a full environment bootstrap in order:

1. k3d
2. dns (wildcard)
3. generate (TLS + Earthly config)
4. helmfile foundational releases (in order): cert-manager, trust-manager, envoy-gateway, registry, postgres, localstack, external-secrets
5. migrate (database seeders)
6. helmfile remaining releases: mailpit, kratos, hydra, temporal
7. hydra (OAuth2 client setup)
8. deploy (services from `deployments`)

You can choose Helmfile action for all helmfile steps:

```bash
uv run python playground/cli/main.py up --helmfile-action=sync
```


Project checks from the playground root:
```bash
just check
```

Pretty exceptions are disabled by default to avoid leaking environment locals. Enable on demand:
```bash
uv run python playground/cli/main.py --pretty setup --list
```

## Configuration

The CLI reads a single CUE file (exported to JSON at runtime):
- Path: `playground/config.cue`

Key top-level fields used by the CLI:
- `registry`: image registry hostname
- `deployments`: service build and deploy parameters
- `tasks`: setup task configuration (per-task subkeys)
- `deps`: shared dependency configuration for setup tasks and utilities

Example snippets:
```cue
registry_host: "registry.projectcatalyst.dev"

deps: {
    db: {
        host:     "postgres-postgresql.postgres.svc.cluster.local"
        port:     5432
        user:     "postgres"
        password: "postgres"
        admin_db: "postgres"
    }
    k8s: {
        kubeconfig: "playground/kubeconfig"
    }
}

tasks: {
    hydra: {
        clients: {
            forge_cli: {
                client_id:   "forge-cli"
                client_name: "Forge CLI (dev)"
                scope:       "openid offline"
                grant_types: ["authorization_code", "refresh_token"]
                response_types: ["code"]
                token_endpoint_auth_method: "none"
                redirect_uris: [
                    "http://127.0.0.1:49152/callback",
                    "http://127.0.0.1:34567/callback",
                ]
                post_logout_redirect_uris: [
                    "http://127.0.0.1:49152/logout",
                ]
            }
        }
    }
}
```

## Commands overview

- `setup --only k3d`: Create or reuse a local k3d cluster with Envoy-compatible ingress, write kubeconfig, wait for Ready.
- `generate`: Generate client TLS certs and Earthly config for mtls to buildkit.
- `setup --only migrate`: Initialize Postgres for dependent apps (Kratos, Hydra, …) via registry-driven db seeders.
- `setup --only deploy`: Build, render, and apply one or more services (Earthly + CUE + kubectl).
- `setup`: Run pluggable setup tasks using the contract-based system. See below.
- `dns pin-wildcard` and `dns pin-registry`: CoreDNS helpers for local routing.
- `setup --only helmfile`: Run Helmfile `apply` (default) or `sync`, with include/skip filters via runtime args.
- `up`: Bring up the full local environment in the correct order (k3d → dns → generate → helmfile foundation → migrate → helmfile apps → hydra → deploy).

## Setupv2 architecture (contract-based tasks)

The setup system uses explicit contracts for parallel execution:
- Tasks live under `playground/cli/setup/tasks/`.
- Each task declares what it `provides` and what it `requires` via the `@task` decorator.
- Tasks are automatically namespaced (e.g., `postgres.dsn`, `kratos.public_url`).
- The system uses networkx to resolve dependencies and execute tasks in parallel where possible.

Execution flow:
1. CLI loads configuration and stores it in a `ConfigState`.
2. The `setup` command auto-discovers tasks under `playground.cli.setup.tasks` and loads them.
3. Tasks are analyzed for dependencies and grouped into parallel execution levels.
4. Tasks within each level run in parallel using ThreadPoolExecutor.
5. Each task gets `(ctx: dict, cfg: dict, log: file)` and returns values to provide to other tasks.
### Runtime task arguments

Some behavior is better controlled at runtime rather than in `config.cue`. The `setup` command accepts task-scoped arguments:

- `--task-arg <task>.<key>=<value>` (repeatable)
- `--task-args-file <path.json>` (JSON object: `{ "<task>": { ... } }`)

Parsing rules:
- JSON objects/arrays in values are parsed when enclosed in `{}`/`[]`.
- `true`/`false` → booleans; integers → ints; comma-separated → list of strings; otherwise string.

Tasks can read their scope via `ctx.runtime.get("<task>", {})`.

Examples:
```bash
uv run python playground/cli/main.py setup --only migrate --task-arg migrate.only=hydra,kratos
uv run python playground/cli/main.py setup --only deploy --task-arg deploy.only=api,frontend
uv run python playground/cli/main.py setup --only deploy --task-arg deploy.skip=renderer
uv run python playground/cli/main.py setup --only hydra --task-arg hydra.clients='{"cli":{"client_id":"forge-cli","redirect_uris":["http://127.0.0.1:49152/callback"]}}'
uv run python playground/cli/main.py setup --task-args-file playground/cli/runtime.json
```

`runtime.json` example:
```json
{
  "migrate": { "only": ["hydra", "kratos"] },
  "deploy": { "only": ["api"], "show_manifest": false },
  "dns": { "domain": "dev.local" }
}
```


Listing or selecting tasks:
```bash
uv run python playground/cli/main.py setup --list
uv run python playground/cli/main.py setup --only k3d
uv run python playground/cli/main.py setup --only hydra
```

### tasks.k3d configuration

Configure the k3d setup task under `tasks.k3d` in `playground/config.cue`:

```cue
tasks: {
    k3d: {
        enabled:       true
        name:          "forge"
        servers:       1
        agents:        0
        http_port:     80
        https_port:    443
        api_port:      0
        kubeconfig_out:"playground/kubeconfig"
        output_json:   "playground/cluster.json"
        force_recreate:false
        assume_yes:    false
        registry_host: "registry.projectcatalyst.dev"
    }
}
```

### Declaring a new setup task

1) Create a file under `playground/cli/setup/tasks/`, e.g. `mytask.py`.
2) Use the `@task` decorator to declare what your task provides and requires.
3) Implement a simple function that takes `(ctx, cfg, log)` parameters.
4) Return a dictionary of values to provide to other tasks.

Minimal example:
```python
from cli.setup import task
from cli.setup.tools import helm

@task("mytask",
      provides={"ready": bool},  # What this task provides
      requires=[],               # What this task needs from others
      config_keys=["tasks.mytask"])  # Config keys to access
def setup_mytask(ctx, cfg, log):
    """Deploy mytask service."""

    # Access filtered config
    task_config = cfg["tasks"]["mytask"]

    # Use setup tools
    helm.install(
        name="mytask",
        chart="mychart/mytask",
        namespace="default",
        values={"config": task_config},
        log=log
    )

    # Return provided values (auto-namespaced as mytask.ready)
    return {"ready": True}
```

Add CUE configuration under `tasks.mytask`:
```cue
tasks: {
    mytask: {
        enabled: true
        message: "hello from setup"
    }
}
```

### Dependency injection

The `Deps` container (in `playground/cli/deps.py`) provides lazy providers:
- `runner`: `CommandRunner` (structured logging, env/cwd control, optional capture, timeouts, redaction)
- `db`: `DatabaseRoot` built from `deps.db` in config
- `k8s`: Pre-auth Kubernetes clients (`core`, `apps`) using `deps.k8s.kubeconfig`
### tasks.helmfile configuration

Configure the Helmfile setup task under `tasks.helmfile` in `playground/config.cue`:

```cue
tasks: {
    helmfile: {
        enabled:      true
        path:         "playground/helmfile/helmfile.yaml"
        environment:  _|_ // optional
        working_dir:  _|_ // optional; defaults to dirname(path)
        timeout_sec:  600
        extra_args:   []
    }
}
```

Runtime examples:

```bash
# Apply everything (default action)
uv run python playground/cli/main.py setup --only helmfile

# Sync only selected releases
uv run python playground/cli/main.py setup --only helmfile \
  --task-arg helmfile.action=sync \
  --task-arg helmfile.only=postgres,ory

# Apply excluding a release (requires helmfile supporting negative selectors)
uv run python playground/cli/main.py setup --only helmfile \
  --task-arg helmfile.skip=temporal

# Add raw selectors and extra args
uv run python playground/cli/main.py setup --only helmfile \
  --task-arg helmfile.selectors='["namespace=auth"]' \
  --task-arg helmfile.args='["--debug"]'
```


Tasks declare what they `provides` and `requires` via the `@task` decorator. The setup system uses networkx to resolve dependencies and execute tasks in parallel where possible.

## Contributing

- Run `just check` from `playground/` to lint and type-check.
- Prefer small, self-contained tasks; keep `run()` idempotent.
- Avoid hardcoding secrets in code; prefer `deps` configuration or environment.
- When adding a new setup task, keep its configuration and Pydantic models local to its module.

## Troubleshooting

- Unexpected stack traces printing env locals? By default we disable pretty exceptions. Use `--pretty` when you need more readable traces; locals remain hidden.
- Ensure required CLIs are installed and on PATH (`docker`, `k3d`, `kubectl`, `mkcert`, `cue`, `earthly`, `go`).
- If `cue export` fails, validate `playground/config.cue` syntax and required fields.

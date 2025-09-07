# Setup V2 Tasks

This directory contains task definitions for the Setup V2 system.

## Task Organization

Tasks can be organized in two formats:

### Legacy Format (Single File)
```
tasks/
├── postgres.py          # Contains PostgresConfig + @task function
└── cert_manager.py      # Contains CertManagerConfig + @task function
```

### New Format (Subpackage) ⭐ **Recommended**
```
tasks/
├── postgres/
│   ├── __init__.py      # Package initialization
│   ├── config.py        # PostgresConfig and related config classes
│   └── task.py          # @task decorated function and task logic
└── cert_manager/
    ├── __init__.py      # Package initialization
    ├── config.py        # CertManagerConfig and related config classes
    └── task.py          # @task decorated function and task logic
```

## Creating New Tasks

### Using Subpackage Format (Recommended)

1. **Create the subpackage directory**:
   ```bash
   mkdir tasks/my_service
   ```

2. **Create `__init__.py`**:
   ```python
   # tasks/my_service/__init__.py
   """
   My Service task package.

   This package contains the My Service deployment task.
   """
   ```

3. **Create `config.py`** with your configuration model:
   ```python
   # tasks/my_service/config.py
   from pydantic import BaseModel

   class MyServiceConfig(BaseModel):
       """Configuration for My Service deployment."""

       namespace: str = "my-service"
       chart: str = "my-service/chart"
       version: str = "1.0.0"
   ```

4. **Create `task.py`** with your task logic:
   ```python
   # tasks/my_service/task.py
   from cli.setupv2 import task
   from cli.setupv2.tools import helm, k8s
   from .config import MyServiceConfig

   @task(
       "my-service",
       requires=["k3d.ready"],  # Dependencies
       provides={"url": str, "ready": bool},  # What this task provides
       config_keys=["my_service"],  # Config keys to access
       timeout_sec=300,
       retry_delays=[10, 30],  # Retry strategy
   )
   def setup_my_service(ctx, cfg, log):
       """Deploy My Service."""

       # Parse configuration
       config = MyServiceConfig.model_validate(cfg["my_service"])

       # Your deployment logic here
       helm.install(
           name="my-service",
           chart=config.chart,
           namespace=config.namespace,
           values={},
           log=log,
       )

       # Return provided values
       return {
           "url": f"https://my-service.{config.namespace}.svc.cluster.local",
           "ready": True,
       }
   ```

### Migration from Legacy Format

Existing tasks can be migrated from single-file format to subpackage format:

1. Create the subpackage directory structure as shown above
2. Move the `Config` class to `config.py`
3. Move the `@task` decorated function to `task.py`
4. Update imports to use relative imports (`from .config import ConfigClass`)
5. Remove the original single-file task

The system automatically detects and loads both formats during the transition period.

## Cluster Setup (Phase 4) ✅

### k3d/ ⭐ **Migrated to Subpackage**
- **Provides**: `cluster_name`, `kubeconfig`, `http_port`, `https_port`, `api_port`, `ready`
- **Config Keys**: `k3d`, `registry`
- **Retry**: 10s, 30s
- **Description**: Creates/manages k3d cluster, configures mkcert CA, sets up kubeconfig
- **Files**: `config.py` (K3dConfig), `task.py` (setup_k3d)
- **Execution**: Level 1 (runs first, before all other tasks)

## Base Infrastructure (Phase 4) ✅

These tasks run in parallel (Level 2) after k3d cluster is ready:

### postgres/ ⭐ **Migrated to Subpackage**
- **Requires**: `k3d.ready`
- **Provides**: `dsn`, `host`, `port`
- **Config Keys**: `postgres`
- **Retry**: 5s, 15s
- **Description**: PostgreSQL database with 10Gi storage
- **Files**: `config.py` (PostgresConfig), `task.py` (setup_postgres)

### cert_manager/ ⭐ **Migrated to Subpackage**
- **Requires**: `k3d.ready`
- **Provides**: `ready`, `webhook_ready`
- **Config Keys**: `cert_manager`
- **Retry**: 10s, 30s
- **Description**: Certificate management with self-signed ClusterIssuer
- **Files**: `config.py` (CertManagerConfig), `task.py` (setup_cert_manager)

### localstack.py
- **Requires**: `k3d.ready`
- **Provides**: `endpoint`, `region`, `aws_access_key`, `aws_secret_key`
- **Config Keys**: None
- **Retry**: 5s, 15s
- **Description**: AWS service simulation (S3, SecretManager, SSM, STS, IAM, DynamoDB, SQS, SNS)

### registry.py
- **Requires**: `k3d.ready`
- **Provides**: `host`, `url`, `username`, `password`
- **Config Keys**: `registry`
- **Retry**: 5s, 15s
- **Description**: Docker registry with 20Gi storage and optional auth

## Configuration Tasks (Phase 4) ✅

These tasks configure cluster services and local development tools:

### dns.py
- **Requires**: `k3d.ready`
- **Provides**: `configured`, `domain`, `envoy_ip`
- **Config Keys**: None (uses defaults)
- **Retry**: 5s, 10s
- **Description**: Configures CoreDNS wildcard DNS to pin *.projectcatalyst.dev to Envoy gateway ClusterIP

### generate.py
- **Requires**: `k3d.ready`
- **Provides**: `client_cert`, `client_key`, `ca_cert`, `earthly_config`, `ready`
- **Config Keys**: None (uses defaults)
- **Retry**: 5s
- **Description**: Generates mTLS client certificates and Earthly configuration for buildkitd

## Test Tasks

### test_task.py
Contains example tasks demonstrating dependency chains and parallel execution patterns.

## Usage

Run all base infrastructure in parallel:
```bash
uv run cli setupv2 --only postgres --only cert-manager --only localstack --only registry
```

Or run everything:
```bash
uv run cli setupv2
```

Check execution plan without running:
```bash
uv run cli setupv2 --dry-run
```

## Auth Stack (Phase 5) ✅

### keycloak/ ⭐ **Migrated to Subpackage**
- **Requires**: `postgres.dsn`, `postgres.host`, `postgres.port`, `external-secrets.ready`, `keycloak-operator.operator_ready`, `cert-manager.ready`
- **Provides**: `base_url`, `admin_url`, `realm_name`, `client_id`, `client_secret`
- **Config Keys**: `keycloak`
- **Retry**: 20s, 30s
- **Description**: Keycloak identity provider with PostgreSQL backend
- **Files**: `config.py` (KeycloakConfig), `task.py` (setup_keycloak)

## Migration Status

All tasks have been successfully migrated to the new subpackage format! ⭐

### Migrated Tasks (Subpackage Format)
- ⭐ `cert_manager/` - Certificate management with ClusterIssuer
- ⭐ `postgres/` - PostgreSQL database with connection details
- ⭐ `keycloak/` - Identity provider with OAuth2/OIDC
- ⭐ `k3d/` - Kubernetes cluster creation and management
- ⭐ `registry/` - Docker registry with authentication
- ⭐ `dns/` - CoreDNS wildcard DNS configuration
- ⭐ `generate/` - TLS certificates and Earthly configuration
- ⭐ `localstack/` - AWS service simulation
- ⭐ `external_secrets/` - Secret management operator
- ⭐ `keycloak_operator/` - Keycloak operator deployment
- ⭐ `trust_manager/` - Certificate authority distribution
- ⭐ `envoy_gateway/` - API routing and ingress management
- ⭐ `mailpit/` - Email testing service
- ⭐ `earthly/` - Earthly remote buildkit instance
- ⭐ `argocd/` - GitOps with Keycloak SSO
- ⭐ `temporal/` - Workflow engine with PostgreSQL
- ⭐ `gitea/` - Git hosting with OAuth2 and repositories
- ⭐ `test_task/` - Test tasks for system verification

## Next Phases

- **Phase 7**: Testing and Validation
- **Phase 8**: Migration Finalization
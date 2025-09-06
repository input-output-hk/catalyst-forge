# Setup V2 Implementation Tasks

## Overview
This document outlines the comprehensive task breakdown for implementing Setup V2, a minimal, contract-based task runner for the Catalyst Forge playground environment. The implementation is organized into 8 major phases with clear dependencies and acceptance criteria.

## Timeline
- **Total Duration**: 5 weeks (as per PRD migration strategy)
- **Parallel Work**: Multiple phases and tasks can execute in parallel where dependencies allow
- **Critical Path**: Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6 → Phase 7 → Phase 8

## Phase Breakdown

### Phase 1: Core Infrastructure (Week 1) ✅
**Goal**: Build the foundation for task registration, dependency resolution, and parallel execution

#### Tasks
- [x] Create setupv2 package structure and directories
  - Create `playground/cli/setupv2/` directory
  - Create `__init__.py`, `core.py`, `tools.py`
  - Create `tasks/`, `values/`, `logs/` subdirectories

- [x] Implement core.py with task decorator and registry (~340 lines)
  - Implement `@task` decorator with parameters: name, requires, provides, config_keys, timeout_sec, retry_delays
  - Create global `_tasks` registry dictionary
  - Implement automatic namespacing for provided keys

- [x] Implement dependency graph resolution with networkx
  - Build directed graph from task requirements
  - Implement cycle detection
  - Generate execution levels using topological generations

- [x] Implement parallel execution with ThreadPoolExecutor
  - Execute tasks within same level in parallel
  - Implement thread-safe context updates
  - Handle failure propagation between levels

- [x] Add retry logic and timeout handling to core runner
  - Implement configurable retry delays
  - Add timeout enforcement per task
  - Log retry attempts with clear messaging

- [x] Implement config filtering and validation system
  - Filter config to only requested keys per task
  - Validate provided values match expected types
  - Ensure required config keys exist

**Acceptance Criteria**:
- ✅ Core can register tasks with contracts
- ✅ Dependencies are correctly resolved into execution levels
- ✅ Tasks execute in parallel within levels
- ✅ Retry and timeout logic works as specified
- ✅ Config filtering prevents tasks from accessing unnecessary data

### Phase 2: Tools Layer (Week 1-2) ✅
**Goal**: Create helper utilities for common operations

#### Tasks
- [x] Implement tools.py with Helm class (~280 lines total)
  - Create `Helm.install()` method with values file generation
  - Add automatic command logging
  - Implement streaming output to log files

- [x] Implement Kubectl class with wait_for functionality
  - Create `Kubectl.wait_for()` method
  - Add configurable timeouts
  - Implement proper error handling

- [x] Add HTTP helpers for API interactions
  - Create helpers for common REST operations
  - Add authentication support
  - Implement retry logic for transient failures

- [x] Implement load_tasks() auto-discovery function
  - Scan `tasks/` directory for Python files
  - Import modules dynamically
  - Skip files starting with underscore

- [x] Add logging infrastructure with automatic log directory management
  - Create `logs/latest/` directory structure
  - Implement per-task log files
  - Add log rotation (remove previous run)

**Acceptance Criteria**:
- ✅ Helm charts can be deployed with automatic logging
- ✅ Kubectl can wait for resources with proper timeouts
- ✅ Tasks are auto-discovered from tasks/ directory
- ✅ Each task execution produces a separate log file
- ✅ Previous logs are cleaned up on new runs

### Phase 3: CLI Integration (Week 2) ✅
**Goal**: Connect setupv2 to the existing CLI system

#### Tasks
- [x] Add setupv2 command to main.py CLI
  - Import setupv2 module
  - Add command handler
  - Pass configuration from CUE

- [x] Implement --dry-run flag for execution planning
  - Show execution levels without running tasks
  - Display task counts and dependencies
  - Exit without executing

- [x] Implement --only flag for selective task execution
  - Filter tasks to specified subset
  - Maintain dependency resolution for filtered tasks
  - Validate all requirements can be satisfied

- [x] Implement --list flag to show available tasks and contracts
  - Display task names with requirements and provisions
  - Show type hints for provided values
  - Format output for readability

**Acceptance Criteria**:
- ✅ `uv run cli setupv2` executes all tasks
- ✅ `--dry-run` shows plan without execution
- ✅ `--only` runs specific tasks with dependencies
- ✅ `--list` displays all available tasks with contracts
- ✅ Proper error messages for invalid task names

### Phase 4: Base Infrastructure Tasks (Week 2) ✅
**Goal**: Migrate foundation services including k3d cluster setup

#### Tasks (COMPLETED)
- [x] Create k3d.py task for cluster management
  - Provides: cluster_name, kubeconfig, http_port, https_port, api_port, ready
  - Create/reuse k3d cluster
  - Configure mkcert CA certificates
  - Setup registry configurations
  - Write kubeconfig and wait for nodes

- [x] Create postgres.py task with retry logic
  - Requires: k3d.ready
  - Provides: dsn, host, port
  - Deploy PostgreSQL with Helm
  - Wait for StatefulSet ready
  - Return connection details

- [x] Create cert-manager.py task
  - Requires: k3d.ready
  - Provides: ready, webhook_ready
  - Deploy cert-manager
  - Wait for webhook availability
  - Configure ClusterIssuer

- [x] Create localstack.py task for AWS simulation
  - Requires: k3d.ready
  - Provides: endpoint, region, aws_access_key, aws_secret_key
  - Deploy LocalStack
  - Configure AWS services
  - Return endpoint URLs

- [x] Create registry.py task for container registry
  - Requires: k3d.ready
  - Provides: host, url, username, password
  - Deploy Docker registry
  - Configure storage and auth
  - Return registry URLs

- [x] Test parallel execution of base infrastructure tasks
  - Verify k3d runs first (Level 1)
  - Verify all other tasks run in parallel (Level 2)
  - Confirm no race conditions
  - Validate performance improvement

**Acceptance Criteria**:
- ✅ k3d cluster created before all other tasks
- ✅ All base services deploy successfully
- ✅ Services run in parallel after k3d (Level 2 execution)
- ✅ Each service provides correct contract values
- ✅ Retry logic handles transient failures
- ✅ Log files contain useful debugging information

### Phase 5: Auth Stack Migration (Week 3) ✅
**Goal**: Migrate authentication services with Keycloak for identity and access management

#### Tasks (COMPLETED)
- [x] Create external-secrets.py task
  - Requires: localstack.endpoint, k3d.ready
  - Provides: ready, cluster_secret_store_ready
  - Deploy external-secrets operator
  - Configure AWS SecretStore for LocalStack
  - Create ClusterSecretStore

- [x] Create keycloak-operator.py task
  - Requires: cert-manager.ready, k3d.ready
  - Provides: operator_ready, operator_namespace
  - Deploy Keycloak operator using YAML manifests
  - Create operator namespace
  - Install CRDs for Keycloak resources
  - Wait for operator deployment

- [x] Create keycloak.py task with postgres dependency
  - Requires: postgres.dsn, external-secrets.ready, keycloak-operator.operator_ready, cert-manager.ready
  - Provides: base_url, admin_url, realm_name, client_id, client_secret
  - Deploy Keycloak instance using operator CR
  - Configure database connection
  - Create default realm (catalyst-forge)
  - Set up admin credentials
  - Configure TLS with cert-manager
  - Create OAuth2/OIDC clients for services (Gitea, ArgoCD, etc.)

- [x] Verify auth stack dependency chain executes correctly
  - Test Level 2 execution (postgres, localstack, cert-manager)
  - Test Level 3 execution (external-secrets, keycloak-operator)
  - Test Level 4 execution (keycloak)

**Acceptance Criteria**:
- ✅ Keycloak operator deploys successfully
- ✅ Keycloak instance provisions with database backend
- ✅ OAuth2/OIDC configuration ready for services
- ✅ Services wait for dependencies before starting
- ✅ Contract values flow correctly between tasks
- ✅ Failed dependencies prevent dependent tasks from running

### Phase 6: Complex Service Migration (Week 3-4)
**Goal**: Migrate services with complex configuration requirements

#### Missing Infrastructure Services
- [ ] Create trust-manager.py task
  - Requires: cert-manager.ready
  - Provides: ready, ca_bundle_ready
  - Deploy trust-manager for cert distribution
  - Configure mkcert CA bundle

- [ ] Create envoy-gateway.py task
  - Requires: cert-manager.ready
  - Provides: gateway_ready, gateway_class, gateway_namespace
  - Deploy Envoy Gateway
  - Configure GatewayClass
  - Set up Gateway resources
  - Configure HTTPRoutes and TCPRoutes

- [ ] Create mailpit.py task for email testing
  - Requires: envoy-gateway.gateway_ready
  - Provides: smtp_host, smtp_port, web_url
  - Deploy Mailpit email catcher
  - Configure SMTP service
  - Set up web UI route

#### Complex Application Services
- [ ] Create gitea.py task combining Helm + configuration logic
  - Requires: postgres.dsn, keycloak.client_id, external-secrets.ready, envoy-gateway.gateway_ready
  - Provides: url, admin_token, ssh_port
  - Deploy Gitea with Helm
  - Configure database connection via ExternalSecret
  - Set up OAuth2/OIDC with Keycloak
  - Configure repository creation
  - Set up webhook configuration

- [ ] Create argocd.py task
  - Requires: envoy-gateway.gateway_ready, external-secrets.ready, keycloak.client_id
  - Provides: url, admin_password, api_url
  - Deploy ArgoCD
  - Configure RBAC
  - Set up repositories
  - Configure SSO with Keycloak OIDC

- [ ] Create temporal.py task
  - Requires: postgres.dsn, external-secrets.ready, envoy-gateway.gateway_ready
  - Provides: frontend_url, grpc_endpoint, web_ui_url
  - Deploy Temporal workflow engine
  - Configure default and visibility stores
  - Set up namespaces
  - Use ExternalSecrets for database credentials

**Acceptance Criteria**:
- [ ] Complex services deploy with all configurations
- [ ] Gitea has OAuth, repos, and webhooks configured
- [ ] ArgoCD loads large value files correctly
- [ ] All Python configuration logic successfully ported
- [ ] Services are fully functional after deployment

### Phase 7: Testing and Validation (Week 4)
**Goal**: Ensure system works correctly and meets performance targets

#### Tasks
- [ ] Test full setup from clean cluster
  - Run `uv run cli setupv2` on fresh k3d cluster
  - Verify all services come up
  - Test service interconnections

- [ ] Verify parallel execution reduces time from ~15min to ~5min
  - Measure baseline with old system
  - Time new system execution
  - Document performance improvements

- [ ] Test retry logic with simulated failures
  - Introduce transient failures
  - Verify retry delays work
  - Confirm eventual success

- [ ] Test --only flag with various task combinations
  - Test single task execution
  - Test multiple task selection
  - Verify dependency inclusion

- [ ] Verify all log files are created with proper content
  - Check log directory structure
  - Verify command logging
  - Confirm error capture

- [ ] Test circular dependency detection
  - Create tasks with circular deps
  - Verify error message clarity
  - Confirm execution stops

**Acceptance Criteria**:
- ✅ Full setup completes in ~5 minutes (3x improvement)
- ✅ All retry scenarios work correctly
- ✅ Selective execution maintains dependencies
- ✅ Logs provide actionable debugging information
- ✅ Circular dependencies are detected and reported

### Phase 8: Migration Finalization (Week 5)
**Goal**: Switch from V1 to V2 as the default system

#### Tasks
- [ ] Update 'just up' command to use setupv2
  - Modify justfile
  - Update command invocation
  - Test full workflow

- [ ] Add deprecation notice to old setup system
  - Add warning messages
  - Document migration path
  - Set removal date

- [ ] Update documentation for platform engineers
  - Create setup guide
  - Document contract system
  - Provide troubleshooting guide

- [ ] Create example task template for engineers
  - Simple infrastructure example
  - Complex service example
  - Best practices guide

- [ ] Remove old Helmfile after validation period
  - Archive old configuration
  - Clean up unused files
  - Update CI/CD pipelines

**Acceptance Criteria**:
- ✅ V2 is the default setup system
- ✅ V1 shows deprecation warnings
- ✅ Documentation is complete and accessible
- ✅ Platform engineers can easily add new services
- ✅ Old system can be safely removed

## Parallel Execution Opportunities

### Within Phases
- **Phase 1**: Task decorator and dependency resolution can be developed in parallel
- **Phase 2**: Helm, Kubectl, and HTTP helper classes can be developed independently
- **Phase 4**: All base infrastructure tasks execute in parallel at runtime
- **Phase 6**: Individual service migrations can be worked on simultaneously

### Between Phases
- **Phase 6 & 7**: Testing can begin as soon as individual services are ready
- **Phase 8 Documentation**: Can start during Phase 7 testing

## Risk Mitigation

### Technical Risks
1. **NetworkX dependency**: Already decided, well-tested library
2. **Threading vs Async**: Decision made for threading due to subprocess GIL release
3. **Migration complexity**: Both systems coexist during transition

### Process Risks
1. **Single user/alpha status**: Low risk, can iterate quickly
2. **15min → 5min goal**: Parallel execution should easily achieve this
3. **Learning curve**: Simple function-based approach minimizes this

## Service Migration Status

### Completed (SetupV2 Tasks)
- ✅ k3d (cluster creation)
- ✅ postgres (database)
- ✅ cert-manager (certificates)
- ✅ localstack (AWS emulation)
- ✅ registry (container registry)
- ✅ external-secrets (secret management)
- ✅ keycloak-operator (identity platform operator)
- ✅ keycloak (identity and access management)

### Missing from SetupV2 (Still in Helmfile)
- ⏳ trust-manager (cert distribution)
- ⏳ envoy-gateway (API gateway and routing)
- ⏳ mailpit (email testing)
- ⏳ temporal (workflow engine)
- ⏳ gitea (git hosting)
- ⏳ argocd (GitOps deployment)

## Success Metrics Tracking

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Execution Time | 15min → 5min | Time full setup with both systems |
| Debuggability | 100% actionable errors | Review error messages during testing |
| Extension Ease | < 30 min to add service | Time platform engineer adding Istio |
| Code Reduction | Remove 600+ lines bash | Count Helmfile LOC removed |
| Engineer Satisfaction | Positive feedback | Survey after migration |

## Notes for Implementation

1. **Keep It Simple**: Resist adding features not in the PRD
2. **Contracts Over Coupling**: Tasks don't know about each other, only contracts
3. **Fail Fast in Levels**: Let level complete, then stop execution
4. **Idempotency**: All tasks must be safe to re-run
5. **Logging First**: Every action must be logged for debugging

## Post-Implementation Review

After Phase 8 completion, conduct a review to:
- Document lessons learned
- Identify areas for V3 improvements
- Create runbook for common issues
- Plan for production considerations (if ever needed)
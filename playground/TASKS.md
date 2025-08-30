### Playground Tasks: Istio Ambient Mesh and Service Lockdown

This checklist tracks the work to deploy Istio Ambient mode in the playground, enable end-to-end encryption between Envoy Gateway and downstream services, and progressively lock down sensitive services (especially Ory).

#### Conventions
- Check off items as they are completed. Keep items idempotent and safe to re-run.
- Manifests live under `playground/helmfile/platform/...` when possible; automation via `playground/justfile`.

---

## Phase 0 — Prep
- [ ] Add Istio Helm repo to `playground/helmfile/helmfile.yaml` repositories (`istio-release.storage.googleapis.com/charts`).
- [ ] Decide monitoring approach (Kiali/Prometheus) for mTLS verification.
- [ ] Validate cluster readiness: k3d up, cert-manager healthy, Envoy Gateway healthy.

## Phase 1 — Install Ambient Mesh (Helmfile)
- [ ] Add `istio/base` release to namespace `istio-system`.
- [ ] Add `istio/istiod` with `profile: ambient` and readiness hooks.
- [ ] Add `istio/cni` with `profile: ambient` (cluster-wide CNI) and readiness hooks.
- [ ] Add `istio/ztunnel` DaemonSet and readiness hooks.
- [ ] Optional: postsync apply of monitoring stack (Kiali/Grafana/Prometheus).

## Phase 2 — Enroll Namespaces into Ambient
- [ ] Label participating namespaces with `istio.io/dataplane-mode=ambient`:
  - [ ] `envoy-gateway-system`
  - [ ] `auth`
  - [ ] `postgres`
  - [ ] `temporal`
  - [ ] `registry`
  - [ ] `mailpit`
- [ ] Exclude infra namespaces (e.g., `kube-system`).

## Phase 3 — Verify mTLS E2E
- [ ] Confirm `ztunnel` running on all nodes.
- [ ] Verify traffic from Envoy Gateway to backends shows mTLS (Kiali padlocks or logs/metrics).
- [ ] Sanity test: reach Kratos/Hydra routes through Envoy and confirm normal behavior under Ambient.

## Phase 4 — Baseline L4 Authorization Policies (Ory-first)
- [ ] Create `playground/helmfile/platform/istio/policies/` directory.
- [ ] Kratos public: allow only Envoy Gateway SA to call public port.
- [ ] Hydra admin: allow only Oathkeeper (and required internal SAs) to call admin port.
- [ ] Oathkeeper API: allow only Envoy Gateway SA to call external-facing API.
- [ ] Apply policies via Helmfile or `kubectl` postsync hooks.

## Phase 5 — L7 Controls via Waypoints (Selective)
- [ ] Decide waypoint scope (per SA vs per namespace) for `auth` services.
- [ ] Create waypoint(s) for `kratos`, `hydra`, and `oathkeeper` service accounts.
- [ ] Add L7 AuthorizationPolicies (paths/methods) behind waypoints as needed.
- [ ] Validate flows continue to work; tighten progressively.

## Phase 6 — Automation in Justfile
- [ ] Add `mesh:install` task to install/upgrade Ambient components via Helmfile.
- [ ] Add `mesh:enroll` task to label target namespaces (idempotent).
- [ ] Add `mesh:monitoring` task to apply monitoring stack.
- [ ] Add `mesh:policies` task to apply base L4 policies.
- [ ] Add `mesh:waypoints` task to apply waypoint manifests.

## Phase 7 — Tests and Safeguards
- [ ] Add positive tests: allowed callers to Ory services still succeed.
- [ ] Add negative tests: disallowed callers are blocked by L4/L7 policies.
- [ ] Document rollback (remove labels, uninstall Ambient components) and recovery steps.

## Documentation
- [ ] Update `playground/README.md` with Ambient setup, commands, and troubleshooting.
- [ ] Document Ory policy model and identities (service accounts) in `playground/OIDC.md` or a new `SECURITY.md`.



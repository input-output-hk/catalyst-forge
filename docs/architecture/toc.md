### **Part I — Conceptual (Narrative Overview)**

1. **What is Catalyst Forge? (One‑pager)**

   * Elevator pitch, problems we solve, measurable outcomes (KPIs)
   * What Forge is / is not (scope boundaries)

2. **Personas & Responsibilities**

   * Product manager, repo maintainer, application engineer, platform engineer, operator/SRE, security
   * What each persona does in Forge (at a glance)

3. **Core Concepts (Mental Model)**

   * Project, Blueprint, Module, Artifact, Release, Rendered Release, Environment, Deployment, GitOps Action/Change/Sync, Trace
   * Simple “term → 1‑line definition” with links (full Glossary remains in Appendix)

4. **From Commit to Cluster (Lifecycle)**

   * High‑level diagram & story: CI → Publish → Release → Render → GitOps → Sync
   * How promotion works across environments

5. **Day‑in‑the‑Life Scenarios**

   * “I merged code” → what happens
   * “I want to promote to staging/prod”
   * “I need to roll back”
   * “I need to see why my change isn’t live”

6. **Guardrails & Principles**

   * Security, immutability, reproducibility, auditability
   * Opinionated conventions (Earthly target groups, repo layout) and how/when to override

7. **What You Get Out‑of‑the‑Box**

   * CLI, web UI, API, remote runners, renderers, operator, GitOps integration
   * Supported ecosystems (Argo CD, Earthly, registries)

8. **Limits & Non‑Goals (v2)**

   * Clear list to set expectations

> *End of PM track. Engineers can stop here or continue to Part II.*

---

### **Part II — Reference Architecture (for Engineers & Platform Devs)**

A. **System Context & Dependencies**

* C4‑L1/L2 style context and container diagrams
* External systems (GitHub, registries, Argo CD, AWS OIDC, runners, Tailscale)

B. **Component Responsibilities & Interfaces**

* Forge API, Operator, Renderer (gRPC), Web Frontend, Remote Runner, CLI
* For each: responsibility, inputs/outputs, failure modes, scaling notes

C. **Domain Model & Persistence**

* Entities and relationships: Repository, Project, Blueprint, Artifact, Module, Module Bundle, Release, Rendered Release, Deployment, Environment, GitOps Action/Change/Sync, Trace
* Where each entity is persisted (DB vs OCI vs Git), immutability rules, identifiers

D. **Workflows (Sequence Diagrams)**

* CI phases (check/build/test/package)
* Publish → Release creation
* Render (dry‑run vs OCI packaged)
* Deployment → GitOps update → Argo Sync → status collection
* Rollback

E. **Configuration & Conventions**

* `blueprint.cue` schema (Root & Project) and examples
* Earthly target grouping and overrides
* Environment definitions and value layering
* GitOps repo layout conventions

F. **APIs & Contracts**

* Public API (HTTP/gRPC) with request/response examples
* CLI commands (FORGE) mapped to API calls
* Event schemas (for publish/release/deploy/sync) and idempotency/locking rules

G. **Rendering Architecture**

* Supported module types (KCL, Helm, git), value resolution, composition
* Rendered Release format, OCI packaging, provenance metadata

H. **Security Architecture**

* AuthN (SSO/OIDC, tokens), AuthZ (roles, scopes), multi‑tenancy boundaries
* Secrets access patterns, least‑privilege for CI/runners/operator
* Supply chain controls (SBOM, attestations, digest pinning)

I. **Observability & Operations**

* Trace propagation, metrics, logs; dashboards; SLOs
* Operating the platform (deploy/upgrade/backup), scaling guidance
* Failure modes & recovery runbooks

J. **Validation & Acceptance**

* Checklists per component and per workflow
* Conformance tests for module renderers and publishers

K. **Migration & Compatibility Notes**

* v1 → v2 (if applicable), deprecations, feature flags


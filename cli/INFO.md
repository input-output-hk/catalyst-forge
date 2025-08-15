Below is a **concise, high‑level end‑to‑end overview** of the Catalyst Forge v2 flow. It’s written so an LLM (with no prior context) can reason about the system and help build components.

---

## 1) Actors & storage

* **GitHub** (source + CI)
* **Earthly** (build engine; remote cache/runner)
* **Forge API** (CRUD for Releases/Deployments/etc.; persistence in Postgres)
* **OCI Registry** (stores: container images, Release Bundles, Rendered Sets)
* **Renderer** (gRPC service: Release + env → manifests or Rendered Set OCI)
* **GitOps repo** (env overlays + tiny pointer files)
* **Argo CD** (pulls pointers, applies manifests)
* **Kubernetes** (target clusters/environments)

---

## 2) What lives where

* **Blueprint (in repo)**: declares *projects*, *modules*, *base values*, and *artifact build targets*.
* **Release Bundle (OCI)**: immutable metadata object that binds a commit + selected artifacts + module lock + base values integrity + **injection map**.
* **Env overlay (`env.cue`, in GitOps)**: per‑environment overrides; never contains images.
* **Pointer files (in GitOps)**:

  * `release.ref` → Release Bundle digest
  * `rendered.ref` (optional) → Rendered Set digest

---

## 3) End‑to‑end sequence (merge/tag → healthy)

```
Developer → GitHub → CI/Earthly → Forge API ↔ Postgres → OCI Registry
                                             ↓
                                     Release Bundle (OCI)
                                             ↓
                        (Promotion) → GitOps pointer PR (release.ref or rendered.ref)
                                             ↓
                           Argo CD CMP/sidecar → Renderer (if needed) → Rendered Set OCI
                                             ↓
                                     Kubernetes apply/sync
                                             ↓
                                      Health reported back
```

---

## 4) Build & Release creation (CI path)

1. **Trigger**

   * On **merge to default** and/or **tag**, CI starts and creates a **Trace** row (correlation id).

2. **Project discovery & build**

   * CI scans the monorepo for **projects** (blueprints).
   * For each project that matches the event, Earthly runs the configured targets (check/build/test/package).
   * Images are built and pushed (multi‑arch as OCI index). **Artifacts** are recorded (name + digest + kind).

3. **Release minting**

   * CI loads the project’s **base values** from the blueprint (no env overlay).
   * CI scans values for `@artifact(name, field)` attributes and produces an **Injection Map** of JSON Pointers.
   * CI resolves the **Module Lock** (each module’s type/version + OCI digest or git ref).
   * CI computes **ValuesHash** and **ContentHash**; optional **ValuesSnapshot** (normalized base values).
   * CI writes a **Release** row and publishes a signed **Release Bundle (OCI)** with:

     * `artifacts{artifact_id → image_name, image_digest, …}`
     * `modules[]` (lock)
     * `values.{hash, snapshot?, source pointer}`
     * `injections[]` (json\_pointer → artifact\_key/field)
   * Release is **sealed** (immutable) and its OCI digest stored.

> Result: An immutable Release that captures “exact assets + exact plan intent”.

---

## 5) Promotion & Deployment

4. **Create Deployment**

   * A user or automation creates a **Deployment** (Release → Environment).
   * Control plane opens/merges a **GitOps PR** that updates a **pointer**:

     * **Option A (pre‑render)**: call Renderer now, publish **Rendered Set (OCI)**, write `rendered.ref`.
     * **Option B (render at sync)**: write `release.ref`; Argo will invoke the Renderer.

5. **Argo sync**

   * Argo’s **CMP/sidecar** reads the pointer(s):

     * If `rendered.ref` exists: pull Rendered Set OCI, verify signature, emit YAML.
     * Else: pull Release Bundle OCI, verify signature, call **Renderer** to get YAML.
   * Argo applies manifests; cluster reconciles.

6. **Health & status**

   * Argo reports sync/health; platform records an **ArgoSync** snapshot and updates the **Deployment** status (healthy/degraded/failed).

---

## 6) Rendering (how values become manifests)

* **Inputs** to Renderer:

  * Release Bundle (OCI) → `artifacts`, `modules`, `values`, `injections`
  * Env overlay (`env.cue` from GitOps)
  * Context: `env`, `project`, `trace_id`

* **Algorithm**:

  1. Acquire base values (from **snapshot** or by refetching repo\@commit and verifying **ValuesHash**).
  2. Apply **Injection Map** (fill JSON Pointer paths with values from `artifacts[name].[field]`).
  3. Merge **env overlay** (closed schema; image paths forbidden).
  4. For each module (KCL/Helm), pass the **final values** (engine‑agnostic single input).
  5. Emit manifests and either:

     * return YAML (legacy path), or
     * package + sign a **Rendered Set (OCI)** and return its digest/ref.

* **Determinism key**: the **intent hash** includes release content, module versions, normalized values, env, and renderer version.

---

## 7) GitOps repo shape (small, stable)

```
k8s/
  dev|preprod|prod/
    <project>/
      env.cue         # overlay (no images)
      release.ref     # oci://…@sha256:…  (Release Bundle)
      rendered.ref    # oci://…@sha256:…  (Rendered Set, optional if pre-rendered)
```

* **Promotion** = change one of these pointers via PR (reviewable, auditable).
* **Rollback** = revert the pointer to a previous digest.

---

## 8) Security & policy

* **Signing**

  * CI signs **images** and **Release Bundles** (cosign keyless via GH OIDC).
  * Renderer signs **Rendered Sets**.

* **Pre‑merge checks** (optional, recommended)

  * All images in the Release are signed by the trusted issuer.
  * Env overlay validates against a **closed schema** (no image edits, no privileged, resource caps, etc.).

* **Admission** (cluster)

  * Enforce “images must be signed” in dev/preprod/prod (progressively).
  * Optionally verify Rendered Set signature/annotations.

---

## 9) Traceability & telemetry

* A **Trace ID** is created at CI start and propagated to: image labels, Release Bundle, Rendered Set, GitOps commits, Argo annotations, and K8s object labels.
* The API stores: Builds, Artifacts, Releases (+ Modules/Injections), Deployments, RenderJobs, GitOpsChanges, ArgoSyncs—each linked by ids and the trace.
* The UI can answer: *“PR → build → artifacts → release → pointer PR → Argo → K8s health/logs.”*

---

## 10) CRUD APIs (to support the flow)

* **Releases**: create/list/get/update/delete; manage modules, injections, and linked artifacts.
* **Deployments**: create/list/get/update/delete; 1:1 **RenderJob**; attach **GitOpsChanges**.
* **Artifacts/Environments/Projects**: list/get (and CRUD where appropriate).

**Route examples (prefix `/api/v1`)**:

```
POST /releases ; GET /releases ; GET /releases/:id ; PATCH /releases/:id ; DELETE /releases/:id
GET  /releases/:id/modules ; POST /releases/:id/modules ; PATCH /releases/:id/modules/:moduleKey ; DELETE …
GET  /releases/:id/injections ; POST /releases/:id/injections ; DELETE /releases/:id/injections/:injID
GET  /releases/:id/artifacts ; POST /releases/:id/artifacts ; DELETE /releases/:id/artifacts/:artifactId

POST /deployments ; GET /deployments ; GET /deployments/:id ; PATCH /deployments/:id ; DELETE /deployments/:id
GET  /deployments/:id/render-job ; POST /deployments/:id/render-job ; PATCH /deployments/:id/render-job
GET  /deployments/:id/gitops-changes ; POST /deployments/:id/gitops-changes
```

---

## 11) Failure & retry (high level)

* **CI**: retryable; Release creation is idempotent by `(project_id, release_key)` or `content_hash`.
* **Renderer**: cache by **intent hash**; safe to re‑invoke.
* **GitOps PR**: normal PR lifecycle; reconcile conflicts; on merge, Argo picks up.
* **Argo**: sync status polled; platform records transitions; failed → visible to user.
* **Immutability**: sealed Releases cannot change key fields (enforced by service + DB trigger later).

---

### Mental model (one line)

**“Code change → Build artifacts → Release Bundle (what & how) → Promotion by updating a pointer in Git → Renderer → Argo → Cluster.”**

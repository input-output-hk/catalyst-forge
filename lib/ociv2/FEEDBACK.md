Absolutely—splitting the concerns cleanly will keep your core OCI utilities **domain‑agnostic** and let `forge-kcl` own all **KCL/KPM‑specific** behaviors.

Below is a pragmatic breakdown of **(A) generalized helpers** that belong in `pkg/oci` and **(B) what `forge-kcl` should implement** on top.

---

## A) What to add (or keep) in `pkg/oci` — **generalized, domain‑agnostic**

These helpers should not know anything about KCL/KPM. They provide a stable, ergonomic layer over ORAS/ggcr/cosign, plus a few safe utilities.

### 1) Ref + Registry utilities

* `Resolve(ctx, ref) (Descriptor, error)` — digest‑first canonicalization.
* `Head(ctx, ref) (Descriptor, error)` — metadata without pulling blobs.
* `ListTags(ctx, repo) ([]string, error)` and `LatestSemverTag(ctx, repo) (string, error)` — optional, but very handy.

### 2) Pack & Push (manifest‑agnostic)

* **Core builder API**:

  ```go
  type LayerSpec struct {
    MediaType    string
    Title        string            // optional: sets OCI title annotation for the layer
    Annotations  map[string]string // extra layer annotations
    Size         int64
    Reader       io.Reader
  }

  type PackOptions struct {
    ArtifactType        string            // required for artifact manifests
    ManifestAnnotations map[string]string // OCI manifest annotations (string map)
    Config              []byte            // optional config payload
    ConfigMediaType     string            // media type for config
    Layers              []LayerSpec       // one or more blobs
    PreferArtifactManifest bool           // try OCI 1.1 artifact manifest
    FallbackImageManifest bool           // fallback to image manifest if needed
  }

  func (c *client) PushArtifact(ctx context.Context, ref string, opts PackOptions) (Descriptor, error)
  ```

  * Handles **artifact manifest v1.1** and **image manifest fallback** automatically.
  * Returns the **manifest digest** as `Descriptor.Ref = "<repo>@sha256:..."`.

* **JSON & TAR conveniences** (already sketched, keep them):

  ```go
  PushJSON(ctx, ref, mt string, payload []byte, ann Annotations) (Descriptor, error)
  PushTar(ctx, ref string, cfg []byte, cfgMT, layerMT string, tar io.Reader, size int64, ann Annotations) (Descriptor, error)
  ```

### 3) Pull (manifest‑agnostic)

* **Structured pull**:

  ```go
  type PulledLayer struct {
    MediaType   string
    Size        int64
    Annotations map[string]string
    Open        func() (io.ReadCloser, error) // lazily stream the blob
  }

  type PullResult struct {
    Descriptor       Descriptor
    ArtifactType     string          // if artifact manifest
    Config           []byte          // nil if none
    ConfigMediaType  string
    Layers           []PulledLayer
    ManifestAnn      map[string]string
  }

  func (c *client) PullArtifact(ctx context.Context, ref string) (PullResult, error)
  ```

  * No KCL assumptions—just returns what’s there.
  * Keep simple helpers for common cases:

    ```go
    PullJSON(ctx, ref, wantMediaType string) ([]byte, Descriptor, error)
    PullTar(ctx, ref, layerMT string) (io.ReadCloser, Descriptor, error)
    ```

### 4) Signing & Attestations (KMS‑centric)

* **Canonical signing** for the **manifest digest**:

  ```go
  func (c *client) SignDigest(ctx context.Context, refOrDigest string) (Descriptor, error)
  func (c *client) VerifyDigest(ctx context.Context, refOrDigest string) (*VerificationReport, error)
  ```

* **Generic DSSE attestation attach/verify**:

  ```go
  func (c *client) Attest(ctx context.Context, subjectRef, predicateType string, predicate []byte) (Descriptor, error)
  func (c *client) VerifyAttestations(ctx context.Context, subjectRef string, wantTypes []string) ([]AttestationReport, error)
  ```

  * Subject is **the same digest** you just pushed/signed.
  * Attestations are format‑agnostic (SLSA or your own predicate).

### 5) Generic validation helpers (policy‑lite, not KCL‑specific)

* **Shape validation** without opinion on domain:

  ```go
  type ArtifactValidationSpec struct {
    RequireArtifactType      string            // empty = don’t check
    RequireConfigMediaType   string            // optional
    AllowedLayerMediaTypes   []string          // OR’d
    RequireLayerCount        *struct{Min,Max int}
    RequireManifestAnn       map[string]bool   // keys that must exist
    EqualManifestAnn         map[string]string // key=value pairs to enforce
    // Layer digest or raw-bytes hashing checks (e.g., sha256 in an annotation)
    LayerChecksumAnnotation  string            // annotation key holding expected sum
    LayerChecksumAlgo        string            // "sha256"
    LayerIndex               int               // which layer to hash (default 0)
  }

  type ArtifactValidationReport struct {
    OK         bool
    Failures   []string
    Descriptor Descriptor
  }

  func (c *client) ValidateArtifact(ctx context.Context, ref string, spec ArtifactValidationSpec) (ArtifactValidationReport, error)
  ```

  * Lets `forge-kcl` compose **KPM‑compat checks** (one tar layer, artifactType=tar, checksum annotation), or your **Forge‑strict** checks (custom artifactType, meta‑blob existence), without baking those rules into `pkg/oci`.

### 6) Minimal tar utilities (generic, safe)

Keep them **generic**, not KCL‑aware:

* `tarutil.WriteDeterministic(root string, include, exclude []string, w io.Writer) (sha256Hex string, err error)`

  * Stable ordering, fixed mtimes/ownership, normalized modes.
* `tarutil.ExtractSafe(r io.Reader, dest string) error`

  * Guards against path traversal, device files, oversized files, etc.

> These are broadly useful across the platform (not just KCL modules).

---

## B) What belongs in **`forge-kcl`** — **KCL/KPM‑specific logic**

This CLI (or library) owns everything specific to **KCL modules** and your chosen **profile(s)** (KPM‑compat and/or Forge‑strict). It should depend on `pkg/oci` for all registry & signing work.

### 1) Packaging rules (KCL semantics)

* Read `kcl.mod` to determine **include/exclude** lists and **entry** file.
* Build the module **tar** using `tarutil.WriteDeterministic`.
* Compute and store the **module checksum** of the tar (sha256).

### 2) Metadata & annotations (KPM & Forge profiles)

* **KPM‑compat profile**:

  * Manifest: **OCI 1.1 artifact manifest**
  * `artifactType = "application/vnd.oci.image.layer.v1.tar"`
  * **One layer**: the module tar, same layer mediaType.
  * Manifest annotations: **name**, **version**, **description**, **checksum** (KPM keys), and OCI `title` = basename of the tar.
* **Forge‑strict profile** (internal preferred):

  * Manifest: **OCI 1.1 artifact manifest**
  * `artifactType = "application/vnd.projectcatalyst.kcl.module.v1+tar"`
  * **Two blobs**:

    * Blob 1 — module tar (`application/vnd.projectcatalyst.kcl.module.v1+tar`)
    * Blob 2 — **meta.json** (`application/vnd.projectcatalyst.kcl.module.meta.v1+json`) with:

      * `module.name`, `version`, `entry`, `kclVersion`
      * `source.repo`, `commit`
      * `pack.checksum`, `include[]`, `exclude[]`
      * optional `schema.values` (JSONSchema)
  * Manifest annotations (small set): `source`, `revision`, `created`, `title`.

> `forge-kcl` selects the profile (`--compat` | `--strict`) and **constructs the correct `PackOptions`** for `pkg/oci.PushArtifact`.

### 3) Signing & attestation policy (KCL flavor)

* After push, call `oci.SignDigest(...)` with your **KMS key alias**.
* Optionally attach a **DSSE/SLSA attestation** that records **GitHub OIDC** (from CI) or **publisher identity**:

  * Subject = module digest
  * Predicate includes repo/ref/sha, run id, module checksum and entry, kcl version.

### 4) Validation policy (KCL)

* For **pull/verify**:

  * Choose the profile you expect (compat/strict) and build a `ArtifactValidationSpec`:

    * ✔ **Compat**: `RequireArtifactType=tar`, `RequireLayerCount=1`, `AllowedLayerMediaTypes=[tar]`, `RequireManifestAnn={name,version,checksum}`, `LayerChecksumAnnotation=<KPM_CHECKSUM_KEY>`.
    * ✔ **Strict**: `RequireArtifactType=…kcl.module.v1+tar`, `RequireLayerCount=2`, `AllowedLayerMediaTypes=[…kcl.module.v1+tar, …kcl.module.meta.v1+json]`, `RequireManifestAnn={title}`, plus meta blob existence.
  * Verify **KMS signature** using your **trust set** (CI/renderer key IDs).
  * (Optional) Verify **attestation** and **GitHub JWT** embedded inside.
* On success, **extract** the module tar to a **cache dir** keyed by **digest** (e.g., `~/.forge/kcl/modules/sha256-.../`).

### 5) Running `kcl`

* Provide `forge-kcl run` to:

  * **pull → verify → extract** (if not cached)
  * Invoke `kcl run -Y ...` or equivalent with `--path <extracted-dir>/<entry>` and pass values file/JSON.
* Provide `forge-kcl inspect` to print module meta (from compat annotations or strict meta.json).
* Provide `forge-kcl verify` to run verification without extraction.

### 6) Tagging & resolution policy (semver)

* `forge-kcl publish --tag v0.11.1` pushes with a tag, but writes/returns the **digest**, and logs both.
* `forge-kcl resolve <repo>` can use `oci.LatestSemverTag` or your **module index** artifact:

  * (Optional) Implement a **module index** OCI artifact (strict JSON mapping semver → digest) to avoid tag listing; update it on publish.

### 7) CLI surface (example)

```
forge-kcl pack     [--compat|--strict] --root ./module --out ./module.tar
forge-kcl publish  [--compat|--strict] --root ./module --ref oci://ghcr.io/org/module --tag v0.11.1 --sign-key awskms://alias/forge-ci
forge-kcl verify   --ref oci://ghcr.io/org/module@sha256:... [--profile compat|strict]
forge-kcl pull     --ref ... --dest /tmp/out [--profile ...]
forge-kcl run      --ref ... --values values.yaml [--profile ...] [--kcl-args ...]
forge-kcl inspect  --ref ...
forge-kcl index    --repo oci://ghcr.io/org/module-index  # optional index management
```

---

## Why this split works

* **`pkg/oci` stays timeless**: pure OCI + signing + validation primitives you’ll reuse for Releases, Rendered Sets, SBOMs, etc.
* **`forge-kcl` moves fast**: you can tweak KPM‑compat vs Forge‑strict profiles, add meta/schema, and control how you run `kcl` without touching the shared OCI layer.
* **Security stays consistent**: single **KMS signature** policy, optional attestations with GitHub OIDC evidence, and shape validation per profile.

If you’d like, I can draft the `PackOptions` calls for both profiles and a `forge-kcl publish` implementation sketch (Go) that wires `tarutil → oci.PushArtifact → oci.SignDigest → oci.Attest`.

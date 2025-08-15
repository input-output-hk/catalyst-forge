Below is a **self‑contained `pkg/oci` package design** you can drop into your repo. It gives you a clean, testable API to **push/pull** OCI artifacts (Release Bundles, Rendered Sets, generic blobs), plus **sign/verify** with Cosign keyless, while hiding registry differences (ECR/GHCR, artifact vs image manifests). It follows **Go option patterns**, uses **contexts & deadlines**, supports **digest‑first semantics**, and keeps **domain out** of the package (no Forge types inside).

> **Dependencies (recommended):**
>
> * Registry I/O: `oras.land/oras-go/v2` (primary) and `github.com/google/go-containerregistry` (fallback & helpers)
> * Signing: `github.com/sigstore/cosign/v2`
> * AWS ECR auth (optional): `github.com/google/go-containerregistry/pkg/authn` + your IRSA flow or AWS SDK if you need it

---

## 1) Package layout

```
pkg/oci/
  client.go        // public interfaces + constructor
  refs.go          // parsing/validation helpers for oci refs
  types.go         // media types, descriptors, error vars
  push_pull.go     // push/pull Release Bundle & Rendered Set helpers
  sign_verify.go   // cosign signing/verification wrappers
  auth.go          // registry auth wiring (docker config, static, ecr helper hook)
  annotations.go   // well-known annotation keys + helpers
  internal/
    orasx.go       // ORAS helpers (artifact vs image manifest fallback)
    ggcrx.go       // ggcr helpers (layers, descriptors)
```

> Keep **Forge domain** (Release JSON, Rendered Set index) outside. This package just moves bytes with the correct **media types** and annotations and returns **digests**.

---

## 2) Public API (concise)

### 2.1 Options & client

```go
// pkg/oci/client.go
package oci

import (
  "context"
  "io"
  "time"
)

type AuthProvider interface {
  // Returns an authn.Authenticator (ggcr) or nil for anonymous.
  Authenticator(registry string) (any, error)
}

type CosignOpts struct {
  Enable         bool
  RekorURL       string            // optional (use default if empty)
  FulcioURL      string            // optional
  Identity       *OIDCIdentity     // optional: enforce issuer/subject on verify
  AllowInsecure  bool              // local dev only
}

type OIDCIdentity struct {
  Issuer  string // e.g., https://token.actions.githubusercontent.com
  Subject string // e.g., repo:org/repo:ref:refs/heads/main or workload identity
}

type ClientOptions struct {
  // Networking
  PlainHTTP         bool          // for local registries only
  Timeout           time.Duration // default 2m
  UserAgent         string

  // Auth & signing
  Auth              AuthProvider  // docker config by default (can be nil)
  Cosign            CosignOpts

  // Registry compatibility
  PreferArtifactManifest bool // try OCI artifact manifest first
  FallbackImageManifest  bool // fallback to image manifest if registry rejects artifacts

  // Observability
  Logger func(msg string, kv ...any) // optional structured logger
}

type Client interface {
  // -------- Generic primitives --------
  Resolve(ctx context.Context, ref string) (Descriptor, error)
  Head(ctx context.Context, ref string) (Descriptor, error)

  // Put/Pull a single JSON blob as an artifact (config-only or small layer)
  PushJSON(ctx context.Context, ref string, mediaType string, payload []byte, ann Annotations) (Descriptor, error)
  PullJSON(ctx context.Context, ref string, wantMediaType string) ([]byte, Descriptor, error)

  // Put/Pull a TAR(GZ) payload as a layer with a JSON config.
  // cfgMT is the config media type (often your custom type or empty-json type).
  // layerMT is your layer type (e.g. Rendered Set tarball).
  PushTar(ctx context.Context, ref string, cfg []byte, cfgMT, layerMT string, tar io.Reader, size int64, ann Annotations) (Descriptor, error)
  PullTar(ctx context.Context, ref string, layerMT string) (io.ReadCloser, Descriptor, error)

  // -------- Release Bundle helpers (thin wrappers over JSON) --------
  PushReleaseBundle(ctx context.Context, ref string, releaseJSON []byte, ann Annotations) (Descriptor, error)
  PullReleaseBundle(ctx context.Context, ref string) ([]byte, Descriptor, error)

  // -------- Rendered Set helpers (tar+json) --------
  PushRenderedSet(ctx context.Context, ref string, indexJSON []byte, tar io.Reader, size int64, ann Annotations) (Descriptor, error)
  PullRenderedSet(ctx context.Context, ref string) (io.ReadCloser, []byte, Descriptor, error)

  // -------- Signing & verification (cosign) --------
  SignDigest(ctx context.Context, refOrDigest string) (Descriptor, error)
  VerifyDigest(ctx context.Context, refOrDigest string) (*VerificationReport, error)
}

func New(opts ClientOptions) (Client, error)
```

### 2.2 Types & errors

```go
// pkg/oci/types.go
package oci

import "time"

type Descriptor struct {
  Ref        string // canonical ref with @sha256:...
  Digest     string // sha256:...
  Size       int64
  MediaType  string
  Annotations map[string]string
  PushedAt   time.Time
}

// Well-known media types for Forge
const (
  MTReleaseConfig   = "application/vnd.forge.release+json"
  MTRenderedIndex   = "application/vnd.forge.rendered.index.v1+json"
  MTRenderedTarGz   = "application/vnd.forge.rendered.layer.v1.tar+gzip"
  MTOCIEmptyJSON    = "application/vnd.oci.empty.v1+json"         // tiny config
)

// Standard error variables for callers to switch on.
var (
  ErrNotFound       = errors.New("oci: not found")
  ErrUnauthorized   = errors.New("oci: unauthorized")
  ErrInsecureRef    = errors.New("oci: insecure ref")
  ErrMediaType      = errors.New("oci: unexpected media type")
  ErrUnsupported    = errors.New("oci: unsupported")
)

type Annotations map[string]string

// Recommended annotations we’ll set/merge if provided:
const (
  AnnSourceRepo     = "org.opencontainers.image.source"
  AnnSourceRev      = "org.opencontainers.image.revision"
  AnnCreated        = "org.opencontainers.image.created"
  AnnTitle          = "org.opencontainers.image.title"
  AnnDescription    = "org.opencontainers.image.description"

  // Forge-specific (suggested)
  AnnForgeKind      = "io.projectcatalyst.forge.kind"     // release|rendered|sbom|...
  AnnForgeProject   = "io.projectcatalyst.forge.project"
  AnnForgeEnv       = "io.projectcatalyst.forge.env"
  AnnForgeTrace     = "io.projectcatalyst.forge.trace"
  AnnForgeRelease   = "io.projectcatalyst.forge.releaseKey"
)
```

### 2.3 Signing results

```go
// pkg/oci/sign_verify.go
package oci

type VerificationReport struct {
  Digest     string
  Signed     bool
  Signers    []SignerIdentity
  BundleVerified bool   // rekor/fulcio
  Errors     []string
}

type SignerIdentity struct {
  Issuer  string
  Subject string
  SANs    []string // alt names if present
  Time    time.Time
}
```

---

## 3) Behavior & best practices baked in

* **Digest‑first:** After every push, resolve to a canonical `ref@sha256:...` and return that in `Descriptor.Ref`. Callers should persist the **digest** (not tag).
* **Artifact‑first, image‑fallback:** Try pushing with **OCI artifact manifest**; if the registry rejects, optionally **fallback** to an **image manifest** with an empty JSON config + your layer. (Compatibility switch via options.)
* **Small JSON as config, large bytes as layer:**

  * Release Bundle → a **config‑only** artifact (or a tiny layer if needed).
  * Rendered Set → config (index JSON) + **single tar.gz layer** for YAMLs.
* **Annotations:** Merge caller annotations and add basic OCI annotations (source, revision, created).
* **Cosign keyless:** `SignDigest` signs the digest using Cosign (keyless if configured), `VerifyDigest` checks signatures and identity constraints (issuer/subject) if provided.
* **Auth:** Out‑of‑the‑box supports docker config (\~/.docker/config.json) via ggcr. You can pass a custom `AuthProvider` (e.g., your IRSA/ECR flow) per registry hostname.
* **Context & timeouts:** All ops accept contexts; default client timeout is 2 minutes, overridable.

---

## 4) Example usage (end‑to‑end)

```go
cli, _ := oci.New(oci.ClientOptions{
  PlainHTTP: false,
  Timeout:   2 * time.Minute,
  UserAgent: "forge-oci/1.0",
  Cosign: oci.CosignOpts{
    Enable: true,
    Identity: &oci.OIDCIdentity{
      Issuer:  "https://token.actions.githubusercontent.com",
      Subject: "repo:input-output-hk/catalyst-forge:ref:refs/heads/main",
    },
  },
})

// 1) Push a Release Bundle JSON
rb := []byte(`{ "releaseKey":"foundry-operator-007", ... }`)
desc, err := cli.PushReleaseBundle(ctx, "oci://123456.dkr.ecr.eu-central-1.amazonaws.com/forge/releases/foundry-operator", rb, oci.Annotations{
  oci.AnnForgeKind: "release",
})
// desc.Ref => oci://...@sha256:DEADBEEF

// 2) Sign it (keyless)
if err == nil {
  _, _ = cli.SignDigest(ctx, desc.Ref) // no-op if Cosign disabled
}

// 3) Push a Rendered Set (index.json + manifests.tar.gz)
rsTar, size := openTar("rendered.tar.gz")
idx := []byte(`{ "modules":[...], "entries":[ ... ] }`)
rs, err := cli.PushRenderedSet(ctx,
    "oci://123456.dkr.ecr.eu-central-1.amazonaws.com/forge/rendered/foundry-operator/preprod",
    idx, rsTar, size,
    oci.Annotations{ oci.AnnForgeKind:"rendered", oci.AnnForgeEnv:"preprod" })

// 4) Later: Pull & verify
raw, meta, _ := cli.PullReleaseBundle(ctx, desc.Ref)
rep, _ := cli.VerifyDigest(ctx, desc.Ref)
```

---

## 5) Key implementation notes

### 5.1 Pushing JSON (Release Bundle)

* If the registry supports **artifact manifest**, set the **config** to your JSON with `MTReleaseConfig` and **no layers**.
* If it rejects, fallback to **image manifest** with **config = MTOCIEmptyJSON** and **one small layer** that contains your JSON; still annotate manifest with your Forge keys.

### 5.2 Pushing Rendered Set

* **Config = `MTRenderedIndex`** (your `index.json`), **Layer = `MTRenderedTarGz`** (tarred YAMLs).
* The layer digest becomes the **content address** for the payload; the manifest digest (returned) is the stable pointer you publish in Git.

### 5.3 Cosign keyless

* For **GitHub Actions**, use the built‑in OIDC token to sign (Cosign supports this).
* `VerifyDigest` should:

  * Resolve signatures for the digest (Cosign)
  * Confirm **issuer/subject** match your policy if provided
  * Optionally require Rekor bundle presence

---

## 6) Code skeleton (files)

> The following snippets show the **core**. They omit imports and some boilerplate for brevity, but are otherwise ready to flesh out.

### `pkg/oci/client.go`

```go
package oci

import (
  "context"
  "time"
)

type client struct {
  opts ClientOptions
  a    AuthProvider
}

func New(opts ClientOptions) (Client, error) {
  if opts.Timeout == 0 {
    opts.Timeout = 2 * time.Minute
  }
  c := &client{
    opts: opts,
    a:    opts.Auth,
  }
  return c, nil
}

func (c *client) Resolve(ctx context.Context, ref string) (Descriptor, error) {
  return c.headOrResolve(ctx, ref, true)
}

func (c *client) Head(ctx context.Context, ref string) (Descriptor, error) {
  return c.headOrResolve(ctx, ref, false)
}
```

### `pkg/oci/refs.go`

```go
package oci

import (
  "fmt"
  "strings"
)

func isDigestRef(ref string) bool {
  return strings.Contains(ref, "@sha256:")
}

func ensureDigest(ref string, d Descriptor) Descriptor {
  if isDigestRef(ref) {
    d.Ref = ref
  } else if d.Digest != "" {
    d.Ref = fmt.Sprintf("%s@%s", strings.TrimSuffix(ref, ":"), d.Digest)
  } else {
    d.Ref = ref
  }
  return d
}
```

### `pkg/oci/types.go` (excerpt)

```go
package oci

import "errors"

const (
  MTReleaseConfig  = "application/vnd.forge.release+json"
  MTRenderedIndex  = "application/vnd.forge.rendered.index.v1+json"
  MTRenderedTarGz  = "application/vnd.forge.rendered.layer.v1.tar+gzip"
  MTOCIEmptyJSON   = "application/vnd.oci.empty.v1+json"
)

var (
  ErrNotFound     = errors.New("oci: not found")
  ErrUnauthorized = errors.New("oci: unauthorized")
  ErrMediaType    = errors.New("oci: unexpected media type")
  ErrUnsupported  = errors.New("oci: unsupported")
)
```

### `pkg/oci/push_pull.go` (excerpt)

```go
package oci

import (
  "context"
  "io"
)

// ---- Generic JSON ----

func (c *client) PushJSON(ctx context.Context, ref, mt string, payload []byte, ann Annotations) (Descriptor, error) {
  // Try artifact manifest first via ORAS:
  d, err := c.orasPushConfigOnly(ctx, ref, mt, payload, ann)
  if err == nil {
    return ensureDigest(ref, d), nil
  }
  if c.opts.FallbackImageManifest {
    // Fallback: push as image manifest with config empty + JSON as a small layer
    d2, err2 := c.ggcrPushJSONLayer(ctx, ref, mt, payload, ann)
    if err2 == nil {
      return ensureDigest(ref, d2), nil
    }
    return Descriptor{}, err2
  }
  return Descriptor{}, err
}

func (c *client) PullJSON(ctx context.Context, ref, wantMT string) ([]byte, Descriptor, error) {
  // Prefer ORAS artifact pull; fallback to layer fetch via ggcr
  b, d, err := c.orasPullConfig(ctx, ref, wantMT)
  if err == nil {
    return b, ensureDigest(ref, d), nil
  }
  return c.ggcrPullJSONLayer(ctx, ref, wantMT)
}

// ---- TAR (Rendered Set) ----

func (c *client) PushTar(ctx context.Context, ref string, cfg []byte, cfgMT, layerMT string, tar io.Reader, size int64, ann Annotations) (Descriptor, error) {
  d, err := c.orasPushConfigAndLayer(ctx, ref, cfg, cfgMT, tar, size, layerMT, ann)
  if err == nil {
    return ensureDigest(ref, d), nil
  }
  if c.opts.FallbackImageManifest {
    d2, err2 := c.ggcrPushConfigAndLayer(ctx, ref, cfg, cfgMT, tar, size, layerMT, ann)
    if err2 == nil {
      return ensureDigest(ref, d2), nil
    }
    return Descriptor{}, err2
  }
  return Descriptor{}, err
}

func (c *client) PullTar(ctx context.Context, ref, layerMT string) (io.ReadCloser, Descriptor, error) {
  rc, d, err := c.orasPullLayer(ctx, ref, layerMT)
  if err == nil {
    return rc, ensureDigest(ref, d), nil
  }
  return c.ggcrPullLayer(ctx, ref, layerMT)
}

// ---- Convenience ----

func (c *client) PushReleaseBundle(ctx context.Context, ref string, releaseJSON []byte, ann Annotations) (Descriptor, error) {
  if ann == nil { ann = Annotations{} }
  ann[AnnForgeKind] = "release"
  return c.PushJSON(ctx, ref, MTReleaseConfig, releaseJSON, ann)
}

func (c *client) PullReleaseBundle(ctx context.Context, ref string) ([]byte, Descriptor, error) {
  return c.PullJSON(ctx, ref, MTReleaseConfig)
}

func (c *client) PushRenderedSet(ctx context.Context, ref string, indexJSON []byte, tar io.Reader, size int64, ann Annotations) (Descriptor, error) {
  if ann == nil { ann = Annotations{} }
  ann[AnnForgeKind] = "rendered"
  return c.PushTar(ctx, ref, indexJSON, MTRenderedIndex, MTRenderedTarGz, tar, size, ann)
}

func (c *client) PullRenderedSet(ctx context.Context, ref string) (io.ReadCloser, []byte, Descriptor, error) {
  // Pull config (index) + layer (tar)
  idx, d1, err := c.PullJSON(ctx, ref, MTRenderedIndex)
  if err != nil { return nil, nil, Descriptor{}, err }
  rc, d2, err := c.PullTar(ctx, ref, MTRenderedTarGz)
  if err != nil { return nil, nil, Descriptor{}, err }
  // Return the layer stream + index json; prefer manifest digest from d2 if present
  if d2.Digest != "" { return rc, idx, d2, nil }
  return rc, idx, d1, nil
}
```

### `pkg/oci/sign_verify.go` (excerpt)

```go
package oci

import "context"

// Uses cosign under the hood; if CosignOpts.Enable=false, SignDigest is a no-op.
func (c *client) SignDigest(ctx context.Context, refOrDigest string) (Descriptor, error) {
  if !c.opts.Cosign.Enable { return Descriptor{Ref: refOrDigest}, nil }
  return c.cosignSign(ctx, refOrDigest)
}

func (c *client) VerifyDigest(ctx context.Context, refOrDigest string) (*VerificationReport, error) {
  if !c.opts.Cosign.Enable {
    return &VerificationReport{Digest: refOrDigest, Signed: false}, nil
  }
  return c.cosignVerify(ctx, refOrDigest, c.opts.Cosign.Identity)
}
```

### `pkg/oci/auth.go` (excerpt)

```go
package oci

// DefaultAuth uses docker config (~/.docker/config.json) via ggcr's authn.
type DefaultAuth struct{}

func (DefaultAuth) Authenticator(registry string) (any, error) {
  // return authn.DefaultKeychain for ggcr; ORAS also supports passing a ggcr authenticator
  return nil, nil // implement with ggcr auth keychain as needed
}
```

### `pkg/oci/annotations.go`

```go
package oci

const (
  AnnSourceRepo  = "org.opencontainers.image.source"
  AnnSourceRev   = "org.opencontainers.image.revision"
  AnnCreated     = "org.opencontainers.image.created"
  AnnTitle       = "org.opencontainers.image.title"
  AnnDescription = "org.opencontainers.image.description"

  AnnForgeKind   = "io.projectcatalyst.forge.kind"
  AnnForgeProject= "io.projectcatalyst.forge.project"
  AnnForgeEnv    = "io.projectcatalyst.forge.env"
  AnnForgeTrace  = "io.projectcatalyst.forge.trace"
  AnnForgeRelease= "io.projectcatalyst.forge.releaseKey"
)
```

> The internal files `internal/orasx.go` and `internal/ggcrx.go` encapsulate the registry implementation details (constructing repositories, choosing manifest kind, mapping errors). Keep those unexported so you can swap implementations later.

---

## 7) Testing strategy

* **Unit tests** with a **fake registry**:

  * `ggcr` provides an in‑memory registry you can spin up for tests, or use `oras-go`’s `content` memory store with a local registry container in CI.
* **Golden tests** for:

  * Release Bundle push/pull (JSON round‑trip; manifest annotations)
  * Rendered Set push/pull (index + tar content type; size/digest match)
  * Fallback: force artifact manifest rejection and assert image manifest path used.
* **Signing tests**: run behind a build tag; verify the client surfaces signer identities and fails on wrong issuer/subject.

---

## 8) Why this shape works for you

* **Keeps consumers simple:** Renderer, Operator, and API just call `PushReleaseBundle`, `PushRenderedSet`, `Pull*`, and `Sign/Verify`.
* **Digest‑based promotion:** Always return canonical `@sha256:…` refs; your GitOps pointers stay stable.
* **Future‑proof:** If registries universally support artifact manifests, turn off fallback and simplify. If you change signing policy, only this package changes.
* **Domain‑agnostic:** Nothing here knows what a Release is; it only ships **bytes with media types** and **returns digests**.

If you’d like, I can fill in the `internal/orasx.go` and `internal/ggcrx.go` with working code using `oras-go` and `go-containerregistry` (including basic docker‑config auth and cosign invocations) sized to your exact registries (ECR + GHCR).

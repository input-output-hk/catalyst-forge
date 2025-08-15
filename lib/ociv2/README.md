# OCI v2 Client Library

A practical Go client for pushing, pulling, signing, and verifying OCI artifacts.

## Table of Contents
- [Installation](#installation)
- [Quick start](#quick-start)
- [Common operations](#common-operations)
  - [Resolve and head](#resolve-and-head)
  - [Push and pull JSON artifacts](#push-and-pull-json-artifacts)
  - [Push and pull tar content](#push-and-pull-tar-content)
  - [Multi-layer artifacts (pack/pull)](#multi-layer-artifacts-packpull)
  - [Tags: list and latest semver](#tags-list-and-latest-semver)
  - [Signing and verification (Cosign)](#signing-and-verification-cosign)
  - [Attestations (DSSE/intoto)](#attestations-dsseintoto)
- [Configuration reference](#configuration-reference)
  - [Networking](#networking)
  - [Authentication](#authentication)
  - [Observability](#observability)
  - [Limits and performance](#limits-and-performance)
- [Troubleshooting](#troubleshooting)
- [Error handling](#error-handling)

## Installation

```bash
go get github.com/input-output-hk/catalyst-forge/lib/ociv2
```

## Quick start

```go
ctx := context.Background()

client, err := ociv2.New(ociv2.ClientOptions{
    Timeout: 2 * time.Minute,
})
if err != nil { log.Fatal(err) }

data := []byte(`{"version":"1.0.0","name":"myapp"}`)

// Push JSON (artifact manifest first, automatic fallback to image manifest)
desc, err := client.PushJSON(ctx,
    "oci://registry.example.com/myapp:latest",
    "application/vnd.myapp.config+json",
    data,
    ociv2.NewAnnotations().WithTitle("My Application").WithVersion("1.0.0"),
)
if err != nil { log.Fatal(err) }

// Pull by canonical digest
doc, _, err := client.PullJSON(ctx, desc.Ref, "application/vnd.myapp.config+json")
if err != nil { log.Fatal(err) }
fmt.Printf("%s\n", string(doc))
```

## Common operations

### Resolve and head
```go
// Resolve returns a full descriptor and canonical digest reference
resolved, err := client.Resolve(ctx, "oci://example.com/repo:tag")
// Head returns descriptor metadata without fetching content
headed, err := client.Head(ctx, "oci://example.com/repo:tag")
```

### Push and pull JSON artifacts
```go
payload := []byte(`{"kind":"release","version":"1.2.3"}`)
ann := ociv2.NewAnnotations().WithForgeKind("release").WithVersion("1.2.3")

pushed, err := client.PushJSON(ctx,
    "oci://example.com/releases/myapp:1.2.3",
    ociv2.MTReleaseConfig,
    payload,
    ann,
)

data, desc, err := client.PullJSON(ctx, pushed.Ref, ociv2.MTReleaseConfig)
_ = desc // descriptor for the manifest
```

### Push and pull tar content
```go
indexJSON := []byte(`{"manifests":[]}`)
var tar io.Reader = myTarReader
size := int64(myTarSize)

pushed, err := client.PushTar(ctx,
    "oci://example.com/rendered/myapp:1.2.3",
    indexJSON,
    ociv2.MTRenderedIndex,
    ociv2.MTRenderedTarGz,
    tar,
    size,
    ociv2.NewAnnotations().WithForgeKind("rendered"),
)

rc, desc, err := client.PullTar(ctx, pushed.Ref, ociv2.MTRenderedTarGz)
defer rc.Close()
```

### Multi-layer artifacts (pack/pull)
```go
layers := []ociv2.LayerSpec{
    { MediaType: "application/vnd.test.layer+tar", Title: "primary", Size: int64(len(buf1)), Reader: bytes.NewReader(buf1) },
    { MediaType: "application/vnd.test.layer+json", Title: "metadata", Size: int64(len(buf2)), Reader: bytes.NewReader(buf2) },
}

opts := ociv2.PackOptions{
    ArtifactType:            "application/vnd.test.artifact+tar",
    ManifestAnnotations:     map[string]string{"app.version":"1.2.3"},
    Layers:                  layers,
    PreferArtifactManifest:  true,
    FallbackImageManifest:   true,
}

pushed, err := client.PushArtifact(ctx, "oci://example.com/artifacts/myapp:1.2.3", opts)

result, err := client.PullArtifact(ctx, pushed.Ref)
first, _ := result.GetLayer(0)
rc, _ := first.Open()
```

### Tags: list and latest semver
```go
tags, err := client.ListTags(ctx, "example.com/myapp")
latest, err := client.LatestSemverTag(ctx, "example.com/myapp", false) // exclude prereleases
```

### Signing and verification (Cosign)
```go
client, _ = ociv2.New(ociv2.ClientOptions{
    Cosign: ociv2.CosignOpts{ Enable: true },
})

sigDesc, err := client.SignDigest(ctx, "example.com/myapp@sha256:...")

report, err := client.VerifyDigest(ctx, sigDesc.Ref)
if report.Signed { /* check report.Signers, report.BundleVerified */ }
```

### Attestations (DSSE/intoto)
```go
attDesc, err := client.Attest(ctx, "example.com/myapp@sha256:...", ociv2.AttestationOptions{
    PredicateType: ociv2.PredicateSLSAProvenanceV1,
    PayloadJSON:   myProvenanceJSON,
})

attReport, err := client.VerifyAttestations(ctx, attDesc.Ref, ociv2.PredicateSLSAProvenanceV1)
```

## Configuration reference

### Networking
- `Timeout` (default: 2m): applied per-operation via context.
- `PlainHTTP`: use HTTP instead of HTTPS. This is off by default. The client will only auto-downgrade to HTTP for loopback hosts (localhost/127.0.0.1/::1). For any other hosts, explicitly set `PlainHTTP: true` if you really want HTTP.
- `HTTPTransport`: optional custom `*http.Transport` for connection pooling.
- `MaxConcurrency`: limits concurrent operations (default: 10).

### Authentication
- Default: Docker config keychain (`~/.docker/config.json`).
- GitHub Container Registry:
  ```go
  client, _ := ociv2.New(ociv2.ClientOptions{ Auth: &ociv2.GitHubAuth{ Token: os.Getenv("GITHUB_TOKEN") } })
  ```
- Static credentials:
  ```go
  client, _ := ociv2.New(ociv2.ClientOptions{ Auth: &ociv2.StaticAuth{ Username: "user", Password: "pass" } })
  ```
- Chain multiple providers: try in order until one works.
  ```go
  client, _ := ociv2.New(ociv2.ClientOptions{ Auth: &ociv2.ChainAuth{ Providers: []ociv2.AuthProvider{ &ociv2.DefaultAuth{}, &ociv2.GitHubAuth{Token: ghToken}, }, }, })
  ```
- Amazon ECR: The `ECRAuth` type is a placeholder and not fully implemented yet. For now, authenticate via `docker login`/AWS CLI so credentials are available in Docker config, or implement a custom provider in your app.

### Observability
- `Logger` or `StructuredLogger` for logs.
- `EnableMetrics` + `MetricsCallback` to receive aggregated metrics per operation and fallback behavior.

### Limits and performance
- `MaxBlobSize` (default: 5GB): upper bound enforced for pushes/pulls.
- `StreamBufferSize` (default: 32KB): buffered streaming for large I/O.
- Prefer digest references when you can to avoid extra resolves.

## Troubleshooting

- Registry doesn’t support artifact manifests (400/415): enable `FallbackImageManifest: true` or use image manifests directly.
- 401 Unauthorized: verify Docker config auth, try the same operation with `docker pull/push`, or set a specific provider (e.g., `GitHubAuth`).
- Signing in CI (GitHub Actions): ensure `permissions: { id-token: write, packages: write }` and `COSIGN_EXPERIMENTAL=1`.
- Timeouts on large artifacts: increase `Timeout` (e.g., `10 * time.Minute`).

## Error handling

Errors are categorized (auth, network, registry, validation, etc.) and include useful context for logging and retries.
```go
_, err := client.PushJSON(ctx, ref, mt, data, nil)
if err != nil {
    var ociErr *ociv2.OCIError
    if errors.As(err, &ociErr) {
        // use ociErr.Category, ociErr.HTTPStatus, etc.
    }
}
```
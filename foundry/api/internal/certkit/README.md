# certkit (AWS PCA-backed certificate signer)

certkit provides a thin, safe facade to issue X.509 certificates via AWS Private CA (PCA), keeping handlers thin and centralizing cryptographic and PCA operations inside the package. It mirrors the shape of `authkit` with clear interfaces, services, and provider adapters.

## Endpoints (mounted by the package)
- POST `/pki/sign` – Issue a certificate from a CSR (PEM). Returns certificate PEM, chain PEM, and certificate ARN.
- GET `/pki/ca` – Returns the CA certificate and chain for the configured PCA.

## Config (certkit.Config)
- `CAArn` (string): ARN of your Private CA
- `Region` (string): AWS region
- Defaults (overridable): `DefaultTemplateArn`, `DefaultSigningAlgo`, `DefaultTTL`
- Policy: `MaxTTL`, `AllowedTemplates`, `AllowedKeyAlgos`, `AllowedKeySizes`, `AllowedSANDomains`, `AllowURISAN`, `AllowIPSAN`
- Behavior: `PollInterval`, `MaxWait`, `RateEnabled`

## Dependencies (certkit.Deps)
- `PCA` (PCAClient): AWS PCA client wrapper (mockable)
- `Limiter` (rate.Limiter, optional): Identity-based rate limiting
- `Clock`, `Logger` (optional)
- `RBAC` (authkit/rbac.Manager, optional): For SAN policy enforcement

## RBAC & SAN conditions
Attach data-driven conditions to the `cert:sign` permission. The issuer passes CSR SANs in `ResourceRef.Attrs`:
- `dns_sans_suffix_in`: `{ "suffixes": [ ".projectcatalyst.io", "*.svc.cluster.local" ] }`
- `uri_sans_prefix_in`: `{ "prefixes": [ "spiffe://org/" ] }`
- `ip_sans_in_cidrs`: `{ "cidrs": [ "10.0.0.0/8", "fd00::/8" ] }`

Admin sets parameters in role entries (DB), not in code.

## Integration example
```go
// Build PCA from AWS default config chain (IRSA, env, etc.)
pca, _ := certkit.BuildPCAFromRegion(ctx, cfg.Region)

deps := certkit.Deps{PCA: pca, RBAC: rbacMgr, Clock: sysClock, Limiter: limiter}
ck, _ := certkit.New(certkit.Config{
    CAArn:              CA_ARN,
    Region:             AWS_REGION,
    DefaultTemplateArn: "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
    DefaultSigningAlgo: "SHA256WITHRSA",
    DefaultTTL:         90*24*time.Hour,
    MaxTTL:             365*24*time.Hour,
    PollInterval:       500*time.Millisecond,
    MaxWait:            30*time.Second,
}, deps)

// Routes
ck.RegisterRoutes(router.Group("/pki"))

// Policy registry (protect issuance)
reg.RequirePermissions([]string{"cert:sign"}, "POST", "/pki/sign")
```

## Errors (HTTP mapping)
- 400: CSR invalid/signature errors
- 403: RBAC deny (SAN policy)
- 429: Rate limit exceeded
- 504: Issuance timeout
- 5xx: Provider/internal errors

## Testing
- Use the fake PCA (`certkit/testing`) for unit/integration tests.
- Issuer and routes are covered by unit and HTTP tests.

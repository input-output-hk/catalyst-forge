# Mock OIDC provider integration for Kratos (playground)

This checklist guides you to deploy a mock OIDC provider, expose it through Envoy Gateway, and configure Ory Kratos to use it for Google-like OIDC flows. We’ll use a fixed client_id/client_secret (non-sensitive in dev) stored in a Kubernetes Secret, and reuse the existing Kratos module features (mapper files, provider secrets, HTTPRoute).

Reference values structure for Kratos Helm chart: https://raw.githubusercontent.com/ory/k8s/refs/heads/master/helm/charts/kratos/values.yaml

## Pre-reqs
- [ ] Envoy Gateway installed and reachable (already provisioned by playground).
- [ ] External DNS or local hosts mapping for the chosen hostnames.
- [ ] Kratos module available at `terraform/ory/kratos` (already in this repo).

## 1) Decide hostnames
- Kratos public: `auth.projectcatalyst.dev` (already variable `kratos_host`).
- Mock OIDC issuer: e.g. `auth-mock.projectcatalyst.dev`.

Tasks:
- [ ] Use wildcard DNS `*.projectcatalyst.dev` which resolves to 127.0.0.1; no hosts changes needed.

## 2) Deploy mock OIDC provider (Deployment + Service)
Create a simple Deployment/Service for a mock OpenID Provider that:
- Serves discovery at `/.well-known/openid-configuration` and JWKS at `/jwks`.
- Issues tokens with the claims your mapper expects (email, email_verified, optional hd/domain).
- Accepts a configured client with redirect URI pointing to Kratos.

Tasks:
- [ ] Create `playground/terraform/mock_oidc.tf` with:
  - [ ] `kubernetes_deployment_v1.mock_oidc` (image: a mock OIDC server, container port 8080 or 443 as supported; configure issuer, clients, and claims via env or mounted config).
  - [ ] `kubernetes_service_v1.mock_oidc` exposing the deployment (ClusterIP, port 80 → container port).
- [ ] Create an HTTPRoute for the mock issuer (see step 3) so it resolves at `https://auth-mock.projectcatalyst.dev`.

Notes:
- Configure a client in the mock OIDC server with:
  - `client_id`: `kratos-mock-client`
  - `client_secret`: `kratos-mock-secret`
  - `redirect_uri`: `https://auth.projectcatalyst.dev/.ory/kratos/public/self-service/methods/oidc/callback/google`
- Emit typical Google-like claims if your mapper checks them (e.g., `email_verified`, `email`, optional `hd`).

## 3) Expose the mock OIDC provider via Envoy Gateway (HTTPRoute)
Tasks:
- [ ] Add a `kubectl_manifest` HTTPRoute (or a rendered template) that:
  - [ ] Sets `parentRefs` to your Envoy Gateway (`var.gateway_name`, `var.namespace`).
  - [ ] Sets `hostnames` to `["auth-mock.projectcatalyst.dev"]`.
  - [ ] Routes `PathPrefix: /` to `Service: mock-oidc` on port 80.
- [ ] Verify `https://auth-mock.projectcatalyst.dev/.well-known/openid-configuration` returns valid metadata.

## 4) Create Kratos OIDC Secret with fixed credentials
We’ll store fixed dev credentials in a Secret that Kratos will mount as files.

Tasks:
- [ ] Add a `kubernetes_secret_v1` named `kratos-oidc-google` in `var.kratos_namespace` with keys:
  - [ ] `client_id`: `kratos-mock-client`
  - [ ] `client_secret`: `kratos-mock-secret`

## 5) Wire the Kratos module to the mock OIDC provider
Tasks (in `playground/terraform/kratos.tf`):
- [ ] Ensure `kratos_config.identity.default_schema_id = "default"` and the identity schema is loaded (already set via `identity_schemas`).
- [ ] Set Kratos OIDC provider to use the mock issuer:
  - In `kratos_config`, set `selfservice.methods.oidc.config.providers[0]` to:
    - `id = "google"` (name Kratos expects by default in the callback path)
    - `provider = "generic"` (or keep `google` and set `issuer_url` explicitly)
    - `issuer_url = "https://auth-mock.projectcatalyst.dev"`
    - `scope = ["openid", "email", "profile"]`
    - `mapper_url` pointing to your Jsonnet (we already mount `google.mapper.jsonnet` via the module)
- [ ] Mount the OIDC client Secret via module input:
  - Set `oidc_provider_secrets = { google = { secret_name = "kratos-oidc-google", client_id_key = "client_id", client_secret_key = "client_secret" } }`.
  - The module will mount files at `/etc/kratos/oidc/google/{client_id,client_secret}` and rewrite provider config to `file://` paths.
- [ ] Keep DSN out of `kratos_config`; inject via the existing `dsn_secret` flow.

## 6) HTTPRoute for Kratos public
Tasks:
- [ ] Confirm `http_route.enabled = true` in the module and `hostnames = [var.kratos_host]`.
- [ ] Default routes `/` to Kratos public service on port 80 (already handled by the module if rules are not provided).

## 7) Test
- [ ] Apply Terraform: `terraform -chdir=playground/terraform init && terraform apply`.
- [ ] Verify mock OP:
  - `curl -k https://auth-mock.projectcatalyst.dev/.well-known/openid-configuration` returns issuer metadata.
- [ ] Hit the Kratos login URL in the frontend; choose OIDC → should redirect to mock OP consent/login, return and create a session.
- [ ] Inspect Kratos logs to confirm token validation and mapper application.

## 8) Optional enhancements
- [ ] Add TLS certs for `auth-mock.local.io` and `auth.local.io` if not using local TLS termination.
- [ ] Parameterize mock OP user claims via ConfigMap values to test different identity shapes.
- [ ] Add cleanup target to remove mock OP resources.

## Rollback (if needed)
- [ ] Delete mock OP Deployment/Service and HTTPRoute.
- [ ] Remove the OIDC Secret and revert Kratos provider to a real IdP configuration.

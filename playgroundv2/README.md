### Playground v2: Local K3d cluster with Helmfile

This folder provisions a complete local environment for development:
- K3d-based Kubernetes cluster (no Multipass/MicroK8s)
- Envoy Gateway installed via Helmfile with TLS (mkcert)
- No hostctl required: we use wildcard DNS on `*.projectcatalyst.dev` -> 127.0.0.1
- In-cluster Docker registry exposed via Envoy (`registry.projectcatalyst.dev`)
- Optional PostgreSQL via Helm
- Mailpit (SMTP testing) via Helm

#### Prerequisites
- macOS or Linux
- Docker, k3d, kubectl, mkcert, uv, python3

#### What gets set up
- K3d cluster named `forge`
- Envoy Gateway (Bitnami chart) with Gateway listeners for HTTP/HTTPS
- TLS via cert-manager Certificate for `*.projectcatalyst.dev`
- In-cluster Docker registry (twuni chart) exposed via HTTPRoute at `registry.projectcatalyst.dev`
- Cluster trusts mkcert CA for pulling from the local registry
- Optional: PostgreSQL with PVC

#### Start / Tear down
```bash
# Bring everything up (idempotent): K3d + Helmfile
just up

# Tear down the local k3d cluster
just down
```

#### Python CLI
```bash
# Show CLI help
uv run python -m playgroundv2.cli.main --help
```

The up command will:
- Create/refresh the K3d cluster and write `playgroundv2/cluster.json`
- Install cert-manager, Envoy Gateway, and Registry via Helmfile
- Bootstrap TLS using mkcert CA and a wildcard Certificate for `*.projectcatalyst.dev`
- Generate local client TLS certs for Earthly (idempotent) under `playgroundv2/.certs/`:
  - `earthly-client.pem`, `earthly-client-key.pem`, and `rootCA.pem`
  - You can disable this with `--no-generate-client-cert`

#### Deployments included
- Envoy Gateway (namespace `envoy-gateway-system`)
- Registry (namespace `registry`)
- Mailpit (namespace `mailpit`)
- PostgreSQL (optional, namespace `databases`)

#### Accessing Mailpit
Once the environment is up, open `https://mailpit.projectcatalyst.dev/`.
SMTP endpoint is available inside the cluster at `mailpit.mailpit.svc.cluster.local:1025`.


#### Building images
```bash
# Build the API server image and push to in-cluster registry as registry.projectcatalyst.dev/api:latest
just earthly api
```



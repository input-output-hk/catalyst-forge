### Playground v2: Single-node MicroK8s on Multipass

This folder provisions a complete local environment for development:
- MicroK8s in a Multipass VM with DNS, Storage, and MetalLB
- Envoy Gateway exposed via MetalLB with TLS (mkcert)
- Public hostnames mapped via hostctl (on host) and /etc/hosts (inside VM)
- In-cluster Docker registry exposed via Envoy (`registry.local.io`)
- Optional PostgreSQL via Helm

#### Prerequisites
- macOS or Linux
- Multipass (`https://multipass.run`)
- OpenTofu (or Terraform), kubectl, jq, mkcert, hostctl
- Docker (optional, for building/pushing images)

#### What gets set up
- MicroK8s VM named `microk8s` with a derived MetalLB range (A.B.C.240-250)
- Envoy Gateway (Bitnami chart) with Gateway listeners for HTTP/HTTPS
- TLS via mkcert wildcard cert for `*.local.io` stored in secret `envoy-gateway-tls`
- Host and VM host mappings (idempotent) for:
  - `gateway.local.io` (Envoy)
  - `registry.local.io` (in-cluster registry)
- In-cluster Docker registry (twuni chart) exposed via HTTPRoute
- VM trusts the mkcert CA for containerd pulls
- Optional: PostgreSQL in `databases` with PVC

#### Start / Tear down
```bash
# Bring everything up (idempotent)
just up

# Tear down fast (delete Multipass VM, clear local state)
just down
```

The up command will:
- Create/refresh the MicroK8s VM and write `playgroundv2/cluster.json`
- Choose a stable IP from the MetalLB range and apply Terraform:
  - Envoy Gateway (OCI chart)
  - Registry (`docker-registry` from `https://helm.twun.io`)
  - HTTPRoute for registry
  - (Optional) PostgreSQL (Bitnami chart)
- Generate mkcert wildcard cert and create the TLS secret
- Install mkcert CA into the VM and restart containerd
- Map `gateway.local.io` and `registry.local.io` on host and VM to the Envoy IP

#### Deployments included
- Envoy Gateway (namespace `envoy-gateway-system`)
- Registry (namespace `registry`)
- PostgreSQL (optional, namespace `databases`)

#### Buildimg images
```bash
# Build the API server image and push to in-cluster registry as registry.local.io/api:latest
just earthly api
```



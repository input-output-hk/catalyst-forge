### Terraform: Envoy Gateway on MicroK8s with MetalLB

This Terraform config installs the Bitnami Envoy Gateway Helm chart and applies minimal Envoy Gateway resources to expose Envoy via a LoadBalancer Service compatible with MetalLB.

#### Prereqs
- A running MicroK8s cluster provisioned via `../bootstrap-microk8s-multipass.sh`
- MetalLB enabled and configured
- Terraform >= 1.5

#### Quick start
```bash
cd playgroundv2/terraform
terraform init
terraform apply -auto-approve
```

After apply, check the external IP:
```bash
kubectl --kubeconfig ../kubeconfig -n envoy-gateway-system get svc
```

#### Variables
- `kubeconfig_path` (default: `../kubeconfig`)
- `namespace` (default: `envoy-gateway-system`)
- `chart_version` (default: unset; use the chart latest)
- `load_balancer_ip` (optional; must be within your MetalLB range)
- `service_annotations` (map; optional)

#### References
- Bitnami Envoy Gateway Helm chart: [Artifact Hub](https://artifacthub.io/packages/helm/bitnami/envoy-gateway)


resource "helm_release" "registry" {
  name             = "registry"
  repository       = "https://helm.twun.io"
  chart            = "docker-registry"
  namespace        = var.registry_namespace
  create_namespace = true

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
  set {
    name  = "persistence.enabled"
    value = "true"
  }
  set {
    name  = "persistence.size"
    value = "20Gi"
  }

  timeout = 600
  wait    = true
}

# HTTPRoute to expose registry via Envoy Gateway
resource "kubectl_manifest" "registry_route" {
  depends_on = [helm_release.registry, kubectl_manifest.gateway]

  yaml_body = templatefile("${path.module}/templates/registry-route.yaml.tftpl", {
    registry_namespace = var.registry_namespace
    gateway_namespace  = local.envoy_gateway_namespace
    gateway_name       = var.gateway_name
    hostname           = var.registry_host
  })
}



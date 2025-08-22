resource "helm_release" "envoy_gateway" {
  name             = "envoy-gateway"
  repository       = "oci://registry-1.docker.io/bitnamicharts"
  chart            = "envoy-gateway"
  namespace        = local.envoy_gateway_namespace
  create_namespace = true
  version          = "2.0.4"

  set {
    name  = "installCRDs"
    value = "true"
  }

  timeout = 600
  wait    = true
}

resource "kubectl_manifest" "envoy_proxy" {
  depends_on = [helm_release.envoy_gateway]

  yaml_body = templatefile("${path.module}/templates/envoyproxy.yaml.tftpl", {
    namespace           = local.envoy_gateway_namespace
    load_balancer_ip    = var.load_balancer_ip
    service_annotations = var.service_annotations
  })
}

resource "kubectl_manifest" "gateway_class" {
  depends_on = [helm_release.envoy_gateway]

  yaml_body = templatefile("${path.module}/templates/gatewayclass.yaml.tftpl", {
    name = var.gateway_class_name
  })
}

resource "kubectl_manifest" "gateway" {
  depends_on = [kubectl_manifest.gateway_class, kubectl_manifest.envoy_proxy]

  yaml_body = templatefile("${path.module}/templates/gateway.yaml.tftpl", {
    name           = var.gateway_name
    namespace      = local.envoy_gateway_namespace
    class          = var.gateway_class_name
    tls_secret_name = var.tls_secret_name
  })
}



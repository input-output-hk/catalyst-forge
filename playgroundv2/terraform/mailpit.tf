resource "helm_release" "mailpit" {
  name             = "mailpit"
  repository       = "https://jouve.github.io/charts/"
  chart            = "mailpit"
  version          = "0.28.0"
  namespace        = var.playground.mailpit.namespace
  create_namespace = true

  # Keep default ports (SMTP 1025, UI 8025); expose via Envoy HTTPRoute
  set {
    name  = "service.type"
    value = "ClusterIP"
  }

  # We route via Envoy Gateway; disable Helm-managed ingress if present
  set {
    name  = "ingress.enabled"
    value = "false"
  }

  set {
    name  = "service.http.name"
    value = "http"
  }

  set {
    name  = "service.smtp.name"
    value = "smtp"
  }

  timeout = 600
  wait    = true
}

resource "kubectl_manifest" "mailpit_route" {
  depends_on = [helm_release.mailpit, kubectl_manifest.gateway]

  yaml_body = templatefile("${path.module}/templates/mailpit-route.yaml.tftpl", {
    mailpit_namespace = var.playground.mailpit.namespace
    gateway_namespace = local.envoy_gateway_namespace
    gateway_name      = var.playground.envoy.gateway_name
    hostname          = var.playground.mailpit.host
    service_name      = var.playground.mailpit.service
    ui_port           = var.playground.mailpit.ui_port
  })
}



resource "helm_release" "localstack" {
  name             = "localstack"
  repository       = "https://localstack.github.io/helm-charts"
  chart            = "localstack"
  namespace        = var.localstack_namespace
  create_namespace = true

  set {
    name  = "service.type"
    value = "ClusterIP"
  }

  dynamic "set" {
    for_each = var.localstack_image_tag == "" ? [] : [1]
    content {
      name  = "localstack.image.tag"
      value = var.localstack_image_tag
    }
  }

  timeout = 600
  wait    = true
}



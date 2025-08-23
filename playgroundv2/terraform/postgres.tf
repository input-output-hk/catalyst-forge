resource "helm_release" "postgres" {
  name             = "postgres"
  repository       = "oci://registry-1.docker.io/bitnamicharts"
  chart            = "postgresql"
  namespace        = var.playground.postgres.namespace
  create_namespace = true
  version          = "16.7.26"

  set {
    name  = "auth.username"
    value = var.playground.postgres.username
  }
  set {
    name  = "auth.password"
    value = var.playground.postgres.password
  }
  set {
    name  = "auth.database"
    value = var.playground.postgres.database
  }
  set {
    name  = "primary.persistence.enabled"
    value = "true"
  }
  set {
    name  = "primary.persistence.size"
    value = var.playground.postgres.storage
  }
  dynamic "set" {
    for_each = var.playground.postgres.storage_class == "" ? [] : [1]
    content {
      name  = "primary.persistence.storageClass"
      value = var.playground.postgres.storage_class
    }
  }

  timeout = 600
  wait    = true
}



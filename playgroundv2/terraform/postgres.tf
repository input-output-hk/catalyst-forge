resource "helm_release" "postgres" {
  name             = "postgres"
  repository       = "oci://registry-1.docker.io/bitnamicharts"
  chart            = "postgresql"
  namespace        = var.postgres_namespace
  create_namespace = true
  version          = "16.7.26"

  set {
    name  = "auth.username"
    value = var.pg_username
  }
  set {
    name  = "auth.password"
    value = var.pg_password
  }
  set {
    name  = "auth.database"
    value = var.pg_database
  }
  set {
    name  = "primary.persistence.enabled"
    value = "true"
  }
  set {
    name  = "primary.persistence.size"
    value = var.pg_storage
  }
  dynamic "set" {
    for_each = var.pg_storage_class == "" ? [] : [1]
    content {
      name  = "primary.persistence.storageClass"
      value = var.pg_storage_class
    }
  }

  timeout = 600
  wait    = true
}



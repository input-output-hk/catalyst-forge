resource "helm_release" "external_secrets" {
  name             = "external-secrets"
  repository       = "https://charts.external-secrets.io"
  chart            = "external-secrets"
  namespace        = var.playground.eso.namespace
  create_namespace = true
  version          = "0.11.0"
  values           = [file("${path.module}/config/eso-values.yaml")]

  # Keep reasonable defaults; allow overriding chart version via variable if desired later
  timeout = 600
  wait    = true
}

# Credentials Secret for ESO AWS provider (LocalStack)
resource "kubernetes_secret_v1" "eso_aws_credentials" {
  metadata {
    name      = var.playground.eso.aws_creds_secret_name
    namespace = var.playground.eso.namespace
  }
  data = {
    "access-key-id"     = var.playground.eso.aws_access_key_id
    "secret-access-key" = var.playground.eso.aws_secret_access_key
  }
  type = "Opaque"
}

# ClusterSecretStore pointing to LocalStack Secrets Manager
resource "kubectl_manifest" "cluster_secret_store" {
  depends_on = [helm_release.external_secrets, kubernetes_secret_v1.eso_aws_credentials]

  validate_schema = false

  yaml_body = <<-YAML
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: cluster-secret-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: ${var.playground.eso.aws_region}
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: ${var.playground.eso.aws_creds_secret_name}
            key: access-key-id
            namespace: ${var.playground.eso.namespace}
          secretAccessKeySecretRef:
            name: ${var.playground.eso.aws_creds_secret_name}
            key: secret-access-key
            namespace: ${var.playground.eso.namespace}
  YAML
}



locals {
  envoy_gateway_namespace = var.namespace
}

resource "helm_release" "envoy_gateway" {
  name             = "envoy-gateway"
  repository       = "oci://registry-1.docker.io/bitnamicharts"
  chart            = "envoy-gateway"
  namespace        = local.envoy_gateway_namespace
  create_namespace = true
  version          = "2.0.4"

  # Ensure CRDs are installed by the chart if supported. If the chart ignores this key, it's harmless.
  set {
    name  = "installCRDs"
    value = "true"
  }

  timeout = 600
  wait    = true
}

# Configure Envoy Gateway to expose the data plane Service as LoadBalancer
# via EnvoyProxy CR. Optionally pin a specific MetalLB IP if provided.
resource "kubectl_manifest" "envoy_proxy" {
  depends_on = [helm_release.envoy_gateway]

  yaml_body = templatefile("${path.module}/templates/envoyproxy.yaml.tftpl", {
    namespace           = local.envoy_gateway_namespace
    load_balancer_ip    = var.load_balancer_ip
    service_annotations = var.service_annotations
  })
}

# Minimal GatewayClass for Envoy Gateway
resource "kubectl_manifest" "gateway_class" {
  depends_on = [helm_release.envoy_gateway]

  yaml_body = templatefile("${path.module}/templates/gatewayclass.yaml.tftpl", {
    name = var.gateway_class_name
  })
}

# Minimal Gateway listening on ports 80/443 in the Envoy Gateway namespace
resource "kubectl_manifest" "gateway" {
  depends_on = [kubectl_manifest.gateway_class, kubectl_manifest.envoy_proxy]

  yaml_body = templatefile("${path.module}/templates/gateway.yaml.tftpl", {
    name      = var.gateway_name
    namespace = local.envoy_gateway_namespace
    class     = var.gateway_class_name
    tls_secret_name = var.tls_secret_name
  })
}

# In-cluster Docker registry (Bitnami chart)
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

# PostgreSQL via Bitnami (single primary, dev credentials)
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



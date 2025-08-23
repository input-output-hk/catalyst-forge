locals {
  http_route_yaml = var.http_route == null || var.http_route.enabled == false ? "" : yamlencode({
    apiVersion = "gateway.networking.k8s.io/v1"
    kind       = "HTTPRoute"
    metadata = {
      name      = coalesce(try(var.http_route.name, null), "${var.name}-public")
      namespace = coalesce(try(var.http_route.namespace, null), var.namespace)
    }
    spec = {
      parentRefs = [merge({ name = var.http_route.parent_ref.name },
        try(var.http_route.parent_ref.namespace, null) == null ? {} : { namespace = var.http_route.parent_ref.namespace },
        try(var.http_route.parent_ref.sectionName, null) == null ? {} : { sectionName = var.http_route.parent_ref.sectionName }
      )]
      hostnames = var.http_route.hostnames
      rules     = try(var.http_route.rules, [
        {
          matches = [{
            path = { type = "PathPrefix", value = "/" }
          }]
          backendRefs = [{
            name = "${helm_release.hydra.name}-public"
            port = 80
          }]
        }
      ])
    }
  })
}

resource "kubectl_manifest" "hydra_http_route" {
  count      = local.http_route_yaml == "" ? 0 : 1
  yaml_body  = local.http_route_yaml
  depends_on = [helm_release.hydra]
}



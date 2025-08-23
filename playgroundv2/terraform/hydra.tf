locals {
  hydra_gateway_namespace = var.playground.envoy.namespace
  hydra_cfg               = yamldecode(file("${path.module}/../config/ory/hydra/hydra.yml"))
}

resource "kubernetes_namespace_v1" "hydra" {
  metadata {
    name = var.playground.hydra.namespace
  }
}

resource "kubernetes_secret_v1" "hydra_dsn" {
  metadata {
    name      = "hydra-dsn"
    namespace = var.playground.hydra.namespace
  }
  data = {
    dsn = base64encode("postgres://foo:bar@pg-sqlproxy-gcloud-sqlproxy:5432/db")
  }
  type = "Opaque"
}

module "hydra" {
  source = "../../terraform/ory/hydra"

  name      = "hydra"
  namespace = var.playground.hydra.namespace

  # Avoid plaintext DSN; inject via env
  dsn_secret = {
    name = kubernetes_secret_v1.hydra_dsn.metadata[0].name
    key  = "dsn"
  }

  # Base Hydra config from provided file; enforce empty DSN and empty secrets.system to avoid plaintext
  hydra_config = merge(
    local.hydra_cfg,
    {
      dsn     = "",
      secrets = merge(try(local.hydra_cfg.secrets, {}), { system = [] })
    }
  )

  # HTTPRoute for public service
  http_route = {
    enabled = true
    parent_ref = {
      name      = var.playground.envoy.gateway_name
      namespace = local.hydra_gateway_namespace
    }
    hostnames = [var.playground.hydra.host]
    rules = [
      {
        matches = [{
          path = { type = "PathPrefix", value = "/hydra" }
        }]
        filters = [{
          type = "URLRewrite"
          urlRewrite = {
            path = {
              type               = "ReplacePrefixMatch"
              replacePrefixMatch = "/"
            }
          }
        }]
        backendRefs = [{
          name = "hydra-public"
          port = 80
        }]
      }
    ]
  }
}



locals {
  gateway_namespace = var.playground.envoy.namespace
  kratos_cfg        = yamldecode(file("${path.module}/../config/ory/kratos/kratos.yml"))
}

resource "kubernetes_namespace_v1" "kratos" {
  metadata {
    name = var.playground.kratos.namespace
  }
}

resource "kubernetes_secret_v1" "kratos_dsn" {
  metadata {
    name      = "kratos-dsn"
    namespace = var.playground.kratos.namespace
  }
  data = {
    dsn = base64encode("postgres://foo:bar@pg-sqlproxy-gcloud-sqlproxy:5432/db")
  }
  type = "Opaque"
}

resource "kubernetes_secret_v1" "kratos_oidc_google" {
  metadata {
    name      = "kratos-oidc-google"
    namespace = var.playground.kratos.namespace
  }
  data = {
    client_id     = base64encode(var.playground.mock_oidc.client_id)
    client_secret = base64encode(var.playground.mock_oidc.client_secret)
  }
  type = "Opaque"
}

module "kratos" {
  source = "../../terraform/ory/kratos"

  name      = "kratos"
  namespace = var.playground.kratos.namespace

  # Avoid plaintext DSN; inject via env
  dsn_secret = {
    name = kubernetes_secret_v1.kratos_dsn.metadata[0].name
    key  = "dsn"
  }

  # Base Kratos config from provided file; enforce empty DSN to avoid plaintext secrets
  kratos_config = merge(
    local.kratos_cfg,
    {
      dsn = "",
      selfservice = {
        methods = {
          oidc = {
            config = {
              providers = [for p in try(local.kratos_cfg.selfservice.methods.oidc.config.providers, []) : p.id == "google" ? merge(p, {
                issuer_url = "https://${var.playground.mock_oidc.host}"
              }) : p]
            }
          }
        }
      }
    }
  )

  # Identity schema content
  identity_schemas = {
    default = file("${path.module}/../config/ory/kratos/identity.schema.json")
  }
  identity_default_schema_id = "default"

  # OIDC mapper files
  oidc_mappers = {
    "google.mapper.jsonnet" = file("${path.module}/../config/ory/kratos/google.mapper.jsonnet")
  }

  oidc_provider_secrets = {
    google = {
      secret_name       = kubernetes_secret_v1.kratos_oidc_google.metadata[0].name
      client_id_key     = "client_id"
      client_secret_key = "client_secret"
    }
  }

  # Automigration on by default
  enable_automigration = true

  # HTTPRoute for public service
  http_route = {
    enabled = true
    parent_ref = {
      name      = var.playground.envoy.gateway_name
      namespace = local.gateway_namespace
      # sectionName can be specified if your gateway uses sections
    }
    hostnames = [var.playground.kratos.host]
    rules = [
      {
        matches = [{
          path = { type = "PathPrefix", value = "/kratos" }
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
          name = "kratos-public"
          port = 80
        }]
      }
    ]
  }
}



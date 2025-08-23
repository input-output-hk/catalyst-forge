locals {
  generated_values_map = var.enable_generate_secrets ? {
    hydra = {
      config = {
        secrets = {
          system = [random_password.hydra["system"].result]
        }
      }
    }
  } : {}

  generated_values = length(local.generated_values_map) > 0 ? yamlencode(local.generated_values_map) : ""

  hydra_config_values = length(var.hydra_config) > 0 ? yamlencode({ hydra = { config = var.hydra_config } }) : ""

  values = compact([
    local.generated_values,
    local.hydra_config_values,
    yamlencode({ hydra = { automigration = { enabled = true } } }),
    var.dsn_secret == null ? "" : yamlencode({
      hydra = {
        deployment = {
          extraEnv = [{
            name      = try(var.dsn_secret.env_name, "DSN")
            valueFrom = { secretKeyRef = { name = var.dsn_secret.name, key = var.dsn_secret.key } }
          }]
        }
        job = {
          extraEnv = [{
            name      = try(var.dsn_secret.env_name, "DSN")
            valueFrom = { secretKeyRef = { name = var.dsn_secret.name, key = var.dsn_secret.key } }
          }]
        }
        cronjob = {
          cleanup = {
            extraEnv = [{
              name      = try(var.dsn_secret.env_name, "DSN")
              valueFrom = { secretKeyRef = { name = var.dsn_secret.name, key = var.dsn_secret.key } }
            }]
          }
        }
      }
    }),
    length(var.values) > 0 ? yamlencode(var.values) : "",
    var.values_yaml,
  ])
}

resource "random_password" "hydra" {
  for_each = var.enable_generate_secrets ? toset(["system"]) : []
  length   = var.secret_length
  special  = false
}

resource "helm_release" "hydra" {
  name             = var.name
  repository       = var.repository
  chart            = "hydra"
  namespace        = var.namespace
  create_namespace = var.create_namespace
  version          = var.chart_version

  values = local.values

  timeout = var.timeout_seconds
  wait    = var.wait
}



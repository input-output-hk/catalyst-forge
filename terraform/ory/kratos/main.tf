locals {
  generated_values_map = var.enable_generate_secrets ? {
    kratos = {
      config = {
        secrets = {
          cookie  = [random_password.kratos["cookie"].result]
          cipher  = [random_password.kratos["cipher"].result]
          default = [random_password.kratos["default"].result]
        }
      }
    }
  } : {}

  generated_values = length(local.generated_values_map) > 0 ? yamlencode(local.generated_values_map) : ""

  kratos_config_values = length(var.kratos_config) > 0 ? yamlencode({ kratos = { config = var.kratos_config } }) : ""

  identity_values = length(var.identity_schemas) > 0 ? yamlencode({
    kratos = {
      identitySchemas = { for k, v in var.identity_schemas : "identity.${k}.schema.json" => v }
      config = {
        identity = {
          default_schema_id = var.identity_default_schema_id
          schemas = [for id, content in var.identity_schemas : {
            id  = id
            url = "base64://${base64encode(content)}"
          }]
        }
      }
    }
  }) : ""

  mapper_volume_mounts = length(var.oidc_mappers) == 0 ? [] : [
    { name = "kratos-mappers", mountPath = "/etc/kratos", readOnly = true }
  ]

  mapper_volumes = length(var.oidc_mappers) == 0 ? [] : [
    { name = "kratos-mappers", configMap = { name = kubernetes_config_map.kratos_mappers[0].metadata[0].name } }
  ]

  oidc_secret_mounts = [for pid, spec in var.oidc_provider_secrets : {
    name      = "kratos-oidc-${pid}"
    mountPath = "/etc/kratos/oidc/${coalesce(try(spec.mount_subdir, null), pid)}"
    readOnly  = true
  }]

  oidc_secret_volumes = [for pid, spec in var.oidc_provider_secrets : {
    name   = "kratos-oidc-${pid}"
    secret = { secretName = spec.secret_name }
  }]

  all_extra_volume_mounts = concat(local.mapper_volume_mounts, local.oidc_secret_mounts)
  all_extra_volumes       = concat(local.mapper_volumes, local.oidc_secret_volumes)

  oidc_provider_overrides = length(var.oidc_provider_secrets) == 0 ? "" : yamlencode({
    kratos = {
      config = {
        selfservice = {
          methods = {
            oidc = {
              config = {
                providers = [for p in try(var.kratos_config.selfservice.methods.oidc.config.providers, []) : contains(keys(var.oidc_provider_secrets), p.id)
                  ? merge(p, {
                      client_id     = "file:///etc/kratos/oidc/${coalesce(try(var.oidc_provider_secrets[p.id].mount_subdir, null), p.id)}/client_id"
                      client_secret = "file:///etc/kratos/oidc/${coalesce(try(var.oidc_provider_secrets[p.id].mount_subdir, null), p.id)}/client_secret"
                    })
                  : p
                ]
              }
            }
          }
        }
      }
    }
  })

  extra_mounts_values = length(local.all_extra_volume_mounts) + length(local.all_extra_volumes) == 0 ? "" : yamlencode({
    kratos = {
      deployment = {
        extraVolumeMounts = local.all_extra_volume_mounts
        extraVolumes      = local.all_extra_volumes
      }
      job = {
        extraVolumeMounts = local.all_extra_volume_mounts
        extraVolumes      = local.all_extra_volumes
      }
    }
  })

  values = compact([
    local.generated_values,
    local.kratos_config_values,
    local.identity_values,
    local.oidc_provider_overrides,
    local.extra_mounts_values,
    yamlencode({ kratos = { automigration = { enabled = var.enable_automigration } } }),
    var.dsn_secret == null ? "" : yamlencode({
      kratos = {
        deployment = {
          extraEnv = [{
            name = try(var.dsn_secret.env_name, "DSN")
            valueFrom = { secretKeyRef = { name = var.dsn_secret.name, key = var.dsn_secret.key } }
          }]
        }
      }
      job = {
        extraEnv = [{
          name = try(var.dsn_secret.env_name, "DSN")
          valueFrom = { secretKeyRef = { name = var.dsn_secret.name, key = var.dsn_secret.key } }
        }]
      }
      cronjob = {
        cleanup = {
          extraEnv = [{
            name = try(var.dsn_secret.env_name, "DSN")
            valueFrom = { secretKeyRef = { name = var.dsn_secret.name, key = var.dsn_secret.key } }
          }]
        }
      }
    }),
    length(var.values) > 0 ? yamlencode(var.values) : "",
    var.values_yaml,
  ])
}

resource "random_password" "kratos" {
  for_each = var.enable_generate_secrets ? toset(["cookie", "cipher", "default"]) : []
  length   = var.secret_length
  special  = false
}

resource "helm_release" "kratos" {
  name             = var.name
  repository       = var.repository
  chart            = "kratos"
  namespace        = var.namespace
  create_namespace = var.create_namespace
  version          = var.chart_version

  values = local.values

  timeout = var.timeout_seconds
  wait    = var.wait
}



resource "kubernetes_config_map" "kratos_mappers" {
  count = length(var.oidc_mappers) == 0 ? 0 : 1
  metadata {
    name      = "${var.name}-mappers"
    namespace = var.namespace
  }
  data = var.oidc_mappers
}



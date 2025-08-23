variable "name" {
  description = "Helm release name for Ory Kratos"
  type        = string
  default     = "kratos"
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.name))
    error_message = "name must be lowercase alphanumeric and hyphens only."
  }
}

variable "namespace" {
  description = "Kubernetes namespace to deploy Kratos into"
  type        = string
  nullable    = false
  validation {
    condition     = length(trimspace(var.namespace)) > 0
    error_message = "namespace must be a non-empty string."
  }
}

variable "repository" {
  description = "Helm repository URL that hosts the Ory Kratos chart"
  type        = string
  default     = "https://k8s.ory.sh/helm/charts"
  validation {
    condition     = can(regex("^https?://", var.repository))
    error_message = "repository must be an http(s) URL."
  }
}

variable "chart_version" {
  description = "Specific Helm chart version for Ory Kratos (optional)"
  type        = string
  default     = null
}

variable "values_yaml" {
  description = "Additional Helm values as a YAML string. Applied after generated defaults to allow overrides."
  type        = string
  default     = ""
}

variable "values" {
  description = "Additional Helm values as a HCL map. Merged before values_yaml; later keys override earlier ones."
  type        = any
  default     = {}
}

variable "create_namespace" {
  description = "Whether to create the namespace if it does not exist"
  type        = bool
  default     = true
}

variable "wait" {
  description = "Whether to wait for all resources to be ready before marking the release successful"
  type        = bool
  default     = true
}

variable "timeout_seconds" {
  description = "Timeout in seconds for the Helm release operations"
  type        = number
  default     = 600
  validation {
    condition     = var.timeout_seconds >= 60
    error_message = "timeout_seconds should be at least 60 seconds."
  }
}

variable "enable_generate_secrets" {
  description = "If true, generate Kratos secrets (default, cookie, cipher) and inject into values"
  type        = bool
  default     = true
}

variable "secret_length" {
  description = "Length for generated Kratos secrets"
  type        = number
  default     = 32
  validation {
    condition     = var.secret_length >= 16
    error_message = "secret_length must be >= 16."
  }
}

variable "identity_schemas" {
  description = "Map of identity schema ID -> JSON schema content. When provided, the module mounts these schemas and configures Kratos to use them."
  type        = map(string)
  default     = {}
}

variable "identity_default_schema_id" {
  description = "Default identity schema ID to use when multiple schemas are provided. Must match a key in identity_schemas when identity_schemas is non-empty."
  type        = string
  default     = "default"
  validation {
    condition     = length(var.identity_schemas) == 0 || contains(keys(var.identity_schemas), var.identity_default_schema_id)
    error_message = "identity_default_schema_id must be a key in identity_schemas when identity_schemas is non-empty."
  }
}

variable "kratos_config" {
  description = "Raw Kratos configuration (merged under kratos.config). Use this to express configuration like in kratos.yml (e.g., log, serve, secrets, selfservice, methods, oidc, session, dsn, version)."
  type        = any
  default     = {}
}

variable "dsn_secret" {
  description = "Secret reference for DSN env injection to avoid storing DSN in plain text config. Specify name, key, and optional env_name (default 'DSN')."
  type = object({
    name     = string
    key      = string
    env_name = optional(string, "DSN")
  })
  default = null
}

variable "enable_automigration" {
  description = "Enable or disable Kratos automigration (kratos.automigration.enabled)."
  type        = bool
  default     = true
}

variable "oidc_mappers" {
  description = "Optional map of filename -> content for OIDC mapper files (e.g., github.mapper.jsonnet). These will be stored in a ConfigMap and mounted at /etc/kratos/."
  type        = map(string)
  default     = {}
}

variable "oidc_provider_secrets" {
  description = "Optional map of OIDC provider ID -> secret mount wiring. Each entry mounts the given Secret at /etc/kratos/oidc/<provider_id>/ and expects keys for client_id and client_secret."
  type = map(object({
    secret_name        = string
    client_id_key      = string
    client_secret_key  = string
    mount_subdir       = optional(string) # defaults to provider ID
  }))
  default = {}
}

variable "http_route" {
  description = "Optional HTTPRoute configuration: parent_ref (name, namespace?, sectionName?), hostnames, and rules. Rules map is passed directly to spec.rules."
  type = object({
    enabled     = bool
    name        = optional(string)
    namespace   = optional(string)
    parent_ref  = object({
      name        = string
      namespace   = optional(string)
      sectionName = optional(string)
    })
    hostnames   = list(string)
    rules       = optional(list(any), [])
  })
  default = null
}



variable "name" {
  description = "Helm release name for Ory Hydra"
  type        = string
  default     = "hydra"
  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.name))
    error_message = "name must be lowercase alphanumeric and hyphens only."
  }
}

variable "namespace" {
  description = "Kubernetes namespace to deploy Hydra into"
  type        = string
  nullable    = false
  validation {
    condition     = length(trimspace(var.namespace)) > 0
    error_message = "namespace must be a non-empty string."
  }
}

variable "repository" {
  description = "Helm repository URL that hosts the Ory charts"
  type        = string
  default     = "https://k8s.ory.sh/helm/charts"
  validation {
    condition     = can(regex("^https?://", var.repository))
    error_message = "repository must be an http(s) URL."
  }
}

variable "chart_version" {
  description = "Specific Helm chart version for Ory Hydra (optional)"
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
  description = "If true, generate Hydra system secret and inject into values"
  type        = bool
  default     = true
}

variable "secret_length" {
  description = "Length for generated Hydra system secret"
  type        = number
  default     = 32
  validation {
    condition     = var.secret_length >= 16
    error_message = "secret_length must be >= 16."
  }
}

variable "hydra_config" {
  description = "Raw Hydra configuration (merged under hydra.config). Mirrors hydra.yml (log, serve, urls, oauth2, strategies, secrets, ttl, ...)."
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



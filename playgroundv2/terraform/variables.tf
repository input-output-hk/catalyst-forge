variable "playground" {
  description = "Unified configuration for the playground"
  type = object({
    global = object({
      base_domain     = string
      kubeconfig_path = string
    })
    vm = object({
      name             = string
      cpus             = number
      memory           = string
      disk             = string
      microk8s_channel = string
      wait_timeout     = number
    })
    networking = object({
      metallb_range = optional(string)
      lb_offset     = number
      hostnames = object({
        gateway  = string
        registry = string
        mailpit  = string
        auth     = string
        forge    = string
        hydra    = string
      })
    })
    envoy = object({
      namespace           = string
      gateway_class_name  = string
      gateway_name        = string
      tls_secret_name     = string
      service_annotations = optional(map(string))
    })
    registry = object({
      namespace = string
      host      = string
      storage   = string
    })
    postgres = object({
      namespace     = string
      username      = string
      password      = string
      database      = string
      storage       = string
      storage_class = string
    })
    localstack = object({
      namespace = string
      image_tag = string
      endpoint  = string
    })
    eso = object({
      namespace             = string
      install_crds          = bool
      aws_creds_secret_name = string
      aws_access_key_id     = string
      aws_secret_access_key = string
      aws_region            = string
    })
    mailpit = object({
      namespace = string
      host      = string
      service   = string
      ui_port   = number
    })
    mock_oidc = object({
      namespace     = string
      host          = string
      client_id     = string
      client_secret = string
      service_port  = number
      image         = string
    })
    kratos = object({
      namespace = string
      host      = string
    })
    hydra = object({
      namespace = string
      host      = string
    })
  })
}

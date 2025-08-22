variable "kubeconfig_path" {
  description = "Path to kubeconfig for connecting to the cluster"
  type        = string
  default     = "../kubeconfig"
}

variable "namespace" {
  description = "Namespace to deploy Envoy Gateway into"
  type        = string
  default     = "envoy-gateway-system"
}

variable "gateway_class_name" {
  description = "Name of the GatewayClass resource"
  type        = string
  default     = "envoy-gateway-class"
}

variable "gateway_name" {
  description = "Name of the Gateway resource"
  type        = string
  default     = "envoy-gateway"
}

variable "load_balancer_ip" {
  description = "Optional static IP to request from MetalLB (must be in the MetalLB pool)"
  type        = string
  default     = ""
}

variable "service_annotations" {
  description = "Optional map of annotations to add to the EnvoyService"
  type        = map(string)
  default     = {}
}

variable "tls_secret_name" {
  description = "Kubernetes Secret name containing TLS cert and key for HTTPS termination"
  type        = string
  default     = "envoy-gateway-tls"
}

variable "registry_namespace" {
  description = "Namespace to deploy the in-cluster Docker registry"
  type        = string
  default     = "registry"
}

variable "registry_host" {
  description = "External hostname for the registry HTTPRoute (must be covered by the TLS cert)"
  type        = string
  default     = "registry.local.io"
}

variable "postgres_namespace" {
  description = "Namespace to deploy PostgreSQL into"
  type        = string
  default     = "default"
}

variable "pg_username" {
  description = "PostgreSQL superuser username (dev only)"
  type        = string
  default     = "postgres"
}

variable "pg_password" {
  description = "PostgreSQL superuser password (dev only)"
  type        = string
  default     = "postgres"
  sensitive   = true
}

variable "pg_database" {
  description = "Default database to create"
  type        = string
  default     = "postgres"
}

variable "pg_storage" {
  description = "Size of the PostgreSQL primary PVC"
  type        = string
  default     = "10Gi"
}

variable "pg_storage_class" {
  description = "Optional storageClassName for PostgreSQL PVC (leave empty to use default)"
  type        = string
  default     = ""
}

variable "localstack_namespace" {
  description = "Namespace to deploy LocalStack into"
  type        = string
  default     = "localstack"
}

variable "localstack_image_tag" {
  description = "Optional LocalStack image tag override (empty for chart default)"
  type        = string
  default     = ""
}

variable "external_secrets_namespace" {
  description = "Namespace to deploy External Secrets Operator into"
  type        = string
  default     = "external-secrets"
}

variable "external_secrets_install_crds" {
  description = "Whether to install CRDs via the Helm chart"
  type        = bool
  default     = true
}

variable "external_secrets_aws_creds_secret_name" {
  description = "Kubernetes Secret name holding AWS credentials for ESO"
  type        = string
  default     = "aws-credentials"
}

variable "eso_aws_access_key_id" {
  description = "AWS access key ID for LocalStack"
  type        = string
  default     = "test"
}

variable "eso_aws_secret_access_key" {
  description = "AWS secret access key for LocalStack"
  type        = string
  default     = "test"
  sensitive   = true
}

variable "eso_aws_region" {
  description = "AWS region to use for LocalStack-backed ESO (arbitrary)"
  type        = string
  default     = "us-east-1"
}

variable "localstack_endpoint" {
  description = "Endpoint URL for LocalStack services"
  type        = string
  default     = "http://localstack.localstack:4566"
}



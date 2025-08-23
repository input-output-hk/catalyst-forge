output "release_name" {
  description = "Name of the Helm release"
  value       = helm_release.hydra.name
}

output "namespace" {
  description = "Namespace Hydra is deployed into"
  value       = helm_release.hydra.namespace
}

output "chart_version" {
  description = "Helm chart version used"
  value       = helm_release.hydra.version
}

output "generated_system_secret" {
  description = "The generated Hydra system secret (empty if not generated)"
  value       = try(random_password.hydra["system"].result, "")
  sensitive   = true
}



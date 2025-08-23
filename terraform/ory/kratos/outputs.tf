output "release_name" {
  description = "Name of the Helm release"
  value       = helm_release.kratos.name
}

output "namespace" {
  description = "Namespace Kratos is deployed into"
  value       = helm_release.kratos.namespace
}

output "chart_version" {
  description = "Helm chart version used"
  value       = helm_release.kratos.version
}

output "generated_cookie_secret" {
  description = "The generated cookie secret (empty if not generated)"
  value       = try(random_password.kratos["cookie"].result, "")
  sensitive   = true
}

output "generated_cipher_secret" {
  description = "The generated cipher secret (empty if not generated)"
  value       = try(random_password.kratos["cipher"].result, "")
  sensitive   = true
}

output "generated_default_secret" {
  description = "The generated default secret (empty if not generated)"
  value       = try(random_password.kratos["default"].result, "")
  sensitive   = true
}



# Get all secrets for an app
data "spiceai_secrets" "app_secrets" {
  app_id = spiceai_app.example.id
}

# Output the list of secret names
output "secret_names" {
  description = "Names of all secrets configured for the app"
  value       = [for secret in data.spiceai_secrets.app_secrets.secrets : secret.name]
}

# Check if a specific secret exists
output "has_database_password" {
  description = "Whether the DATABASE_PASSWORD secret exists"
  value       = contains([for s in data.spiceai_secrets.app_secrets.secrets : s.name], "DATABASE_PASSWORD")
}

# Get secret metadata (note: values are not returned - they are masked)
output "secrets_info" {
  description = "Metadata about all secrets"
  value = [for secret in data.spiceai_secrets.app_secrets.secrets : {
    id         = secret.id
    name       = secret.name
    created_at = secret.created_at
    updated_at = secret.updated_at
  }]
}
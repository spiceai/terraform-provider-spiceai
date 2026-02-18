# Get API keys for an app
data "spiceai_api_keys" "example" {
  app_id = spiceai_app.example.id
}

# Use the primary API key
output "primary_api_key" {
  description = "The primary API key for the app"
  value       = data.spiceai_api_keys.example.api_key
  sensitive   = true
}

# Use the secondary API key (useful for key rotation)
output "secondary_api_key" {
  description = "The secondary API key for the app"
  value       = data.spiceai_api_keys.example.api_key_2
  sensitive   = true
}
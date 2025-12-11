# Get details about an existing app by ID
data "spiceai_app" "example" {
  id = "12345"
}

# Use data from an existing app
output "app_name" {
  description = "The name of the app"
  value       = data.spiceai_app.example.name
}

output "app_visibility" {
  description = "The visibility of the app"
  value       = data.spiceai_app.example.visibility
}

output "app_region" {
  description = "The region where the app is deployed"
  value       = data.spiceai_app.example.region
}

output "app_replicas" {
  description = "The number of replicas configured for the app"
  value       = data.spiceai_app.example.replicas
}

output "app_image_tag" {
  description = "The runtime image tag for the app"
  value       = data.spiceai_app.example.image_tag
}

# Reference an app created by another resource
data "spiceai_app" "from_resource" {
  id = spiceai_app.example.id
}

# Use data source to get app API key for other configurations
output "app_api_key" {
  description = "The API key for the app"
  value       = data.spiceai_app.from_resource.api_key
  sensitive   = true
}
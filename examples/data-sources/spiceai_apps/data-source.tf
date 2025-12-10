# List all apps in the organization
data "spiceai_apps" "all" {}

# Output the count of apps
output "app_count" {
  description = "The total number of apps in the organization"
  value       = length(data.spiceai_apps.all.apps)
}

# Output all app names
output "app_names" {
  description = "List of all app names"
  value       = [for app in data.spiceai_apps.all.apps : app.name]
}

# Output all app IDs
output "app_ids" {
  description = "List of all app IDs"
  value       = [for app in data.spiceai_apps.all.apps : app.id]
}

# Filter apps by visibility
output "public_apps" {
  description = "List of public app names"
  value       = [for app in data.spiceai_apps.all.apps : app.name if app.visibility == "public"]
}

output "private_apps" {
  description = "List of private app names"
  value       = [for app in data.spiceai_apps.all.apps : app.name if app.visibility == "private"]
}

# Get apps with their configurations
output "apps_with_config" {
  description = "Map of app names to their replica counts"
  value = {
    for app in data.spiceai_apps.all.apps : app.name => {
      replicas  = app.replicas
      image_tag = app.image_tag
      region    = app.region
    }
  }
}

# Find a specific app by name
locals {
  target_app_name = "my-app"
  target_app = [
    for app in data.spiceai_apps.all.apps : app
    if app.name == local.target_app_name
  ]
}

output "found_app_id" {
  description = "ID of the found app (if exists)"
  value       = length(local.target_app) > 0 ? local.target_app[0].id : null
}
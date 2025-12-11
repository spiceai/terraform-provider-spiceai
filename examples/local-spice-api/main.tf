# Test configuration for Spice.ai Terraform Provider
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}

# Provider configuration
provider "spiceai" {
  # Optional: Custom API endpoint (defaults to https://api.spice.ai)
  api_endpoint = "http://localhost:8080"

  # Optional: Custom OAuth endpoint (defaults to https://spice.ai/api/oauth/token)
  # oauth_endpoint = "https://spice.ai/api/oauth/token"
  oauth_endpoint = "https://dev.spice.ai/api/oauth/token"
}


# Local values
locals {
  app_name = "terraform-test-app-local-2"
}

# Create a test app with full configuration
resource "spiceai_app" "test" {
  name        = local.app_name
  description = "Test app for Terraform provider validation, updated description"
  visibility  = "private"

  # Spicepod configuration from template file with app name
  spicepod = templatefile("${path.module}/spicepod.yaml.tftpl", {
    app_name = local.app_name
  })

  # # Runtime configuration
  image_tag = "1.10.0-enterprise-models"
  replicas  = 3
  # production_branch = "main"
}

resource "spiceai_deployment" "test" {
  app_id = spiceai_app.test.id

  # Trigger new deployment when app configuration changes
  triggers = {
    spicepod  = spiceai_app.test.spicepod
    image_tag = spiceai_app.test.image_tag
    replicas  = spiceai_app.test.replicas
    image_tag = spiceai_app.test.image_tag
  }

  # Use app defaults
  debug = false
}

# Outputs for verification
# output "app_id" {
#   description = "The ID of the test app"
#   value       = spiceai_app.test.id
# }

# output "app_description" {
#   description = "The description of the test app"
#   value       = spiceai_app.test.description
# }

# output "app_name" {
#   description = "The name of the test app"
#   value       = spiceai_app.test.name
# }

# output "app_region" {
#   description = "The region of the test app"
#   value       = spiceai_app.test.region
# }

# output "app_cluster_id" {
#   description = "The cluster ID of the test app"
#   value       = spiceai_app.test.cluster_id
# }

# output "app_api_key" {
#   description = "The API key for the test app"
#   value       = spiceai_app.test.api_key
#   sensitive   = true
# }

# # Data source: List available regions
# data "spiceai_regions" "all" {}

# # Data source: List available container images
# data "spiceai_container_images" "stable" {
#   channel = "stable"
# }

# # Data source: Get API keys for the test app
# # data "spiceai_api_keys" "test" {
# #   app_id = spiceai_app.test.id
# # }

# output "available_regions" {
#   description = "List of available deployment regions"
#   value       = data.spiceai_regions.all.regions[*].region
# }

# output "default_region" {
#   description = "The default deployment region"
#   value       = data.spiceai_regions.all.default
# }

# output "available_image_tags" {
#   description = "List of available container image tags"
#   value       = data.spiceai_container_images.stable.images[*].tag
# }

# output "default_image_tag" {
#   description = "The default container image tag"
#   value       = data.spiceai_container_images.stable.default
# }

# output "api_key_primary" {
#   description = "Primary API key for the test app"
#   value       = data.spiceai_api_keys.test.api_key
#   sensitive   = true
# }

# output "api_key_secondary" {
#   description = "Secondary API key for the test app"
#   value       = data.spiceai_api_keys.test.api_key_2
#   sensitive   = true
# }

# # Create a deployment for the test app
# resource "spiceai_deployment" "test" {
#   app_id = spiceai_app.test.id

#   # Use app defaults
#   debug = false
# }

# # Data source: Read back the app we created
# data "spiceai_app" "test" {
#   id = spiceai_app.test.id
# }

# # Data source: List all apps
# data "spiceai_apps" "all" {}

# output "deployment_id" {
#   description = "The ID of the deployment"
#   value       = spiceai_deployment.test.id
# }

# output "deployment_status" {
#   description = "The status of the deployment"
#   value       = spiceai_deployment.test.status
# }

# output "data_source_app_name" {
#   description = "App name from data source"
#   value       = data.spiceai_app.test.name
# }

# output "all_apps_count" {
#   description = "Total number of apps in the organization"
#   value       = length(data.spiceai_apps.all.apps)
# }

# Test configuration for Spice.ai Terraform Provider
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}

# Provider configuration using environment variables
# Set SPICEAI_CLIENT_ID and SPICEAI_CLIENT_SECRET before running
provider "spiceai" {}

# Create a test app with full configuration
resource "spiceai_app" "test" {
  name        = "terraform-test-app"
  description = "Test app for Terraform provider validation"
  visibility  = "private"

  # Spicepod configuration
  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: terraform-test-app
    datasets:
      - name: test_dataset
        from: s3://spiceai-demo-datasets/taxi_trips/2024/
        params:
          file_format: parquet
  YAML

  # Runtime configuration
  image_tag         = "latest"
  replicas          = 1
  production_branch = "main"
}

# Create a deployment for the test app
resource "spiceai_deployment" "test" {
  app_id = spiceai_app.test.id

  # Use app defaults
  debug = false
}

# Data source: Read back the app we created
data "spiceai_app" "test" {
  id = spiceai_app.test.id
}

# Data source: List all apps
data "spiceai_apps" "all" {}

# Outputs for verification
output "app_id" {
  description = "The ID of the test app"
  value       = spiceai_app.test.id
}

output "app_name" {
  description = "The name of the test app"
  value       = spiceai_app.test.name
}

output "app_api_key" {
  description = "The API key for the test app"
  value       = spiceai_app.test.api_key
  sensitive   = true
}

output "deployment_id" {
  description = "The ID of the deployment"
  value       = spiceai_deployment.test.id
}

output "deployment_status" {
  description = "The status of the deployment"
  value       = spiceai_deployment.test.status
}

output "data_source_app_name" {
  description = "App name from data source"
  value       = data.spiceai_app.test.name
}

output "all_apps_count" {
  description = "Total number of apps in the organization"
  value       = length(data.spiceai_apps.all.apps)
}
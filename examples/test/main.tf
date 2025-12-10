# Simple test configuration for local development
# 
# Prerequisites:
# 1. Set environment variables:
#    export SPICEAI_CLIENT_ID="your-client-id"
#    export SPICEAI_CLIENT_SECRET="your-client-secret"
#
# 2. Configure ~/.terraformrc with dev_overrides (see ../.terraformrc.example)
#
# 3. Build the provider:
#    cd ../.. && go build -o terraform-provider-spiceai .
#
# Usage:
#    terraform plan
#    terraform apply

terraform {
  required_providers {
    spiceai = {
      source = "spiceai/spiceai"
    }
  }
}

provider "spiceai" {
  # Credentials are read from environment variables:
  # - SPICEAI_CLIENT_ID
  # - SPICEAI_CLIENT_SECRET
}

# Test 1: List all existing apps (data source)
data "spiceai_apps" "all" {}

output "existing_apps" {
  description = "List of all existing apps"
  value       = [for app in data.spiceai_apps.all.apps : { id = app.id, name = app.name }]
}

# Test 2: Create a new app
resource "spiceai_app" "test" {
  name        = "terraform-test-app"
  description = "Test app created by Terraform provider"
  visibility  = "private"
}

output "created_app_id" {
  description = "ID of the created app"
  value       = spiceai_app.test.id
}

output "created_app_api_key" {
  description = "API key of the created app"
  value       = spiceai_app.test.api_key
  sensitive   = true
}

# Test 3: Apply configuration to the app
resource "spiceai_app_config" "test" {
  app_id = spiceai_app.test.id

  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: terraform-test-app
  YAML

  replicas = 1
}

# Test 4: Create a deployment (uncomment to test)
# resource "spiceai_deployment" "test" {
#   app_id = spiceai_app.test.id
#   
#   depends_on = [spiceai_app_config.test]
# }
#
# output "deployment_id" {
#   value = spiceai_deployment.test.id
# }
#
# output "deployment_status" {
#   value = spiceai_deployment.test.status
# }
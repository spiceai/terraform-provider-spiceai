# Configure the Spice.ai provider
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}

# Provider configuration using OAuth client credentials
# You can set these via environment variables:
#   SPICEAI_CLIENT_ID
#   SPICEAI_CLIENT_SECRET
provider "spiceai" {
  # client_id     = "your-client-id"      # Or use SPICEAI_CLIENT_ID env var
  # client_secret = "your-client-secret"  # Or use SPICEAI_CLIENT_SECRET env var
  # api_endpoint  = "https://api.spice.ai" # Optional, defaults to production API
}

# Create a new Spice.ai app with configuration
resource "spiceai_app" "example" {
  name        = "my-terraform-app"
  description = "An app created and managed by Terraform"
  visibility  = "private"

  # Spicepod configuration (YAML or JSON string)
  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: my-terraform-app
    datasets:
      - name: my_dataset
        from: s3://my-bucket/data/
        params:
          file_format: parquet
  YAML

  # Runtime configuration
  image_tag             = "latest"
  replicas              = 1
  region                = "us-east-1"
  production_branch     = "main"
}

# Create a deployment for the app
resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id

  # Optional: override these for this specific deployment
  # image_tag = "v0.18.0"
  # replicas  = 2
  # debug     = false

  # Optional: Git information for tracking
  # branch         = "main"
  # commit_sha     = "abc123"
  # commit_message = "Deploy via Terraform"
}

# Data source: Get details about an existing app by ID
data "spiceai_app" "existing" {
  id = spiceai_app.example.id
}

# Data source: List all apps in the organization
data "spiceai_apps" "all" {}

# Outputs
output "app_id" {
  description = "The ID of the created app"
  value       = spiceai_app.example.id
}

output "app_api_key" {
  description = "The API key for the app"
  value       = spiceai_app.example.api_key
  sensitive   = true
}

output "deployment_id" {
  description = "The ID of the deployment"
  value       = spiceai_deployment.example.id
}

output "deployment_status" {
  description = "The current status of the deployment"
  value       = spiceai_deployment.example.status
}

output "all_apps" {
  description = "List of all apps in the organization"
  value       = [for app in data.spiceai_apps.all.apps : app.name]
}
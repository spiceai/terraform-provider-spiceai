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
  image_tag         = "latest"
  replicas          = 1
  region            = "us-east-1"
  production_branch = "main"
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

# Create secrets for the app
resource "spiceai_secret" "database_password" {
  app_id = spiceai_app.example.id
  name   = "DATABASE_PASSWORD"
  value  = var.database_password
}

resource "spiceai_secret" "api_token" {
  app_id = spiceai_app.example.id
  name   = "EXTERNAL_API_TOKEN"
  value  = var.api_token
}

# Add members to the organization
resource "spiceai_member" "developer" {
  username = "johndoe"
  roles    = ["member"]
}

resource "spiceai_member" "admin" {
  username = "janedoe"
  roles    = ["admin", "member"]
}

# Data source: Get details about an existing app by ID
data "spiceai_app" "existing" {
  id = spiceai_app.example.id
}

# Data source: List all apps in the organization
data "spiceai_apps" "all" {}

# Variables for sensitive values
variable "database_password" {
  type        = string
  description = "Database password for the app"
  sensitive   = true
  default     = ""
}

variable "api_token" {
  type        = string
  description = "External API token for the app"
  sensitive   = true
  default     = ""
}

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

output "secret_ids" {
  description = "The IDs of the created secrets"
  value = {
    database_password = spiceai_secret.database_password.id
    api_token         = spiceai_secret.api_token.id
  }
}

output "member_ids" {
  description = "The user IDs of added members"
  value = {
    developer = spiceai_member.developer.user_id
    admin     = spiceai_member.admin.user_id
  }
}

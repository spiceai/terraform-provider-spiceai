# Basic deployment using app defaults
resource "spiceai_deployment" "basic" {
  app_id = spiceai_app.example.id
}

# Deployment with custom settings
resource "spiceai_deployment" "custom" {
  app_id = spiceai_app.example.id

  # Override runtime settings for this deployment
  image_tag = "v0.18.0"
  replicas  = 3
  debug     = false
}

# Deployment with git tracking information
resource "spiceai_deployment" "with_git_info" {
  app_id = spiceai_app.example.id

  # Git information for tracking and rollback
  branch         = "main"
  commit_sha     = "abc123def456789"
  commit_message = "Deploy new feature via Terraform"

  # Runtime overrides
  image_tag = "latest"
  replicas  = 2
}

# Production deployment with all options
resource "spiceai_deployment" "production" {
  app_id = spiceai_app.production.id

  image_tag      = "v0.18.0"
  replicas       = 5
  debug          = false
  branch         = "release/v1.0"
  commit_sha     = "abc123def456789"
  commit_message = "Production release v1.0"
}
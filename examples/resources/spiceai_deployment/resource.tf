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

# Deployment of a revision of the connected GitHub repository.
# branch and commit_sha choose what to deploy. An app with no
# connected repository refuses them.
resource "spiceai_deployment" "release" {
  app_id = spiceai_app.production.id

  branch     = "release/v1.0"
  commit_sha = var.release_commit_sha # the full 40-character commit SHA

  # Runtime overrides
  image_tag = "v0.18.0"
  replicas  = 5
}
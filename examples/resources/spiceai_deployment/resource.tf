resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id

  # Optional: Override settings for this deployment
  # image_tag      = "v0.18.0"
  # replicas       = 2
  # debug          = false

  # Optional: Git tracking information
  # branch         = "main"
  # commit_sha     = "abc123def456"
  # commit_message = "Deploy via Terraform"

  depends_on = [spiceai_app_config.example]
}
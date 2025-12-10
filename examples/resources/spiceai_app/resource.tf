# Basic app with minimal configuration
resource "spiceai_app" "basic" {
  name        = "my-basic-app"
  description = "A basic Spice.ai app"
  visibility  = "private"
}

# Full app with spicepod and runtime configuration
resource "spiceai_app" "full" {
  name        = "my-full-app"
  description = "A fully configured Spice.ai app"
  visibility  = "private"

  # Spicepod configuration (YAML or JSON)
  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: my-full-app
    datasets:
      - name: taxi_trips
        from: s3://spiceai-demo-datasets/taxi_trips/2024/
        params:
          file_format: parquet
    models:
      - name: my_model
        from: openai:gpt-4
  YAML

  # Runtime configuration
  image_tag             = "latest"
  replicas              = 2
  node_group            = "default"
  region                = "us-east-1"
  storage_claim_size_gb = 10.0
  production_branch     = "main"
}

# App with JSON spicepod configuration
resource "spiceai_app" "json_config" {
  name        = "my-json-app"
  description = "An app with JSON spicepod configuration"
  visibility  = "public"

  spicepod = jsonencode({
    version = "v1beta1"
    kind    = "Spicepod"
    name    = "my-json-app"
    datasets = [
      {
        name = "my_dataset"
        from = "postgres://mydb/table"
      }
    ]
  })

  replicas = 1
}
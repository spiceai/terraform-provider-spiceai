# Basic app with minimal configuration
resource "spiceai_app" "basic" {
  name        = "my-basic-app"
  description = "A basic Spice.ai app"
  visibility  = "private"
  cname       = "us-west-2-prod-aws-data" # Required: region identifier from spiceai_regions data source
}

# Full app with spicepod and runtime configuration
resource "spiceai_app" "full" {
  name        = "my-full-app"
  description = "A fully configured Spice.ai app"
  visibility  = "private"
  cname       = "us-west-2-prod-aws-data"

  # Spicepod configuration (YAML or JSON)
  spicepod = <<-YAML
    version: v1
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
  region                = "us-east-2"
  storage_claim_size_gb = 10.0
  production_branch     = "main"
}

# App with JSON spicepod configuration
resource "spiceai_app" "json_config" {
  name        = "my-json-app"
  description = "An app with JSON spicepod configuration"
  visibility  = "public"
  cname       = "us-west-2-prod-aws-data"

  spicepod = jsonencode({
    version = "v1"
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

# App using regions data source to get cname
data "spiceai_regions" "available" {}

resource "spiceai_app" "with_region_lookup" {
  name        = "my-dynamic-region-app"
  description = "An app using dynamic region lookup"
  visibility  = "private"
  cname       = data.spiceai_regions.available.regions[0].cname
}

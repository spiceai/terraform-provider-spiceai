resource "spiceai_app_config" "example" {
  app_id = spiceai_app.example.id

  # Spicepod configuration (YAML or JSON)
  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: my-app
    datasets:
      - name: taxi_trips
        from: s3://spiceai-demo-datasets/taxi_trips/2024/
        params:
          file_format: parquet
  YAML

  # Runtime configuration
  image_tag             = "latest"
  replicas              = 2
  node_group            = "default"
  region                = "us-east-1"
  storage_claim_size_gb = 10.0
  production_branch     = "main"
}
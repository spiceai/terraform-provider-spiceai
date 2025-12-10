data "spiceai_app" "example" {
  id = "12345"
}

output "app_name" {
  value = data.spiceai_app.example.name
}

output "app_api_key" {
  value     = data.spiceai_app.example.api_key
  sensitive = true
}
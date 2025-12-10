data "spiceai_apps" "all" {}

output "all_app_names" {
  value = [for app in data.spiceai_apps.all.apps : app.name]
}
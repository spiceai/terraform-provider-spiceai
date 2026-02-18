# Get available stable container images
data "spiceai_container_images" "stable" {
  channel = "stable"
}

# Get available enterprise container images
data "spiceai_container_images" "enterprise" {
  channel = "enterprise"
}

# Output the default image tag
output "default_image_tag" {
  description = "The default container image tag"
  value       = data.spiceai_container_images.stable.default
}

# Output all available image tags
output "available_image_tags" {
  description = "List of available container image tags"
  value       = [for image in data.spiceai_container_images.stable.images : image.tag]
}
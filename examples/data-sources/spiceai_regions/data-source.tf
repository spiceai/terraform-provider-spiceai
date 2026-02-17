# Get all available deployment regions
data "spiceai_regions" "all" {}

# Get only production regions
data "spiceai_regions" "prod" {
  env = "prod"
}

# Output the default region
output "default_region" {
  description = "The default deployment region"
  value       = data.spiceai_regions.all.default
}

# Output all available region identifiers
output "available_regions" {
  description = "List of available deployment regions"
  value       = [for region in data.spiceai_regions.all.regions : region.region]
}

# Output region details
output "regions_info" {
  description = "Detailed information about available regions"
  value = [for region in data.spiceai_regions.all.regions : {
    name     = region.name
    region   = region.region
    cname    = region.cname
    provider = region.provider_name
  }]
}
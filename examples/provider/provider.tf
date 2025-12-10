# Configure the Spice.ai provider
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}

# Provider configuration using variables
provider "spiceai" {
  # OAuth client credentials for authentication
  # These can also be set via environment variables:
  #   SPICEAI_CLIENT_ID
  #   SPICEAI_CLIENT_SECRET
  client_id     = var.spiceai_client_id
  client_secret = var.spiceai_client_secret

  # Optional: Custom API endpoint (defaults to https://api.spice.ai)
  # api_endpoint = "https://api.spice.ai"

  # Optional: Custom OAuth endpoint (defaults to https://spice.ai/api/oauth/token)
  # oauth_endpoint = "https://spice.ai/api/oauth/token"
}

# Variables for credentials
variable "spiceai_client_id" {
  description = "OAuth client ID for Spice.ai API authentication"
  type        = string
  sensitive   = true
}

variable "spiceai_client_secret" {
  description = "OAuth client secret for Spice.ai API authentication"
  type        = string
  sensitive   = true
}

# Alternative: Provider configuration using environment variables only
# provider "spiceai" {
#   # Credentials are automatically read from:
#   # - SPICEAI_CLIENT_ID
#   # - SPICEAI_CLIENT_SECRET
# }
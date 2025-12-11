# Basic secret for storing a database password
resource "spiceai_secret" "database_password" {
  app_id = spiceai_app.example.id
  name   = "DATABASE_PASSWORD"
  value  = var.database_password
}

# Secret for an external API token
resource "spiceai_secret" "api_token" {
  app_id = spiceai_app.example.id
  name   = "EXTERNAL_API_TOKEN"
  value  = var.external_api_token
}

# Secret for S3 access credentials
resource "spiceai_secret" "aws_access_key" {
  app_id = spiceai_app.example.id
  name   = "AWS_ACCESS_KEY_ID"
  value  = var.aws_access_key_id
}

resource "spiceai_secret" "aws_secret_key" {
  app_id = spiceai_app.example.id
  name   = "AWS_SECRET_ACCESS_KEY"
  value  = var.aws_secret_access_key
}

# Example variables (define these in your variables.tf)
variable "database_password" {
  type      = string
  sensitive = true
}

variable "external_api_token" {
  type      = string
  sensitive = true
}

variable "aws_access_key_id" {
  type      = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}
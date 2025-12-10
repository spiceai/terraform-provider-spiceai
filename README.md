# Terraform Provider for Spice.ai

This Terraform provider allows you to manage [Spice.ai Cloud](https://spice.ai) resources including apps and deployments.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0 (or [OpenTofu](https://opentofu.org/) >= 1.0)
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- A Spice.ai account with OAuth client credentials

## Installation

### From Terraform Registry (Recommended)

```hcl
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}
```

### Building from Source

```bash
git clone https://github.com/spiceai/terraform-provider-spiceai.git
cd terraform-provider-spiceai
go build -o terraform-provider-spiceai
```

Then move the binary to your Terraform plugins directory or use a [dev override](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

## Authentication

The provider uses OAuth client credentials to authenticate with the Spice.ai API. You can obtain these credentials from your Spice.ai account.

### Configuration Options

```hcl
provider "spiceai" {
  client_id      = "your-client-id"      # Or use SPICEAI_CLIENT_ID env var
  client_secret  = "your-client-secret"  # Or use SPICEAI_CLIENT_SECRET env var
  api_endpoint   = "https://api.spice.ai" # Optional
  oauth_endpoint = "https://spice.ai/api/oauth/token" # Optional
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `SPICEAI_CLIENT_ID` | OAuth client ID |
| `SPICEAI_CLIENT_SECRET` | OAuth client secret |
| `SPICEAI_API_ENDPOINT` | API endpoint (default: `https://api.spice.ai`) |
| `SPICEAI_OAUTH_ENDPOINT` | OAuth token endpoint (default: `https://spice.ai/api/oauth/token`) |

## Resources

### spiceai_app

Manages a Spice.ai app and its configuration. This resource combines app creation with spicepod and runtime configuration.

```hcl
resource "spiceai_app" "example" {
  name        = "my-app"
  description = "My Spice.ai application"
  visibility  = "private"

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
```

#### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | The name of the app (min 4 chars, alphanumeric and hyphens only). Changing this forces a new resource. |
| `description` | string | No | A description of the app |
| `visibility` | string | No | The visibility (`public` or `private`, default: `private`) |
| `spicepod` | string | No | Spicepod configuration (YAML or JSON string) |
| `image_tag` | string | No | Spice.ai runtime image tag (e.g., `latest`, `v0.18.0`) |
| `replicas` | int | No | Number of replicas (1-10) |
| `node_group` | string | No | Node group for deployment |
| `region` | string | No | Deployment region |
| `storage_claim_size_gb` | float | No | Storage claim size in GB |
| `production_branch` | string | No | Production branch name for git-based deployments |

#### Read-Only Attributes

| Name | Description |
|------|-------------|
| `id` | The unique identifier of the app |
| `created_at` | Timestamp when the app was created |
| `api_key` | The API key for the app (sensitive) |

### spiceai_deployment

Creates a deployment for a Spice.ai app. Deployments are immutable - any changes will trigger creation of a new deployment.

```hcl
resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id

  # Optional overrides
  image_tag      = "v0.18.0"
  replicas       = 2
  branch         = "main"
  commit_sha     = "abc123def456"
  commit_message = "Deploy from Terraform"
  debug          = false
}
```

#### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `app_id` | string | Yes | The ID of the app to deploy. Changing this forces a new deployment. |
| `image_tag` | string | No | Override image tag for this deployment. Changing this forces a new deployment. |
| `replicas` | int | No | Override replicas (1-10). Changing this forces a new deployment. |
| `branch` | string | No | Git branch name. Changing this forces a new deployment. |
| `commit_sha` | string | No | Git commit SHA. Changing this forces a new deployment. |
| `commit_message` | string | No | Git commit message. Changing this forces a new deployment. |
| `debug` | bool | No | Enable debug mode (default: false). Changing this forces a new deployment. |

#### Read-Only Attributes

| Name | Description |
|------|-------------|
| `id` | The unique identifier of the deployment |
| `status` | Current status (`queued`, `deploying`, `running`, `failed`, `stopped`) |
| `created_at` | Timestamp when the deployment was created |
| `started_at` | Timestamp when the deployment started running |
| `finished_at` | Timestamp when the deployment finished |
| `error_message` | Error message if deployment failed |

> **Note:** Deployments are immutable. Any changes to deployment parameters will trigger a replacement (new deployment).

## Data Sources

### spiceai_app

Retrieves details about an existing app by ID.

```hcl
data "spiceai_app" "example" {
  id = "12345"
}

output "app_name" {
  value = data.spiceai_app.example.name
}

output "app_replicas" {
  value = data.spiceai_app.example.replicas
}
```

### spiceai_apps

Lists all apps in the organization.

```hcl
data "spiceai_apps" "all" {}

output "app_names" {
  value = [for app in data.spiceai_apps.all.apps : app.name]
}

output "private_apps" {
  value = [for app in data.spiceai_apps.all.apps : app.name if app.visibility == "private"]
}
```

## Example Usage

```hcl
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}

provider "spiceai" {
  # Credentials are read from environment variables:
  # SPICEAI_CLIENT_ID and SPICEAI_CLIENT_SECRET
}

# Create an app with configuration
resource "spiceai_app" "example" {
  name        = "my-terraform-app"
  description = "Managed by Terraform"
  visibility  = "private"

  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: my-terraform-app
    datasets:
      - name: taxi_trips
        from: s3://spiceai-demo-datasets/taxi_trips/2024/
        params:
          file_format: parquet
  YAML

  image_tag = "latest"
  replicas  = 1
}

# Deploy the app
resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id
}

output "app_id" {
  value = spiceai_app.example.id
}

output "app_api_key" {
  value     = spiceai_app.example.api_key
  sensitive = true
}

output "deployment_status" {
  value = spiceai_deployment.example.status
}
```

## Import

Resources can be imported using their IDs:

```bash
# Import an app
terraform import spiceai_app.example 12345

# Import a deployment (format: app_id/deployment_id)
terraform import spiceai_deployment.example 12345/67890
```

## Development

### Building

```bash
go build -o terraform-provider-spiceai
```

### Testing

```bash
go test ./...
```

### Running Locally

Create a `~/.terraformrc` file with a dev override:

```hcl
provider_installation {
  dev_overrides {
    "spiceai/spiceai" = "/path/to/terraform-provider-spiceai"
  }
  direct {}
}
```

### Generating Documentation

```bash
go generate ./...
```

## License

MPL-2.0. See [LICENSE](LICENSE) file.
# Terraform Provider for Spice.ai

This Terraform provider allows you to manage [Spice.ai](https://spice.ai) resources including apps, app configurations, and deployments.

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

Manages a Spice.ai app.

```hcl
resource "spiceai_app" "example" {
  name        = "my-app"
  description = "My Spice.ai application"
  visibility  = "private" # or "public"
}
```

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | The name of the app (min 4 chars, alphanumeric and hyphens only) |
| `description` | string | No | A description of the app |
| `visibility` | string | No | The visibility (`public` or `private`, default: `private`) |

#### Read-Only Attributes

- `id` - The unique identifier of the app
- `region` - The region where the app is deployed
- `created_at` - Timestamp when the app was created
- `api_key` - The API key for the app (sensitive)

### spiceai_app_config

Applies configuration to a Spice.ai app.

```hcl
resource "spiceai_app_config" "example" {
  app_id = spiceai_app.example.id

  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: my-app
    datasets:
      - name: my_dataset
        from: s3://bucket/path/
  YAML

  image_tag           = "latest"
  replicas            = 2
  node_group          = "default"
  region              = "us-east-1"
  storage_claim_size_gb = 10.0
  production_branch   = "main"
}
```

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `app_id` | string | Yes | The ID of the app to configure |
| `spicepod` | string | No | Spicepod configuration (YAML or JSON) |
| `image_tag` | string | No | Spice.ai runtime image tag |
| `replicas` | int | No | Number of replicas (1-10) |
| `node_group` | string | No | Node group for deployment |
| `region` | string | No | Deployment region |
| `storage_claim_size_gb` | float | No | Storage claim size in GB |
| `production_branch` | string | No | Production branch name |
| `description` | string | No | App description |
| `visibility` | string | No | App visibility |

### spiceai_deployment

Creates a deployment for a Spice.ai app.

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

  depends_on = [spiceai_app_config.example]
}
```

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `app_id` | string | Yes | The ID of the app to deploy |
| `image_tag` | string | No | Override image tag for this deployment |
| `replicas` | int | No | Override replicas (1-10) |
| `branch` | string | No | Git branch name |
| `commit_sha` | string | No | Git commit SHA |
| `commit_message` | string | No | Git commit message |
| `debug` | bool | No | Enable debug mode (default: false) |

#### Read-Only Attributes

- `id` - The unique identifier of the deployment
- `status` - Current status (`queued`, `deploying`, `running`, `failed`, `stopped`)
- `created_at` - Timestamp when the deployment was created
- `started_at` - Timestamp when the deployment started
- `finished_at` - Timestamp when the deployment finished
- `error_message` - Error message if deployment failed

> **Note:** Deployments are immutable. Any changes to deployment parameters will trigger a replacement (new deployment).

## Data Sources

### spiceai_app

Retrieves details about an existing app.

```hcl
data "spiceai_app" "example" {
  id = "12345"
}

output "app_name" {
  value = data.spiceai_app.example.name
}
```

### spiceai_apps

Lists all apps in the organization.

```hcl
data "spiceai_apps" "all" {}

output "app_names" {
  value = [for app in data.spiceai_apps.all.apps : app.name]
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

provider "spiceai" {}

# Create an app
resource "spiceai_app" "example" {
  name        = "my-terraform-app"
  description = "Managed by Terraform"
  visibility  = "private"
}

# Configure the app with a spicepod
resource "spiceai_app_config" "example" {
  app_id = spiceai_app.example.id

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

  replicas = 1
}

# Deploy the app
resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id
  
  depends_on = [spiceai_app_config.example]
}

output "app_id" {
  value = spiceai_app.example.id
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

# Import an app config (uses app ID)
terraform import spiceai_app_config.example 12345

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

## License

MPL-2.0. See [LICENSE](LICENSE) file.
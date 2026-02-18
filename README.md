# Terraform Provider for Spice.ai

[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet?logo=terraform)](https://registry.terraform.io/providers/spiceai/spiceai/latest)
[![CI](https://github.com/spiceai/terraform-provider-spiceai/actions/workflows/test.yml/badge.svg)](https://github.com/spiceai/terraform-provider-spiceai/actions/workflows/test.yml)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-brightgreen.svg)](LICENSE)

Terraform provider for managing [Spice.ai Cloud](https://spice.ai) resources.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0 (or [OpenTofu](https://opentofu.org/) >= 1.0)
- A [Spice.ai Cloud](https://spice.ai) account with [OAuth client credentials](https://docs.spice.ai/api/management#id-2.-oauth-2.0-client-credentials)

## Installation

Add the provider to your Terraform configuration:

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

## Authentication

The provider authenticates using OAuth 2.0 client credentials. Create an OAuth client in [Spice.ai Cloud](https://docs.spice.ai/api/management#id-2.-oauth-2.0-client-credentials) to obtain a client ID and secret.

### Configuration

```hcl
provider "spiceai" {
  client_id     = var.spiceai_client_id     # Or set SPICEAI_CLIENT_ID env var
  client_secret = var.spiceai_client_secret # Or set SPICEAI_CLIENT_SECRET env var
}
```

### Environment Variables

| Variable                       | Description                                                        |
| ------------------------------ | ------------------------------------------------------------------ |
| `SPICEAI_CLIENT_ID`            | OAuth client ID                                                    |
| `SPICEAI_CLIENT_SECRET`        | OAuth client secret                                                |
| `SPICEAI_API_ENDPOINT`         | API endpoint (default: `https://api.spice.ai`)                     |
| `SPICEAI_OAUTH_ENDPOINT`       | OAuth token endpoint (default: `https://spice.ai/api/oauth/token`) |

## Example

```hcl
terraform {
  required_providers {
    spiceai = {
      source  = "spiceai/spiceai"
      version = "~> 0.1"
    }
  }
}

resource "spiceai_app" "app" {
  name  = "my-spice-cloud-app"
  cname = "us-west-2-prod-aws-data"
}

resource "spiceai_deployment" "deploy" {
  app_id = spiceai_app.app.id
}
```

## Documentation

For detailed documentation on all resources, data sources, and their attributes, see the [docs](docs/) directory:

- **Resources:** [spiceai_app](docs/resources/app.md) · [spiceai_deployment](docs/resources/deployment.md) · [spiceai_secret](docs/resources/secret.md) · [spiceai_member](docs/resources/member.md)
- **Data Sources:** [spiceai_app](docs/data-sources/app.md) · [spiceai_apps](docs/data-sources/apps.md) · [spiceai_regions](docs/data-sources/regions.md) · [spiceai_secrets](docs/data-sources/secrets.md) · [spiceai_members](docs/data-sources/members.md) · [spiceai_api_keys](docs/data-sources/api_keys.md) · [spiceai_container_images](docs/data-sources/container_images.md)

## License

MPL-2.0. See [LICENSE](LICENSE) file.

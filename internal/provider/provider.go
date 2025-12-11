// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure SpiceAIProvider satisfies various provider interfaces.
var _ provider.Provider = &SpiceAIProvider{}
var _ provider.ProviderWithFunctions = &SpiceAIProvider{}

// SpiceAIProvider defines the provider implementation.
type SpiceAIProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// SpiceAIProviderModel describes the provider data model.
type SpiceAIProviderModel struct {
	ClientID               types.String `tfsdk:"client_id"`
	ClientSecret           types.String `tfsdk:"client_secret"`
	APIEndpoint            types.String `tfsdk:"api_endpoint"`
	OAuthEndpoint          types.String `tfsdk:"oauth_endpoint"`
	VercelProtectionBypass types.String `tfsdk:"vercel_protection_bypass"`
}

func (p *SpiceAIProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spiceai"

	resp.Version = p.version
}

func (p *SpiceAIProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The Spice.ai provider allows you to manage Spice.ai Cloud resources.

## Authentication

The provider uses OAuth client credentials for authentication. You can obtain these from the Spice.ai Cloud portal.

Credentials can be provided via:
- Provider configuration block
- Environment variables (recommended for CI/CD)

## Example Usage

` + "```hcl" + `
provider "spiceai" {
  # Credentials can also be set via environment variables:
  # SPICEAI_CLIENT_ID and SPICEAI_CLIENT_SECRET
  client_id     = var.spiceai_client_id
  client_secret = var.spiceai_client_secret
}
` + "```",
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				MarkdownDescription: "The OAuth client ID for Spice.ai API authentication. Can also be set via the `SPICEAI_CLIENT_ID` environment variable.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The OAuth client secret for Spice.ai API authentication. Can also be set via the `SPICEAI_CLIENT_SECRET` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_endpoint": schema.StringAttribute{
				MarkdownDescription: "The Spice.ai API endpoint. Defaults to `https://api.spice.ai`. Can also be set via the `SPICEAI_API_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"oauth_endpoint": schema.StringAttribute{
				MarkdownDescription: "The Spice.ai OAuth token endpoint. Defaults to `https://spice.ai/api/oauth/token`. Can also be set via the `SPICEAI_OAUTH_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"vercel_protection_bypass": schema.StringAttribute{
				MarkdownDescription: "Optional bypass token for Vercel deployment protection. When set, adds the `x-vercel-protection-bypass` header to API requests. Can also be set via the `SPICEAI_VERCEL_PROTECTION_BYPASS` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *SpiceAIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data SpiceAIProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from config or environment variables
	clientID := getConfigValue(data.ClientID, "SPICEAI_CLIENT_ID")
	clientSecret := getConfigValue(data.ClientSecret, "SPICEAI_CLIENT_SECRET")
	apiEndpoint := getConfigValue(data.APIEndpoint, "SPICEAI_API_ENDPOINT")
	oauthEndpoint := getConfigValue(data.OAuthEndpoint, "SPICEAI_OAUTH_ENDPOINT")
	vercelProtectionBypass := getConfigValue(data.VercelProtectionBypass, "SPICEAI_VERCEL_PROTECTION_BYPASS")

	// Validate required configuration
	if clientID == "" {
		resp.Diagnostics.AddError(
			"Missing Client ID",
			"The provider cannot create the Spice.ai API client as there is a missing or empty value for the Spice.ai OAuth client ID. "+
				"Set the client_id value in the configuration or use the SPICEAI_CLIENT_ID environment variable.",
		)
	}

	if clientSecret == "" {
		resp.Diagnostics.AddError(
			"Missing Client Secret",
			"The provider cannot create the Spice.ai API client as there is a missing or empty value for the Spice.ai OAuth client secret. "+
				"Set the client_secret value in the configuration or use the SPICEAI_CLIENT_SECRET environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create the Spice.ai API client
	spiceClient := client.NewSpiceAIClient(clientID, clientSecret, apiEndpoint, oauthEndpoint, vercelProtectionBypass)

	// Make the client available to resources and data sources
	resp.DataSourceData = spiceClient
	resp.ResourceData = spiceClient
}

func (p *SpiceAIProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
		NewDeploymentResource,
		NewSecretResource,
		NewMemberResource,
	}
}

func (p *SpiceAIProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAppDataSource,
		NewAppsDataSource,
		NewRegionsDataSource,
		NewContainerImagesDataSource,
		NewAPIKeysDataSource,
		NewSecretsDataSource,
		NewMembersDataSource,
	}
}

func (p *SpiceAIProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SpiceAIProvider{
			version: version,
		}
	}
}

// getConfigValue returns the value from the config if set, otherwise from the environment variable.
func getConfigValue(configValue types.String, envVar string) string {
	if !configValue.IsNull() && !configValue.IsUnknown() {
		return configValue.ValueString()
	}
	return os.Getenv(envVar)
}

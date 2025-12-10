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
	ClientID      types.String `tfsdk:"client_id"`
	ClientSecret  types.String `tfsdk:"client_secret"`
	APIEndpoint   types.String `tfsdk:"api_endpoint"`
	OAuthEndpoint types.String `tfsdk:"oauth_endpoint"`
}

func (p *SpiceAIProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spiceai"
	resp.Version = p.version
}

func (p *SpiceAIProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Spice.ai provider allows you to manage Spice.ai resources such as apps and deployments.",
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
	var clientID string
	if !data.ClientID.IsNull() && !data.ClientID.IsUnknown() {
		clientID = data.ClientID.ValueString()
	}
	if clientID == "" {
		clientID = os.Getenv("SPICEAI_CLIENT_ID")
	}

	var clientSecret string
	if !data.ClientSecret.IsNull() && !data.ClientSecret.IsUnknown() {
		clientSecret = data.ClientSecret.ValueString()
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("SPICEAI_CLIENT_SECRET")
	}

	var apiEndpoint string
	if !data.APIEndpoint.IsNull() && !data.APIEndpoint.IsUnknown() {
		apiEndpoint = data.APIEndpoint.ValueString()
	}
	if apiEndpoint == "" {
		apiEndpoint = os.Getenv("SPICEAI_API_ENDPOINT")
	}

	var oauthEndpoint string
	if !data.OAuthEndpoint.IsNull() && !data.OAuthEndpoint.IsUnknown() {
		oauthEndpoint = data.OAuthEndpoint.ValueString()
	}
	if oauthEndpoint == "" {
		oauthEndpoint = os.Getenv("SPICEAI_OAUTH_ENDPOINT")
	}

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
	spiceClient := client.NewSpiceAIClient(clientID, clientSecret, apiEndpoint, oauthEndpoint)

	// Make the client available to resources and data sources
	resp.DataSourceData = spiceClient
	resp.ResourceData = spiceClient
}

func (p *SpiceAIProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
		NewAppConfigResource,
		NewDeploymentResource,
	}
}

func (p *SpiceAIProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAppDataSource,
		NewAppsDataSource,
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

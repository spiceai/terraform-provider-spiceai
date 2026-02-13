// Copyright (c) Spice AI, Inc. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &APIKeysDataSource{}

func NewAPIKeysDataSource() datasource.DataSource {
	return &APIKeysDataSource{}
}

// APIKeysDataSource defines the data source implementation.
type APIKeysDataSource struct {
	client *client.SpiceAIClient
}

// APIKeysDataSourceModel describes the data source data model.
type APIKeysDataSourceModel struct {
	AppID   types.String `tfsdk:"app_id"`
	APIKey  types.String `tfsdk:"api_key"`
	APIKey2 types.String `tfsdk:"api_key_2"`
}

func (d *APIKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *APIKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the API keys for a Spice.ai app. Each app has two API keys to support key rotation.",

		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the app.",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The primary API key for the app.",
				Computed:            true,
				Sensitive:           true,
			},
			"api_key_2": schema.StringAttribute{
				MarkdownDescription: "The secondary API key for the app.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *APIKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.SpiceAIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.SpiceAIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *APIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	apiKeys, err := d.client.GetAPIKeys(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read API keys, got error: %s", err))
		return
	}

	data.APIKey = types.StringValue(apiKeys.APIKey)
	data.APIKey2 = types.StringValue(apiKeys.APIKey2)

	tflog.Trace(ctx, "read api_keys data source", map[string]interface{}{
		"app_id": appID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

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
var _ datasource.DataSource = &SecretsDataSource{}

func NewSecretsDataSource() datasource.DataSource {
	return &SecretsDataSource{}
}

// SecretsDataSource defines the data source implementation.
type SecretsDataSource struct {
	client *client.SpiceAIClient
}

// SecretsDataSourceModel describes the data source data model.
type SecretsDataSourceModel struct {
	AppID   types.String      `tfsdk:"app_id"`
	Secrets []SecretDataModel `tfsdk:"secrets"`
}

// SecretDataModel describes a single secret in the data source.
type SecretDataModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *SecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secrets"
}

func (d *SecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all secrets for a Spice.ai app. Note that secret values are not returned (they are masked).",

		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the app.",
				Required:            true,
			},
			"secrets": schema.ListNestedAttribute{
				MarkdownDescription: "List of secrets for the app.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the secret.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the secret.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The timestamp when the secret was created.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "The timestamp when the secret was last updated.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *SecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SecretsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	secrets, err := d.client.ListSecrets(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list secrets, got error: %s", err))
		return
	}

	data.Secrets = make([]SecretDataModel, len(secrets))
	for i, secret := range secrets {
		data.Secrets[i] = SecretDataModel{
			ID:        types.StringValue(strconv.FormatInt(secret.ID, 10)),
			Name:      types.StringValue(secret.Name),
			CreatedAt: types.StringValue(secret.CreatedAt),
			UpdatedAt: types.StringValue(secret.UpdatedAt),
		}
	}

	tflog.Trace(ctx, "read secrets data source", map[string]interface{}{
		"app_id":        appID,
		"secrets_count": len(secrets),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

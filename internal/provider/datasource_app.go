// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &AppDataSource{}

func NewAppDataSource() datasource.DataSource {
	return &AppDataSource{}
}

// AppDataSource defines the data source implementation.
type AppDataSource struct {
	client *client.SpiceAIClient
}

// AppDataSourceModel describes the data source data model.
type AppDataSourceModel struct {
	ID               types.String        `tfsdk:"id"`
	Name             types.String        `tfsdk:"name"`
	Description      types.String        `tfsdk:"description"`
	Visibility       types.String        `tfsdk:"visibility"`
	Region           types.String        `tfsdk:"region"`
	CreatedAt        types.String        `tfsdk:"created_at"`
	ProductionBranch types.String        `tfsdk:"production_branch"`
	APIKey           types.String        `tfsdk:"api_key"`
	Config           *AppConfigDataModel `tfsdk:"config"`
}

// AppConfigDataModel describes the app config in the data source.
type AppConfigDataModel struct {
	Spicepod           types.String  `tfsdk:"spicepod"`
	ImageTag           types.String  `tfsdk:"image_tag"`
	Replicas           types.Int64   `tfsdk:"replicas"`
	NodeGroup          types.String  `tfsdk:"node_group"`
	StorageClaimSizeGB types.Float64 `tfsdk:"storage_claim_size_gb"`
}

func (d *AppDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *AppDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves details about a Spice.ai app by its ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the app.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the app.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the app.",
				Computed:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The visibility of the app (public or private).",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region where the app is deployed.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the app was created.",
				Computed:            true,
			},
			"production_branch": schema.StringAttribute{
				MarkdownDescription: "The production branch for the app.",
				Computed:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The API key for the app.",
				Computed:            true,
				Sensitive:           true,
			},
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the app.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"spicepod": schema.StringAttribute{
						MarkdownDescription: "The spicepod configuration as a JSON string.",
						Computed:            true,
					},
					"image_tag": schema.StringAttribute{
						MarkdownDescription: "The Spice.ai runtime image tag.",
						Computed:            true,
					},
					"replicas": schema.Int64Attribute{
						MarkdownDescription: "The number of replicas.",
						Computed:            true,
					},
					"node_group": schema.StringAttribute{
						MarkdownDescription: "The node group for the app.",
						Computed:            true,
					},
					"storage_claim_size_gb": schema.Float64Attribute{
						MarkdownDescription: "The storage claim size in GB.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *AppDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (d *AppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AppDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Call the API
	app, err := d.client.GetApp(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read app, got error: %s", err))
		return
	}

	if app == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("App with ID %d not found", appID))
		return
	}

	// Map response to model
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))
	data.Name = types.StringValue(app.Name)
	data.Description = types.StringValue(app.Description)
	data.Visibility = types.StringValue(app.Visibility)
	data.Region = types.StringValue(app.Region)
	data.CreatedAt = types.StringValue(app.CreatedAt)
	data.ProductionBranch = types.StringValue(app.ProductionBranch)
	data.APIKey = types.StringValue(app.APIKey)

	// Map config if available
	if app.Config != nil {
		configModel := &AppConfigDataModel{
			ImageTag:           types.StringValue(app.Config.ImageTag),
			Replicas:           types.Int64Value(int64(app.Config.Replicas)),
			NodeGroup:          types.StringValue(app.Config.NodeGroup),
			StorageClaimSizeGB: types.Float64Value(app.Config.StorageClaimSizeGB),
		}

		// Handle spicepod - it comes as an interface, convert to string if present
		if app.Config.Spicepod != nil {
			if spicepodStr, ok := app.Config.Spicepod.(string); ok {
				configModel.Spicepod = types.StringValue(spicepodStr)
			} else {
				// It's a JSON object, marshal it
				if spicepodBytes, err := json.Marshal(app.Config.Spicepod); err == nil {
					configModel.Spicepod = types.StringValue(string(spicepodBytes))
				} else {
					configModel.Spicepod = types.StringNull()
				}
			}
		} else {
			configModel.Spicepod = types.StringNull()
		}

		data.Config = configModel
	}

	tflog.Trace(ctx, "read an app data source", map[string]interface{}{
		"id":   app.ID,
		"name": app.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

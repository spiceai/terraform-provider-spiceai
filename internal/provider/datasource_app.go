// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	// Identity
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	// Basic configuration
	Description      types.String `tfsdk:"description"`
	Visibility       types.String `tfsdk:"visibility"`
	ProductionBranch types.String `tfsdk:"production_branch"`
	Tags             types.Map    `tfsdk:"tags"`
	Cname            types.String `tfsdk:"cname"`

	// Spicepod configuration
	Spicepod types.String `tfsdk:"spicepod"`

	// Runtime configuration
	Registry           types.String  `tfsdk:"registry"`
	Image              types.String  `tfsdk:"image"`
	ImageTag           types.String  `tfsdk:"image_tag"`
	UpdateChannel      types.String  `tfsdk:"update_channel"`
	Replicas           types.Int64   `tfsdk:"replicas"`
	NodeGroup          types.String  `tfsdk:"node_group"`
	Region             types.String  `tfsdk:"region"`
	StorageClaimSizeGB types.Float64 `tfsdk:"storage_claim_size_gb"`

	// Read-only attributes
	CreatedAt types.String `tfsdk:"created_at"`
	ClusterID types.String `tfsdk:"cluster_id"`
	APIKey    types.String `tfsdk:"api_key"`
}

func (d *AppDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *AppDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves details about an existing Spice.ai app by its ID.",

		Attributes: map[string]schema.Attribute{
			// Identity attributes
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the app.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the app.",
				Computed:            true,
			},

			// Basic configuration attributes
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the app.",
				Computed:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The visibility of the app (`public` or `private`).",
				Computed:            true,
			},
			"production_branch": schema.StringAttribute{
				MarkdownDescription: "The production branch for the app.",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for the app.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"cname": schema.StringAttribute{
				MarkdownDescription: "The region identifier (cname) for the app.",
				Computed:            true,
			},

			// Spicepod configuration
			"spicepod": schema.StringAttribute{
				MarkdownDescription: "The spicepod configuration as a JSON string.",
				Computed:            true,
			},

			// Runtime configuration attributes
			"registry": schema.StringAttribute{
				MarkdownDescription: "Registry for the spiced image.",
				Computed:            true,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image name for the spiced container.",
				Computed:            true,
			},
			"image_tag": schema.StringAttribute{
				MarkdownDescription: "The Spice.ai runtime image tag.",
				Computed:            true,
			},
			"update_channel": schema.StringAttribute{
				MarkdownDescription: "Update channel for the spicepod.",
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
			"region": schema.StringAttribute{
				MarkdownDescription: "The region where the app is deployed.",
				Computed:            true,
			},
			"storage_claim_size_gb": schema.Float64Attribute{
				MarkdownDescription: "The storage claim size in GB.",
				Computed:            true,
			},

			// Read-only attributes
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The timestamp when the app was created.",
				Computed:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The Kubernetes cluster identifier where the app is deployed.",
				Computed:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "The API key for the app.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *AppDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

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
	d.mapAppToModel(&data, app)

	tflog.Trace(ctx, "read app data source", map[string]interface{}{
		"id":   app.ID,
		"name": app.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapAppToModel maps an API App response to the data source model.
func (d *AppDataSource) mapAppToModel(data *AppDataSourceModel, app *client.App) {
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))
	data.Name = types.StringValue(app.Name)
	data.Description = types.StringValue(app.Description)
	data.Visibility = types.StringValue(app.Visibility)
	data.ProductionBranch = types.StringValue(app.ProductionBranch)

	// Map tags
	if len(app.Tags) > 0 {
		tagElements := make(map[string]attr.Value)
		for k, v := range app.Tags {
			tagElements[k] = types.StringValue(v)
		}
		data.Tags = types.MapValueMust(types.StringType, tagElements)
	} else {
		data.Tags = types.MapNull(types.StringType)
	}

	data.CreatedAt = types.StringValue(app.CreatedAt)
	data.ClusterID = types.StringValue(app.ClusterID)
	data.APIKey = types.StringValue(app.APIKey)

	// Map cname
	if app.Cname != "" {
		data.Cname = types.StringValue(app.Cname)
	} else {
		data.Cname = types.StringNull()
	}

	// Region is inside config
	if app.Config != nil && app.Config.Region != "" {
		data.Region = types.StringValue(app.Config.Region)
	} else {
		data.Region = types.StringNull()
	}

	// Map config fields if available
	if app.Config != nil {
		data.Registry = types.StringValue(app.Config.Registry)
		data.Image = types.StringValue(app.Config.Image)
		data.ImageTag = types.StringValue(app.Config.ImageTag)
		data.UpdateChannel = types.StringValue(app.Config.UpdateChannel)
		data.Replicas = types.Int64Value(int64(app.Config.Replicas))
		data.NodeGroup = types.StringValue(app.Config.NodeGroup)
		data.StorageClaimSizeGB = types.Float64Value(app.Config.StorageClaimSizeGB)

		// Handle spicepod - convert to JSON string if present
		if app.Config.Spicepod != nil {
			if spicepodStr, ok := app.Config.Spicepod.(string); ok {
				data.Spicepod = types.StringValue(spicepodStr)
			} else {
				if spicepodBytes, err := json.Marshal(app.Config.Spicepod); err == nil {
					data.Spicepod = types.StringValue(string(spicepodBytes))
				} else {
					data.Spicepod = types.StringNull()
				}
			}
		} else {
			data.Spicepod = types.StringNull()
		}
	} else {
		data.Registry = types.StringNull()
		data.Image = types.StringNull()
		data.ImageTag = types.StringNull()
		data.UpdateChannel = types.StringNull()
		data.Replicas = types.Int64Null()
		data.NodeGroup = types.StringNull()
		data.StorageClaimSizeGB = types.Float64Null()
		data.Spicepod = types.StringNull()
	}
}

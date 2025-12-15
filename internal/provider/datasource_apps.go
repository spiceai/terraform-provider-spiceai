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
var _ datasource.DataSource = &AppsDataSource{}

func NewAppsDataSource() datasource.DataSource {
	return &AppsDataSource{}
}

// AppsDataSource defines the data source implementation.
type AppsDataSource struct {
	client *client.SpiceAIClient
}

// AppsDataSourceModel describes the data source data model.
type AppsDataSourceModel struct {
	Apps []AppModel `tfsdk:"apps"`
}

// AppModel describes an app in the list.
type AppModel struct {
	// Identity
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	// Basic configuration
	Description      types.String `tfsdk:"description"`
	Visibility       types.String `tfsdk:"visibility"`
	ProductionBranch types.String `tfsdk:"production_branch"`
	Tags             types.Map    `tfsdk:"tags"`

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

func (d *AppsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apps"
}

func (d *AppsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all Spice.ai apps in the authenticated organization.",

		Attributes: map[string]schema.Attribute{
			"apps": schema.ListNestedAttribute{
				MarkdownDescription: "List of apps in the organization.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						// Identity attributes
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the app.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *AppsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AppsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apps, err := d.client.ListApps(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list apps, got error: %s", err))
		return
	}

	// Map response to model
	data.Apps = make([]AppModel, len(apps))
	for i, app := range apps {
		data.Apps[i] = d.mapAppToModel(&app)
	}

	tflog.Trace(ctx, "read apps data source", map[string]interface{}{
		"count": len(apps),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapAppToModel maps an API App response to the data source model.
func (d *AppsDataSource) mapAppToModel(app *client.App) AppModel {
	model := AppModel{
		ID:               types.StringValue(strconv.FormatInt(app.ID, 10)),
		Name:             types.StringValue(app.Name),
		Description:      types.StringValue(app.Description),
		Visibility:       types.StringValue(app.Visibility),
		ProductionBranch: types.StringValue(app.ProductionBranch),
		CreatedAt:        types.StringValue(app.CreatedAt),
		ClusterID:        types.StringValue(app.ClusterID),
		APIKey:           types.StringValue(app.APIKey),
	}

	// Map tags
	if len(app.Tags) > 0 {
		tagElements := make(map[string]attr.Value)
		for k, v := range app.Tags {
			tagElements[k] = types.StringValue(v)
		}
		model.Tags = types.MapValueMust(types.StringType, tagElements)
	} else {
		model.Tags = types.MapNull(types.StringType)
	}

	// Region can be at top level or inside config
	if app.Region != "" {
		model.Region = types.StringValue(app.Region)
	} else if app.Config != nil && app.Config.Region != "" {
		model.Region = types.StringValue(app.Config.Region)
	} else {
		model.Region = types.StringNull()
	}

	// Map config fields if available
	if app.Config != nil {
		model.Registry = types.StringValue(app.Config.Registry)
		model.Image = types.StringValue(app.Config.Image)
		model.ImageTag = types.StringValue(app.Config.ImageTag)
		model.UpdateChannel = types.StringValue(app.Config.UpdateChannel)
		model.Replicas = types.Int64Value(int64(app.Config.Replicas))
		model.NodeGroup = types.StringValue(app.Config.NodeGroup)
		model.StorageClaimSizeGB = types.Float64Value(app.Config.StorageClaimSizeGB)

		// Handle spicepod - convert to JSON string if present
		if app.Config.Spicepod != nil {
			if spicepodStr, ok := app.Config.Spicepod.(string); ok {
				model.Spicepod = types.StringValue(spicepodStr)
			} else {
				if spicepodBytes, err := json.Marshal(app.Config.Spicepod); err == nil {
					model.Spicepod = types.StringValue(string(spicepodBytes))
				} else {
					model.Spicepod = types.StringNull()
				}
			}
		} else {
			model.Spicepod = types.StringNull()
		}
	} else {
		model.Registry = types.StringNull()
		model.Image = types.StringNull()
		model.ImageTag = types.StringNull()
		model.UpdateChannel = types.StringNull()
		model.Replicas = types.Int64Null()
		model.NodeGroup = types.StringNull()
		model.StorageClaimSizeGB = types.Float64Null()
		model.Spicepod = types.StringNull()
	}

	return model
}

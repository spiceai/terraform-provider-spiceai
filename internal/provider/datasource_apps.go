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
	Apps []AppDataModel `tfsdk:"apps"`
}

// AppDataModel describes an app in the list.
type AppDataModel struct {
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

func (d *AppsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apps"
}

func (d *AppsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all Spice.ai apps in the authenticated organization.",

		Attributes: map[string]schema.Attribute{
			"apps": schema.ListNestedAttribute{
				MarkdownDescription: "List of apps.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the app.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *AppsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AppsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call the API
	apps, err := d.client.ListApps(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list apps, got error: %s", err))
		return
	}

	// Map response to model
	data.Apps = make([]AppDataModel, len(apps))
	for i, app := range apps {
		appModel := AppDataModel{
			ID:               types.StringValue(strconv.FormatInt(app.ID, 10)),
			Name:             types.StringValue(app.Name),
			Description:      types.StringValue(app.Description),
			Visibility:       types.StringValue(app.Visibility),
			Region:           types.StringValue(app.Region),
			CreatedAt:        types.StringValue(app.CreatedAt),
			ProductionBranch: types.StringValue(app.ProductionBranch),
			APIKey:           types.StringValue(app.APIKey),
		}

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

			appModel.Config = configModel
		}

		data.Apps[i] = appModel
	}

	tflog.Trace(ctx, "read apps data source", map[string]interface{}{
		"count": len(apps),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

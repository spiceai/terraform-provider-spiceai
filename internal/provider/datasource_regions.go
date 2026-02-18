// Copyright (c) Spice AI, Inc. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RegionsDataSource{}

func NewRegionsDataSource() datasource.DataSource {
	return &RegionsDataSource{}
}

// RegionsDataSource defines the data source implementation.
type RegionsDataSource struct {
	client *client.SpiceAIClient
}

// RegionsDataSourceModel describes the data source data model.
type RegionsDataSourceModel struct {
	Env     types.String  `tfsdk:"env"`
	Regions []RegionModel `tfsdk:"regions"`
	Default types.String  `tfsdk:"default"`
}

// RegionModel describes a region in the list.
type RegionModel struct {
	Name         types.String `tfsdk:"name"`
	Region       types.String `tfsdk:"region"`
	Provider     types.String `tfsdk:"provider"`
	ProviderName types.String `tfsdk:"provider_name"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	Disabled     types.Bool   `tfsdk:"disabled"`
	Cname        types.String `tfsdk:"cname"`
}

func (d *RegionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

func (d *RegionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of available deployment regions for Spice.ai apps.",

		Attributes: map[string]schema.Attribute{
			"env": schema.StringAttribute{
				MarkdownDescription: "Filter by environment: `prod` or `dev`. If not specified, returns all regions.",
				Optional:            true,
			},
			"default": schema.StringAttribute{
				MarkdownDescription: "The default region identifier.",
				Computed:            true,
			},
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "List of available regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The display name of the region (e.g., `US East (Ohio)`).",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "The region identifier (e.g., `us-east-2`).",
							Computed:            true,
						},
						"provider": schema.StringAttribute{
							MarkdownDescription: "The cloud provider identifier (e.g., `aws`).",
							Computed:            true,
						},
						"provider_name": schema.StringAttribute{
							MarkdownDescription: "The cloud provider display name (e.g., `AWS`).",
							Computed:            true,
						},
						"is_default": schema.BoolAttribute{
							MarkdownDescription: "Whether this is the default region.",
							Computed:            true,
						},
						"disabled": schema.BoolAttribute{
							MarkdownDescription: "Whether this region is disabled.",
							Computed:            true,
						},
						"cname": schema.StringAttribute{
							MarkdownDescription: "The CNAME for the region (e.g., `us-west-2-prod-aws-data`).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RegionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RegionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	env := ""
	if !data.Env.IsNull() && !data.Env.IsUnknown() {
		env = data.Env.ValueString()
	}

	regionsResp, err := d.client.ListRegions(ctx, env)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list regions, got error: %s", err))
		return
	}

	// Map response to model
	data.Regions = make([]RegionModel, len(regionsResp.Regions))
	for i, region := range regionsResp.Regions {
		data.Regions[i] = RegionModel{
			Name:         types.StringValue(region.Name),
			Region:       types.StringValue(region.Region),
			Provider:     types.StringValue(region.Provider),
			ProviderName: types.StringValue(region.ProviderName),
			IsDefault:    types.BoolValue(region.IsDefault),
			Disabled:     types.BoolValue(region.Disabled),
			Cname:        types.StringValue(region.CName),
		}
	}
	data.Default = types.StringValue(regionsResp.Default)

	tflog.Trace(ctx, "read regions data source", map[string]interface{}{
		"count":   len(regionsResp.Regions),
		"default": regionsResp.Default,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

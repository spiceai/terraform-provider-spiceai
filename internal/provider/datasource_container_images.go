// Copyright (c) HashiCorp, Inc.
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
var _ datasource.DataSource = &ContainerImagesDataSource{}

func NewContainerImagesDataSource() datasource.DataSource {
	return &ContainerImagesDataSource{}
}

// ContainerImagesDataSource defines the data source implementation.
type ContainerImagesDataSource struct {
	client *client.SpiceAIClient
}

// ContainerImagesDataSourceModel describes the data source data model.
type ContainerImagesDataSourceModel struct {
	Channel types.String          `tfsdk:"channel"`
	Images  []ContainerImageModel `tfsdk:"images"`
	Default types.String          `tfsdk:"default"`
}

// ContainerImageModel describes a container image.
type ContainerImageModel struct {
	Name    types.String `tfsdk:"name"`
	Tag     types.String `tfsdk:"tag"`
	Channel types.String `tfsdk:"channel"`
}

func (d *ContainerImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_images"
}

func (d *ContainerImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of available Spice.ai runtime container images.",

		Attributes: map[string]schema.Attribute{
			"channel": schema.StringAttribute{
				MarkdownDescription: "Release channel to filter by (`stable` or `enterprise`). Defaults to `stable`.",
				Optional:            true,
			},
			"default": schema.StringAttribute{
				MarkdownDescription: "The default image tag.",
				Computed:            true,
			},
			"images": schema.ListNestedAttribute{
				MarkdownDescription: "List of available container images.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The full image name (e.g., `spiceai/spiceai:1.5.0-models`).",
							Computed:            true,
						},
						"tag": schema.StringAttribute{
							MarkdownDescription: "The image tag (e.g., `1.5.0-models`).",
							Computed:            true,
						},
						"channel": schema.StringAttribute{
							MarkdownDescription: "The release channel (`stable` or `enterprise`).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ContainerImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ContainerImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerImagesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channel := ""
	if !data.Channel.IsNull() && !data.Channel.IsUnknown() {
		channel = data.Channel.ValueString()
	}

	imagesResp, err := d.client.ListContainerImages(ctx, channel)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list container images, got error: %s", err))
		return
	}

	// Map response to model
	data.Images = make([]ContainerImageModel, len(imagesResp.Images))
	for i, image := range imagesResp.Images {
		data.Images[i] = ContainerImageModel{
			Name:    types.StringValue(image.Name),
			Tag:     types.StringValue(image.Tag),
			Channel: types.StringValue(image.Channel),
		}
	}
	data.Default = types.StringValue(imagesResp.Default)

	tflog.Trace(ctx, "read container images data source", map[string]interface{}{
		"count":   len(imagesResp.Images),
		"default": imagesResp.Default,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Copyright (c) HashiCorp, Inc.
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
var _ datasource.DataSource = &MembersDataSource{}

func NewMembersDataSource() datasource.DataSource {
	return &MembersDataSource{}
}

// MembersDataSource defines the data source implementation.
type MembersDataSource struct {
	client *client.SpiceAIClient
}

// MembersDataSourceModel describes the data source data model.
type MembersDataSourceModel struct {
	Members []MemberDataModel `tfsdk:"members"`
}

// MemberDataModel describes a single member in the data source.
type MemberDataModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.Int64  `tfsdk:"user_id"`
	Username  types.String `tfsdk:"username"`
	Roles     types.List   `tfsdk:"roles"`
	IsOwner   types.Bool   `tfsdk:"is_owner"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *MembersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_members"
}

func (d *MembersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all members in the organization.",

		Attributes: map[string]schema.Attribute{
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "List of members in the organization.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the member (same as user_id).",
							Computed:            true,
						},
						"user_id": schema.Int64Attribute{
							MarkdownDescription: "The user ID of the member.",
							Computed:            true,
						},
						"username": schema.StringAttribute{
							MarkdownDescription: "The username of the member.",
							Computed:            true,
						},
						"roles": schema.ListAttribute{
							MarkdownDescription: "The roles assigned to the member.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"is_owner": schema.BoolAttribute{
							MarkdownDescription: "Whether the member is the organization owner.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "The timestamp when the member joined the organization.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *MembersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MembersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	members, err := d.client.ListMembers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list members, got error: %s", err))
		return
	}

	data.Members = make([]MemberDataModel, len(members))
	for i, member := range members {
		// Convert []string to types.List
		rolesList, diags := types.ListValueFrom(ctx, types.StringType, member.Roles)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.Members[i] = MemberDataModel{
			ID:        types.StringValue(strconv.FormatInt(member.UserID, 10)),
			UserID:    types.Int64Value(member.UserID),
			Username:  types.StringValue(member.Username),
			Roles:     rolesList,
			IsOwner:   types.BoolValue(member.IsOwner),
			CreatedAt: types.StringValue(member.CreatedAt),
		}
	}

	tflog.Trace(ctx, "read members data source", map[string]interface{}{
		"members_count": len(members),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

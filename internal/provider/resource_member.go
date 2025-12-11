// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &MemberResource{}
var _ resource.ResourceWithImportState = &MemberResource{}

func NewMemberResource() resource.Resource {
	return &MemberResource{}
}

// MemberResource defines the resource implementation.
type MemberResource struct {
	client *client.SpiceAIClient
}

// MemberResourceModel describes the resource data model.
type MemberResourceModel struct {
	ID        types.String `tfsdk:"id"`
	UserID    types.Int64  `tfsdk:"user_id"`
	Username  types.String `tfsdk:"username"`
	Roles     types.List   `tfsdk:"roles"`
	IsOwner   types.Bool   `tfsdk:"is_owner"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *MemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member"
}

func (r *MemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages an organization member in Spice.ai.

Members represent users belonging to an organization. This resource allows you to add users to your organization and manage their roles.

## Example Usage

` + "```hcl" + `
resource "spiceai_member" "developer" {
  username = "johndoe"
  roles    = ["member"]
}

resource "spiceai_member" "admin" {
  username = "janedoe"
  roles    = ["admin", "member"]
}
` + "```" + `

## Import

Members can be imported using their user ID:

` + "```shell" + `
terraform import spiceai_member.example 123
` + "```" + `

~> **Note:** Organization owners cannot be modified or removed via this resource.
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the member (same as user_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The user ID of the member.",
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the user to add as a member. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.ListAttribute{
				MarkdownDescription: "The roles assigned to the member. Common roles include `admin` and `member`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"is_owner": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the member is the organization owner. Owners cannot be modified or removed.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the member was added to the organization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *MemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.SpiceAIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.SpiceAIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *MemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert roles from types.List to []string
	var roles []string
	if !data.Roles.IsNull() && !data.Roles.IsUnknown() {
		resp.Diagnostics.Append(data.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	addReq := &client.AddMemberRequest{
		Username: data.Username.ValueString(),
		Roles:    roles,
	}

	member, err := r.client.AddMember(ctx, addReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add member: %s", err))
		return
	}

	r.mapMemberToModel(ctx, member, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.GetMember(ctx, data.UserID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read member: %s", err))
		return
	}

	if member == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapMemberToModel(ctx, member, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if trying to modify an owner
	var stateData MemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if stateData.IsOwner.ValueBool() {
		resp.Diagnostics.AddError(
			"Cannot Modify Owner",
			"Organization owners cannot be modified via Terraform.",
		)
		return
	}

	// Convert roles from types.List to []string
	var roles []string
	if !data.Roles.IsNull() && !data.Roles.IsUnknown() {
		resp.Diagnostics.Append(data.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateReq := &client.UpdateMemberRequest{
		Roles: roles,
	}

	member, err := r.client.UpdateMember(ctx, data.UserID.ValueInt64(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update member: %s", err))
		return
	}

	r.mapMemberToModel(ctx, member, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if trying to delete an owner
	if data.IsOwner.ValueBool() {
		resp.Diagnostics.AddError(
			"Cannot Remove Owner",
			"Organization owners cannot be removed via Terraform.",
		)
		return
	}

	err := r.client.DeleteMember(ctx, data.UserID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove member: %s", err))
		return
	}
}

func (r *MemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: user_id
	userID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Could not parse user_id as integer: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
}

// mapMemberToModel maps a client.Member to MemberResourceModel.
func (r *MemberResource) mapMemberToModel(ctx context.Context, member *client.Member, model *MemberResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(strconv.FormatInt(member.UserID, 10))
	model.UserID = types.Int64Value(member.UserID)
	model.Username = types.StringValue(member.Username)
	model.IsOwner = types.BoolValue(member.IsOwner)
	model.CreatedAt = types.StringValue(member.CreatedAt)

	// Convert []string to types.List
	rolesList, diagsConv := types.ListValueFrom(ctx, types.StringType, member.Roles)
	diags.Append(diagsConv...)
	model.Roles = rolesList
}

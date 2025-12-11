// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

// SecretResource defines the resource implementation.
type SecretResource struct {
	client *client.SpiceAIClient
}

// SecretResourceModel describes the resource data model.
type SecretResourceModel struct {
	ID        types.String `tfsdk:"id"`
	AppID     types.Int64  `tfsdk:"app_id"`
	Name      types.String `tfsdk:"name"`
	Value     types.String `tfsdk:"value"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a secret for a Spice.ai app.

Secrets store sensitive configuration values (API keys, passwords, connection strings) for use in your Spicepod configuration. Secret values are encrypted at rest.

## Example Usage

` + "```hcl" + `
resource "spiceai_secret" "database_password" {
  app_id = spiceai_app.example.id
  name   = "DATABASE_PASSWORD"
  value  = var.database_password
}

resource "spiceai_secret" "api_token" {
  app_id = spiceai_app.example.id
  name   = "EXTERNAL_API_TOKEN"
  value  = var.api_token
}
` + "```" + `

## Import

Secrets can be imported using the format ` + "`app_id/secret_name`" + `:

` + "```shell" + `
terraform import spiceai_secret.example 123/DATABASE_PASSWORD
` + "```" + `

~> **Note:** The secret value cannot be imported and must be set after import.
`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the secret.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the app this secret belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the secret. Must start with a letter or underscore and contain only letters, numbers, and underscores. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the secret. This value is encrypted at rest and will not be returned by the API.",
				Required:            true,
				Sensitive:           true,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the secret was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the secret was last updated.",
			},
		},
	}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CreateSecretRequest{
		Name:  data.Name.ValueString(),
		Value: data.Value.ValueString(),
	}

	secret, err := r.client.CreateOrUpdateSecret(ctx, data.AppID.ValueInt64(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create secret: %s", err))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(secret.ID, 10))
	data.CreatedAt = types.StringValue(secret.CreatedAt)
	data.UpdatedAt = types.StringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.GetSecret(ctx, data.AppID.ValueInt64(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read secret: %s", err))
		return
	}

	if secret == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(secret.ID, 10))
	data.CreatedAt = types.StringValue(secret.CreatedAt)
	data.UpdatedAt = types.StringValue(secret.UpdatedAt)
	// Note: Value is not returned by the API (masked), so we keep the state value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.CreateSecretRequest{
		Name:  data.Name.ValueString(),
		Value: data.Value.ValueString(),
	}

	secret, err := r.client.CreateOrUpdateSecret(ctx, data.AppID.ValueInt64(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update secret: %s", err))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(secret.ID, 10))
	data.CreatedAt = types.StringValue(secret.CreatedAt)
	data.UpdatedAt = types.StringValue(secret.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSecret(ctx, data.AppID.ValueInt64(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete secret: %s", err))
		return
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: app_id/secret_name
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'app_id/secret_name', got: %s", req.ID),
		)
		return
	}

	appID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid App ID",
			fmt.Sprintf("Could not parse app_id as integer: %s", parts[0]),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), appID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	// Value must be set manually after import since it's not returned by the API
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), "")...)
}

// int64RequiresReplace returns a plan modifier that requires replacement when the value changes.
func int64RequiresReplace() planmodifier.Int64 {
	return &int64RequiresReplacePlanModifier{}
}

type int64RequiresReplacePlanModifier struct{}

func (m *int64RequiresReplacePlanModifier) Description(ctx context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m *int64RequiresReplacePlanModifier) MarkdownDescription(ctx context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m *int64RequiresReplacePlanModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() {
		return
	}

	if req.PlanValue.IsUnknown() {
		return
	}

	if !req.StateValue.Equal(req.PlanValue) {
		resp.RequiresReplace = true
	}
}

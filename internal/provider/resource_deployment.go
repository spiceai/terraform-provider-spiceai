// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DeploymentResource{}
var _ resource.ResourceWithImportState = &DeploymentResource{}

func NewDeploymentResource() resource.Resource {
	return &DeploymentResource{}
}

// DeploymentResource defines the resource implementation.
type DeploymentResource struct {
	client *client.SpiceAIClient
}

// DeploymentResourceModel describes the resource data model.
type DeploymentResourceModel struct {
	ID            types.String `tfsdk:"id"`
	AppID         types.String `tfsdk:"app_id"`
	ImageTag      types.String `tfsdk:"image_tag"`
	Replicas      types.Int64  `tfsdk:"replicas"`
	Branch        types.String `tfsdk:"branch"`
	CommitSHA     types.String `tfsdk:"commit_sha"`
	CommitMessage types.String `tfsdk:"commit_message"`
	Debug         types.Bool   `tfsdk:"debug"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
	StartedAt     types.String `tfsdk:"started_at"`
	FinishedAt    types.String `tfsdk:"finished_at"`
	ErrorMessage  types.String `tfsdk:"error_message"`
}

func (r *DeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *DeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Creates a deployment for a Spice.ai app.

A deployment uses the app's current spicepod configuration and deploys it to the Spice.ai cloud infrastructure. Deployments are immutable - any changes to deployment parameters will create a new deployment.

## Example Usage

` + "```hcl" + `
resource "spiceai_deployment" "example" {
  app_id = spiceai_app.example.id

  # Optional: Override settings for this deployment
  image_tag = "v0.18.0"
  replicas  = 2
  debug     = false

  # Optional: Git tracking information
  branch         = "main"
  commit_sha     = "abc123def456"
  commit_message = "Deploy via Terraform"
}
` + "```",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the deployment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the app to deploy. Changing this forces a new deployment to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_tag": schema.StringAttribute{
				MarkdownDescription: "Override the Spice.ai runtime image tag for this deployment. If not specified, uses the app's configured image tag. Changing this forces a new deployment to be created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "Override the number of replicas for this deployment. Must be between 1 and 10. If not specified, uses the app's configured replicas. Changing this forces a new deployment to be created.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
				PlanModifiers: []planmodifier.Int64{
					int64PlanModifierRequiresReplace{},
				},
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "Git branch name associated with this deployment. Changing this forces a new deployment to be created.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"commit_sha": schema.StringAttribute{
				MarkdownDescription: "Git commit SHA associated with this deployment. Changing this forces a new deployment to be created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"commit_message": schema.StringAttribute{
				MarkdownDescription: "Git commit message associated with this deployment. Changing this forces a new deployment to be created.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"debug": schema.BoolAttribute{
				MarkdownDescription: "Enable debug mode for this deployment. Changing this forces a new deployment to be created.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolPlanModifierRequiresReplace{},
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The current status of the deployment. Possible values: `queued`, `deploying`, `running`, `failed`, `stopped`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the deployment was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"started_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the deployment started running.",
			},
			"finished_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the deployment finished.",
			},
			"error_message": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Error message if the deployment failed.",
			},
		},
	}
}

func (r *DeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	createReq := &client.CreateDeploymentRequest{}

	if !data.ImageTag.IsNull() && !data.ImageTag.IsUnknown() {
		createReq.ImageTag = data.ImageTag.ValueString()
	}

	if !data.Replicas.IsNull() && !data.Replicas.IsUnknown() {
		replicas := int(data.Replicas.ValueInt64())
		createReq.Replicas = &replicas
	}

	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		createReq.Branch = data.Branch.ValueString()
	}

	if !data.CommitSHA.IsNull() && !data.CommitSHA.IsUnknown() {
		createReq.CommitSHA = data.CommitSHA.ValueString()
	}

	if !data.CommitMessage.IsNull() && !data.CommitMessage.IsUnknown() {
		createReq.CommitMessage = data.CommitMessage.ValueString()
	}

	if !data.Debug.IsNull() && !data.Debug.IsUnknown() {
		debug := data.Debug.ValueBool()
		createReq.Debug = &debug
	}

	deployment, err := r.client.CreateDeployment(ctx, appID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create deployment, got error: %s", err))
		return
	}

	r.mapDeploymentToModel(&data, deployment)

	tflog.Trace(ctx, "created deployment", map[string]interface{}{
		"id":     deployment.ID,
		"app_id": appID,
		"status": deployment.Status,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	deploymentID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Deployment ID", fmt.Sprintf("Unable to parse deployment ID: %s", err))
		return
	}

	deployment, err := r.client.GetDeployment(ctx, appID, deploymentID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read deployment, got error: %s", err))
		return
	}

	if deployment == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapDeploymentToModel(&data, deployment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Deployments are immutable - any changes require replacement
	// This should not be called due to RequiresReplace plan modifiers
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Deployments are immutable and cannot be updated. Any changes require creating a new deployment.",
	)
}

func (r *DeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Deployments cannot be deleted via API, they just get superseded by new deployments
	// We just remove from Terraform state
	tflog.Trace(ctx, "removed deployment from state (deployments cannot be deleted)", map[string]interface{}{
		"id":     data.ID.ValueString(),
		"app_id": data.AppID.ValueString(),
	})
}

func (r *DeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: app_id/deployment_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in format 'app_id/deployment_id', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// mapDeploymentToModel maps an API Deployment response to the Terraform model.
func (r *DeploymentResource) mapDeploymentToModel(data *DeploymentResourceModel, deployment *client.Deployment) {
	data.ID = types.StringValue(strconv.FormatInt(deployment.ID, 10))
	data.Status = types.StringValue(deployment.Status)

	if deployment.ImageTag != "" {
		data.ImageTag = types.StringValue(deployment.ImageTag)
	}

	if deployment.Replicas > 0 {
		data.Replicas = types.Int64Value(int64(deployment.Replicas))
	}

	if deployment.CreatedAt != "" {
		data.CreatedAt = types.StringValue(deployment.CreatedAt)
	}

	if deployment.CommitSHA != "" {
		data.CommitSHA = types.StringValue(deployment.CommitSHA)
	}

	if deployment.CommitMessage != "" {
		data.CommitMessage = types.StringValue(deployment.CommitMessage)
	}

	if deployment.StartedAt != "" {
		data.StartedAt = types.StringValue(deployment.StartedAt)
	} else {
		data.StartedAt = types.StringNull()
	}

	if deployment.FinishedAt != "" {
		data.FinishedAt = types.StringValue(deployment.FinishedAt)
	} else {
		data.FinishedAt = types.StringNull()
	}

	if deployment.ErrorMessage != "" {
		data.ErrorMessage = types.StringValue(deployment.ErrorMessage)
	} else {
		data.ErrorMessage = types.StringNull()
	}
}

// Custom plan modifiers for Int64 and Bool that require replacement

type int64PlanModifierRequiresReplace struct{}

func (m int64PlanModifierRequiresReplace) Description(ctx context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m int64PlanModifierRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m int64PlanModifierRequiresReplace) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() {
		return
	}

	if req.PlanValue.IsUnknown() {
		return
	}

	if req.StateValue.Equal(req.PlanValue) {
		return
	}

	resp.RequiresReplace = true
}

type boolPlanModifierRequiresReplace struct{}

func (m boolPlanModifierRequiresReplace) Description(ctx context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m boolPlanModifierRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m boolPlanModifierRequiresReplace) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() {
		return
	}

	if req.PlanValue.IsUnknown() {
		return
	}

	if req.StateValue.Equal(req.PlanValue) {
		return
	}

	resp.RequiresReplace = true
}

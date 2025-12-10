// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AppConfigResource{}
var _ resource.ResourceWithImportState = &AppConfigResource{}

func NewAppConfigResource() resource.Resource {
	return &AppConfigResource{}
}

// AppConfigResource defines the resource implementation.
type AppConfigResource struct {
	client *client.SpiceAIClient
}

// AppConfigResourceModel describes the resource data model.
type AppConfigResourceModel struct {
	ID                 types.String  `tfsdk:"id"`
	AppID              types.String  `tfsdk:"app_id"`
	Spicepod           types.String  `tfsdk:"spicepod"`
	ImageTag           types.String  `tfsdk:"image_tag"`
	Replicas           types.Int64   `tfsdk:"replicas"`
	NodeGroup          types.String  `tfsdk:"node_group"`
	Region             types.String  `tfsdk:"region"`
	StorageClaimSizeGB types.Float64 `tfsdk:"storage_claim_size_gb"`
	ProductionBranch   types.String  `tfsdk:"production_branch"`
	Description        types.String  `tfsdk:"description"`
	Visibility         types.String  `tfsdk:"visibility"`
}

func (r *AppConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_config"
}

func (r *AppConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Applies configuration to a Spice.ai app. This resource allows you to configure the spicepod, runtime settings, replicas, and other app configurations.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the app configuration (same as app_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the app to configure.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"spicepod": schema.StringAttribute{
				MarkdownDescription: "The spicepod configuration as a YAML or JSON string. This defines the data sources, models, and other spicepod settings.",
				Optional:            true,
			},
			"image_tag": schema.StringAttribute{
				MarkdownDescription: "The Spice.ai runtime image tag to use for deployments.",
				Optional:            true,
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "The number of replicas for the app. Must be between 1 and 10.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"node_group": schema.StringAttribute{
				MarkdownDescription: "The node group for the app deployment.",
				Optional:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region for the app deployment.",
				Optional:            true,
			},
			"storage_claim_size_gb": schema.Float64Attribute{
				MarkdownDescription: "The storage claim size in GB for the app.",
				Optional:            true,
			},
			"production_branch": schema.StringAttribute{
				MarkdownDescription: "The production branch for the app.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the app.",
				Optional:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The visibility of the app. Valid values are `public` or `private`.",
				Optional:            true,
			},
		},
	}
}

func (r *AppConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *AppConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Build update request
	updateReq := r.buildUpdateRequest(&data)

	// Call the API
	app, err := r.client.UpdateApp(ctx, appID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to apply app configuration, got error: %s", err))
		return
	}

	// Set the ID to the app ID
	data.ID = data.AppID

	// Update computed fields from response
	r.updateModelFromApp(&data, app)

	tflog.Trace(ctx, "applied app configuration", map[string]interface{}{
		"app_id": appID,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Call the API
	app, err := r.client.GetApp(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read app configuration, got error: %s", err))
		return
	}

	// If the app was not found, remove from state
	if app == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model from app response
	r.updateModelFromApp(&data, app)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.AppID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid App ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Build update request
	updateReq := r.buildUpdateRequest(&data)

	// Call the API
	app, err := r.client.UpdateApp(ctx, appID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update app configuration, got error: %s", err))
		return
	}

	// Update model from response
	r.updateModelFromApp(&data, app)

	tflog.Trace(ctx, "updated app configuration", map[string]interface{}{
		"app_id": appID,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For app config, we don't delete the app itself, just remove from state
	// The configuration remains on the app until explicitly changed
	tflog.Trace(ctx, "removed app configuration from state", map[string]interface{}{
		"app_id": data.AppID.ValueString(),
	})
}

func (r *AppConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using app_id
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// buildUpdateRequest creates an UpdateAppRequest from the model
func (r *AppConfigResource) buildUpdateRequest(data *AppConfigResourceModel) *client.UpdateAppRequest {
	updateReq := &client.UpdateAppRequest{}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		updateReq.Description = data.Description.ValueString()
	}

	if !data.Visibility.IsNull() && !data.Visibility.IsUnknown() {
		updateReq.Visibility = data.Visibility.ValueString()
	}

	if !data.ProductionBranch.IsNull() && !data.ProductionBranch.IsUnknown() {
		updateReq.ProductionBranch = data.ProductionBranch.ValueString()
	}

	if !data.Spicepod.IsNull() && !data.Spicepod.IsUnknown() {
		// Try to parse as JSON first, if that fails, send as string (YAML)
		spicepodStr := data.Spicepod.ValueString()
		var spicepodJSON interface{}
		if err := json.Unmarshal([]byte(spicepodStr), &spicepodJSON); err == nil {
			updateReq.Spicepod = spicepodJSON
		} else {
			// Send as YAML string
			updateReq.Spicepod = spicepodStr
		}
	}

	if !data.ImageTag.IsNull() && !data.ImageTag.IsUnknown() {
		updateReq.ImageTag = data.ImageTag.ValueString()
	}

	if !data.Replicas.IsNull() && !data.Replicas.IsUnknown() {
		replicas := int(data.Replicas.ValueInt64())
		updateReq.Replicas = &replicas
	}

	if !data.NodeGroup.IsNull() && !data.NodeGroup.IsUnknown() {
		updateReq.NodeGroup = data.NodeGroup.ValueString()
	}

	if !data.Region.IsNull() && !data.Region.IsUnknown() {
		updateReq.Region = data.Region.ValueString()
	}

	if !data.StorageClaimSizeGB.IsNull() && !data.StorageClaimSizeGB.IsUnknown() {
		size := data.StorageClaimSizeGB.ValueFloat64()
		updateReq.StorageClaimSizeGB = &size
	}

	return updateReq
}

// updateModelFromApp updates the model with values from the API response
func (r *AppConfigResource) updateModelFromApp(data *AppConfigResourceModel, app *client.App) {
	// Only update fields that are returned by the API and not explicitly set
	if app.Description != "" {
		data.Description = types.StringValue(app.Description)
	}

	if app.Visibility != "" {
		data.Visibility = types.StringValue(app.Visibility)
	}

	if app.ProductionBranch != "" {
		data.ProductionBranch = types.StringValue(app.ProductionBranch)
	}

	if app.Region != "" {
		data.Region = types.StringValue(app.Region)
	}

	// Update config fields if available
	if app.Config != nil {
		if app.Config.ImageTag != "" {
			data.ImageTag = types.StringValue(app.Config.ImageTag)
		}

		if app.Config.Replicas > 0 {
			data.Replicas = types.Int64Value(int64(app.Config.Replicas))
		}

		if app.Config.NodeGroup != "" {
			data.NodeGroup = types.StringValue(app.Config.NodeGroup)
		}

		if app.Config.StorageClaimSizeGB > 0 {
			data.StorageClaimSizeGB = types.Float64Value(app.Config.StorageClaimSizeGB)
		}

		// Handle spicepod - convert to JSON string if it's an object
		if app.Config.Spicepod != nil {
			if spicepodBytes, err := json.Marshal(app.Config.Spicepod); err == nil {
				data.Spicepod = types.StringValue(string(spicepodBytes))
			}
		}
	}
}

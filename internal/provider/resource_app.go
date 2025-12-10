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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}

func NewAppResource() resource.Resource {
	return &AppResource{}
}

// AppResource defines the resource implementation.
type AppResource struct {
	client *client.SpiceAIClient
}

// AppResourceModel describes the resource data model.
type AppResourceModel struct {
	// Identity
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	// Basic configuration
	Description      types.String `tfsdk:"description"`
	Visibility       types.String `tfsdk:"visibility"`
	ProductionBranch types.String `tfsdk:"production_branch"`

	// Spicepod configuration
	Spicepod types.String `tfsdk:"spicepod"`

	// Runtime configuration
	ImageTag           types.String  `tfsdk:"image_tag"`
	Replicas           types.Int64   `tfsdk:"replicas"`
	NodeGroup          types.String  `tfsdk:"node_group"`
	Region             types.String  `tfsdk:"region"`
	StorageClaimSizeGB types.Float64 `tfsdk:"storage_claim_size_gb"`

	// Read-only attributes
	CreatedAt types.String `tfsdk:"created_at"`
	ClusterID types.String `tfsdk:"cluster_id"`
	APIKey    types.String `tfsdk:"api_key"`
}

func (r *AppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages a Spice.ai app and its configuration.

Apps are the primary organizational unit in Spice.ai for deploying and managing spicepods. This resource creates an app and configures its spicepod, runtime settings, and deployment parameters.

## Example Usage

` + "```hcl" + `
resource "spiceai_app" "example" {
  name        = "my-terraform-app"
  description = "An app created and managed by Terraform"
  visibility  = "private"

  # Spicepod configuration (YAML or JSON)
  spicepod = <<-YAML
    version: v1beta1
    kind: Spicepod
    name: my-app
    datasets:
      - name: taxi_trips
        from: s3://spiceai-demo-datasets/taxi_trips/2024/
        params:
          file_format: parquet
  YAML

  # Runtime configuration
  image_tag             = "latest"
  replicas              = 2
  node_group            = "default"
  region                = "us-east-1"
  storage_claim_size_gb = 10.0
  production_branch     = "main"
}
` + "```",

		Attributes: map[string]schema.Attribute{
			// Identity attributes
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the app.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the app. Must be at least 4 characters and contain only letters, numbers, and hyphens. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Basic configuration attributes
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the app.",
				Optional:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The visibility of the app. Valid values are `public` or `private`. Defaults to `private`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("private"),
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private"),
				},
			},
			"production_branch": schema.StringAttribute{
				MarkdownDescription: "The production branch for the app. Used for git-based deployments.",
				Optional:            true,
				Computed:            true,
			},

			// Spicepod configuration
			"spicepod": schema.StringAttribute{
				MarkdownDescription: "The spicepod configuration as a YAML or JSON string. This defines the datasets, models, and other spicepod settings for the app.",
				Optional:            true,
				Computed:            true,
			},

			// Runtime configuration attributes
			"image_tag": schema.StringAttribute{
				MarkdownDescription: "The Spice.ai runtime image tag to use for deployments (e.g., `latest`, `v0.18.0`).",
				Optional:            true,
				Computed:            true,
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "The number of replicas for the app. Must be between 1 and 10.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"node_group": schema.StringAttribute{
				MarkdownDescription: "The node group for the app deployment.",
				Optional:            true,
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region for the app deployment.",
				Optional:            true,
				Computed:            true,
			},
			"storage_claim_size_gb": schema.Float64Attribute{
				MarkdownDescription: "The storage claim size in GB for the app.",
				Optional:            true,
				Computed:            true,
			},

			// Read-only attributes
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the app was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Kubernetes cluster identifier where the app is deployed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The API key for the app. This is used to authenticate requests to the app's endpoints.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create the app
	createReq := &client.CreateAppRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Visibility:  data.Visibility.ValueString(),
	}

	app, err := r.client.CreateApp(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create app, got error: %s", err))
		return
	}

	// Set the ID immediately so we can update configuration
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))

	tflog.Trace(ctx, "created app", map[string]interface{}{
		"id":   app.ID,
		"name": app.Name,
	})

	// Step 2: Apply configuration if any config attributes are set
	if r.hasConfigAttributes(&data) {
		updateReq := r.buildUpdateRequest(&data)

		app, err = r.client.UpdateApp(ctx, app.ID, updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to apply app configuration, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "applied app configuration", map[string]interface{}{
			"id": app.ID,
		})
	}

	// Map response to model
	r.mapAppToModel(&data, app)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	app, err := r.client.GetApp(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read app, got error: %s", err))
		return
	}

	if app == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapAppToModel(&data, app)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	updateReq := r.buildUpdateRequest(&data)

	app, err := r.client.UpdateApp(ctx, appID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update app, got error: %s", err))
		return
	}

	r.mapAppToModel(&data, app)

	tflog.Trace(ctx, "updated app", map[string]interface{}{
		"id":   app.ID,
		"name": app.Name,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	err = r.client.DeleteApp(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete app, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted app", map[string]interface{}{
		"id": appID,
	})
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// hasConfigAttributes checks if any configuration attributes are set.
func (r *AppResource) hasConfigAttributes(data *AppResourceModel) bool {
	return !data.Spicepod.IsNull() ||
		!data.ImageTag.IsNull() ||
		!data.Replicas.IsNull() ||
		!data.NodeGroup.IsNull() ||
		!data.Region.IsNull() ||
		!data.StorageClaimSizeGB.IsNull() ||
		!data.ProductionBranch.IsNull()
}

// buildUpdateRequest creates an UpdateAppRequest from the model.
func (r *AppResource) buildUpdateRequest(data *AppResourceModel) *client.UpdateAppRequest {
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

// mapAppToModel maps an API App response to the Terraform model.
func (r *AppResource) mapAppToModel(data *AppResourceModel, app *client.App) {
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))
	data.Name = types.StringValue(app.Name)

	if app.Description != "" {
		data.Description = types.StringValue(app.Description)
	} else {
		data.Description = types.StringNull()
	}

	if app.Visibility != "" {
		data.Visibility = types.StringValue(app.Visibility)
	} else {
		data.Visibility = types.StringNull()
	}

	if app.ProductionBranch != "" {
		data.ProductionBranch = types.StringValue(app.ProductionBranch)
	} else {
		data.ProductionBranch = types.StringNull()
	}

	// Region can be at top level or inside config
	if app.Region != "" {
		data.Region = types.StringValue(app.Region)
	} else if app.Config != nil && app.Config.Region != "" {
		data.Region = types.StringValue(app.Config.Region)
	} else {
		data.Region = types.StringNull()
	}

	if app.ClusterID != "" {
		data.ClusterID = types.StringValue(app.ClusterID)
	} else {
		data.ClusterID = types.StringNull()
	}

	if app.CreatedAt != "" {
		data.CreatedAt = types.StringValue(app.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}

	if app.APIKey != "" {
		data.APIKey = types.StringValue(app.APIKey)
	} else {
		data.APIKey = types.StringNull()
	}

	// Map config fields if available
	if app.Config != nil {
		if app.Config.ImageTag != "" {
			data.ImageTag = types.StringValue(app.Config.ImageTag)
		} else {
			data.ImageTag = types.StringNull()
		}

		if app.Config.Replicas > 0 {
			data.Replicas = types.Int64Value(int64(app.Config.Replicas))
		} else {
			data.Replicas = types.Int64Null()
		}

		if app.Config.NodeGroup != "" {
			data.NodeGroup = types.StringValue(app.Config.NodeGroup)
		} else {
			data.NodeGroup = types.StringNull()
		}

		if app.Config.StorageClaimSizeGB > 0 {
			data.StorageClaimSizeGB = types.Float64Value(app.Config.StorageClaimSizeGB)
		} else {
			data.StorageClaimSizeGB = types.Float64Null()
		}

		if app.Config.Spicepod != nil {
			if spicepodBytes, err := json.Marshal(app.Config.Spicepod); err == nil {
				data.Spicepod = types.StringValue(string(spicepodBytes))
			} else {
				data.Spicepod = types.StringNull()
			}
		} else {
			data.Spicepod = types.StringNull()
		}
	} else {
		// No config returned, set all config fields to null
		data.ImageTag = types.StringNull()
		data.Replicas = types.Int64Null()
		data.NodeGroup = types.StringNull()
		data.StorageClaimSizeGB = types.Float64Null()
		data.Spicepod = types.StringNull()
	}
}

// Copyright (c) Spice AI, Inc. 2025, 2026
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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

// SpicepodStringType is a custom string type that implements semantic equality
// for spicepod configurations, treating equivalent YAML and JSON as equal.
type SpicepodStringType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = SpicepodStringType{}

func (t SpicepodStringType) Equal(o attr.Type) bool {
	other, ok := o.(SpicepodStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t SpicepodStringType) String() string {
	return "SpicepodStringType"
}

func (t SpicepodStringType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return SpicepodStringValue{StringValue: in}, nil
}

func (t SpicepodStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}
	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("error converting string value: %v", diags)
	}
	return stringValuable, nil
}

func (t SpicepodStringType) ValueType(ctx context.Context) attr.Value {
	return SpicepodStringValue{}
}

// SpicepodStringValue is a custom string value that implements semantic equality
// for spicepod configurations.
type SpicepodStringValue struct {
	basetypes.StringValue
}

var _ basetypes.StringValuable = SpicepodStringValue{}
var _ basetypes.StringValuableWithSemanticEquals = SpicepodStringValue{}

func (v SpicepodStringValue) Equal(o attr.Value) bool {
	other, ok := o.(SpicepodStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v SpicepodStringValue) Type(ctx context.Context) attr.Type {
	return SpicepodStringType{}
}

// StringSemanticEquals implements semantic equality for spicepod strings.
// It normalizes both YAML and JSON to a common format for comparison.
func (v SpicepodStringValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	newValue, ok := newValuable.(SpicepodStringValue)
	if !ok {
		return false, nil
	}

	// If either is null or unknown, use standard equality
	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return v.Equal(newValue), nil
	}

	// Normalize both values to JSON for comparison
	oldNormalized := normalizeSpicepodToJSON(v.ValueString())
	newNormalized := normalizeSpicepodToJSON(newValue.ValueString())

	return oldNormalized == newNormalized, nil
}

// normalizeSpicepodToJSON converts a spicepod string (YAML or JSON) to a normalized JSON string.
func normalizeSpicepodToJSON(s string) string {
	// Try to parse as JSON first
	var data interface{}
	if err := json.Unmarshal([]byte(s), &data); err == nil {
		// Already JSON, normalize it
		normalized, err := json.Marshal(data)
		if err != nil {
			return s
		}
		return string(normalized)
	}

	// Try to parse as YAML
	if err := yaml.Unmarshal([]byte(s), &data); err == nil {
		// Convert to JSON
		normalized, err := json.Marshal(data)
		if err != nil {
			return s
		}
		return string(normalized)
	}

	// Return original if neither works
	return s
}

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
	Tags             types.Map    `tfsdk:"tags"`

	// Region identifier (required for create)
	Cname types.String `tfsdk:"cname"`

	// Spicepod configuration
	Spicepod SpicepodStringValue `tfsdk:"spicepod"`

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
  cname       = "us-west-2-prod-aws-data"  # Required: region identifier from spiceai_regions data source

  # Spicepod configuration (YAML or JSON)
  spicepod = <<-YAML
    version: v1
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
  region                = "us-east-2"
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
			"tags": schema.MapAttribute{
				MarkdownDescription: "Key-value tags for the app.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			// Region identifier (required for create)
			"cname": schema.StringAttribute{
				MarkdownDescription: "The region identifier (cname) for the app. This is required when creating an app and determines where the app is deployed. Get available values from the `spiceai_regions` data source. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Spicepod configuration
			"spicepod": schema.StringAttribute{
				MarkdownDescription: "The spicepod configuration as a YAML or JSON string. This defines the datasets, models, and other spicepod settings for the app.",
				Optional:            true,
				Computed:            true,
				CustomType:          SpicepodStringType{},
			},

			// Runtime configuration attributes
			"registry": schema.StringAttribute{
				MarkdownDescription: "Registry for the spiced image.",
				Optional:            true,
				Computed:            true,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image name for the spiced container.",
				Optional:            true,
				Computed:            true,
			},
			"image_tag": schema.StringAttribute{
				MarkdownDescription: "The Spice.ai runtime image tag to use for deployments (e.g., `latest`, `v0.18.0`).",
				Optional:            true,
				Computed:            true,
			},
			"update_channel": schema.StringAttribute{
				MarkdownDescription: "Update channel for the spicepod. Valid values are `stable`, `nightly`, `internal`, `internal-sandbox`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("stable", "nightly", "internal", "internal-sandbox"),
				},
			},
			"replicas": schema.Int64Attribute{
				MarkdownDescription: "The number of replicas for the app. Must be between 0 and 10.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 10),
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
		Cname:       data.Cname.ValueString(),
		Description: data.Description.ValueString(),
		Visibility:  data.Visibility.ValueString(),
	}

	// Add tags if provided
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make(map[string]string)
		data.Tags.ElementsAs(ctx, &tags, false)
		createReq.Tags = tags
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

	// Map response to model, preserving user's spicepod to avoid inconsistent result errors
	r.mapAppToModel(&data, app, true)

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

	// During Read, use API values (don't preserve user spicepod)
	r.mapAppToModel(&data, app, false)

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

	// Map response to model, preserving user's spicepod to avoid inconsistent result errors
	r.mapAppToModel(&data, app, true)

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
		!data.Registry.IsNull() ||
		!data.Image.IsNull() ||
		!data.ImageTag.IsNull() ||
		!data.UpdateChannel.IsNull() ||
		!data.Replicas.IsNull() ||
		!data.NodeGroup.IsNull() ||
		!data.Region.IsNull() ||
		!data.StorageClaimSizeGB.IsNull() ||
		!data.ProductionBranch.IsNull() ||
		!data.Tags.IsNull()
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
			// Already valid JSON
			updateReq.Spicepod = spicepodJSON
		} else {
			// Try to parse as YAML and convert to JSON-compatible structure
			var yamlData interface{}
			if err := yaml.Unmarshal([]byte(spicepodStr), &yamlData); err == nil {
				// Convert YAML to JSON-compatible structure (yaml.v3 uses map[string]interface{})
				updateReq.Spicepod = yamlData
			} else {
				// If neither JSON nor YAML, send as raw string (will likely fail on API side)
				updateReq.Spicepod = spicepodStr
			}
		}
	}

	if !data.Registry.IsNull() && !data.Registry.IsUnknown() {
		updateReq.Registry = data.Registry.ValueString()
	}

	if !data.Image.IsNull() && !data.Image.IsUnknown() {
		updateReq.Image = data.Image.ValueString()
	}

	if !data.ImageTag.IsNull() && !data.ImageTag.IsUnknown() {
		updateReq.ImageTag = data.ImageTag.ValueString()
	}

	if !data.UpdateChannel.IsNull() && !data.UpdateChannel.IsUnknown() {
		updateReq.UpdateChannel = data.UpdateChannel.ValueString()
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

	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		tags := make(map[string]string)
		for k, v := range data.Tags.Elements() {
			if strVal, ok := v.(types.String); ok {
				tags[k] = strVal.ValueString()
			}
		}
		updateReq.Tags = tags
	}

	return updateReq
}

// mapAppToModel maps an API App response to the Terraform model.
// mapAppToModel maps the API response to the Terraform model.
// If preserveUserSpicepod is true, the user's original spicepod value is preserved
// (used during Create/Update to avoid "inconsistent result after apply" errors).
// If false, the API response is used (for Read operations).
func (r *AppResource) mapAppToModel(data *AppResourceModel, app *client.App, preserveUserSpicepod bool) {
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

	// Map tags - only update if tags were configured by the user
	// If tags is null in config, preserve null to avoid "inconsistent result after apply" error
	if !data.Tags.IsNull() {
		if len(app.Tags) > 0 {
			tagElements := make(map[string]attr.Value)
			for k, v := range app.Tags {
				tagElements[k] = types.StringValue(v)
			}
			data.Tags = types.MapValueMust(types.StringType, tagElements)
		} else {
			data.Tags = types.MapNull(types.StringType)
		}
	}
	// If data.Tags is null (not configured), leave it as null

	// Map cname from API response
	if app.Cname != "" {
		data.Cname = types.StringValue(app.Cname)
	}

	// Region is inside config
	if app.Config != nil && app.Config.Region != "" {
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
		if app.Config.Registry != "" {
			data.Registry = types.StringValue(app.Config.Registry)
		} else {
			data.Registry = types.StringNull()
		}

		if app.Config.Image != "" {
			data.Image = types.StringValue(app.Config.Image)
		} else {
			data.Image = types.StringNull()
		}

		if app.Config.ImageTag != "" {
			data.ImageTag = types.StringValue(app.Config.ImageTag)
		} else {
			data.ImageTag = types.StringNull()
		}

		if app.Config.UpdateChannel != "" {
			data.UpdateChannel = types.StringValue(app.Config.UpdateChannel)
		} else {
			data.UpdateChannel = types.StringNull()
		}

		data.Replicas = types.Int64Value(int64(app.Config.Replicas))

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
				apiSpicepodJSON := string(spicepodBytes)
				// During Create/Update, preserve the user's original spicepod value
				// to avoid "inconsistent result after apply" errors from Terraform.
				// The semantic equality will handle comparison during plan/refresh.
				if preserveUserSpicepod && !data.Spicepod.IsNull() && !data.Spicepod.IsUnknown() {
					// Keep the user's original value (YAML or JSON) - don't overwrite
				} else {
					// During Read or when user didn't provide a value, use API response
					data.Spicepod = SpicepodStringValue{StringValue: types.StringValue(apiSpicepodJSON)}
				}
			} else {
				if !preserveUserSpicepod {
					data.Spicepod = SpicepodStringValue{StringValue: types.StringNull()}
				}
			}
		} else {
			if !preserveUserSpicepod {
				data.Spicepod = SpicepodStringValue{StringValue: types.StringNull()}
			}
		}
	} else {
		// No config returned, set all config fields to null
		data.Registry = types.StringNull()
		data.Image = types.StringNull()
		data.ImageTag = types.StringNull()
		data.UpdateChannel = types.StringNull()
		data.Replicas = types.Int64Null()
		data.NodeGroup = types.StringNull()
		data.StorageClaimSizeGB = types.Float64Null()
		data.Spicepod = SpicepodStringValue{StringValue: types.StringNull()}
	}
}

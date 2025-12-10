// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-spiceai/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Visibility  types.String `tfsdk:"visibility"`
	Region      types.String `tfsdk:"region"`
	CreatedAt   types.String `tfsdk:"created_at"`
	APIKey      types.String `tfsdk:"api_key"`
}

func (r *AppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Spice.ai app. Apps are the primary organizational unit in Spice.ai for deploying and managing spicepods.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the app.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the app. Must be at least 4 characters and contain only letters, numbers, and hyphens.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the app.",
				Optional:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The visibility of the app. Valid values are `public` or `private`. Defaults to `private`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("private"),
			},
			"region": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The region where the app is deployed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The timestamp when the app was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The API key for the app.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request
	createReq := &client.CreateAppRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Visibility:  data.Visibility.ValueString(),
	}

	// Call the API
	app, err := r.client.CreateApp(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create app, got error: %s", err))
		return
	}

	// Map response to model
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))
	data.Name = types.StringValue(app.Name)
	data.Description = types.StringValue(app.Description)
	data.Visibility = types.StringValue(app.Visibility)
	data.Region = types.StringValue(app.Region)
	data.CreatedAt = types.StringValue(app.CreatedAt)
	data.APIKey = types.StringValue(app.APIKey)

	tflog.Trace(ctx, "created an app resource", map[string]interface{}{
		"id":   app.ID,
		"name": app.Name,
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Call the API
	app, err := r.client.GetApp(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read app, got error: %s", err))
		return
	}

	// If the app was not found, remove from state
	if app == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Map response to model
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))
	data.Name = types.StringValue(app.Name)
	data.Description = types.StringValue(app.Description)
	data.Visibility = types.StringValue(app.Visibility)
	data.Region = types.StringValue(app.Region)
	data.CreatedAt = types.StringValue(app.CreatedAt)
	data.APIKey = types.StringValue(app.APIKey)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Create update request
	updateReq := &client.UpdateAppRequest{
		Description: data.Description.ValueString(),
		Visibility:  data.Visibility.ValueString(),
	}

	// Call the API
	app, err := r.client.UpdateApp(ctx, appID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update app, got error: %s", err))
		return
	}

	// Map response to model
	data.ID = types.StringValue(strconv.FormatInt(app.ID, 10))
	data.Name = types.StringValue(app.Name)
	data.Description = types.StringValue(app.Description)
	data.Visibility = types.StringValue(app.Visibility)
	data.Region = types.StringValue(app.Region)
	data.CreatedAt = types.StringValue(app.CreatedAt)
	data.APIKey = types.StringValue(app.APIKey)

	tflog.Trace(ctx, "updated an app resource", map[string]interface{}{
		"id":   app.ID,
		"name": app.Name,
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse app ID
	appID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse app ID: %s", err))
		return
	}

	// Call the API
	err = r.client.DeleteApp(ctx, appID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete app, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an app resource", map[string]interface{}{
		"id": appID,
	})
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

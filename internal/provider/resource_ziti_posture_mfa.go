// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiPostureMfaResource{}
var _ resource.ResourceWithImportState = &ZitiPostureMfaResource{}

func NewZitiPostureMfaResource() resource.Resource {
	return &ZitiPostureMfaResource{}
}

// ZitiPostureMfaResource defines the resource implementation.
type ZitiPostureMfaResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiPostureMfaResourceModel describes the resource data model.
type ZitiPostureMfaResourceModel struct {
	ID types.String `tfsdk:"id"`

	Name           types.String `tfsdk:"name"`
	RoleAttributes types.List   `tfsdk:"role_attributes"`
	Tags           types.Map    `tfsdk:"tags"`

	IgnoreLegacyEndpoints types.Bool  `tfsdk:"ignore_legacy_endpoints"`
	PromptOnUnlock        types.Bool  `tfsdk:"prompt_on_unlock"`
	PromptOnWake          types.Bool  `tfsdk:"prompt_on_wake"`
	TimeoutSeconds        types.Int64 `tfsdk:"timeout_seconds"`
}

func (r *ZitiPostureMfaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_mfa"
}

func (r *ZitiPostureMfaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A resource to define a host.v1 config of Ziti",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Required:            true,
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
			"ignore_legacy_endpoints": schema.BoolAttribute{
				MarkdownDescription: "Controls whether legacy endpoints are ignored for this mfa check",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"prompt_on_unlock": schema.BoolAttribute{
				MarkdownDescription: "Controls whether user is prompted to pass mfa check after a device unlock. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"prompt_on_wake": schema.BoolAttribute{
				MarkdownDescription: "Controls whether user is prompted to pass mfa check after a device wake. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "Time after which controls when mfa check times out. Defaults to -1, which indicates no limit.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
			},
			"tags": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags of the service.",
				Optional:            true,
				Computed:            true,
				Default:             mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
		},
	}
}

func (r *ZitiPostureMfaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*edge_apis.ManagementApiClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *apis.ManagementApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ZitiPostureMfaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiPostureMfaResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
	tags := TagsFromAttributes(plan.Tags.Elements())

	postureCheckMfaProperties := rest_model.PostureCheckMfaProperties{
		IgnoreLegacyEndpoints: plan.IgnoreLegacyEndpoints.ValueBool(),
		PromptOnUnlock:        plan.PromptOnUnlock.ValueBool(),
		PromptOnWake:          plan.PromptOnWake.ValueBool(),
		TimeoutSeconds:        plan.TimeoutSeconds.ValueInt64(),
	}
	postureCheckCreate := rest_model.PostureCheckMfaCreate{
		PostureCheckMfaProperties: postureCheckMfaProperties,
	}

	postureCheckCreate.SetName(&name)
	postureCheckCreate.SetRoleAttributes(&roleAttributes)
	postureCheckCreate.SetTags(tags)

	params := posture_checks.NewCreatePostureCheckParams()

	params.PostureCheck = &postureCheckCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreatePostureCheck req")

	data, err := r.client.API.PostureChecks.CreatePostureCheck(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Creating Ziti Edge Posture Check from API",
			"Could not create Ziti Edge Posture Check "+plan.ID.ValueString()+": "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(data.Payload.Data.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiPostureMfaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiPostureMfaResourceModel
	var newState ZitiPostureMfaResourceModel

	tflog.Info(ctx, "Reading Ziti Edge Posture Check from API")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := posture_checks.NewDetailPostureCheckParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.PostureChecks.DetailPostureCheck(params, nil)
	if _, ok := err.(*posture_checks.DetailPostureCheckNotFound); ok {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Posture Check from API",
			"Could not read Ziti Posture Check ID "+state.ID.ValueString()+": "+err.Error(),
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	posture_check, _ := data.Payload.Data().(*rest_model.PostureCheckMfaDetail)
	name := posture_check.Name()
	newState.Name = types.StringValue(*name)

	newState.Tags, _ = NativeMapToTerraformMap(ctx, types.StringType, posture_check.Tags().SubTags)
	newState.RoleAttributes, _ = NativeListToTerraformTypedList(ctx, types.StringType, []string(*posture_check.RoleAttributes()))

	newState.IgnoreLegacyEndpoints = types.BoolValue(posture_check.PostureCheckMfaProperties.IgnoreLegacyEndpoints)
	newState.PromptOnUnlock = types.BoolValue(posture_check.PostureCheckMfaProperties.PromptOnUnlock)
	newState.PromptOnWake = types.BoolValue(posture_check.PostureCheckMfaProperties.PromptOnWake)
	newState.TimeoutSeconds = types.Int64Value(posture_check.PostureCheckMfaProperties.TimeoutSeconds)

	newState.ID = state.ID
	state = newState

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiPostureMfaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiPostureMfaResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
	tags := TagsFromAttributes(plan.Tags.Elements())

	postureCheckMfaProperties := rest_model.PostureCheckMfaPropertiesPatch{
		IgnoreLegacyEndpoints: plan.IgnoreLegacyEndpoints.ValueBoolPointer(),
		PromptOnUnlock:        plan.PromptOnUnlock.ValueBoolPointer(),
		PromptOnWake:          plan.PromptOnWake.ValueBoolPointer(),
		TimeoutSeconds:        plan.TimeoutSeconds.ValueInt64Pointer(),
	}

	postureCheckUpdate := rest_model.PostureCheckMfaPatch{
		PostureCheckMfaPropertiesPatch: postureCheckMfaProperties,
	}

	postureCheckUpdate.SetName(name)
	postureCheckUpdate.SetRoleAttributes(&roleAttributes)
	postureCheckUpdate.SetTags(tags)

	jsonObj, _ := json.Marshal(postureCheckUpdate)
	tflog.Info(ctx, string(jsonObj))

	params := posture_checks.NewPatchPostureCheckParams()

	params.ID = plan.ID.ValueString()
	params.PostureCheck = &postureCheckUpdate

	tflog.Debug(ctx, "Assigned all the params. Making UpdatePostureCheck req")

	_, err := r.client.API.PostureChecks.PatchPostureCheck(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Updating Ziti Edge Posture Check from API",
			"Could not create Ziti Edge Posture Check "+plan.ID.ValueString()+": "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiPostureMfaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiPostureMfaResourceModel

	tflog.Debug(ctx, "Deleting Ziti Service Edge Router Policy")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := posture_checks.NewDeletePostureCheckParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.PostureChecks.DeletePostureCheck(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Deleting Ziti Posture check from API",
			"Could not delete Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiPostureMfaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

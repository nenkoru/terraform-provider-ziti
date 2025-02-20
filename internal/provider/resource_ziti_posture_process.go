// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiPostureProcessResource{}
var _ resource.ResourceWithImportState = &ZitiPostureProcessResource{}

func NewZitiPostureProcessResource() resource.Resource {
	return &ZitiPostureProcessResource{}
}

var ProcessModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"path":               types.StringType,
		"os_type":            types.StringType,
		"hashes":             types.ListType{ElemType: types.StringType},
		"signer_fingerprint": types.StringType,
	},
}

// ZitiPostureProcessResource defines the resource implementation.
type ZitiPostureProcessResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiPostureProcessResourceModel describes the resource data model.
type ZitiPostureProcessResourceModel struct {
	ID types.String `tfsdk:"id"`

	Name           types.String `tfsdk:"name"`
	RoleAttributes types.List   `tfsdk:"role_attributes"`
	Tags           types.Map    `tfsdk:"tags"`
	Process        types.Object `tfsdk:"process"`
}

func (r *ZitiPostureProcessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_process"
}

func (r *ZitiPostureProcessResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"process": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"path": schema.StringAttribute{
						Required: true,
					},
					"os_type": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("Windows", "WindowsServer", "Android", "iOS", "Linux", "macOS"),
						},
					},
					"hashes": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "A list of file hashes",
						Optional:            true,
						Computed:            true,
						Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
					},
					"signer_fingerprint": schema.StringAttribute{
						MarkdownDescription: "A list of file sign fingerprints",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString(""),
					},
				},
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
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

func (r *ZitiPostureProcessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiPostureProcessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiPostureProcessResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
	tags := TagsFromAttributes(plan.Tags.Elements())
	var process rest_model.Process
	GenericFromObject[rest_model.Process](convertKeysToCamel(AttributesToNativeTypes(ctx, plan.Process.Attributes())), &process)
	postureCheckCreate := rest_model.PostureCheckProcessCreate{
		Process: &process,
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

func (r *ZitiPostureProcessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiPostureProcessResourceModel
	var newState ZitiPostureProcessResourceModel

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

	posture_check, _ := data.Payload.Data().(*rest_model.PostureCheckProcessDetail)
	name := posture_check.Name()
	newState.Name = types.StringValue(*name)

	newState.Tags, _ = NativeMapToTerraformMap(ctx, types.StringType, posture_check.Tags().SubTags)
	newState.RoleAttributes, _ = NativeListToTerraformTypedList(ctx, types.StringType, []string(*posture_check.RoleAttributes()))

	if posture_check.Process != nil {
		processco, _ := JsonStructToObject(ctx, *posture_check.Process, true, false)
		processco = convertKeysToSnake(processco)

		delete(processco, "hashes")
		delete(processco, "signer_fingerprint")
		delete(processco, "os_type")

		objectMap := NativeBasicTypedAttributesToTerraform(ctx, processco, ProcessModel.AttrTypes)
		objectMap["hashes"], _ = NativeListToTerraformTypedList(ctx, types.StringType, posture_check.Process.Hashes)
		objectMap["signer_fingerprint"] = types.StringValue(posture_check.Process.SignerFingerprint)
		objectMap["os_type"] = types.StringValue(string(*posture_check.Process.OsType))

		object, _ := types.ObjectValue(ProcessModel.AttrTypes, objectMap)
		newState.Process = object
	} else {
		newState.Process = types.ObjectNull(ProcessModel.AttrTypes)

	}
	newState.ID = state.ID
	state = newState

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiPostureProcessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiPostureProcessResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
	tags := TagsFromAttributes(plan.Tags.Elements())
	var process rest_model.Process
	GenericFromObject[rest_model.Process](convertKeysToCamel(AttributesToNativeTypes(ctx, plan.Process.Attributes())), &process)
	postureCheckUpdate := rest_model.PostureCheckProcessPatch{
		Process: &process,
	}
	postureCheckUpdate.SetName(name)
	postureCheckUpdate.SetRoleAttributes(&roleAttributes)
	postureCheckUpdate.SetTags(tags)
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

func (r *ZitiPostureProcessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiPostureProcessResourceModel

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

func (r *ZitiPostureProcessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

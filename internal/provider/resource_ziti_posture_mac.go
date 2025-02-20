// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
    //"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiPostureMacAddressesResource{}
var _ resource.ResourceWithImportState = &ZitiPostureMacAddressesResource{}

func NewZitiPostureMacAddressesResource() resource.Resource {
	return &ZitiPostureMacAddressesResource{}
}

// ZitiPostureMacAddressesResource defines the resource implementation.
type ZitiPostureMacAddressesResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiPostureMacAddressesResourceModel describes the resource data model.
type ZitiPostureMacAddressesResourceModel struct {
	ID                     types.String `tfsdk:"id"`

	Name                   types.String `tfsdk:"name"`
    RoleAttributes  types.List  `tfsdk:"role_attributes"`
    Tags    types.Map    `tfsdk:"tags"`
    MacAddresses types.List `tfsdk:"mac_addresses"`
}


func (r *ZitiPostureMacAddressesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_mac_addresses"
}

func (r *ZitiPostureMacAddressesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
            "mac_addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of mac addresses",
				Required:            true,
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
                Computed:   true,
                Default:    mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
		},
	}
}

func (r *ZitiPostureMacAddressesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiPostureMacAddressesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiPostureMacAddressesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

    var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
    tags := TagsFromAttributes(plan.Tags.Elements())
	postureCheckCreate := rest_model.PostureCheckMacAddressCreate{
        MacAddresses:  ElementsToListOfStrings(plan.MacAddresses.Elements()),
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

func (r *ZitiPostureMacAddressesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiPostureMacAddressesResourceModel
    var newState ZitiPostureMacAddressesResourceModel

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

    posture_check, _ := data.Payload.Data().(*rest_model.PostureCheckMacAddressDetail)
    name := posture_check.Name()
	newState.Name = types.StringValue(*name)

    newState.Tags, _ = NativeMapToTerraformMap(ctx, types.StringType, posture_check.Tags().SubTags)
    newState.RoleAttributes, _ = NativeListToTerraformTypedList(ctx, types.StringType, []string(*posture_check.RoleAttributes()))

    newState.MacAddresses, _ = NativeListToTerraformTypedList(ctx, types.StringType, posture_check.MacAddresses)
    
    newState.ID = state.ID
    state = newState

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiPostureMacAddressesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiPostureMacAddressesResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

    var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
    tags := TagsFromAttributes(plan.Tags.Elements())

    postureCheckUpdate := rest_model.PostureCheckMacAddressPatch{
        MacAddresses:  ElementsToListOfStrings(plan.MacAddresses.Elements()),
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

func (r *ZitiPostureMacAddressesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiPostureMacAddressesResourceModel

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


func (r *ZitiPostureMacAddressesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

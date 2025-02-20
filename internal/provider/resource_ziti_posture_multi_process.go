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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiPostureMultiProcessResource{}
var _ resource.ResourceWithImportState = &ZitiPostureMultiProcessResource{}

func NewZitiPostureMultiProcessResource() resource.Resource {
	return &ZitiPostureMultiProcessResource{}
}

// ZitiPostureMultiProcessResource defines the resource implementation.
type ZitiPostureMultiProcessResource struct {
	client *edge_apis.ManagementApiClient
}

var ProcessMultiModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"path":          types.StringType,
		"os_type":          types.StringType,
        "hashes": types.ListType{ElemType: types.StringType},
        "signer_fingerprints": types.ListType{ElemType: types.StringType},
	},
}
// ZitiPostureMultiProcessResourceModel describes the resource data model.
type ZitiPostureMultiProcessResourceModel struct {
	ID                     types.String `tfsdk:"id"`

	Name                   types.String `tfsdk:"name"`
    RoleAttributes  types.List  `tfsdk:"role_attributes"`
    Tags    types.Map    `tfsdk:"tags"`
    Processes  types.List  `tfsdk:"processes"`
    Semantic  types.String  `tfsdk:"semantic"`
}


func (r *ZitiPostureMultiProcessResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_multi_process"
}

func (r *ZitiPostureMultiProcessResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
            "processes": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
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
                        "signer_fingerprints": schema.ListAttribute{
                            ElementType:         types.StringType,
                            MarkdownDescription: "A list of file sign fingerprints",
                            Optional:            true,
                            Computed:            true,
                            Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
                        },
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
            "semantic": schema.StringAttribute{
				MarkdownDescription: "Semantic for posture checks of the service",
                Optional:   true,
                Computed: true,
                Default:    stringdefault.StaticString("AllOf"),
                Validators: []validator.String{
                    stringvalidator.OneOf("AllOf", "AnyOf"),
                },
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

func (r *ZitiPostureMultiProcessResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiPostureMultiProcessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiPostureMultiProcessResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

    var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
    processes := ElementsToListOfStructsPointers[rest_model.ProcessMulti](ctx, plan.Processes.Elements())
	postureCheckCreate := rest_model.PostureCheckProcessMultiCreate{
        Semantic: &semantic,
        Processes:  processes,
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

func (r *ZitiPostureMultiProcessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiPostureMultiProcessResourceModel
    var newState ZitiPostureMultiProcessResourceModel

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

    posture_check, _ := data.Payload.Data().(*rest_model.PostureCheckProcessMultiDetail)
    name := posture_check.Name()
	newState.Name = types.StringValue(*name)
    newState.Semantic = types.StringValue(string(*posture_check.Semantic))

    newState.Tags, _ = NativeMapToTerraformMap(ctx, types.StringType, posture_check.Tags().SubTags)
    newState.RoleAttributes, _ = NativeListToTerraformTypedList(ctx, types.StringType, []string(*posture_check.RoleAttributes()))

    if posture_check.Processes != nil {
		var objects []attr.Value
		for _, processMulti := range posture_check.Processes {
			processMultico, _ := JsonStructToObject(ctx, processMulti, true, false)
            processMultico = convertKeysToSnake(processMultico)
            
			objectMap := NativeBasicTypedAttributesToTerraform(ctx, processMultico, ProcessMultiModel.AttrTypes)
            objectMap["hashes"], _ = NativeListToTerraformTypedList(ctx, types.StringType, processMulti.Hashes)
            objectMap["signer_fingerprints"], _ = NativeListToTerraformTypedList(ctx, types.StringType, processMulti.SignerFingerprints)
            objectMap["os_type"] = types.StringValue(string(*processMulti.OsType))

			object, _ := types.ObjectValue(ProcessMultiModel.AttrTypes, objectMap)
			objects = append(objects, object)
		}

		processes, _ := types.ListValueFrom(ctx, ProcessMultiModel, objects)
		newState.Processes = processes
	} else {
		newState.Processes = types.ListNull(ProcessMultiModel)
	}
    newState.ID = state.ID
    state = newState

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiPostureMultiProcessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiPostureMultiProcessResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

    var roleAttributes rest_model.Attributes = ElementsToListOfStrings(plan.RoleAttributes.Elements())

	name := plan.Name.ValueString()
    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
    processes := ElementsToListOfStructsPointers[rest_model.ProcessMulti](ctx, plan.Processes.Elements())
	postureCheckUpdate := rest_model.PostureCheckProcessMultiUpdate{
        Semantic: &semantic,
        Processes:  processes,
	}
    postureCheckUpdate.SetName(&name)
    postureCheckUpdate.SetRoleAttributes(&roleAttributes)
    postureCheckUpdate.SetTags(tags)
	params := posture_checks.NewUpdatePostureCheckParams()
    
    params.ID = plan.ID.ValueString()
	params.PostureCheck = &postureCheckUpdate

	tflog.Debug(ctx, "Assigned all the params. Making UpdatePostureCheck req")

	_, err := r.client.API.PostureChecks.UpdatePostureCheck(params, nil)
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

func (r *ZitiPostureMultiProcessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiPostureMultiProcessResourceModel

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


func (r *ZitiPostureMultiProcessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

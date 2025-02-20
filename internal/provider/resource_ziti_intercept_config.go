// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiInterceptConfigResource{}
var _ resource.ResourceWithImportState = &ZitiInterceptConfigResource{}

func NewZitiInterceptConfigResource() resource.Resource {
	return &ZitiInterceptConfigResource{}
}

// ZitiInterceptConfigResource defines the resource implementation.
type ZitiInterceptConfigResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiInterceptConfigResourceModel describes the resource data model.
var PortRangeModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"low":  types.Int32Type,
		"high": types.Int32Type,
	},
}

var DialOptionsModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"connect_timeout_seconds": types.StringType,
		"identity":                types.StringType,
	},
}

type ZitiInterceptConfigResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Addresses    types.List   `tfsdk:"addresses"`
	DialOptions  types.Object `tfsdk:"dial_options"`
	PortRanges   types.List   `tfsdk:"port_ranges"`
	Protocols    types.List   `tfsdk:"protocols"`
	SourceIP     types.String `tfsdk:"source_ip"`
	ConfigTypeId types.String `tfsdk:"config_type_id"`
	ID           types.String `tfsdk:"id"`
}

func (r *ZitiInterceptConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intercept_config_v1"
}

func (r *ZitiInterceptConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A resource to define a host.v1 config of Ziti",

		Attributes: map[string]schema.Attribute{
			"addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed addresses that could be forwarded.",
				Required:            true,
			},
			"dial_options": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"connect_timeout_seconds": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("5s"),
					},
					"identity": schema.StringAttribute{
						Optional: true,
					},
				},
			},
			"protocols": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed protocols that could be forwarded.",
				Required:            true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("tcp", "udp"),
					),
				},
			},
			"port_ranges": schema.ListNestedAttribute{
				Default:  listdefault.StaticValue(types.ListNull(AllowedPortRangeModel)),
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"low": schema.Int32Attribute{
							Required: true,
							Validators: []validator.Int32{
								int32validator.Between(1, 65535),
							},
						},
						"high": schema.Int32Attribute{
							Required: true,
							Validators: []validator.Int32{
								int32validator.Between(1, 65535),
							},
						},
					},
				},
				MarkdownDescription: "An array of allowed ports that could be forwarded.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Id of a config",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_ip": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "SourceIp of a config",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of a config",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_type_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("g7cIWbcGg"),
				MarkdownDescription: "configTypeId",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ZitiInterceptConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type DialOptionsDTO struct {
	ConnectTimeoutSeconds *int32  `json:"connectTimeoutSeconds,omitempty"`
	Identity              *string `json:"identity,omitempty"`
}

type InterceptConfigDTO struct {
	Addresses   *[]string         `json:"addresses,omitempty"`
	DialOptions *DialOptionsDTO   `json:"dialOptions,omitempty"`
	PortRanges  *[]ConfigPortsDTO `json:"portRanges,omitempty"`
	Protocols   *[]string         `json:"protocols,omitempty"`
	SourceIP    *string           `json:"sourceIp,omitempty"`
}

func AttributesToDialOptionsStruct(ctx context.Context, attr map[string]attr.Value) DialOptionsDTO {
	var dialOptions DialOptionsDTO
	attrsNative := AttributesToNativeTypes(ctx, attr)
	attrsNative = convertKeysToCamel(attrsNative)
	GenericFromObject(attrsNative, &dialOptions)
	return dialOptions

}

func (dto *InterceptConfigDTO) ConvertToZitiResourceModel(ctx context.Context) ZitiInterceptConfigResourceModel {

	res := ZitiInterceptConfigResourceModel{
		Addresses: convertStringList(ctx, dto.Addresses, types.StringType),
		Protocols: convertStringList(ctx, dto.Protocols, types.StringType),
		SourceIP:  types.StringPointerValue(dto.SourceIP),
	}

	if dto.PortRanges != nil {
		var objects []attr.Value
		for _, allowedRange := range *dto.PortRanges {
			allowedRangeco, _ := JsonStructToObject(ctx, allowedRange, true, false)

			objectMap := NativeBasicTypedAttributesToTerraform(ctx, allowedRangeco, PortRangeModel.AttrTypes)
			object, _ := basetypes.NewObjectValue(PortRangeModel.AttrTypes, objectMap)
			objects = append(objects, object)
		}
		allowedPortRanges, _ := types.ListValueFrom(ctx, PortRangeModel, objects)
		res.PortRanges = allowedPortRanges
	} else {
		res.PortRanges = types.ListNull(PortRangeModel)
	}

	if dto.DialOptions != nil {
		dialOptionsObject, _ := JsonStructToObject(ctx, *dto.DialOptions, true, false)
		dialOptionsObject = convertKeysToSnake(dialOptionsObject)

		dialOptionsMap := NativeBasicTypedAttributesToTerraform(ctx, dialOptionsObject, DialOptionsModel.AttrTypes)

		dialOptionsTf, err := basetypes.NewObjectValue(DialOptionsModel.AttrTypes, dialOptionsMap)
		if err != nil {
			oneerr := err[0]
			tflog.Debug(ctx, "Error converting dialOptionsMap to an object: "+oneerr.Summary()+" | "+oneerr.Detail())
		}
		res.DialOptions = dialOptionsTf
	} else {
		res.DialOptions = types.ObjectNull(DialOptionsModel.AttrTypes)
	}

	return res
}

func (r *ZitiInterceptConfigResourceModel) ToInterceptConfigDTO(ctx context.Context) InterceptConfigDTO {
	dialOptions := AttributesToDialOptionsStruct(ctx, r.DialOptions.Attributes())

	interceptConfigDto := InterceptConfigDTO{
		Addresses:   ElementsToStringArray(r.Addresses.Elements()),
		DialOptions: &dialOptions,
		Protocols:   ElementsToStringArray(r.Protocols.Elements()),
		SourceIP:    r.SourceIP.ValueStringPointer(),
	}

	if len(r.PortRanges.Elements()) > 0 {
		allowedPortRanges := ElementsToListOfStructs[ConfigPortsDTO](ctx, r.PortRanges.Elements())
		interceptConfigDto.PortRanges = &allowedPortRanges
	}

	return interceptConfigDto
}

func (r *ZitiInterceptConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiInterceptConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	requestObject, err := JsonStructToObject(ctx, plan.ToInterceptConfigDTO(ctx), true, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling Ziti Config from API",
			"Could not create Ziti Config "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	jsonObj, _ := json.Marshal(requestObject)
	tflog.Debug(ctx, string(jsonObj))

	name := plan.Name.ValueString()
	configTypeId := plan.ConfigTypeId.ValueString()
	configCreate := rest_model.ConfigCreate{
		ConfigTypeID: &configTypeId,
		Name:         &name,
		Data:         requestObject,
	}
	params := config.NewCreateConfigParams()
	params.Config = &configCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreateConfig req")

	data, err := r.client.API.Config.CreateConfig(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Creating Ziti Config from API",
			"Could not create Ziti Config "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(data.Payload.Data.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiInterceptConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiInterceptConfigResourceModel

	tflog.Debug(ctx, "Reading Ziti config")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := config.NewDetailConfigParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.Config.DetailConfig(params, nil)
	if _, ok := err.(*config.DetailConfigNotFound); ok {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Got response from detail ziti config")
	responseData, ok := data.Payload.Data.Data.(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Error casting a response from a ziti controller to a dictionary",
			"Could not cast a response from ziti to a dictionary",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var hostConfigDto InterceptConfigDTO
	GenericFromObject(responseData, &hostConfigDto)
	newState := hostConfigDto.ConvertToZitiResourceModel(ctx)

	jsonObj, _ := json.Marshal(hostConfigDto)
	tflog.Debug(ctx, "RESPONSE DETAIL")
	tflog.Debug(ctx, string(jsonObj))

	if resp.Diagnostics.HasError() {
		return
	}
	name := data.Payload.Data.Name
	newState.Name = types.StringValue(*name)

	newState.ID = state.ID
	newState.ConfigTypeId = state.ConfigTypeId
	state = newState

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiInterceptConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiInterceptConfigResourceModel

	tflog.Debug(ctx, "Updating Ziti config")
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return

	}
	requestObject, err := JsonStructToObject(ctx, plan.ToInterceptConfigDTO(ctx), true, true)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling Ziti Config from API",
			"Could not create Ziti Config "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	jsonObj, _ := json.Marshal(requestObject)
	tflog.Debug(ctx, "UPDATE REQUEST OBJECT")
	tflog.Debug(ctx, string(jsonObj))

	name := plan.Name.ValueString()
	configUpdate := rest_model.ConfigUpdate{
		Name: &name,
		Data: requestObject,
	}

	params := config.NewUpdateConfigParams()
	params.ID = plan.ID.ValueString()
	params.Config = &configUpdate

	_, err = r.client.API.Config.UpdateConfig(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Updating Ziti Config from API",
			"Could not update Ziti Config "+plan.ID.ValueString()+": "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiInterceptConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiInterceptConfigResourceModel

	tflog.Debug(ctx, "Deleting Ziti config")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := config.NewDeleteConfigParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.Config.DeleteConfig(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Creating Ziti Config from API",
			"Could not create Ziti Config "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.State.RemoveResource(ctx)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiInterceptConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

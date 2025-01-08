// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	//"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiHostConfigResource{}
var _ resource.ResourceWithImportState = &ZitiHostConfigResource{}

func NewZitiHostConfigResource() resource.Resource {
	return &ZitiHostConfigResource{}
}

// ZitiHostConfigResource defines the resource implementation.
type ZitiHostConfigResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiHostConfigResourceModel describes the resource data model.
var AllowedPortRangeModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"low":  types.Int32Type,
		"high": types.Int32Type,
	},
}

var ListenOptionsModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"bind_using_edge_identity": types.BoolType,
		"connect_timeout":          types.StringType,
		"cost":                     types.Int32Type,
		"max_connections":          types.Int32Type,
		"precedence":               types.StringType,
	},
}

// {"trigger":"fail","duration":"30s","consecutiveEvents":2,"action":"mark unhealthy"}
var CheckActionModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"trigger":            types.StringType,
		"duration":           types.StringType,
		"action":             types.StringType,
		"consecutive_events": types.Int32Type,
	},
}

// {"address":"localhost","interval":"5s","timeout":"10s"}
var PortCheckModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"address":  types.StringType,
		"interval": types.StringType,
		"timeout":  types.StringType,
		"actions":  types.ListType{ElemType: CheckActionModel},
	},
}

// {"url":"https://localhost/health","method":"GET","body":"", "expectStatus": 200, "expectInBody": "test", interval: "5s", "timeout": "10s", "actions": [{}..]}
var HTTPCheckModel = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"url":            types.StringType,
		"method":         types.StringType,
		"body":           types.StringType,
		"expect_status":  types.Int32Type,
		"expect_in_body": types.StringType,
		"interval":       types.StringType,
		"timeout":        types.StringType,
		"actions":        types.ListType{ElemType: CheckActionModel},
	},
}

type ZitiHostConfigResourceModel struct {
	Name                   types.String `tfsdk:"name"`
	Address                types.String `tfsdk:"address"`
	ConfigTypeId           types.String `tfsdk:"config_type_id"`
	Port                   types.Int32  `tfsdk:"port"`
	Protocol               types.String `tfsdk:"protocol"`
	ForwardProtocol        types.Bool   `tfsdk:"forward_protocol"`
	ForwardPort            types.Bool   `tfsdk:"forward_port"`
	ForwardAddress         types.Bool   `tfsdk:"forward_address"`
	AllowedProtocols       types.List   `tfsdk:"allowed_protocols"`
	AllowedAddresses       types.List   `tfsdk:"allowed_addresses"`
	AllowedSourceAddresses types.List   `tfsdk:"allowed_source_addresses"`
	AllowedPortRanges      types.List   `tfsdk:"allowed_port_ranges"`
	ListenOptions          types.Object `tfsdk:"listen_options"`
	PortChecks             types.List   `tfsdk:"port_checks"`
	HTTPChecks             types.List   `tfsdk:"http_checks"`
	ID                     types.String `tfsdk:"id"`
}

func (r *ZitiHostConfigResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("address"),
			path.MatchRoot("forward_address"),
		),
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("protocol"),
			path.MatchRoot("forward_protocol"),
		),
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("port"),
			path.MatchRoot("forward_port"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("address"),
			path.MatchRoot("forward_address"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("protocol"),
			path.MatchRoot("forward_protocol"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("port"),
			path.MatchRoot("forward_port"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("forward_protocol"),
			path.MatchRoot("allowed_protocols"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("forward_port"),
			path.MatchRoot("allowed_port_ranges"),
		),
		resourcevalidator.RequiredTogether(
			path.MatchRoot("forward_protocol"),
			path.MatchRoot("allowed_protocols"),
		),
	}
}

func (r *ZitiHostConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_config_v1"
}

func (r *ZitiHostConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A resource to define a host.v1 config of Ziti",

		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "A target host config address towards which traffic would be relayed.",
				Optional:            true,
			},
			"port": schema.Int32Attribute{
				MarkdownDescription: "A port of a target address towards which traffic would be relayed",
				Optional:            true,
				Validators: []validator.Int32{
					int32validator.Between(1, 65535),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "A protocol which config would be allowed to receive",
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "udp"),
				},
				Optional: true,
			},
			"forward_protocol": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to forward allowedProtocols",
				Optional:            true,
			},
			"forward_port": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to forward allowedPortRanges",
				Optional:            true,
			},
			"forward_address": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to forward allowedAddresses",
				Optional:            true,
			},
			"allowed_addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed addresses that could be forwarded.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
			"allowed_source_addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed source addresses that could be forwarded.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
			"listen_options": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"bind_using_edge_identity": schema.BoolAttribute{
						Optional: true,
					},
					"connect_timeout": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("5s"),
					},
					"cost": schema.Int32Attribute{
						Optional: true,
						Computed: true,
						Default:  int32default.StaticInt32(0),
						Validators: []validator.Int32{
							int32validator.Between(0, 65535),
						},
					},
					"max_connections": schema.Int32Attribute{
						Optional: true,
						Computed: true,
						Default:  int32default.StaticInt32(65535),
						Validators: []validator.Int32{
							int32validator.Between(1, 65535),
						},
					},
					"precedence": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("default"),
						Validators: []validator.String{
							stringvalidator.OneOf("default", "required", "failed"),
						},
					},
				},
			},
			"http_checks": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required: true,
						},
						"method": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("GET", "PUT", "POST", "PATCH"),
							},
						},
						"body": schema.StringAttribute{
							Optional: true,
						},
						"expect_status": schema.Int32Attribute{
							Optional: true,
							Computed: true,
							Default:  int32default.StaticInt32(200),
							Validators: []validator.Int32{
								int32validator.Between(1, 1000),
							},
						},
						"expect_in_body": schema.StringAttribute{
							Optional: true,
						},
						"interval": schema.StringAttribute{
							Required: true,
						},
						"timeout": schema.StringAttribute{
							Required: true,
						},
						"actions": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"trigger": schema.StringAttribute{
										Required: true,
										Validators: []validator.String{
											stringvalidator.OneOf("pass", "fail", "change"),
										},
									},
									"duration": schema.StringAttribute{
										Required: true,
									},
									"action": schema.StringAttribute{
										Required: true,
										Validators: []validator.String{
											stringvalidator.Any(
												stringvalidator.OneOf("mark unhealthy", "mark healthy", "send event"),
												stringvalidator.RegexMatches(
													regexp.MustCompile(`^(increase|decrease) cost (-?\d+)$`),
													"must have a valid syntax(eg 'increase cost 100')",
												),
											),
										},
									},
									"consecutive_events": schema.Int32Attribute{
										Optional: true,
										Computed: true,
										Default:  int32default.StaticInt32(1),
									},
								},
							},
							MarkdownDescription: "An array of actions to take upon health check result.",
							Required:            true,
						},
					},
				},
			},
			"port_checks": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Required: true,
						},
						"interval": schema.StringAttribute{
							Required: true,
						},
						"timeout": schema.StringAttribute{
							Required: true,
						},
						"actions": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"trigger": schema.StringAttribute{
										Required: true,
										Validators: []validator.String{
											stringvalidator.OneOf("pass", "fail", "change"),
										},
									},
									"duration": schema.StringAttribute{
										Required: true,
									},
									"action": schema.StringAttribute{
										Required: true,
										Validators: []validator.String{
											stringvalidator.Any(
												stringvalidator.OneOf("mark unhealthy", "mark healthy", "send event"),
												stringvalidator.RegexMatches(
													regexp.MustCompile(`^(increase|decrease) cost (-?\d+)$`),
													"must have a valid syntax(eg 'increase cost 100')",
												),
											),
										},
									},
									"consecutive_events": schema.Int32Attribute{
										Optional: true,
										Computed: true,
										Default:  int32default.StaticInt32(1),
									},
								},
							},
							MarkdownDescription: "An array of actions to take upon health check result.",
							Required:            true,
						},
					},
				},
			},
			"allowed_protocols": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed protocols that could be forwarded.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("tcp", "udp"),
					),
				},
			},
			"allowed_port_ranges": schema.ListNestedAttribute{
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
				Default:             stringdefault.StaticString("NH5p4FpGR"),
				MarkdownDescription: "configTypeId",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ZitiHostConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type HostConfigAllowedPortsDTO struct {
	Low  int32 `json:"low,omitempty"`
	High int32 `json:"high,omitempty"`
}

type ListenOptionsDTO struct {
	BindUsingEdgeIdentity *bool   `json:"bindUsingEdgeIdentity,omitempty"`
	ConnectTimeout        *string `json:"connectTimeout,omitempty"`
	Cost                  *int32  `json:"cost,omitempty"`
	MaxConnections        *int32  `json:"maxConnections,omitempty"`
	Precedence            *string `json:"precedence,omitempty"`
}

type CheckActionDTO struct {
	Trigger           *string `json:"trigger"`
	Duration          *string `json:"duration"`
	ConsecutiveEvents *int32  `json:"consecutiveEvents,omitempty"`
	Action            *string `json:"action"`
}

type HTTPCheckDTO struct {
	Url          *string           `json:"url"`
	Method       *string           `json:"method"`
	Body         *string           `json:"body,omitempty"`
	ExpectStatus *int32            `json:"expectStatus,omitempty"`
	ExpectInBody *string           `json:"expectInBody,omitempty"`
	Interval     *string           `json:"interval"`
	Timeout      *string           `json:"timeout"`
	Actions      *[]CheckActionDTO `json:"actions"`
}

type PortCheckDTO struct {
	Address  *string           `json:"address"`
	Interval *string           `json:"interval"`
	Timeout  *string           `json:"timeout"`
	Actions  *[]CheckActionDTO `json:"actions"`
}

type HostConfigDTO struct {
	Address                *string                      `json:"address,omitempty"`
	Port                   *int32                       `json:"port,omitempty"`
	Protocol               *string                      `json:"protocol,omitempty"`
	ForwardProtocol        *bool                        `json:"forwardProtocol,omitempty"`
	ForwardPort            *bool                        `json:"forwardPort,omitempty"`
	ForwardAddress         *bool                        `json:"forwardAddress,omitempty"`
	AllowedProtocols       *[]string                    `json:"allowedProtocols,omitempty"`
	AllowedAddresses       *[]string                    `json:"allowedAddresses,omitempty"`
	AllowedSourceAddresses *[]string                    `json:"allowedSourceAddresses,omitempty"`
	AllowedPortRanges      *[]HostConfigAllowedPortsDTO `json:"allowedPortRanges,omitempty"`
	ListenOptions          *ListenOptionsDTO            `json:"listenOptions,omitempty"`
	HTTPChecks             *[]HTTPCheckDTO              `json:"httpChecks,omitempty"`
	PortChecks             *[]PortCheckDTO              `json:"portChecks,omitempty"`
}



func AttributesToListenOptionsStruct(attr map[string]attr.Value) ListenOptionsDTO {
	var listenOptions ListenOptionsDTO
	attrsNative := AttributesToNativeTypes(attr)
	attrsNative = convertKeysToCamel(attrsNative)
	GenericFromObject(attrsNative, &listenOptions)
	return listenOptions

}

func HandleActions(attr map[string]attr.Value) *[]CheckActionDTO {
	if value, exists := attr["actions"]; exists {
		if valueList, ok := value.(types.List); ok {
			actionsArray := []CheckActionDTO{}
			for _, v := range valueList.Elements() {
				if valueObject, ok := v.(types.Object); ok {
					var checkAction CheckActionDTO
					attrsNative := AttributesToNativeTypes(valueObject.Attributes())
					attrsNative = convertKeysToCamel(attrsNative)

                    GenericFromObject(attrsNative, &checkAction)
					actionsArray = append(actionsArray, checkAction)
				}
			}
			if len(actionsArray) > 0 {
				return &actionsArray
			}
		}
	}
	return nil
}

func AttributesToStruct[T any](attr map[string]attr.Value)  T {
	var result T
	attrsNative := AttributesToNativeTypes(attr)
	attrsNative = convertKeysToCamel(attrsNative)
	GenericFromObject(attrsNative, &result)

	// Set the actions field, assuming all such structs have Actions
	// Modify this part if the Actions handling differs per type.
    fieldValue := reflect.ValueOf(&result).Elem().FieldByName("Actions")
    if fieldValue.IsValid() && fieldValue.CanSet() {
        fieldValue.Set(reflect.ValueOf(HandleActions(attr)))
    }

	return result
}


func ElementsToListOfStructs(ctx context.Context, elements []attr.Value) []HostConfigAllowedPortsDTO {
	if len(elements) != 0 {
		elementsArray := []HostConfigAllowedPortsDTO{}
		for _, v := range elements {
			var hostConfigAllowedPorts HostConfigAllowedPortsDTO
			if val, ok := v.(types.Object); ok {
				attrsNative := AttributesToNativeTypes(val.Attributes())
				GenericFromObject(attrsNative, &hostConfigAllowedPorts)
				elementsArray = append(elementsArray, hostConfigAllowedPorts)
			}
		}
		return elementsArray
	}
	return []HostConfigAllowedPortsDTO{}
}




func convertCheckActionToTerraformList(ctx context.Context, actions *[]CheckActionDTO) (types.List, diag.Diagnostics) {
    var actionsTf []attr.Value
    for _, item := range *actions {

        actionObject, _ := JsonStructToObject(ctx, item, true, false)
        actionObject = convertKeysToSnake(actionObject)

        actionMap := NativeBasicTypedAttributesToTerraform(ctx, actionObject, CheckActionModel.AttrTypes)

        actionTf, err := basetypes.NewObjectValue(CheckActionModel.AttrTypes, actionMap)
        if err != nil {
            oneerr := err[0]
            tflog.Debug(ctx, "Error converting actionMap to an object: "+oneerr.Summary()+" | "+oneerr.Detail())

        }
        actionsTf = append(actionsTf, actionTf)
    }
    if len(actionsTf) > 0 {
        actionsList, err := types.ListValueFrom(ctx, CheckActionModel, actionsTf)
        return actionsList, err
    } else {
        return types.ListNull(CheckActionModel), nil

    }
}

func convertChecksToTerraformList(ctx context.Context, checks interface{}, modelAttrs map[string]attr.Type, checkModel attr.Type) types.List {
	var objects []attr.Value

	checkList := reflect.ValueOf(checks)

	for i := 0; i < checkList.Len(); i++ {
		check := checkList.Index(i).Interface()
		checkObject, _ := JsonStructToObject(ctx, check, true, false)
		checkObject = convertKeysToSnake(checkObject)
		delete(checkObject, "actions")
		checkMap := NativeBasicTypedAttributesToTerraform(ctx, checkObject, modelAttrs)

		actionsValue := reflect.ValueOf(check).FieldByName("Actions").Interface()
        if actions, ok := actionsValue.(*[]CheckActionDTO); ok {
            actionsList, err := convertCheckActionToTerraformList(ctx, actions)
            if err != nil {
                tflog.Debug(ctx, "Error converting an array of actions to a list")
            }
            checkMap["actions"] = actionsList

            checkTf, err := basetypes.NewObjectValue(modelAttrs, checkMap)
            if err != nil {
                tflog.Debug(ctx, "Error converting checkMap to ObjectValue")
            }

            objects = append(objects, checkTf)
        }
		
	}

	checksTf, _ := types.ListValueFrom(ctx, checkModel, objects)
	
    return checksTf

}

func (dto *HostConfigDTO) ConvertToZitiResourceModel(ctx context.Context) ZitiHostConfigResourceModel {

    res := ZitiHostConfigResourceModel{
        Address: types.StringPointerValue(dto.Address),
        Port: types.Int32PointerValue(dto.Port),
        Protocol: types.StringPointerValue(dto.Protocol),
        ForwardProtocol: types.BoolPointerValue(dto.ForwardProtocol),
        ForwardPort: types.BoolPointerValue(dto.ForwardPort),
        ForwardAddress: types.BoolPointerValue(dto.ForwardAddress),

    }
    res.AllowedProtocols = convertStringList(ctx, dto.AllowedProtocols, types.StringType)
    res.AllowedAddresses = convertStringList(ctx, dto.AllowedAddresses, types.StringType)
    res.AllowedSourceAddresses = convertStringList(ctx, dto.AllowedSourceAddresses, types.StringType)


	if dto.AllowedPortRanges != nil {
		var objects []attr.Value
		for _, allowedRange := range *dto.AllowedPortRanges {
			allowedRangeco, _ := JsonStructToObject(ctx, allowedRange, true, false)

			objectMap := NativeBasicTypedAttributesToTerraform(ctx, allowedRangeco, AllowedPortRangeModel.AttrTypes)
			object, _ := basetypes.NewObjectValue(AllowedPortRangeModel.AttrTypes, objectMap)
			objects = append(objects, object)
		}
		allowedPortRanges, _ := types.ListValueFrom(ctx, AllowedPortRangeModel, objects)
		res.AllowedPortRanges = allowedPortRanges
	} else {
		res.AllowedPortRanges = types.ListNull(AllowedPortRangeModel)
	}

	if dto.ListenOptions != nil {
		listenOptionsObject, _ := JsonStructToObject(ctx, *dto.ListenOptions, true, false)
		listenOptionsObject = convertKeysToSnake(listenOptionsObject)

		listenOptionsMap := NativeBasicTypedAttributesToTerraform(ctx, listenOptionsObject, ListenOptionsModel.AttrTypes)

		listenOptionsTf, err := basetypes.NewObjectValue(ListenOptionsModel.AttrTypes, listenOptionsMap)
		if err != nil {
			oneerr := err[0]
			tflog.Debug(ctx, "Error converting listenOptionsMap to an object: "+oneerr.Summary()+" | "+oneerr.Detail())
		}
		res.ListenOptions = listenOptionsTf
	} else {
		res.ListenOptions = types.ObjectNull(ListenOptionsModel.AttrTypes)
	}

    if dto.HTTPChecks != nil {
        res.HTTPChecks = convertChecksToTerraformList(ctx, *dto.HTTPChecks, HTTPCheckModel.AttrTypes, HTTPCheckModel)
    } else {
        res.HTTPChecks = types.ListNull(HTTPCheckModel)

    }
    
    if dto.PortChecks != nil {
        res.PortChecks = convertChecksToTerraformList(ctx, *dto.PortChecks, PortCheckModel.AttrTypes, PortCheckModel)
    } else {
        res.PortChecks = types.ListNull(PortCheckModel)
    }
	

	return res
}

func (r *ZitiHostConfigResourceModel) ToHostConfigDTO(ctx context.Context) HostConfigDTO {
	listenOptions := AttributesToListenOptionsStruct(r.ListenOptions.Attributes())
	var portChecks []PortCheckDTO
	for _, v := range r.PortChecks.Elements() {
		if v, ok := v.(types.Object); ok {
            portCheck := AttributesToStruct[PortCheckDTO](v.Attributes())
			portChecks = append(portChecks, portCheck)
		}
	}
	var httpChecks []HTTPCheckDTO
	for _, v := range r.HTTPChecks.Elements() {
		if v, ok := v.(types.Object); ok {
            httpCheck := AttributesToStruct[HTTPCheckDTO](v.Attributes())
			httpChecks = append(httpChecks, httpCheck)
		}
	}

	hostConfigDto := HostConfigDTO{
		Address:       r.Address.ValueStringPointer(),
		Protocol:      r.Protocol.ValueStringPointer(),
		ListenOptions: &listenOptions,
		PortChecks:    &portChecks,
		HTTPChecks:    &httpChecks,
        ForwardAddress: r.ForwardAddress.ValueBoolPointer(),
        ForwardPort: r.ForwardPort.ValueBoolPointer(),
        ForwardProtocol: r.ForwardProtocol.ValueBoolPointer(),
        Port:   r.Port.ValueInt32Pointer(),
        AllowedProtocols: ElementsToStringArray(r.AllowedProtocols.Elements()),
        AllowedAddresses: ElementsToStringArray(r.AllowedAddresses.Elements()),
        AllowedSourceAddresses: ElementsToStringArray(r.AllowedSourceAddresses.Elements()),
	}


	if len(r.AllowedPortRanges.Elements()) > 0 {
		allowedPortRanges := ElementsToListOfStructs(ctx, r.AllowedPortRanges.Elements())
		hostConfigDto.AllowedPortRanges = &allowedPortRanges
	}

	return hostConfigDto
}

// func mapPlanToJsonCreateData(plan ZitiHostConfigResourceModel, jsonObj *gabs.Container) (*gabs.Container, error) {
// }
func jsonSetPIfNotZero[T comparable](value T, path string, jsonObj *gabs.Container) (*gabs.Container, error) {
	if !IsZero(value) {
		return jsonObj.SetP(value, path)
	}
	return nil, nil
}

func (r *ZitiHostConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiHostConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}
	requestObject, err := JsonStructToObject(ctx, plan.ToHostConfigDTO(ctx), true, true)
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

func (r *ZitiHostConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiHostConfigResourceModel

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

	var hostConfigDto HostConfigDTO
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

func (r *ZitiHostConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiHostConfigResourceModel

	tflog.Debug(ctx, "Updating Ziti config")
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return

	}
	requestObject, err := JsonStructToObject(ctx, plan.ToHostConfigDTO(ctx), true, true)
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

func (r *ZitiHostConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiHostConfigResourceModel

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

func (r *ZitiHostConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

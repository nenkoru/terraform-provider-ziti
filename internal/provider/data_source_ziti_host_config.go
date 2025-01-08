// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiHostConfigDataSource{}

func NewZitiHostConfigDataSource() datasource.DataSource {
	return &ZitiHostConfigDataSource{}
}

// ZitiHostConfigDataSource defines the data source implementation.
type ZitiHostConfigDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiHostConfigDataSourceModel describes the data source data model.
type ZitiHostConfigDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Filter                    types.String `tfsdk:"filter"`
    MostRecent  types.Bool  `tfsdk:"most_recent"`

    Name                   types.String `tfsdk:"name"`
	ConfigTypeID           types.String `tfsdk:"config_type_id"`
	Address                types.String `tfsdk:"address"`
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
}

func (r *ZitiHostConfigDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("filter"),
			path.MatchRoot("name"),
		),
		datasourcevalidator.Conflicting(
			path.MatchRoot("id"),
			path.MatchRoot("filter"),
            path.MatchRoot("name"),
		),
	}
}
func (d *ZitiHostConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_config_v1"
}

func (d *ZitiHostConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Ziti Host Config Data Source",

		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "ZitiQl filter query",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Example identifier",
				Computed:            true,
                Optional: true,
			},
            "name": schema.StringAttribute{
				Computed:            true,
                Optional:   true,
				MarkdownDescription: "Name of a config",
			},
            "most_recent": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to get the first result from the filter query",
                Optional: true,
			},

            "address": schema.StringAttribute{
				MarkdownDescription: "A target host config address towards which traffic would be relayed.",
				Computed:            true,
			},
			"port": schema.Int32Attribute{
				MarkdownDescription: "A port of a target address towards which traffic would be relayed",
				Computed:            true,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "A protocol which config would be allowed to receive",
				Computed: true,
			},
			"forward_protocol": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to forward allowedProtocols",
				Computed: true,
			},
			"forward_port": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to forward allowedPortRanges",
				Computed:            true,
			},
			"forward_address": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to forward allowedAddresses",
				Computed:            true,
			},
			"allowed_addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed addresses that could be forwarded.",
				Computed:            true,
			},
			"allowed_source_addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed source addresses that could be forwarded.",
				Computed:            true,
			},
			"listen_options": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"bind_using_edge_identity": schema.BoolAttribute{
						Computed: true,
					},
					"connect_timeout": schema.StringAttribute{
						Computed: true,
					},
					"cost": schema.Int32Attribute{
						Computed: true,
					},
					"max_connections": schema.Int32Attribute{
						Computed: true,
					},
					"precedence": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"http_checks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Computed: true,
						},
						"method": schema.StringAttribute{
							Computed: true,
						},
						"body": schema.StringAttribute{
							Computed: true,
						},
						"expect_status": schema.Int32Attribute{
							Computed: true,
						},
						"expect_in_body": schema.StringAttribute{
							Computed: true,
						},
						"interval": schema.StringAttribute{
							Computed: true,
						},
						"timeout": schema.StringAttribute{
							Computed: true,
						},
						"actions": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"trigger": schema.StringAttribute{
										Computed: true,
									},
									"duration": schema.StringAttribute{
										Computed: true,
									},
									"action": schema.StringAttribute{
										Computed: true,
									},
									"consecutive_events": schema.Int32Attribute{
										Computed: true,
									},
								},
							},
							MarkdownDescription: "An array of actions to take upon health check result.",
							Computed:            true,
						},
					},
				},
			},
			"port_checks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Computed: true,
						},
						"interval": schema.StringAttribute{
							Computed: true,
						},
						"timeout": schema.StringAttribute{
							Computed: true,
						},
						"actions": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"trigger": schema.StringAttribute{
										Computed: true,
									},
									"duration": schema.StringAttribute{
										Computed: true,
									},
									"action": schema.StringAttribute{
										Computed: true,
									},
									"consecutive_events": schema.Int32Attribute{
										Computed: true,
									},
								},
							},
							MarkdownDescription: "An array of actions to take upon health check result.",
							Computed:            true,
						},
					},
				},
			},
			"allowed_protocols": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed protocols that could be forwarded.",
				Computed:            true,
			},
			"allowed_port_ranges": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"low": schema.Int32Attribute{
							Computed: true,
						},
						"high": schema.Int32Attribute{
							Computed: true,
						},
					},
				},
				MarkdownDescription: "An array of allowed ports that could be forwarded.",
			},
			
			"config_type_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "configTypeId",
			},
		},
	}
}

func (r *ZitiHostConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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


func ResourceModelToDataSourceModel(resourceModel ZitiHostConfigResourceModel) ZitiHostConfigDataSourceModel {
    dataSourceModel := ZitiHostConfigDataSourceModel{
        Name: resourceModel.Name,
        Address: resourceModel.Address,
        Port:   resourceModel.Port,
        Protocol:   resourceModel.Protocol,
        ForwardProtocol:    resourceModel.ForwardProtocol,
        ForwardPort:    resourceModel.ForwardPort,
        ForwardAddress: resourceModel.ForwardAddress,
        AllowedProtocols:   resourceModel.AllowedProtocols,
        AllowedAddresses:   resourceModel.AllowedAddresses,
        AllowedSourceAddresses: resourceModel.AllowedSourceAddresses,
        AllowedPortRanges:  resourceModel.AllowedPortRanges,
        ListenOptions:  resourceModel.ListenOptions,
        PortChecks: resourceModel.PortChecks,
        HTTPChecks: resourceModel.HTTPChecks,
    }
    return dataSourceModel

}
func (d *ZitiHostConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiHostConfigDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}


    params := config.NewListConfigsParams()
    var limit int64 = 1000
    var offset int64 = 0
    params.Limit = &limit
    params.Offset = &offset
    filter := ""
    if state.ID.ValueString() != "" {
        filter = "id = \"" + state.ID.ValueString() + "\""
    } else if state.Name.ValueString() != "" {
        filter = "name = \"" + state.Name.ValueString() + "\""
    } else {
        filter = state.Filter.ValueString()
    }

    filter = filter + " and type = \"NH5p4FpGR\"" //host.v1 config
    params.Filter = &filter
    data, err := d.client.API.Config.ListConfigs(params, nil)
    if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	configLists := data.Payload.Data
    if len(configLists) > 1 && !state.MostRecent.ValueBool() {
        resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: " + filter,
		)
    }
    if len(configLists) == 0 {
        resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
            "Try to relax the filter expression: " + filter,
		)
    }
    if resp.Diagnostics.HasError() {
		return
	}
    configList := configLists[0]
	responseData, ok := configList.Data.(map[string]interface{})
    if !ok {
		resp.Diagnostics.AddError(
			"Error casting a response from a ziti controller to a dictionary",
			"Could not cast a response from ziti to a dictionary",
		)
	}
    if resp.Diagnostics.HasError() {
		return
	}

	var hostConfigDto HostConfigDTO
	GenericFromObject(responseData, &hostConfigDto)

	resourceState := hostConfigDto.ConvertToZitiResourceModel(ctx)
    newState := ResourceModelToDataSourceModel(resourceState)

    newState.ID = types.StringValue(*configList.BaseEntity.ID)
    newState.Filter = state.Filter
    newState.MostRecent = state.MostRecent
    newState.ConfigTypeID = types.StringValue(*configList.ConfigTypeID)
	// Save data into Terraform state
	state = newState
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

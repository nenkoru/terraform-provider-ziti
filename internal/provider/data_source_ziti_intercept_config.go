// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiInterceptConfigDataSource{}

func NewZitiInterceptConfigDataSource() datasource.DataSource {
	return &ZitiInterceptConfigDataSource{}
}

// ZitiInterceptConfigDataSource defines the data source implementation.
type ZitiInterceptConfigDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiInterceptConfigDataSourceModel describes the data source data model.
type ZitiInterceptConfigDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Filter     types.String `tfsdk:"filter"`
	MostRecent types.Bool   `tfsdk:"most_recent"`

	Name         types.String `tfsdk:"name"`
	Addresses    types.List   `tfsdk:"addresses"`
	DialOptions  types.Object `tfsdk:"dial_options"`
	PortRanges   types.List   `tfsdk:"port_ranges"`
	Protocols    types.List   `tfsdk:"protocols"`
	SourceIP     types.String `tfsdk:"source_ip"`
	ConfigTypeID types.String `tfsdk:"config_type_id"`
}

func (r *ZitiInterceptConfigDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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
func (d *ZitiInterceptConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intercept_config_v1"
}

func (d *ZitiInterceptConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Ziti Intercept Config Data Source",

		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "ZitiQl filter query",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Example identifier",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Name of a config",
			},
			"most_recent": schema.BoolAttribute{
				MarkdownDescription: "A flag which controls whether to get the first result from the filter query",
				Optional:            true,
			},

			"addresses": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed addresses that could be forwarded.",
				Computed:            true,
			},
			"dial_options": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"connect_timeout_seconds": schema.StringAttribute{
						Computed: true,
					},
					"identity": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"protocols": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed protocols that could be forwarded.",
				Computed:            true,
			},
			"port_ranges": schema.ListNestedAttribute{
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
			"source_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "configTypeId",
			},
		},
	}
}

func (r *ZitiInterceptConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func resourceModelToDataSourceModel(resourceModel ZitiInterceptConfigResourceModel) ZitiInterceptConfigDataSourceModel {
	dataSourceModel := ZitiInterceptConfigDataSourceModel{
		Name:        resourceModel.Name,
		Addresses:   resourceModel.Addresses,
		DialOptions: resourceModel.DialOptions,
		PortRanges:  resourceModel.PortRanges,
		Protocols:   resourceModel.Protocols,
		SourceIP:    resourceModel.SourceIP,
	}
	return dataSourceModel

}
func (d *ZitiInterceptConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiInterceptConfigDataSourceModel

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

	filter = filter + " and type = \"g7cIWbcGg\"" //intercept.v1 config
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
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: "+filter,
		)
	}
	if len(configLists) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
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

	var interceptConfigDto InterceptConfigDTO
	GenericFromObject(responseData, &interceptConfigDto)

	resourceState := interceptConfigDto.ConvertToZitiResourceModel(ctx)
	newState := resourceModelToDataSourceModel(resourceState)

	newState.ID = types.StringValue(*configList.BaseEntity.ID)
	newState.Filter = state.Filter
	newState.MostRecent = state.MostRecent
	newState.ConfigTypeID = types.StringValue(*configList.ConfigTypeID)
	// Save data into Terraform state
	state = newState
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

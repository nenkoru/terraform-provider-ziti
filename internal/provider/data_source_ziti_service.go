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
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiServiceDataSource{}

func NewZitiServiceDataSource() datasource.DataSource {
	return &ZitiServiceDataSource{}
}

// ZitiServiceDataSource defines the resource implementation.
type ZitiServiceDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServiceDataSourceModel describes the resource data model.

type ZitiServiceDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Filter     types.String `tfsdk:"filter"`
	MostRecent types.Bool   `tfsdk:"most_recent"`

	Name                    types.String `tfsdk:"name"`
	Configs                 types.List   `tfsdk:"configs"`
	EncryptionRequired      types.Bool   `tfsdk:"encryption_required"`
	MaxIdleTimeMilliseconds types.Int64  `tfsdk:"max_idle_milliseconds"`
	RoleAttributes          types.List   `tfsdk:"role_attributes"`
	TerminatorStrategy      types.String `tfsdk:"terminator_strategy"`
}

func (d *ZitiServiceDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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
func (d *ZitiServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ZitiServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A datasource to define a service of Ziti",

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

			"terminator_strategy": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Computed:            true,
			},
			"max_idle_milliseconds": schema.Int64Attribute{
				MarkdownDescription: "Time after which idle circuit will be terminated. Defaults to 0, which indicates no limit on idle circuits",
				Computed:            true,
			},
			"encryption_required": schema.BoolAttribute{
				MarkdownDescription: "Controls end-to-end encryption for the service (default true)",
				Computed:            true,
			},
			"configs": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Configuration id or names to be associated with the new service",
				Computed:            true,
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Computed:            true,
			},
		},
	}
}

func (d *ZitiServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *ZitiServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiServiceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service.NewListServicesParams()
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

	params.Filter = &filter
	data, err := d.client.API.Service.ListServices(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	serviceLists := data.Payload.Data
	if len(serviceLists) > 1 && !state.MostRecent.ValueBool() {
		resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: "+filter,
		)
	}
	if len(serviceLists) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	serviceDetail := serviceLists[0]

	name := serviceDetail.Name
	state.Name = types.StringValue(*name)

	configs, _ := types.ListValueFrom(ctx, types.StringType, serviceDetail.Configs)
	state.Configs = configs

	state.EncryptionRequired = types.BoolValue(*serviceDetail.EncryptionRequired)
	state.MaxIdleTimeMilliseconds = types.Int64Value(*serviceDetail.MaxIdleTimeMillis)

	roleAttributes, _ := types.ListValueFrom(ctx, types.StringType, serviceDetail.RoleAttributes)
	state.RoleAttributes = roleAttributes

	state.TerminatorStrategy = types.StringValue(*serviceDetail.TerminatorStrategy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

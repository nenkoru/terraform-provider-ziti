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
	"github.com/openziti/edge-api/rest_management_api_client/edge_router_policy"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiEdgeRouterPolicyDataSource{}

func NewZitiEdgeRouterPolicyDataSource() datasource.DataSource {
	return &ZitiEdgeRouterPolicyDataSource{}
}

// ZitiEdgeRouterPolicyDataSource defines the resource implementation.
type ZitiEdgeRouterPolicyDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiEdgeRouterPolicyDataSourceModel describes the resource data model.

type ZitiEdgeRouterPolicyDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Filter     types.String `tfsdk:"filter"`
	MostRecent types.Bool   `tfsdk:"most_recent"`
	Name       types.String `tfsdk:"name"`

	EdgeRouterRoles types.List   `tfsdk:"edge_router_roles"`
	IdentityRoles   types.List   `tfsdk:"identity_roles"`
	Semantic        types.String `tfsdk:"semantic"`
	Tags            types.Map    `tfsdk:"tags"`
}

func (d *ZitiEdgeRouterPolicyDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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
func (d *ZitiEdgeRouterPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_router_policy"
}

func (d *ZitiEdgeRouterPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A datasource to define a service edge router policy of Ziti",

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

			"edge_router_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Edge router roles list.",
				Computed:            true,
			},
			"identity_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Service roles list.",
				Computed:            true,
			},
			"semantic": schema.StringAttribute{
				MarkdownDescription: "Semantic for posture checks of the service",
				Computed:            true,
			},
			"tags": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags of the service.",
				Computed:            true,
			},
		},
	}
}

func (d *ZitiEdgeRouterPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZitiEdgeRouterPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiEdgeRouterPolicyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := edge_router_policy.NewListEdgeRouterPoliciesParams()
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
	data, err := d.client.API.EdgeRouterPolicy.ListEdgeRouterPolicies(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	edgeRouterPolicies := data.Payload.Data
	if len(edgeRouterPolicies) > 1 && !state.MostRecent.ValueBool() {
		resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: "+filter,
		)
	}
	if len(edgeRouterPolicies) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	edgeRouterPolicy := edgeRouterPolicies[0]

	name := edgeRouterPolicy.Name
	state.Name = types.StringValue(*name)
	state.ID = types.StringValue(*edgeRouterPolicy.ID)

	if len(edgeRouterPolicy.EdgeRouterRoles) > 0 {
		edgeRouterRoles, _ := types.ListValueFrom(ctx, types.StringType, edgeRouterPolicy.EdgeRouterRoles)
		state.EdgeRouterRoles = edgeRouterRoles
	} else {
		state.EdgeRouterRoles = types.ListNull(types.StringType)
	}

	if len(edgeRouterPolicy.IdentityRoles) > 0 {
		serviceRoles, _ := types.ListValueFrom(ctx, types.StringType, edgeRouterPolicy.IdentityRoles)
		state.IdentityRoles = serviceRoles
	} else {
		state.IdentityRoles = types.ListNull(types.StringType)
	}

	if len(edgeRouterPolicy.BaseEntity.Tags.SubTags) != 0 {
		tags, diag := types.MapValueFrom(ctx, types.StringType, edgeRouterPolicy.BaseEntity.Tags.SubTags)
		resp.Diagnostics = append(resp.Diagnostics, diag...)
		state.Tags = tags
	} else {
		state.Tags = types.MapNull(types.StringType)
	}

	state.Semantic = types.StringValue(string(*edgeRouterPolicy.Semantic))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

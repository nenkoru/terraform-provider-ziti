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
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiServicePolicyDataSource{}

func NewZitiServicePolicyDataSource() datasource.DataSource {
	return &ZitiServicePolicyDataSource{}
}

// ZitiServicePolicyDataSource defines the resource implementation.
type ZitiServicePolicyDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServicePolicyDataSourceModel describes the resource data model.

type ZitiServicePolicyDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Filter     types.String `tfsdk:"filter"`
	MostRecent types.Bool   `tfsdk:"most_recent"`
	Name       types.String `tfsdk:"name"`

	IdentityRoles     types.List   `tfsdk:"identity_roles"`
	ServiceRoles      types.List   `tfsdk:"service_roles"`
	PostureCheckRoles types.List   `tfsdk:"posture_check_roles"`
	Type              types.String `tfsdk:"type"`
	Semantic          types.String `tfsdk:"semantic"`
	Tags              types.Map    `tfsdk:"tags"`
}

func (d *ZitiServicePolicyDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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
func (d *ZitiServicePolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_policy"
}

func (d *ZitiServicePolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

			"identity_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Identity roles list.",
				Computed:            true,
			},
			"service_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Service roles list.",
				Computed:            true,
			},
			"posture_check_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Posture check roles list.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of the service policy",
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

func (d *ZitiServicePolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZitiServicePolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiServicePolicyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service_policy.NewListServicePoliciesParams()
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
	data, err := d.client.API.ServicePolicy.ListServicePolicies(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	servicePolicies := data.Payload.Data
	if len(servicePolicies) > 1 && !state.MostRecent.ValueBool() {
		resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: "+filter,
		)
	}
	if len(servicePolicies) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	servicePolicy := servicePolicies[0]

	name := servicePolicy.Name
	state.Name = types.StringValue(*name)
	state.ID = types.StringValue(*servicePolicy.ID)

	if len(servicePolicy.IdentityRoles) > 0 {
		identityRoles, _ := types.ListValueFrom(ctx, types.StringType, servicePolicy.IdentityRoles)
		state.IdentityRoles = identityRoles
	} else {
		state.IdentityRoles = types.ListNull(types.StringType)
	}

	if len(servicePolicy.ServiceRoles) > 0 {
		serviceRoles, _ := types.ListValueFrom(ctx, types.StringType, servicePolicy.ServiceRoles)
		state.ServiceRoles = serviceRoles
	} else {
		state.ServiceRoles = types.ListNull(types.StringType)
	}

	if len(servicePolicy.PostureCheckRoles) > 0 {
		postureCheckRoles, _ := types.ListValueFrom(ctx, types.StringType, servicePolicy.PostureCheckRoles)
		state.PostureCheckRoles = postureCheckRoles
	} else {
		state.PostureCheckRoles = types.ListNull(types.StringType)
	}

	if len(servicePolicy.BaseEntity.Tags.SubTags) != 0 {
		tags, diag := types.MapValueFrom(ctx, types.StringType, servicePolicy.BaseEntity.Tags.SubTags)
		resp.Diagnostics = append(resp.Diagnostics, diag...)
		state.Tags = tags
	} else {
		state.Tags = types.MapNull(types.StringType)
	}

	state.Type = types.StringValue(string(*servicePolicy.Type))
	state.Semantic = types.StringValue(string(*servicePolicy.Semantic))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

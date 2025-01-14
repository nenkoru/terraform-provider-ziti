// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiIdentityDataSource{}

func NewZitiIdentityDataSource() datasource.DataSource {
	return &ZitiIdentityDataSource{}
}

// ZitiIdentityDataSource defines the datasource implementation.
type ZitiIdentityDataSource struct {
	client *edge_apis.ManagementApiClient
}

func (d *ZitiIdentityDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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

// ZitiIdentityDataSourceModel describes the datasource data model.
type ZitiIdentityDataSourceModel struct {
    ID                     types.String `tfsdk:"id"`
	Filter                    types.String `tfsdk:"filter"`
    MostRecent  types.Bool  `tfsdk:"most_recent"`


	Name                   types.String `tfsdk:"name"`
    AppData    types.Map    `tfsdk:"app_data"`
    AuthPolicyID    types.String    `tfsdk:"auth_policy_id"`
    DefaultHostingCost  types.Int64 `tfsdk:"default_hosting_cost"`
    DefaultHostingPrecedence    types.String    `tfsdk:"default_hosting_precedence"`
    ExternalID  types.String    `tfsdk:"external_id"`
    IsAdmin types.Bool  `tfsdk:"is_admin"`
    RoleAttributes  types.List  `tfsdk:"role_attributes"`
    ServiceHostingCosts types.Map `tfsdk:"service_hosting_costs"`
    ServiceHostingPrecedence    types.Map    `tfsdk:"service_hosting_precedence"`
    Tags    types.Map    `tfsdk:"tags"`
    Type    types.String    `tfsdk:"type"`
}


func (d *ZitiIdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (d *ZitiIdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A datasource to define an identity of Ziti",

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

            "auth_policy_id": schema.StringAttribute{
				MarkdownDescription: "Auth policy id",
                Computed:   true,
			},
			"default_hosting_cost": schema.Int64Attribute{
				MarkdownDescription: "Default cost of the service identity is going to host. Defaults to 0, which indicates no additional cost applied",
                Computed:   true,
			},
            "default_hosting_precedence": schema.StringAttribute{
				MarkdownDescription: "Default precedence for the service identity is going to host. Defaults to 'default'.",
                Computed: true,
			},
            "external_id": schema.StringAttribute{
				MarkdownDescription: "External id of the identity. Might be used to have an id of this identity from an external system(eg identity provider)",
                Computed: true,
			},
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Controls whether an identity is going to have admin rights in the Edge Management API(default false)",
                Computed:   true,
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Computed:            true,
			},
            "service_hosting_costs": schema.MapAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "A mapping of service names to their hosting cost for this identity",
                Computed:   true,
			},
            "service_hosting_precedence": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A mapping of service names to their hosting precedence for this identity",
                Computed:   true,
			},
            "tags": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags of the identity",
                Computed:   true,
			},
            "app_data": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "AppData of the identity",
                Computed:   true,
			},
            "type": schema.StringAttribute{
				MarkdownDescription: "Type of the identity.",
                Computed: true,
			},

		},
	}
}

func (d *ZitiIdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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


func (d *ZitiIdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiIdentityDataSourceModel

	tflog.Debug(ctx, "Reading Ziti Identity")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

    params := identity.NewListIdentitiesParams()
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

	data, err := d.client.API.Identity.ListIdentities(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service from API",
			"Could not read Ziti Service ID "+state.ID.ValueString()+": "+err.Error(),
		)
	}

    identities := data.Payload.Data
    if len(identities) > 1 && !state.MostRecent.ValueBool() {
        resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: " + filter,
		)
    }
    if len(identities) == 0 {
        resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
            "Try to relax the filter expression: " + filter,
		)
    }
	if resp.Diagnostics.HasError() {
		return
	}

    identityDetail := identities[0]
	name := identityDetail.Name
	state.Name = types.StringValue(*name)
    state.ID = types.StringValue(*identityDetail.ID)


    if len(identityDetail.AppData.SubTags) != 0 {
        appData, diag := types.MapValueFrom(ctx, types.StringType, identityDetail.AppData.SubTags)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.AppData = appData
    } else {
        state.AppData = types.MapNull(types.StringType)
    }

    state.AuthPolicyID = types.StringValue(*identityDetail.AuthPolicyID)
    state.DefaultHostingCost = types.Int64Value(int64(*identityDetail.DefaultHostingCost))
    state.DefaultHostingPrecedence = types.StringValue(string(identityDetail.DefaultHostingPrecedence))


    if identityDetail.ExternalID != nil {
        state.ExternalID = types.StringValue(*identityDetail.ExternalID)
    } else {
        state.ExternalID = types.StringNull()
    }
    state.IsAdmin = types.BoolValue(*identityDetail.IsAdmin)

    if identityDetail.RoleAttributes != nil {
        roleAttributes, diag := types.ListValueFrom(ctx, types.StringType, identityDetail.RoleAttributes)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.RoleAttributes = roleAttributes
    } else {
        state.RoleAttributes = types.ListNull(types.StringType)
    }

    if len(identityDetail.ServiceHostingCosts) > 0 {
        serviceHostingCosts, diag := types.MapValueFrom(ctx, types.Int64Type, identityDetail.ServiceHostingCosts)
        resp.Diagnostics = append(resp.Diagnostics, diag...)

        state.ServiceHostingCosts = serviceHostingCosts
    } else {
        state.ServiceHostingCosts = types.MapNull(types.Int64Type)
    }

    if len(identityDetail.ServiceHostingPrecedences) > 0 {
        serviceHostingPrecedence, diag := types.MapValueFrom(ctx, types.StringType, identityDetail.ServiceHostingPrecedences)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.ServiceHostingPrecedence = serviceHostingPrecedence
    } else {
        state.ServiceHostingPrecedence = types.MapNull(types.StringType)
    }

    if len(identityDetail.BaseEntity.Tags.SubTags) != 0 {
        tags, diag := types.MapValueFrom(ctx, types.StringType, identityDetail.BaseEntity.Tags.SubTags)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.Tags = tags
    } else {
        state.Tags = types.MapNull(types.StringType)
    }

    state.Type = types.StringValue(identityDetail.Type.Name)

    if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

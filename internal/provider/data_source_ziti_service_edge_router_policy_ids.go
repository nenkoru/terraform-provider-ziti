// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/edge-api/rest_management_api_client/service_edge_router_policy"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiServiceEdgeRouterPolicyIdsDataSource{}

func NewZitiServiceEdgeRouterPolicyIdsDataSource() datasource.DataSource {
	return &ZitiServiceEdgeRouterPolicyIdsDataSource{}
}

// ZitiServiceEdgeRouterPolicyIdsDataSource defines the resource implementation.
type ZitiServiceEdgeRouterPolicyIdsDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServiceEdgeRouterPolicyIdsDataSourceModel describes the resource data model.

type ZitiServiceEdgeRouterPolicyIdsDataSourceModel struct {
    IDS     types.List  `tfsdk:"ids"`
	Filter                    types.String `tfsdk:"filter"`
}

func (d *ZitiServiceEdgeRouterPolicyIdsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_edge_router_policy_ids"
}

func (d *ZitiServiceEdgeRouterPolicyIdsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = CommonIdsDataSourceSchema
}

func (d *ZitiServiceEdgeRouterPolicyIdsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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


func (d *ZitiServiceEdgeRouterPolicyIdsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiServiceEdgeRouterPolicyIdsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}


    params := service_edge_router_policy.NewListServiceEdgeRouterPoliciesParams()
    var limit int64 = 1000
    var offset int64 = 0
    params.Limit = &limit
    params.Offset = &offset

    filter := state.Filter.ValueString()
    params.Filter = &filter
    data, err := d.client.API.ServiceEdgeRouterPolicy.ListServiceEdgeRouterPolicies(params, nil)
    if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service Edge Router Policies from API",
			"Could not read Ziti Service Edge Router Policies IDs "+state.Filter.ValueString()+": "+err.Error(),
		)
		return
	}

	serviceEdgeRouterPolicies := data.Payload.Data
    if len(serviceEdgeRouterPolicies) == 0 {
        resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
            "Try to relax the filter expression: " + filter,
		)
    }
    if resp.Diagnostics.HasError() {
		return
	}

    var ids []string
    for _, serviceEdgeRouterPolicy := range serviceEdgeRouterPolicies {
        ids = append(ids, *serviceEdgeRouterPolicy.ID)
    }

    idsList, _ := types.ListValueFrom(ctx, types.StringType, ids)
    state.IDS = idsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}


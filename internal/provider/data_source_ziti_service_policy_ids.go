// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiServicePolicyIdsDataSource{}

func NewZitiServicePolicyIdsDataSource() datasource.DataSource {
	return &ZitiServicePolicyIdsDataSource{}
}

// ZitiServicePolicyIdsDataSource defines the resource implementation.
type ZitiServicePolicyIdsDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServicePolicyIdsDataSourceModel describes the resource data model.

type ZitiServicePolicyIdsDataSourceModel struct {
	IDS    types.List   `tfsdk:"ids"`
	Filter types.String `tfsdk:"filter"`
}

func (d *ZitiServicePolicyIdsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_policy_ids"
}

func (d *ZitiServicePolicyIdsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = CommonIdsDataSourceSchema
}

func (d *ZitiServicePolicyIdsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZitiServicePolicyIdsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiServicePolicyIdsDataSourceModel

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

	filter := state.Filter.ValueString()
	params.Filter = &filter
	data, err := d.client.API.ServicePolicy.ListServicePolicies(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service Policies from API",
			"Could not read Ziti Service Policies IDs "+state.Filter.ValueString()+": "+err.Error(),
		)
		return
	}

	servicePolicies := data.Payload.Data
	if len(servicePolicies) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var ids []string
	for _, servicePolicy := range servicePolicies {
		ids = append(ids, *servicePolicy.ID)
	}

	idsList, _ := types.ListValueFrom(ctx, types.StringType, ids)
	state.IDS = idsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

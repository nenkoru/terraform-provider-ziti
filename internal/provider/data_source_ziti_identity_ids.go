// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiIdentityIdsDataSource{}

func NewZitiIdentityIdsDataSource() datasource.DataSource {
	return &ZitiIdentityIdsDataSource{}
}

// ZitiIdentityIdsDataSource defines the resource implementation.
type ZitiIdentityIdsDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiIdentityIdsDataSourceModel describes the resource data model.

type ZitiIdentityIdsDataSourceModel struct {
	IDS    types.List   `tfsdk:"ids"`
	Filter types.String `tfsdk:"filter"`
}

func (d *ZitiIdentityIdsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_ids"
}

func (d *ZitiIdentityIdsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A datasource to define a service of Ziti",

		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "ZitiQl filter query",
				Optional:            true,
			},
			"ids": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "An array of allowed addresses that could be forwarded.",
				Computed:            true,
			},
		},
	}
}

func (d *ZitiIdentityIdsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZitiIdentityIdsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiIdentityIdsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := identity.NewListIdentitiesParams()
	var limit int64 = 1000
	var offset int64 = 0
	params.Limit = &limit
	params.Offset = &offset

	filter := state.Filter.ValueString()
	params.Filter = &filter
	data, err := d.client.API.Identity.ListIdentities(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Services IDs "+state.Filter.ValueString()+": "+err.Error(),
		)
		return
	}

	identities := data.Payload.Data
	if len(identities) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var ids []string
	for _, identity := range identities {
		ids = append(ids, *identity.ID)
	}

	idsList, _ := types.ListValueFrom(ctx, types.StringType, ids)
	state.IDS = idsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

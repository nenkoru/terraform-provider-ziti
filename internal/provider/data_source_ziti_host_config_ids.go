// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiHostConfigIdsDataSource{}

func NewZitiHostConfigIdsDataSource() datasource.DataSource {
	return &ZitiHostConfigIdsDataSource{}
}

// ZitiHostConfigIdsDataSource defines the data source implementation.
type ZitiHostConfigIdsDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiHostConfigIdsDataSourceModel describes the data source data model.
type ZitiHostConfigIdsDataSourceModel struct {
	Filter                    types.String `tfsdk:"filter"`

    IDS     types.List  `tfsdk:"ids"`
}

func (d *ZitiHostConfigIdsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_config_v1_ids"
}

func (d *ZitiHostConfigIdsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = CommonIdsDataSourceSchema
}

func (r *ZitiHostConfigIdsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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


func (d *ZitiHostConfigIdsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiHostConfigIdsDataSourceModel

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

    filter := state.Filter.ValueString()
    filter = filter + " and type = \"NH5p4FpGR\"" //host.v1 config
    params.Filter = &filter

    data, err := d.client.API.Config.ListConfigs(params, nil)
    if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.Filter.ValueString()+": "+err.Error(),
		)
		return
	}

	configLists := data.Payload.Data
    if len(configLists) == 0 {
        resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
            "Try to relax the filter expression: " + filter,
		)
    }
    if resp.Diagnostics.HasError() {
		return
	}
    var ids []string
    for _, configList := range configLists {
        ids = append(ids, *configList.ID)
    }

    idsList, _ := types.ListValueFrom(ctx, types.StringType, ids)

    state.IDS = idsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

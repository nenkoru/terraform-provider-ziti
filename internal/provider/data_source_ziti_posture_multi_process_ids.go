// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiPostureMultiProcessIdsDataSource{}

func NewZitiPostureMultiProcessIdsDataSource() datasource.DataSource {
	return &ZitiPostureMultiProcessIdsDataSource{}
}

// ZitiPostureMultiProcessIdsDataSource defines the resource implementation.
type ZitiPostureMultiProcessIdsDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiPostureMultiProcessIdsDataSourceModel describes the resource data model.

type ZitiPostureMultiProcessIdsDataSourceModel struct {
    IDS     types.List  `tfsdk:"ids"`
	Filter                    types.String `tfsdk:"filter"`
}

func (d *ZitiPostureMultiProcessIdsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_multi_process_ids"
}

func (d *ZitiPostureMultiProcessIdsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = CommonIdsDataSourceSchema
}

func (d *ZitiPostureMultiProcessIdsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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


func (d *ZitiPostureMultiProcessIdsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiPostureMultiProcessIdsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}


    params := posture_checks.NewListPostureChecksParams()
    var limit int64 = 1000
    var offset int64 = 0
    params.Limit = &limit
    params.Offset = &offset

    filter := state.Filter.ValueString()
    params.Filter = &filter
    data, err := d.client.API.PostureChecks.ListPostureChecks(params, nil)
    if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Services IDs "+state.Filter.ValueString()+": "+err.Error(),
		)
		return
	}

	postureChecks := data.Payload.Data()
    if len(postureChecks) == 0 {
        resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
            "Try to relax the filter expression: " + filter,
		)
    }
    if resp.Diagnostics.HasError() {
		return
	}

    var ids []string
    for _, postureCheck := range postureChecks {
        if _, ok := postureCheck.(*rest_model.PostureCheckProcessMultiDetail); ok {
            ids = append(ids, *postureCheck.ID())
        }
    }

    idsList, _ := types.ListValueFrom(ctx, types.StringType, ids)
    state.IDS = idsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}


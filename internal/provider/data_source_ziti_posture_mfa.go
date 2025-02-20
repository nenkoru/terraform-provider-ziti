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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiPostureMfaDataSource{}

func NewZitiPostureMfaDataSource() datasource.DataSource {
	return &ZitiPostureMfaDataSource{}
}

// ZitiPostureMfaDataSource defines the resource implementation.
type ZitiPostureMfaDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiPostureMfaDataSourceModel describes the resource data model.

type ZitiPostureMfaDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Filter     types.String `tfsdk:"filter"`
	MostRecent types.Bool   `tfsdk:"most_recent"`
	Name       types.String `tfsdk:"name"`

	RoleAttributes types.List `tfsdk:"role_attributes"`
	Tags           types.Map  `tfsdk:"tags"`

	IgnoreLegacyEndpoints types.Bool  `tfsdk:"ignore_legacy_endpoints"`
	PromptOnUnlock        types.Bool  `tfsdk:"prompt_on_unlock"`
	PromptOnWake          types.Bool  `tfsdk:"prompt_on_wake"`
	TimeoutSeconds        types.Int64 `tfsdk:"timeout_seconds"`
}

func (d *ZitiPostureMfaDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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
func (d *ZitiPostureMfaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_mfa"
}

func (d *ZitiPostureMfaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

			"ignore_legacy_endpoints": schema.BoolAttribute{
				MarkdownDescription: "Controls whether legacy endpoints are ignored for this mfa check",
				Optional:            true,
				Computed:            true,
			},
			"prompt_on_unlock": schema.BoolAttribute{
				MarkdownDescription: "Controls whether user is prompted to pass mfa check after a device unlock. Defaults to true.",
				Optional:            true,
				Computed:            true,
			},
			"prompt_on_wake": schema.BoolAttribute{
				MarkdownDescription: "Controls whether user is prompted to pass mfa check after a device wake. Defaults to true.",
				Optional:            true,
				Computed:            true,
			},
			"timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "Time after which controls when mfa check times out. Defaults to -1, which indicates no limit.",
				Optional:            true,
				Computed:            true,
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
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

func (d *ZitiPostureMfaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ZitiPostureMfaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ZitiPostureMfaDataSourceModel

	tflog.Info(ctx, "Reading Ziti Edge Posture Check from API")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := posture_checks.NewListPostureChecksParams()
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
	data, err := d.client.API.PostureChecks.ListPostureChecks(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var posture_checks []rest_model.PostureCheckMfaDetail
	for _, postureCheck := range data.Payload.Data() {
		if processCheck, ok := postureCheck.(*rest_model.PostureCheckMfaDetail); ok {
			posture_checks = append(posture_checks, *processCheck)
		}
	}
	if len(posture_checks) > 1 && !state.MostRecent.ValueBool() {
		resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: "+filter,
		)
	}
	if len(posture_checks) == 0 {
		resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
			"Try to relax the filter expression: "+filter,
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	posture_check := posture_checks[0]
	name := posture_check.Name()
	state.Name = types.StringValue(*name)

	state.Tags, _ = NativeMapToTerraformMap(ctx, types.StringType, posture_check.Tags().SubTags)
	state.RoleAttributes, _ = NativeListToTerraformTypedList(ctx, types.StringType, []string(*posture_check.RoleAttributes()))

	state.IgnoreLegacyEndpoints = types.BoolValue(posture_check.PostureCheckMfaProperties.IgnoreLegacyEndpoints)
	state.PromptOnUnlock = types.BoolValue(posture_check.PostureCheckMfaProperties.PromptOnUnlock)
	state.PromptOnWake = types.BoolValue(posture_check.PostureCheckMfaProperties.PromptOnWake)
	state.TimeoutSeconds = types.Int64Value(posture_check.PostureCheckMfaProperties.TimeoutSeconds)

	state.ID = types.StringValue(*posture_check.ID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

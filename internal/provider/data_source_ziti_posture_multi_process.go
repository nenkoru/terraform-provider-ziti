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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/openziti/edge-api/rest_management_api_client/posture_checks"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ZitiPostureMultiProcessDataSource{}

func NewZitiPostureMultiProcessDataSource() datasource.DataSource {
	return &ZitiPostureMultiProcessDataSource{}
}

// ZitiPostureMultiProcessDataSource defines the resource implementation.
type ZitiPostureMultiProcessDataSource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiPostureMultiProcessDataSourceModel describes the resource data model.

type ZitiPostureMultiProcessDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Filter                    types.String `tfsdk:"filter"`
    MostRecent  types.Bool  `tfsdk:"most_recent"`
	Name                   types.String `tfsdk:"name"`

    RoleAttributes  types.List  `tfsdk:"role_attributes"`
    Tags    types.Map    `tfsdk:"tags"`
    Processes  types.List  `tfsdk:"processes"`
    Semantic  types.String  `tfsdk:"semantic"`
}

func (d *ZitiPostureMultiProcessDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
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
func (d *ZitiPostureMultiProcessDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_check_multi_process"
}

func (d *ZitiPostureMultiProcessDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

            "processes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Computed: true,
						},
						"os_type": schema.StringAttribute{
							Computed: true,
						},
                        "hashes": schema.ListAttribute{
                            ElementType:         types.StringType,
                            MarkdownDescription: "A list of file hashes",
                            Computed:            true,
                        },
                        "signer_fingerprints": schema.ListAttribute{
                            ElementType:         types.StringType,
                            MarkdownDescription: "A list of file sign fingerprints",
                            Computed:            true,
                        },
					},
				},
			},
            "role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Computed:            true,
			},
            "semantic": schema.StringAttribute{
				MarkdownDescription: "Semantic for posture checks of the service",
                Computed: true,
			},
            "tags": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags of the service.",
                Computed:   true,
			},
		},
	}
}

func (d *ZitiPostureMultiProcessDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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


func (d *ZitiPostureMultiProcessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var state ZitiPostureMultiProcessDataSourceModel

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

    posture_checks := data.Payload.Data()
    if len(posture_checks) > 1 && !state.MostRecent.ValueBool() {
        resp.Diagnostics.AddError(
			"Multiple items returned from API upon filter execution!",
			"Try to narrow down the filter expression, or set most_recent to true to get the first result: " + filter,
		)
    }
    if len(posture_checks) == 0 {
        resp.Diagnostics.AddError(
			"No items returned from API upon filter execution!",
            "Try to relax the filter expression: " + filter,
		)
    }
    if resp.Diagnostics.HasError() {
		return
	}
    posture_check := posture_checks[0].(*rest_model.PostureCheckProcessMultiDetail)
    name := posture_check.Name()
	state.Name = types.StringValue(*name)
    state.Semantic = types.StringValue(string(*posture_check.Semantic))

    state.Tags, _ = NativeMapToTerraformMap(ctx, types.StringType, posture_check.Tags().SubTags)
    state.RoleAttributes, _ = NativeListToTerraformTypedList(ctx, types.StringType, []string(*posture_check.RoleAttributes()))

    if posture_check.Processes != nil {
		var objects []attr.Value
		for _, processMulti := range posture_check.Processes {
			processMultico, _ := JsonStructToObject(ctx, processMulti, true, false)
            processMultico = convertKeysToSnake(processMultico)
            
            delete(processMultico, "hashes")
            delete(processMultico, "signer_fingerprints")
            delete(processMultico, "os_type")
            
			objectMap := NativeBasicTypedAttributesToTerraform(ctx, processMultico, ProcessMultiModel.AttrTypes)
            objectMap["hashes"], _ = NativeListToTerraformTypedList(ctx, types.StringType, processMulti.Hashes)
            objectMap["signer_fingerprints"], _ = NativeListToTerraformTypedList(ctx, types.StringType, processMulti.SignerFingerprints)
            objectMap["os_type"] = types.StringValue(string(*processMulti.OsType))

			object, _ := types.ObjectValue(ProcessMultiModel.AttrTypes, objectMap)
			objects = append(objects, object)
		}

		processes, _ := types.ListValueFrom(ctx, ProcessMultiModel, objects)
		state.Processes = processes
	} else {
		state.Processes = types.ListNull(ProcessMultiModel)
	}
    state.ID = state.ID
    state = state

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}


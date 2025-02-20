// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiServiceResource{}
var _ resource.ResourceWithImportState = &ZitiServiceResource{}

func NewZitiServiceResource() resource.Resource {
	return &ZitiServiceResource{}
}

// ZitiServiceResource defines the resource implementation.
type ZitiServiceResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServiceResourceModel describes the resource data model.

type ZitiServiceResourceModel struct {
	Name                    types.String `tfsdk:"name"`
	Configs                 types.List   `tfsdk:"configs"`
	EncryptionRequired      types.Bool   `tfsdk:"encryption_required"`
	MaxIdleTimeMilliseconds types.Int64  `tfsdk:"max_idle_milliseconds"`
	RoleAttributes          types.List   `tfsdk:"role_attributes"`
	TerminatorStrategy      types.String `tfsdk:"terminator_strategy"`

	ID types.String `tfsdk:"id"`
}

func (r *ZitiServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ZitiServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A resource to define a host.v1 config of Ziti",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Required:            true,
			},
			"terminator_strategy": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("smartrouting"),
			},
			"max_idle_milliseconds": schema.Int64Attribute{
				MarkdownDescription: "Time after which idle circuit will be terminated. Defaults to 0, which indicates no limit on idle circuits",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"encryption_required": schema.BoolAttribute{
				MarkdownDescription: "Controls end-to-end encryption for the service (default true)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"configs": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Configuration id or names to be associated with the new service",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
		},
	}
}

func (r *ZitiServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var configs []string
	for _, value := range plan.Configs.Elements() {
		if config, ok := value.(types.String); ok {
			configs = append(configs, config.ValueString())
		}
	}
	encryptionRequired := plan.EncryptionRequired.ValueBool()
	maxIdleMilliseconds := plan.MaxIdleTimeMilliseconds.ValueInt64()
	name := plan.Name.ValueString()
	var roleAttributes []string
	for _, value := range plan.RoleAttributes.Elements() {
		if roleAttribute, ok := value.(types.String); ok {
			roleAttributes = append(roleAttributes, roleAttribute.ValueString())
		}
	}

	terminatorStrategy := plan.TerminatorStrategy.ValueString()
	serviceCreate := rest_model.ServiceCreate{
		Configs:            configs,
		EncryptionRequired: &encryptionRequired,
		MaxIdleTimeMillis:  maxIdleMilliseconds,
		Name:               &name,
		RoleAttributes:     roleAttributes,
		TerminatorStrategy: terminatorStrategy,
	}
	params := service.NewCreateServiceParams()
	params.Service = &serviceCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreateService req")

	data, err := r.client.API.Service.CreateService(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Creating Ziti Service from API",
			"Could not create Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(data.Payload.Data.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiServiceResourceModel

	tflog.Debug(ctx, "Reading Ziti Service")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service.NewDetailServiceParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.Service.DetailService(params, nil)
	if _, ok := err.(*service.DetailServiceNotFound); ok {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service from API",
			"Could not read Ziti Service ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
	name := data.Payload.Data.Name
	state.Name = types.StringValue(*name)

	configs, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.Configs)
	state.Configs = configs

	state.EncryptionRequired = types.BoolValue(*data.Payload.Data.EncryptionRequired)
	state.MaxIdleTimeMilliseconds = types.Int64Value(*data.Payload.Data.MaxIdleTimeMillis)

	roleAttributes, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.RoleAttributes)
	state.RoleAttributes = roleAttributes

	state.TerminatorStrategy = types.StringValue(*data.Payload.Data.TerminatorStrategy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiServiceResourceModel

	tflog.Debug(ctx, "Updating Ziti Service")
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var configs []string
	for _, value := range plan.Configs.Elements() {
		if config, ok := value.(types.String); ok {
			configs = append(configs, config.ValueString())
		}
	}
	encryptionRequired := plan.EncryptionRequired.ValueBool()
	maxIdleMilliseconds := plan.MaxIdleTimeMilliseconds.ValueInt64()
	name := plan.Name.ValueString()
	var roleAttributes []string
	for _, value := range plan.RoleAttributes.Elements() {
		if roleAttribute, ok := value.(types.String); ok {
			roleAttributes = append(roleAttributes, roleAttribute.ValueString())
		}
	}

	terminatorStrategy := plan.TerminatorStrategy.ValueString()
	serviceUpdate := rest_model.ServiceUpdate{
		Configs:            configs,
		EncryptionRequired: encryptionRequired,
		MaxIdleTimeMillis:  maxIdleMilliseconds,
		Name:               &name,
		RoleAttributes:     roleAttributes,
		TerminatorStrategy: terminatorStrategy,
	}
	params := service.NewUpdateServiceParams()
	params.Service = &serviceUpdate
	params.ID = plan.ID.ValueString()

	tflog.Debug(ctx, "Assigned all the params. Making UpdateService req")

	_, err := r.client.API.Service.UpdateService(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Updating Ziti Service from API",
			"Could not create Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiServiceResourceModel

	tflog.Debug(ctx, "Deleting Ziti Service")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service.NewDeleteServiceParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.Service.DeleteService(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Deleting Ziti Service from API",
			"Could not delete Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.State.RemoveResource(ctx)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

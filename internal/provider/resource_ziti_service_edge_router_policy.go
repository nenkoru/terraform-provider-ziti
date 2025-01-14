// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/service_edge_router_policy"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiServiceEdgeRouterPolicyResource{}
var _ resource.ResourceWithImportState = &ZitiServiceEdgeRouterPolicyResource{}

func NewZitiServiceEdgeRouterPolicyResource() resource.Resource {
	return &ZitiServiceEdgeRouterPolicyResource{}
}

// ZitiServiceEdgeRouterPolicyResource defines the resource implementation.
type ZitiServiceEdgeRouterPolicyResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServiceEdgeRouterPolicyResourceModel describes the resource data model.
type ZitiServiceEdgeRouterPolicyResourceModel struct {
	ID                     types.String `tfsdk:"id"`

	Name                   types.String `tfsdk:"name"`
    EdgeRouterRoles   types.List  `tfsdk:"edge_router_roles"`
    ServiceRoles   types.List  `tfsdk:"service_roles"`
    Semantic  types.String  `tfsdk:"semantic"`
    Tags    types.Map    `tfsdk:"tags"`
}


func (r *ZitiServiceEdgeRouterPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_edge_router_policy"
}

func (r *ZitiServiceEdgeRouterPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
            "edge_router_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Edge Router roles list.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
            "service_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Service roles list.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
            "semantic": schema.StringAttribute{
				MarkdownDescription: "Semantic for posture checks of the service",
                Optional:   true,
                Computed: true,
                Default:    stringdefault.StaticString("AllOf"),
                Validators: []validator.String{
                    stringvalidator.OneOf("AllOf", "AnyOf"),
                },
			},
            "tags": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags of the service.",
				Optional:            true,
                Computed:   true,
                Default:    mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
		},
	}
}

func (r *ZitiServiceEdgeRouterPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiServiceEdgeRouterPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiServiceEdgeRouterPolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}


	name := plan.Name.ValueString()
    var edgeRouterRoles rest_model.Roles
    for _, value := range plan.EdgeRouterRoles.Elements() {
        if edgeRouterRole, ok := value.(types.String); ok {
            edgeRouterRoles = append(edgeRouterRoles, edgeRouterRole.ValueString())
        }
    }

    var serviceRoles rest_model.Roles
    for _, value := range plan.ServiceRoles.Elements() {
        if serviceRole, ok := value.(types.String); ok {
            serviceRoles = append(serviceRoles, serviceRole.ValueString())
        }
    }

    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
	serviceEdgeRouterPolicyCreate := rest_model.ServiceEdgeRouterPolicyCreate{
        EdgeRouterRoles:  edgeRouterRoles,
        Name: &name,
        Semantic: &semantic,
        ServiceRoles:   serviceRoles,
        Tags:   tags,

	}
	params := service_edge_router_policy.NewCreateServiceEdgeRouterPolicyParams()
	params.Policy = &serviceEdgeRouterPolicyCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreateEdgeRouterServicePolicy req")

	data, err := r.client.API.ServiceEdgeRouterPolicy.CreateServiceEdgeRouterPolicy(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Creating Ziti Edge Router Service Policy from API",
			"Could not create Ziti Edge Router Service Policy "+plan.ID.ValueString()+": "+err.Error(),
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

func (r *ZitiServiceEdgeRouterPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiServiceEdgeRouterPolicyResourceModel

	tflog.Debug(ctx, "Reading Ziti Edge Router Service Policy ")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service_edge_router_policy.NewDetailServiceEdgeRouterPolicyParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.ServiceEdgeRouterPolicy.DetailServiceEdgeRouterPolicy(params, nil)
	if _, ok := err.(*service_edge_router_policy.DetailServiceEdgeRouterPolicyNotFound); ok {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service Edge Router Policy from API",
			"Could not read Ziti Service ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Payload.Data.Name
	state.Name = types.StringValue(*name)

    if len(data.Payload.Data.EdgeRouterRoles) > 0 {
        edgeRouterRoles, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.EdgeRouterRoles)
        state.EdgeRouterRoles = edgeRouterRoles
    } else {
        state.EdgeRouterRoles = types.ListNull(types.StringType)
    }

    if len(data.Payload.Data.ServiceRoles) > 0 {
        serviceRoles, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.ServiceRoles)
        state.ServiceRoles = serviceRoles
    } else {
        state.ServiceRoles = types.ListNull(types.StringType)
    }

    if len(data.Payload.Data.BaseEntity.Tags.SubTags) != 0 {
        tags, diag := types.MapValueFrom(ctx, types.StringType, data.Payload.Data.BaseEntity.Tags.SubTags)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.Tags = tags
    } else {
        state.Tags = types.MapNull(types.StringType)
    }

    state.Semantic = types.StringValue(string(*data.Payload.Data.Semantic))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiServiceEdgeRouterPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiServiceEdgeRouterPolicyResourceModel

	tflog.Debug(ctx, "Updating Ziti Service Policy")
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}


	name := plan.Name.ValueString()
    var edgeRouterRoles rest_model.Roles
    for _, value := range plan.EdgeRouterRoles.Elements() {
        if edgeRouter, ok := value.(types.String); ok {
            edgeRouterRoles = append(edgeRouterRoles, edgeRouter.ValueString())
        }
    }

    var serviceRoles rest_model.Roles
    for _, value := range plan.ServiceRoles.Elements() {
        if serviceRole, ok := value.(types.String); ok {
            serviceRoles = append(serviceRoles, serviceRole.ValueString())
        }
    }

    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
	serviceEdgeRouterPolicyUpdate := rest_model.ServiceEdgeRouterPolicyUpdate{
        EdgeRouterRoles:  edgeRouterRoles,
        Name: &name,
        Semantic: &semantic,
        ServiceRoles:   serviceRoles,
        Tags:   tags,

	}
	params := service_edge_router_policy.NewUpdateServiceEdgeRouterPolicyParams()
    params.ID = plan.ID.ValueString()
	params.Policy = &serviceEdgeRouterPolicyUpdate

	tflog.Debug(ctx, "Assigned all the params. Making UpdateServiceEdgeRouterPolicy req")

	_, err := r.client.API.ServiceEdgeRouterPolicy.UpdateServiceEdgeRouterPolicy(params, nil)
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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiServiceEdgeRouterPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiServiceEdgeRouterPolicyResourceModel

	tflog.Debug(ctx, "Deleting Ziti Service Edge Router Policy")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service_edge_router_policy.NewDeleteServiceEdgeRouterPolicyParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.ServiceEdgeRouterPolicy.DeleteServiceEdgeRouterPolicy(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Deleting Ziti Service Policyfrom API",
			"Could not delete Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

    resp.State.RemoveResource(ctx)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}


func (r *ZitiServiceEdgeRouterPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

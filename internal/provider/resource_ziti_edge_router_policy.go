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
	"github.com/openziti/edge-api/rest_management_api_client/edge_router_policy"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiEdgeRouterPolicyResource{}
var _ resource.ResourceWithImportState = &ZitiEdgeRouterPolicyResource{}

func NewZitiEdgeRouterPolicyResource() resource.Resource {
	return &ZitiEdgeRouterPolicyResource{}
}

// ZitiEdgeRouterPolicyResource defines the resource implementation.
type ZitiEdgeRouterPolicyResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiEdgeRouterPolicyResourceModel describes the resource data model.
type ZitiEdgeRouterPolicyResourceModel struct {
	ID                     types.String `tfsdk:"id"`

	Name                   types.String `tfsdk:"name"`
    EdgeRouterRoles   types.List  `tfsdk:"edge_router_roles"`
    IdentityRoles   types.List  `tfsdk:"identity_roles"`
    Semantic  types.String  `tfsdk:"semantic"`
    Tags    types.Map    `tfsdk:"tags"`
}


func (r *ZitiEdgeRouterPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_router_policy"
}

func (r *ZitiEdgeRouterPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
            "identity_roles": schema.ListAttribute{
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

func (r *ZitiEdgeRouterPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiEdgeRouterPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiEdgeRouterPolicyResourceModel

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

    var identityRoles rest_model.Roles
    for _, value := range plan.IdentityRoles.Elements() {
        if identityRole, ok := value.(types.String); ok {
            identityRoles = append(identityRoles, identityRole.ValueString())
        }
    }

    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
	EdgeRouterPolicyCreate := rest_model.EdgeRouterPolicyCreate{
        EdgeRouterRoles:  edgeRouterRoles,
        Name: &name,
        Semantic: &semantic,
        IdentityRoles:   identityRoles,
        Tags:   tags,

	}
	params := edge_router_policy.NewCreateEdgeRouterPolicyParams()
	params.Policy = &EdgeRouterPolicyCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreateEdgeRouterServicePolicy req")

	data, err := r.client.API.EdgeRouterPolicy.CreateEdgeRouterPolicy(params, nil)
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

func (r *ZitiEdgeRouterPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiEdgeRouterPolicyResourceModel

	tflog.Debug(ctx, "Reading Ziti Edge Router Service Policy ")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := edge_router_policy.NewDetailEdgeRouterPolicyParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.EdgeRouterPolicy.DetailEdgeRouterPolicy(params, nil)
	if _, ok := err.(*edge_router_policy.DetailEdgeRouterPolicyNotFound); ok {
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

    if len(data.Payload.Data.IdentityRoles) > 0 {
        identityRoles, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.IdentityRoles)
        state.IdentityRoles = identityRoles
    } else {
        state.IdentityRoles = types.ListNull(types.StringType)
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

func (r *ZitiEdgeRouterPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiEdgeRouterPolicyResourceModel

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

    var identityRoles rest_model.Roles
    for _, value := range plan.IdentityRoles.Elements() {
        if identityRole, ok := value.(types.String); ok {
            identityRoles = append(identityRoles, identityRole.ValueString())
        }
    }

    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
	EdgeRouterPolicyUpdate := rest_model.EdgeRouterPolicyUpdate{
        EdgeRouterRoles:  edgeRouterRoles,
        Name: &name,
        Semantic: &semantic,
        IdentityRoles:   identityRoles,
        Tags:   tags,

	}
	params := edge_router_policy.NewUpdateEdgeRouterPolicyParams()
    params.ID = plan.ID.ValueString()
	params.Policy = &EdgeRouterPolicyUpdate

	tflog.Debug(ctx, "Assigned all the params. Making UpdateServiceEdgeRouterPolicy req")

	_, err := r.client.API.EdgeRouterPolicy.UpdateEdgeRouterPolicy(params, nil)
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

func (r *ZitiEdgeRouterPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiEdgeRouterPolicyResourceModel

	tflog.Debug(ctx, "Deleting Ziti Service Edge Router Policy")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := edge_router_policy.NewDeleteEdgeRouterPolicyParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.EdgeRouterPolicy.DeleteEdgeRouterPolicy(params, nil)
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


func (r *ZitiEdgeRouterPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

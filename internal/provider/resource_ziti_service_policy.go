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
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiServicePolicyResource{}
var _ resource.ResourceWithImportState = &ZitiServicePolicyResource{}

func NewZitiServicePolicyResource() resource.Resource {
	return &ZitiServicePolicyResource{}
}

// ZitiServicePolicyResource defines the resource implementation.
type ZitiServicePolicyResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiServicePolicyResourceModel describes the resource data model.
type ZitiServicePolicyResourceModel struct {
	ID                     types.String `tfsdk:"id"`

	Name                   types.String `tfsdk:"name"`
    IdentityRoles   types.List  `tfsdk:"identity_roles"`
    ServiceRoles   types.List  `tfsdk:"service_roles"`
    PostureCheckRoles   types.List  `tfsdk:"posture_check_roles"`
    Type  types.String  `tfsdk:"type"`
    Semantic  types.String  `tfsdk:"semantic"`
    Tags    types.Map    `tfsdk:"tags"`
}


func (r *ZitiServicePolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_policy"
}

func (r *ZitiServicePolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
            "identity_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Identity roles list.",
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
            "posture_check_roles": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Posture check roles list.",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
            "type": schema.StringAttribute{
				MarkdownDescription: "Type of the service policy",
                Required:   true,
                Validators: []validator.String{
                    stringvalidator.OneOf("Dial", "Bind"),
                },
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

func (r *ZitiServicePolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiServicePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiServicePolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}


	name := plan.Name.ValueString()
    var identityRoles rest_model.Roles
    for _, value := range plan.IdentityRoles.Elements() {
        if identityRole, ok := value.(types.String); ok {
            identityRoles = append(identityRoles, identityRole.ValueString())
        }
    }

    var serviceRoles rest_model.Roles
    for _, value := range plan.ServiceRoles.Elements() {
        if serviceRole, ok := value.(types.String); ok {
            serviceRoles = append(serviceRoles, serviceRole.ValueString())
        }
    }

    var postureCheckRoles rest_model.Roles
    for _, value := range plan.PostureCheckRoles.Elements() {
        if postureCheckRole, ok := value.(types.String); ok {
            postureCheckRoles = append(postureCheckRoles, postureCheckRole.ValueString())
        }
    }

    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    type_ := rest_model.DialBind(plan.Type.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
	servicePolicyCreate := rest_model.ServicePolicyCreate{
        IdentityRoles:  identityRoles,
        Name: &name,
        PostureCheckRoles: postureCheckRoles,
        Semantic: &semantic,
        ServiceRoles:   serviceRoles,
        Tags:   tags,
        Type:   &type_,

	}
	params := service_policy.NewCreateServicePolicyParams()
	params.Policy = &servicePolicyCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreateService req")

	data, err := r.client.API.ServicePolicy.CreateServicePolicy(params, nil)
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

func (r *ZitiServicePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiServicePolicyResourceModel

	tflog.Debug(ctx, "Reading Ziti Service")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service_policy.NewDetailServicePolicyParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.ServicePolicy.DetailServicePolicy(params, nil)
	if _, ok := err.(*service_policy.DetailServicePolicyNotFound); ok {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service Policy from API",
			"Could not read Ziti Service ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Payload.Data.Name
	state.Name = types.StringValue(*name)

    if len(data.Payload.Data.IdentityRoles) > 0 {
        identityRoles, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.IdentityRoles)
        state.IdentityRoles = identityRoles
    } else {
        state.IdentityRoles = types.ListNull(types.StringType)
    }

    if len(data.Payload.Data.ServiceRoles) > 0 {
        serviceRoles, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.ServiceRoles)
        state.ServiceRoles = serviceRoles
    } else {
        state.ServiceRoles = types.ListNull(types.StringType)
    }

    if len(data.Payload.Data.PostureCheckRoles) > 0 {
        postureCheckRoles, _ := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.PostureCheckRoles)
        state.PostureCheckRoles = postureCheckRoles
    } else {
        state.PostureCheckRoles = types.ListNull(types.StringType)
    }

    if len(data.Payload.Data.BaseEntity.Tags.SubTags) != 0 {
        tags, diag := types.MapValueFrom(ctx, types.StringType, data.Payload.Data.BaseEntity.Tags.SubTags)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.Tags = tags
    } else {
        state.Tags = types.MapNull(types.StringType)
    }

    state.Type = types.StringValue(string(*data.Payload.Data.Type))
    state.Semantic = types.StringValue(string(*data.Payload.Data.Semantic))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiServicePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiServicePolicyResourceModel

	tflog.Debug(ctx, "Updating Ziti Service Policy")
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}


	name := plan.Name.ValueString()
    var identityRoles rest_model.Roles
    for _, value := range plan.IdentityRoles.Elements() {
        if identityRole, ok := value.(types.String); ok {
            identityRoles = append(identityRoles, identityRole.ValueString())
        }
    }

    var serviceRoles rest_model.Roles
    for _, value := range plan.ServiceRoles.Elements() {
        if serviceRole, ok := value.(types.String); ok {
            serviceRoles = append(serviceRoles, serviceRole.ValueString())
        }
    }

    var postureCheckRoles rest_model.Roles
    for _, value := range plan.PostureCheckRoles.Elements() {
        if postureCheckRole, ok := value.(types.String); ok {
            postureCheckRoles = append(postureCheckRoles, postureCheckRole.ValueString())
        }
    }

    semantic := rest_model.Semantic(plan.Semantic.ValueString())
    type_ := rest_model.DialBind(plan.Type.ValueString())
    tags := TagsFromAttributes(plan.Tags.Elements())
	servicePolicyUpdate := rest_model.ServicePolicyUpdate{
        IdentityRoles:  identityRoles,
        Name: &name,
        PostureCheckRoles: postureCheckRoles,
        Semantic: &semantic,
        ServiceRoles:   serviceRoles,
        Tags:   tags,
        Type:   &type_,

	}
	params := service_policy.NewUpdateServicePolicyParams()
    params.ID = plan.ID.ValueString()
	params.Policy = &servicePolicyUpdate

	tflog.Debug(ctx, "Assigned all the params. Making UpdateServicePolicy req")

	_, err := r.client.API.ServicePolicy.UpdateServicePolicy(params, nil)
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

func (r *ZitiServicePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiServicePolicyResourceModel

	tflog.Debug(ctx, "Deleting Ziti Service Policy")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	params := service_policy.NewDeleteServicePolicyParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.ServicePolicy.DeleteServicePolicy(params, nil)
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


func (r *ZitiServicePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

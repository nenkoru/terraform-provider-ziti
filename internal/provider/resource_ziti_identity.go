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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/edge-apis"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiIdentityResource{}
var _ resource.ResourceWithImportState = &ZitiIdentityResource{}

func NewZitiIdentityResource() resource.Resource {
	return &ZitiIdentityResource{}
}

// ZitiIdentityResource defines the resource implementation.
type ZitiIdentityResource struct {
	client *edge_apis.ManagementApiClient
}


// ZitiIdentityResourceModel describes the resource data model.
type ZitiIdentityResourceModel struct {
	Name                   types.String `tfsdk:"name"`
    AppData    types.Map    `tfsdk:"app_data"`
    AuthPolicyID    types.String    `tfsdk:"auth_policy_id"`
    DefaultHostingCost  types.Int64 `tfsdk:"default_hosting_cost"`
    DefaultHostingPrecedence    types.String    `tfsdk:"default_hosting_precedence"`
    ExternalID  types.String    `tfsdk:"external_id"`
    IsAdmin types.Bool  `tfsdk:"is_admin"`
    RoleAttributes  types.List  `tfsdk:"role_attributes"`
    ServiceHostingCosts types.Map `tfsdk:"service_hosting_costs"`
    ServiceHostingPrecedence    types.Map    `tfsdk:"service_hosting_precedence"`
    Tags    types.Map    `tfsdk:"tags"`
    Type    types.String    `tfsdk:"type"`

	ID                     types.String `tfsdk:"id"`
}


func (r *ZitiIdentityResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (r *ZitiIdentityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A resource to define an identity of Ziti",

		Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
				MarkdownDescription: "Id of the identity",
				Computed:            true,
                PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the service",
				Required:            true,
			},
            "auth_policy_id": schema.StringAttribute{
				MarkdownDescription: "Auth policy id",
				Optional:            true,
                Default:  stringdefault.StaticString("default"),
                Computed:   true,
			},
			"default_hosting_cost": schema.Int64Attribute{
				MarkdownDescription: "Default cost of the service identity is going to host. Defaults to 0, which indicates no additional cost applied",
				Optional:            true,
                Computed:   true,
                Default:  int64default.StaticInt64(0),
                Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
            "default_hosting_precedence": schema.StringAttribute{
				MarkdownDescription: "Default precedence for the service identity is going to host. Defaults to 'default'.",
                Default:  stringdefault.StaticString("default"),
                Optional: true,
                Computed: true,
                Validators: []validator.String{
                    stringvalidator.OneOf("default", "required", "failed"),
                },
			},
            "external_id": schema.StringAttribute{
				MarkdownDescription: "External id of the identity. Might be used to have an id of this identity from an external system(eg identity provider)",
                Optional: true,
			},
			"is_admin": schema.BoolAttribute{
				MarkdownDescription: "Controls whether an identity is going to have admin rights in the Edge Management API(default false)",
				Optional:            true,
                Computed:   true,
                Default:    booldefault.StaticBool(false),
			},
			"role_attributes": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of role attributes",
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListNull(types.StringType)),
			},
            "service_hosting_costs": schema.MapAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "A mapping of service names to their hosting cost for this identity",
				Optional:            true,
                Computed:   true,
                Default:    mapdefault.StaticValue(types.MapNull(types.Int64Type)),
			},
            "service_hosting_precedence": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A mapping of service names to their hosting precedence for this identity",
				Optional:            true,
                Computed:   true,
                Default:    mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
            "tags": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags of the identity",
				Optional:            true,
                Computed:   true,
                Default:    mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
            "app_data": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "AppData of the identity",
				Optional:            true,
                Computed:   true,
                Default:    mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
            "type": schema.StringAttribute{
				MarkdownDescription: "Type of the identity.",
                Default:  stringdefault.StaticString("Default"),
                Optional: true,
                Computed: true,
                Validators: []validator.String{
                    stringvalidator.OneOf("User", "Device", "Service", "Router", "Default"),
                },
			},

		},
	}
}

func (r *ZitiIdentityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiIdentityResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}


    var roleAttributes rest_model.Attributes
    for _, value := range plan.RoleAttributes.Elements() {
        if roleAttribute, ok := value.(types.String); ok {
            roleAttributes = append(roleAttributes, roleAttribute.ValueString())
        }
    }

    appData := TagsFromAttributes(plan.AppData.Elements())
    tags := TagsFromAttributes(plan.Tags.Elements())

	name := plan.Name.ValueString()
	authPolicyId := plan.AuthPolicyID.ValueString()
    defaultHostingCost := rest_model.TerminatorCost(plan.DefaultHostingCost.ValueInt64())
    defaultHostingPrecedence := rest_model.TerminatorPrecedence(plan.DefaultHostingPrecedence.ValueString())
    externalId := plan.ExternalID.ValueString()
    isAdmin := plan.IsAdmin.ValueBool()
    serviceHostingCosts := make(rest_model.TerminatorCostMap)
    for key, value := range AttributesToNativeTypes(plan.ServiceHostingCosts.Elements()) {
        if val, ok := value.(int64); ok {
            cost := rest_model.TerminatorCost(val)
            serviceHostingCosts[key] = &cost
        }
    }
    serviceHostingPrecedences := make(rest_model.TerminatorPrecedenceMap)
    for key, value := range AttributesToNativeTypes(plan.ServiceHostingPrecedence.Elements()) {
        if val, ok := value.(string); ok {
            serviceHostingPrecedences[key] = rest_model.TerminatorPrecedence(val)
        }
    }
    type_ := rest_model.IdentityType(plan.Type.ValueString())
	identityCreate := rest_model.IdentityCreate{
        AppData:    appData,
        AuthPolicyID:   &authPolicyId,
        DefaultHostingCost: &defaultHostingCost,
        DefaultHostingPrecedence: defaultHostingPrecedence,
        ExternalID: &externalId,
        IsAdmin:    &isAdmin,
		Name:         &name,
        RoleAttributes: &roleAttributes,
        ServiceHostingCosts:    serviceHostingCosts,
        ServiceHostingPrecedences:    serviceHostingPrecedences,
        Tags:   tags,
        Type:   &type_,
	}

	params := identity.NewCreateIdentityParams()
	params.Identity = &identityCreate

	tflog.Debug(ctx, "Assigned all the params. Making CreateIdentity req")

	data, err := r.client.API.Identity.CreateIdentity(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Creating Ziti Identity from API",
			"Could not create Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(data.Payload.Data.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ZitiIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiIdentityResourceModel

	tflog.Debug(ctx, "Reading Ziti Identity")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

    params := identity.NewDetailIdentityParams()
	params.ID = state.ID.ValueString()
	data, err := r.client.API.Identity.DetailIdentity(params, nil)
	if _, ok := err.(*identity.DetailIdentityNotFound); ok {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Reading Ziti Service from API",
			"Could not read Ziti Service ID "+state.ID.ValueString()+": "+err.Error(),
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Payload.Data.Name
	state.Name = types.StringValue(*name)


    if len(data.Payload.Data.AppData.SubTags) != 0 {
        appData, diag := types.MapValueFrom(ctx, types.StringType, data.Payload.Data.AppData.SubTags)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.AppData = appData
    } else {
        state.AppData = types.MapNull(types.StringType)
    }

    state.AuthPolicyID = types.StringValue(*data.Payload.Data.AuthPolicyID)
    state.DefaultHostingCost = types.Int64Value(int64(*data.Payload.Data.DefaultHostingCost))
    state.DefaultHostingPrecedence = types.StringValue(string(data.Payload.Data.DefaultHostingPrecedence))



    if data.Payload.Data.ExternalID != nil {
        state.ExternalID = types.StringValue(*data.Payload.Data.ExternalID)
    } else {
        state.ExternalID = types.StringNull()
    }
    state.IsAdmin = types.BoolValue(*data.Payload.Data.IsAdmin)

    if data.Payload.Data.RoleAttributes != nil {
        roleAttributes, diag := types.ListValueFrom(ctx, types.StringType, data.Payload.Data.RoleAttributes)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.RoleAttributes = roleAttributes
    } else {
        state.RoleAttributes = types.ListNull(types.StringType)
    }

    if len(data.Payload.Data.ServiceHostingCosts) > 0 {
        serviceHostingCosts, diag := types.MapValueFrom(ctx, types.Int64Type, data.Payload.Data.ServiceHostingCosts)
        resp.Diagnostics = append(resp.Diagnostics, diag...)

        state.ServiceHostingCosts = serviceHostingCosts
    } else {
        state.ServiceHostingCosts = types.MapNull(types.Int64Type)
    }

    if len(data.Payload.Data.ServiceHostingPrecedences) > 0 {
        serviceHostingPrecedence, diag := types.MapValueFrom(ctx, types.StringType, data.Payload.Data.ServiceHostingPrecedences)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.ServiceHostingPrecedence = serviceHostingPrecedence
    } else {
        state.ServiceHostingPrecedence = types.MapNull(types.StringType)
    }

    if len(data.Payload.Data.BaseEntity.Tags.SubTags) != 0 {
        tags, diag := types.MapValueFrom(ctx, types.StringType, data.Payload.Data.BaseEntity.Tags.SubTags)
        resp.Diagnostics = append(resp.Diagnostics, diag...)
        state.Tags = tags
    } else {
        state.Tags = types.MapNull(types.StringType)
    }

    state.Type = types.StringValue(data.Payload.Data.Type.Name)

    if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r *ZitiIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ZitiIdentityResourceModel

	tflog.Debug(ctx, "Updating Ziti Identity")

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

    var roleAttributes rest_model.Attributes
    for _, value := range plan.RoleAttributes.Elements() {
        if roleAttribute, ok := value.(types.String); ok {
            roleAttributes = append(roleAttributes, roleAttribute.ValueString())
        }
    }

    appData := TagsFromAttributes(plan.AppData.Elements())
    tags := TagsFromAttributes(plan.Tags.Elements())

	name := plan.Name.ValueString()
	authPolicyId := plan.AuthPolicyID.ValueString()
    defaultHostingCost := rest_model.TerminatorCost(plan.DefaultHostingCost.ValueInt64())
    defaultHostingPrecedence := rest_model.TerminatorPrecedence(plan.DefaultHostingPrecedence.ValueString())
    
    externalId := plan.ExternalID.ValueString()
    isAdmin := plan.IsAdmin.ValueBool()
    serviceHostingCosts := make(rest_model.TerminatorCostMap)
    for key, value := range AttributesToNativeTypes(plan.ServiceHostingCosts.Elements()) {
        if val, ok := value.(int64); ok {
            cost := rest_model.TerminatorCost(val)
            serviceHostingCosts[key] = &cost
        }
    }
    serviceHostingPrecedences := make(rest_model.TerminatorPrecedenceMap)
    for key, value := range AttributesToNativeTypes(plan.ServiceHostingPrecedence.Elements()) {
        if val, ok := value.(string); ok {
            serviceHostingPrecedences[key] = rest_model.TerminatorPrecedence(val)
        }
    }
    type_ := rest_model.IdentityType(plan.Type.ValueString())
	identityUpdate := rest_model.IdentityUpdate{
        AppData:    appData,
        AuthPolicyID:   &authPolicyId,
        DefaultHostingCost: &defaultHostingCost,
        DefaultHostingPrecedence: defaultHostingPrecedence,
        ExternalID: &externalId,
        IsAdmin:    &isAdmin,
		Name:         &name,
        RoleAttributes: &roleAttributes,
        ServiceHostingCosts:    serviceHostingCosts,
        ServiceHostingPrecedences:    serviceHostingPrecedences,
        Tags:   tags,
        Type:   &type_,
	}

	params := identity.NewUpdateIdentityParams()
	params.Identity = &identityUpdate
    params.ID = plan.ID.ValueString()

	tflog.Debug(ctx, "Assigned all the params. Making UpdateIdentity req")

	_, err := r.client.API.Identity.UpdateIdentity(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Updating Ziti Identity from API",
			"Could not create Ziti Service "+plan.ID.ValueString()+": "+err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *ZitiIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ZitiIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

    if resp.Diagnostics.HasError() {
		return
	}
    params := identity.NewDeleteIdentityParams()
	params.ID = plan.ID.ValueString()

	_, err := r.client.API.Identity.DeleteIdentity(params, nil)
	if err != nil {
		err = rest_util.WrapErr(err)
		resp.Diagnostics.AddError(
			"Error Deleting Ziti Identity from API",
			"Could not delete Ziti Identity "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

    resp.State.RemoveResource(ctx)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}



func (r *ZitiIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

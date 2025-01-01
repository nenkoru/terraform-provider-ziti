// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

    "github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/sdk-golang/edge-apis"
    "github.com/openziti/edge-api/rest_management_api_client/config"
    "github.com/openziti/edge-api/rest_model"
    "github.com/openziti/edge-api/rest_util"

)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ZitiHostConfigResource{}
var _ resource.ResourceWithImportState = &ZitiHostConfigResource{}

func NewZitiHostConfigResource() resource.Resource {
	return &ZitiHostConfigResource{}
}

// ZitiHostConfigResource defines the resource implementation.
type ZitiHostConfigResource struct {
	client *edge_apis.ManagementApiClient
}

// ZitiHostConfigResourceModel describes the resource data model.
type ZitiHostConfigResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Address  types.String `tfsdk:"address"`
	Port     types.Int32 `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
	ID       types.String `tfsdk:"id"`
}

func (r *ZitiHostConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_config"
    tflog.Debug(ctx, resp.TypeName)
}

func (r *ZitiHostConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A resource to define a host.v1 config of Ziti",

		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "A target host config address towards which traffic would be relayed.",
				Optional:            true,
			},
			"port": schema.Int32Attribute{
				MarkdownDescription: "A port of a target address towards which traffic would be relayed",
				Optional:            true,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "A protocol which config would be allowed to receive",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Id of a config",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
            "name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of a config",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ZitiHostConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZitiHostConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ZitiHostConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}


    jsonObj := gabs.New()
    jsonObj.SetP(plan.Address.ValueString(), "address")
    jsonObj.SetP(plan.Port.ValueInt32(), "port")
    jsonObj.SetP(plan.Protocol.ValueString(), "protocol")

    tflog.Trace(ctx, jsonObj.String())

    var name string
    var configTypeId string
    var configCreate rest_model.ConfigCreate
    name = plan.Name.ValueString()
    configTypeId = "NH5p4FpGR"

    configCreate.ConfigTypeID = &configTypeId
    configCreate.Name = &name
    configCreate.Data = jsonObj

    params := config.NewCreateConfigParams()
    params.Config = &configCreate

    tflog.Debug(ctx, "Assigned all the params. Making CreateConfig req")

    data, err := r.client.API.Config.CreateConfig(params, nil)
    if err != nil {
        err = rest_util.WrapErr(err)
        resp.Diagnostics.AddError(
			"Error Creating Ziti Config from API",
			"Could not create Ziti Config "+plan.ID.ValueString()+": "+err.Error(),
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

func (r *ZitiHostConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ZitiHostConfigResourceModel

	tflog.Debug(ctx, "Reading Ziti config")
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

    params := config.NewDetailConfigParams()
    params.ID = state.ID.ValueString()
    data, err := r.client.API.Config.DetailConfig(params, nil)
    if err != nil {
        err = rest_util.WrapErr(err)
        resp.Diagnostics.AddError(
			"Error Reading Ziti Config from API",
			"Could not read Ziti Config ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
    }
    if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Got response from detail ziti config")
    responseData, ok := data.Payload.Data.Data.(map[string]interface{})
    if !ok {
        resp.Diagnostics.AddError(
            "Error casting a response from a ziti controller to a dictionary",
            "Could not cast a response from ziti to a dictionary",
        )
        return
    }

    if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, responseData["address"].(string))
	tflog.Debug(ctx, responseData["protocol"].(string))
    address, ok := responseData["address"].(string)
    if !ok {
        resp.Diagnostics.AddError(
            "Error casting address to string",
            "Could not cast an address from ziti to a string",
        )

    }
    portRaw, ok := responseData["port"].(float64)
    if !ok {
        resp.Diagnostics.AddError(
            "Error casting port to int32",
            "Could not cast a port from ziti to a int32",
        )

    }
    port := int32(portRaw)
    protocol, ok := responseData["protocol"].(string)
    if !ok {
        resp.Diagnostics.AddError(
            "Error casting protocol to string",
            "Could not cast a protocol from ziti to a string",
        )

    }

    name := data.Payload.Data.Name
    if resp.Diagnostics.HasError() {
		return
	}
    state.Address = types.StringValue(address)
    state.Port = types.Int32Value(port)
    state.Protocol = types.StringValue(protocol)
    state.Name = types.StringValue(*name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

    
}

func (r *ZitiHostConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZitiHostConfigResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZitiHostConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZitiHostConfigResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *ZitiHostConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"crypto/x509"
	"encoding/base64"
	"github.com/fullsailor/pkcs7"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/sdk-golang/ziti"
)

// Ensure ZitiProvider satisfies various provider interfaces.
var _ provider.Provider = &ZitiProvider{}
var _ provider.ProviderWithFunctions = &ZitiProvider{}

// ZitiProvider defines the provider implementation.
type ZitiProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ZitiProviderModel describes the provider data model.
type ZitiProviderModel struct {
	Endpoint types.String `tfsdk:"mgmt_endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	CaPool   types.String `tfsdk:"capool"`
}

func (p *ZitiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ziti"
	resp.Version = p.version
}

func (p *ZitiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"mgmt_endpoint": schema.StringAttribute{
				MarkdownDescription: "An endpoint pointing to Ziti Edge Management API URL",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "A username of an identity that is able to perform admin actions",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "A password of an identity that is able to perform admin actions",
				Optional:            true,
				Sensitive:           true,
			},
			"capool": schema.StringAttribute{
				MarkdownDescription: "A base64 encoded CA Pool of the Edge Management API.",
				Optional:            true,
			},
		},
	}
}

func emptyTotpCallback(ch chan string) {
	ch <- "" // Send an empty string
	close(ch)
}

func (p *ZitiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ZitiProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown Ziti Edge Management API URL",
			"The provider cannot create the Ziti Edge API client as there is an unknown configuration value for the Ziti Edge Management API URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ZITI_EDGE_MGMT_URL environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Ziti Edge Management API Username",
			"The provider cannot create the Ziti Edge API client as there is an unknown configuration value for the Ziti Edge Management API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ZITI_EDGE_MGMT_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown Ziti Edge Management API Password",
			"The provider cannot create the Ziti Edge API client as there is an unknown configuration value for the Ziti Edge Management API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ZITI_EDGE_MGMT_PASSWORD environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("ZITI_EDGE_MGMT_URL")
	username := os.Getenv("ZITI_EDGE_MGMT_USERNAME")
	password := os.Getenv("ZITI_EDGE_MGMT_PASSWORD")
	capool := os.Getenv("ZITI_EDGE_MGMT_CAPOOL")

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if !config.CaPool.IsNull() {
		capool = config.CaPool.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Ziti Edge Management API URL",
			"The provider cannot create the Ziti Edge Management API client as there is a missing or empty value for the HashiCups API host. "+
				"Set the host value in the configuration or use the ZITI_EDGE_MGMT_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Ziti Edge Management API Username",
			"The provider cannot create the Ziti Edge Management API client as there is a missing or empty value for the HashiCups API username. "+
				"Set the username value in the configuration or use the ZITI_EDGE_MGMT_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Ziti Edge Management API Password",
			"The provider cannot create the Ziti Edge Management API client as there is a missing or empty value for the HashiCups API password. "+
				"Set the password value in the configuration or use the ZITI_EDGE_MGMT_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if capool == "" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("capool"),
			"Missing CA Pool value to verify, will retrieve CA Pool and trust it implicitly!",
			"Make sure to provide CA Pool manually in configuration or use the ZITI_EDGE_MGMT_CAPOOL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)

	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "ziti_mgmt_endpoint", endpoint)
	ctx = tflog.SetField(ctx, "ziti_username", username)
	ctx = tflog.SetField(ctx, "ziti_password", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "ziti_password")

	tflog.Debug(ctx, "Creating Ziti client")

	apiUrl, _ := url.Parse(endpoint)

	// Note that GetControllerWellKnownCaPool() does not verify the authenticity of the controller, it is assumed
	// this is handled in some other way.
	var caPool *x509.CertPool
	if capool == "" {
		// Parse the full URL
		parsedUrl, err := url.Parse(endpoint)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to parse an endpoint url, make sure its a valid url!",
				"The provider cannot parse an endpoint pointing to a edge management api. Make sure its a valid rfc complaint url",
			)
			return
		}
		// Construct the base URL
		baseUrl := fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Host)
		caPool, err = ziti.GetControllerWellKnownCaPool(baseUrl)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to retrieve well-known certs from ziti edge controller",
				"The provider cannot retrieve well-known cert pool from ziti edge controller. Make sure well-known certs are served at .well-known/est/cacerts",
			)
			return

		}
	} else {
		certData, err := base64.StdEncoding.DecodeString(string(capool))
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("capool"),
				"Unable to decode capool value using base64!",
				"The provider cannot decode base64-encoded capool value. Make sure the capool value is an actual base64 encoded capool",
			)
			return
		}
		certs, err := pkcs7.Parse(certData)

		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("capool"),
				"Unable to parse decoded base64 capool into an actual pkcs7 structure.",
				"The provider cannot parse decoded base64 capool into a pkcs7 structure. Make sure that the encoded base64 value is the actual chain",
			)
		}

		caPool = x509.NewCertPool()
		for _, cert := range certs.Certificates {
			caPool.AddCert(cert)
		}

	}

	if resp.Diagnostics.HasError() {
		return
	}

	credentials := edge_apis.NewUpdbCredentials(username, password)
	credentials.CaPool = caPool

	var apiUrls []*url.URL
	apiUrls = append(apiUrls, apiUrl)

	//Note: the CA pool can be provided here or during the Authenticate(<creds>) call. It is allowed here to enable
	//      calls to REST API endpoints that do not require authentication.
	managementClient := edge_apis.NewManagementApiClient(apiUrls, credentials.GetCaPool(), emptyTotpCallback)

	//"configTypes" are string identifiers of configuration that can be requested by clients. Developers may
	//specify their own in order to provide distributed identity and/or service specific configurations.
	//
	//See: https://openziti.io/docs/learn/core-concepts/config-store/overview
	//Example: configTypes = []string{"myCustomAppConfigType"}
	var configTypes []string

	_, err := managementClient.Authenticate(credentials, configTypes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client for Ziti Edge Management API",
			"The provider cannot create a client for a Ziti Edge Management API",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = managementClient
	resp.ResourceData = managementClient

	tflog.Info(ctx, "Configured Ziti Edge Management client", map[string]any{"success": true})
}

func (p *ZitiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZitiHostConfigResource,
		NewZitiInterceptConfigResource,
        NewZitiServiceResource,
        NewZitiIdentityResource,
        NewZitiServicePolicyResource,
        NewZitiServiceEdgeRouterPolicyResource,
	}
}

func (p *ZitiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewZitiHostConfigDataSource,
		NewZitiHostConfigIdsDataSource,

        NewZitiInterceptConfigDataSource,
		NewZitiInterceptConfigIdsDataSource,

        NewZitiServiceDataSource,
        NewZitiServiceIdsDataSource,

        NewZitiIdentityDataSource,
        NewZitiIdentityIdsDataSource,

        NewZitiServicePolicyDataSource,
        NewZitiServicePolicyIdsDataSource,

        NewZitiServiceEdgeRouterPolicyDataSource,
        NewZitiServiceEdgeRouterPolicyIdsDataSource,

	}
}

func (p *ZitiProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ZitiProvider{
			version: version,
		}
	}
}

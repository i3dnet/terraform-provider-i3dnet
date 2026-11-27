package provider

import (
	"context"
	"fmt"
	"os"

	"terraform-provider-i3dnet/internal/cdn_api"
	"terraform-provider-i3dnet/internal/network_api"
	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = (*i3dnetProvider)(nil)

func New() provider.Provider {
	return &i3dnetProvider{}
}

type i3dnetProvider struct{}

func (p *i3dnetProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The `i3dnet` provider is used to interact with the resources supported by i3D.net. " +
			"The provider needs to be configured with the proper credentials before it can be used.\n\nUse the navigation " +
			"to the left to read about the available resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("API Key for i3D.net One API. May also be provided via `%s` environment variable.", envForApiKey),
				Optional:            true, // optional because it can be configured via env variables as well
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "API base URL. By default it's using `https://api.i3d.net` API URL",
				Optional:            true, // optional. if not specified it will use the prod api URL
			},
			"cdn_api_key": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("API key for the i3D.net CDN API, used by the `i3dnet_cdn_*` resources and data sources. "+
					"It is a different key from `api_key`. May also be provided via `%s` environment variable.", envForCDNApiKey),
				Optional:  true,
				Sensitive: true,
			},
			"cdn_base_url": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("CDN API base URL. By default it's using `%s`.", cdn_api.DefaultBaseURL),
				Optional:            true,
			},
		},
	}
}

// i3dnetProviderModel maps provider schema data to a Go type.
type i3dnetProviderModel struct {
	APIKey     types.String `tfsdk:"api_key"`
	BaseURL    types.String `tfsdk:"base_url"`
	CDNAPIKey  types.String `tfsdk:"cdn_api_key"`
	CDNBaseURL types.String `tfsdk:"cdn_base_url"`
}

const (
	envForApiKey    = "FLEXMETAL_API_KEY"
	envForCDNApiKey = "I3D_CDN_API_KEY"
)

func (p *i3dnetProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config i3dnetProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.
	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown API kEY",
			"The provider cannot create the i3Dnet API client as there is an unknown configuration value for the api_key "+
				fmt.Sprintf("Either target apply the source of the value first, set the value statically in the configuration, or use the %s environment variable.", envForApiKey),
		)
	}

	if config.CDNAPIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("cdn_api_key"),
			"Unknown CDN API key",
			"The provider cannot create the CDN API client as there is an unknown configuration value for the cdn_api_key. "+
				fmt.Sprintf("Either target apply the source of the value first, set the value statically in the configuration, or use the %s environment variable.", envForCDNApiKey),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	apiKey := os.Getenv(envForApiKey)
	if !config.APIKey.IsNull() {
		tflog.Info(ctx, "using api key from provider configuration")
		apiKey = config.APIKey.ValueString()
	}

	cdnAPIKey := os.Getenv(envForCDNApiKey)
	if !config.CDNAPIKey.IsNull() {
		cdnAPIKey = config.CDNAPIKey.ValueString()
	}

	// The One API key is required unless the provider is configured for the
	// CDN only. Resources that need a missing key report it themselves.
	if apiKey == "" && cdnAPIKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing i3D.net API Key",
			"The provider cannot create the API client as there is a missing or empty value for the API key. "+
				fmt.Sprintf("Set the api_key value in the configuration or use the %s environment variable. ", envForApiKey)+
				fmt.Sprintf("To use only the CDN resources, set cdn_api_key or the %s environment variable instead. ", envForCDNApiKey)+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	data := &providerClients{}

	if apiKey != "" {
		client, err := one_api.NewClient(apiKey, config.BaseURL.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Could not initialize i3D.net API client",
				fmt.Sprintf("error: %s", err))
			return
		}

		networkClient, err := network_api.NewAPI(apiKey, config.BaseURL.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Could not initialize i3D.net Network API client",
				fmt.Sprintf("error: %s", err))
			return
		}

		data.oneAPI = client
		data.network = networkClient
	}

	if cdnAPIKey != "" {
		cdnClient, err := cdn_api.NewClient(cdnAPIKey, config.CDNBaseURL.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("cdn_base_url"),
				"Could not initialize i3D.net CDN API client",
				fmt.Sprintf("error: %s", err))
			return
		}

		data.cdn = cdnClient
	}

	// Make the API clients available during DataSource and Resource type Configure methods.
	resp.DataSourceData = data
	resp.ResourceData = data
}

// providerClients is what the provider hands to every resource and data source
// through ProviderData.
//
// A client is nil when its API key is not configured; the *FromProviderData
// helpers report that to the resource or data source that needs it.
type providerClients struct {
	oneAPI  *one_api.Client
	network *network_api.ClientWithResponses
	cdn     *cdn_api.Client
}

func (p *i3dnetProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "i3dnet"
}

func (p *i3dnetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSshKeyDataSource,
		NewTagsDataSource,
		NewLocationsDataSource,
		NewFlexvmCloudDataSource,
		NewFlexvmNodeDataSource,
		NewFlexvmNodesDataSource,
		NewFabricsDataSource,
		NewNetworkDataSource,
		NewNetworksDataSource,
		NewNetworkPrefixesDataSource,
		NewNetworkIPAddressesDataSource,
		NewCDNPredefinedAllowlistsDataSource,
	}
}

func (p *i3dnetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
		NewSshKeyResource,
		NewTagResource,
		NewFlexvmVMResource,
		NewFlexvmCloudResource,
		NewFlexvmNodeResource,
		NewCDNResource,
	}
}

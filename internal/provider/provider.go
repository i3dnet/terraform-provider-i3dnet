package provider

import (
	"context"
	"fmt"
	"os"

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
		},
	}
}

// i3dnetProviderModel maps provider schema data to a Go type.
type i3dnetProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

const (
	envForApiKey = "FLEXMETAL_API_KEY"
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

	// If any of the expected configurations are missing, return errors with provider-specific guidance.
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing i3D.net API Key",
			"The provider cannot create the API client as there is a missing or empty value for the API key. "+
				fmt.Sprintf("Set the api_key value in the configuration or use the %s environment variable. ", envForApiKey)+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := one_api.NewClient(apiKey, config.BaseURL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not initialize i3D.net API client",
			fmt.Sprintf("error: %s", err))
	}

	// Make the API client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *i3dnetProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "i3dnet"
}

func (p *i3dnetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSshKeyDataSource,
		NewTagsDataSource,
		NewLocationsDataSource,
		NewVmPlanDataSource,
		NewVmPoolDataSource,
		NewVmImageDataSource,
	}
}

func (p *i3dnetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
		NewSshKeyResource,
		NewTagResource,
		NewVmPoolResource,
		NewVmResource,
	}
}

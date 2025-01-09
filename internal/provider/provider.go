package provider

import (
	"context"
	"fmt"
	"os"

	"terraform-provider-i3d/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = (*i3dProvider)(nil)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &i3dProvider{}
	}
}

type i3dProvider struct{}

func (p *i3dProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional: true, // optional because it can be configured via env variables as well
			},
			"base_url": schema.StringAttribute{
				Optional: true, // optional. if not specified it will use the prod api URL
			},
		},
	}
}

// i3dProviderModel maps provider schema data to a Go type.
type i3dProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

const (
	envForApiKey = "FLEXMETAL_API_KEY"
)

func (p *i3dProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config i3dProviderModel
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
			"The provider cannot create the i3D API client as there is an unknown configuration value for the api_key "+
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
			"Missing i3D API Key",
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
			"Could not initialize i3d API client",
			fmt.Sprintf("error: %s", err))
	}

	// Make the API client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *i3dProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "i3d"
}

func (p *i3dProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewServersDataSource,
		NewSshKeyDataSource,
	}
}

func (p *i3dProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
		NewSshKeyResource,
	}
}

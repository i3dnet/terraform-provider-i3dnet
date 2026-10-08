package provider

import (
	"context"
	"strings"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// configuredProviderData runs the real provider Configure with api_key and
// base_url set and returns what it hands to resources and data sources.
func configuredProviderData(t *testing.T) (resourceData, dataSourceData any) {
	t.Helper()

	ctx := context.Background()
	p := New()

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	raw := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
		"api_key":  tftypes.NewValue(tftypes.String, "key"),
		"base_url": tftypes.NewValue(tftypes.String, "https://api-testing.example"),
	})

	var resp provider.ConfigureResponse
	p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider Configure: %v", resp.Diagnostics)
	}

	data, ok := resp.ResourceData.(*providerClients)
	if !ok {
		t.Fatalf("ResourceData is %T, want *providerClients", resp.ResourceData)
	}
	if data.oneAPI == nil || data.network == nil {
		t.Fatalf("provider data = %+v, want both clients set", data)
	}
	if resp.DataSourceData != resp.ResourceData {
		t.Error("DataSourceData and ResourceData differ, want the same clients for both")
	}

	return resp.ResourceData, resp.DataSourceData
}

type configureCheck struct {
	name      string
	configure func(providerData any) diag.Diagnostics
}

// everyConfigurable lists the Configure method of every registered resource
// and data source, so a new registration is covered without editing the test.
func everyConfigurable(t *testing.T) []configureCheck {
	t.Helper()

	ctx := context.Background()
	p := New()

	var checks []configureCheck

	for _, newResource := range p.Resources(ctx) {
		r := newResource()

		var meta resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "i3dnet"}, &meta)

		rc, ok := r.(resource.ResourceWithConfigure)
		if !ok {
			t.Errorf("resource %s has no Configure method", meta.TypeName)
			continue
		}
		checks = append(checks, configureCheck{
			name: "resource " + meta.TypeName,
			configure: func(providerData any) diag.Diagnostics {
				var resp resource.ConfigureResponse
				rc.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &resp)
				return resp.Diagnostics
			},
		})
	}

	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()

		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "i3dnet"}, &meta)

		dc, ok := d.(datasource.DataSourceWithConfigure)
		if !ok {
			t.Errorf("data source %s has no Configure method", meta.TypeName)
			continue
		}
		checks = append(checks, configureCheck{
			name: "data source " + meta.TypeName,
			configure: func(providerData any) diag.Diagnostics {
				var resp datasource.ConfigureResponse
				dc.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &resp)
				return resp.Diagnostics
			},
		})
	}

	if len(checks) == 0 {
		t.Fatal("no resources or data sources registered")
	}

	return checks
}

// Every resource and data source accepts what the provider's Configure hands
// out. A leftover type assertion on the old *one_api.Client fails here.
func TestEveryResourceAndDataSourceAcceptsTheProviderData(t *testing.T) {
	resourceData, dataSourceData := configuredProviderData(t)

	for _, c := range everyConfigurable(t) {
		data := resourceData
		if strings.HasPrefix(c.name, "data source ") {
			data = dataSourceData
		}
		if diags := c.configure(data); diags.HasError() {
			t.Errorf("%s: Configure with the provider data: %v", c.name, diags)
		}
	}
}

// Terraform calls Configure before the provider is configured; that must not
// be an error.
func TestEveryResourceAndDataSourceAcceptsNilProviderData(t *testing.T) {
	for _, c := range everyConfigurable(t) {
		if diags := c.configure(nil); diags.HasError() {
			t.Errorf("%s: Configure with nil provider data: %v", c.name, diags)
		}
	}
}

func TestEveryResourceAndDataSourceRejectsTheWrongProviderDataType(t *testing.T) {
	// The type the provider handed out before it carried two clients.
	oldData, err := one_api.NewClient("key", "")
	if err != nil {
		t.Fatalf("one_api.NewClient: %v", err)
	}

	for _, c := range everyConfigurable(t) {
		diags := c.configure(oldData)
		if !diags.HasError() {
			t.Errorf("%s: Configure with %T: no error, want an Unexpected Configure Type diagnostic", c.name, oldData)
			continue
		}
		if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, "providerClients") {
			t.Errorf("%s: diagnostic detail = %q, want it to name the expected type", c.name, detail)
		}
	}
}

func TestClientExtractorsReturnTheirClient(t *testing.T) {
	resourceData, _ := configuredProviderData(t)
	want := resourceData.(*providerClients)

	var diags diag.Diagnostics
	if got := clientFromProviderData(resourceData, &diags); got != want.oneAPI {
		t.Errorf("clientFromProviderData = %p, want the One API client %p", got, want.oneAPI)
	}
	if got := networkClientFromProviderData(resourceData, &diags); got != want.network {
		t.Errorf("networkClientFromProviderData = %p, want the Network API client %p", got, want.network)
	}
	if diags.HasError() {
		t.Errorf("diagnostics: %v", diags)
	}

	if got := networkClientFromProviderData("not provider data", &diags); got != nil {
		t.Errorf("networkClientFromProviderData(wrong type) = %v, want nil", got)
	}
	if !diags.HasError() {
		t.Error("networkClientFromProviderData(wrong type) added no diagnostic")
	}
}

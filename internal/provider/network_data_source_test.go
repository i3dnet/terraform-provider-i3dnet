package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	dsNetworkID         = "0b9c2f7e-4d51-4a8e-9f3a-1c2d3e4f5a6b"
	testRoutingDomainID = "7f1e2d3c-4b5a-4968-8776-655443322110"
)

// configWith builds a config value for schema s: every attribute null except
// the string attributes in set.
func configWith(t *testing.T, ctx context.Context, s datasource.SchemaResponse, set map[string]string) tftypes.Value {
	t.Helper()

	objType, ok := s.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is %T, want tftypes.Object", s.Schema.Type().TerraformType(ctx))
	}

	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, ty := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(ty, nil)
	}
	for name, v := range set {
		if _, ok := objType.AttributeTypes[name]; !ok {
			t.Fatalf("schema has no attribute %q", name)
		}
		values[name] = tftypes.NewValue(tftypes.String, v)
	}

	return tftypes.NewValue(objType, values)
}

// readDataSource runs d.Read with the string arguments in set against handler.
func readDataSource(t *testing.T, newDS func(*network_api.ClientWithResponses) datasource.DataSource, set map[string]string, handler http.HandlerFunc) *datasource.ReadResponse {
	t.Helper()

	ctx := context.Background()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	api, err := network_api.NewAPI("key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}
	d := newDS(api)

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema
	null := tftypes.NewValue(s.Type().TerraformType(ctx), nil)

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: null}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s, Raw: configWith(t, ctx, schemaResp, set)}}, resp)

	return resp
}

func readNetwork(t *testing.T, id string, handler http.HandlerFunc) *datasource.ReadResponse {
	t.Helper()

	return readDataSource(t, func(api *network_api.ClientWithResponses) datasource.DataSource {
		return &networkDataSource{client: api}
	}, map[string]string{"id": id}, handler)
}

// routedNetworkJSON is a private routed network i3D.net set up for the account.
func routedNetworkJSON(id, fabric, name string) string {
	return fmt.Sprintf(`{"id":%q,"fabric":%q,"name":%q,`+
		`"connectivity":{"type":"private","routing":{"enabled":true,"routingDomainId":%q}},`+
		`"addressing":{"source":"provider"},"state":"active","vlanId":812,"retention":"ephemeral",`+
		`"createdAt":"2026-09-01T10:00:00Z","updatedAt":"2026-09-02T11:30:15Z",`+
		`"links":{"self":{"href":"/networks/%s"}}}`, id, fabric, name, testRoutingDomainID, id)
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func TestNetworkDataSourceReadsTheNetwork(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resp := readNetwork(t, dsNetworkID, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if want := "/network/v1/networks/" + dsNetworkID; r.URL.Path != want {
			t.Errorf("request path = %q, want %q", r.URL.Path, want)
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "key" {
			t.Errorf("PRIVATE-TOKEN = %q, want key", got)
		}
		writeJSON(w, routedNetworkJSON(dsNetworkID, "rotterdam-a", "db-backend"))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d requests, want 1", n)
	}

	var got networkModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}

	wantString(t, "id", got.ID, dsNetworkID)
	wantString(t, "fabric", got.Fabric, "rotterdam-a")
	wantString(t, "connectivity", got.Connectivity, "private")
	wantString(t, "name", got.Name, "db-backend")
	wantString(t, "addressing_source", got.AddressingSource, "provider")
	wantString(t, "state", got.State, "active")
	wantString(t, "retention", got.Retention, "ephemeral")
	wantString(t, "routing_domain_id", got.RoutingDomainID, testRoutingDomainID)
	wantString(t, "created_at", got.CreatedAt, "2026-09-01T10:00:00Z")
	wantString(t, "updated_at", got.UpdatedAt, "2026-09-02T11:30:15Z")
	if !got.VlanID.Equal(types.Int64Value(812)) {
		t.Errorf("vlan_id = %v, want 812", got.VlanID)
	}
	if !got.RoutingEnabled.Equal(types.BoolValue(true)) {
		t.Errorf("routing_enabled = %v, want true", got.RoutingEnabled)
	}
}

// Terraform rejects a read that changes a configured value, so an upper-case
// id stays as configured even though the API answers in lower case.
func TestNetworkDataSourceKeepsTheConfiguredIDCase(t *testing.T) {
	t.Parallel()

	upper := strings.ToUpper(dsNetworkID)
	resp := readNetwork(t, upper, func(w http.ResponseWriter, r *http.Request) {
		if want := "/network/v1/networks/" + dsNetworkID; r.URL.Path != want {
			t.Errorf("request path = %q, want %q", r.URL.Path, want)
		}
		writeJSON(w, routedNetworkJSON(dsNetworkID, "rotterdam-a", "db-backend"))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var got networkModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}
	wantString(t, "id", got.ID, upper)
}

// Scenario "Unknown id".
func TestNetworkDataSourceFailsOnANotFound(t *testing.T) {
	t.Parallel()

	resp := readNetwork(t, dsNetworkID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"code":"E000404","detail":"network not found","traceId":"4bf92f3577b34da6a3ce929d0e0e4736"}`))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	d := resp.Diagnostics.Errors()[0]
	if want := "network not found on this account"; d.Summary() != want {
		t.Errorf("summary = %q, want %q", d.Summary(), want)
	}
	if !strings.Contains(d.Detail(), "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Errorf("detail = %q, want it to contain the traceId", d.Detail())
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("state = %v, want no state on a not found", resp.State.Raw)
	}
}

func TestNetworkDataSourceReportsAServerError(t *testing.T) {
	t.Parallel()

	resp := readNetwork(t, dsNetworkID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"title":"Internal Server Error","status":500,"code":"E000001","detail":"read failed","traceId":"0af7651916cd43dd8448eb211c80319c"}`))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	d := resp.Diagnostics.Errors()[0]
	if d.Summary() != "Internal Server Error" {
		t.Errorf("summary = %q, want the problem title", d.Summary())
	}
	for _, want := range []string{"Unable to read i3D.net network", "read failed", "E000001", "0af7651916cd43dd8448eb211c80319c"} {
		if !strings.Contains(d.Detail(), want) {
			t.Errorf("detail = %q, want it to contain %q", d.Detail(), want)
		}
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("state = %v, want no state on an error", resp.State.Raw)
	}
}

// Scenario "Network created outside Terraform", end to end through the
// provider: an upper-case id of an i3D-provisioned network reads its
// vlan_id and routing_domain_id and keeps the id as configured.
func TestNetworkDataSourceThroughTheProvider(t *testing.T) {
	var badRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/network/v1/networks/"+dsNetworkID ||
			r.Header.Get("PRIVATE-TOKEN") != "unit-test-key" {
			badRequests.Add(1)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		writeJSON(w, routedNetworkJSON(dsNetworkID, "rotterdam-a", "legacy"))
	}))
	t.Cleanup(srv.Close)

	upper := strings.ToUpper(dsNetworkID)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "i3dnet" {
  api_key  = "unit-test-key"
  base_url = %q
}

data "i3dnet_network" "legacy" {
  id = %q
}
`, srv.URL, upper),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_network.legacy", "id", upper),
					resource.TestCheckResourceAttr("data.i3dnet_network.legacy", "vlan_id", "812"),
					resource.TestCheckResourceAttr("data.i3dnet_network.legacy", "routing_domain_id", testRoutingDomainID),
					resource.TestCheckResourceAttr("data.i3dnet_network.legacy", "routing_enabled", "true"),
					resource.TestCheckResourceAttr("data.i3dnet_network.legacy", "retention", "ephemeral"),
				),
			},
		},
	})

	if n := badRequests.Load(); n != 0 {
		t.Errorf("%d requests missed GET /network/v1/networks/{id} or the PRIVATE-TOKEN header", n)
	}
}

func TestNetworkDataSourceRejectsAnInvalidID(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "i3dnet" {
  api_key  = "unit-test-key"
  base_url = %q
}

data "i3dnet_network" "bad" {
  id = "not-a-uuid"
}
`, srv.URL),
				ExpectError: regexp.MustCompile(`must be a UUID`),
			},
		},
	})

	if n := calls.Load(); n != 0 {
		t.Errorf("made %d requests, want none for an invalid id", n)
	}
}

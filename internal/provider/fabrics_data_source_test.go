package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// readFabrics runs the fabrics data source Read against handler and returns
// the response.
func readFabrics(t *testing.T, handler http.HandlerFunc) *datasource.ReadResponse {
	t.Helper()

	ctx := context.Background()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	api, err := network_api.NewAPI("key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}
	d := &fabricsDataSource{client: api}

	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema
	null := tftypes.NewValue(s.Type().TerraformType(ctx), nil)

	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: null}}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s, Raw: null}}, resp)

	return resp
}

func writeFabricsPage(w http.ResponseWriter, next string, items ...string) {
	links := `"self":{"href":"/fabrics?limit=100"}`
	if next != "" {
		links += fmt.Sprintf(`,"next":{"href":%q}`, next)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"items":[%s],"links":{%s}}`, strings.Join(items, ","), links)
}

func fabricJSON(slug, displayName, metro string) string {
	return fmt.Sprintf(`{"slug":%q,"displayName":%q,"metro":%q}`, slug, displayName, metro)
}

func TestFabricsDataSourceReadsEveryPage(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resp := readFabrics(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/network/v1/fabrics" {
			t.Errorf("request path = %q, want /network/v1/fabrics", r.URL.Path)
		}
		switch calls.Add(1) {
		case 1:
			writeFabricsPage(w, "/fabrics?limit=100&cursor=abc",
				fabricJSON("rotterdam-a", "Rotterdam A", "Rotterdam"),
				fabricJSON("rotterdam-b", "Rotterdam B", "Rotterdam"))
		default:
			if got := r.URL.Query().Get("cursor"); got != "abc" {
				t.Errorf("page 2 cursor = %q, want abc", got)
			}
			writeFabricsPage(w, "", fabricJSON("tokyo-a", "Tokyo A", "Tokyo"))
		}
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var got fabricsDataSourceModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}

	want := []fabricModel{
		{Slug: types.StringValue("rotterdam-a"), DisplayName: types.StringValue("Rotterdam A"), Metro: types.StringValue("Rotterdam")},
		{Slug: types.StringValue("rotterdam-b"), DisplayName: types.StringValue("Rotterdam B"), Metro: types.StringValue("Rotterdam")},
		{Slug: types.StringValue("tokyo-a"), DisplayName: types.StringValue("Tokyo A"), Metro: types.StringValue("Tokyo")},
	}
	if len(got.Fabrics) != len(want) {
		t.Fatalf("fabrics = %v, want %d entries", got.Fabrics, len(want))
	}
	for i := range want {
		if !got.Fabrics[i].Slug.Equal(want[i].Slug) ||
			!got.Fabrics[i].DisplayName.Equal(want[i].DisplayName) ||
			!got.Fabrics[i].Metro.Equal(want[i].Metro) {
			t.Errorf("fabrics[%d] = %+v, want %+v", i, got.Fabrics[i], want[i])
		}
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("made %d requests, want 2", n)
	}
}

func TestFabricsDataSourceReadsAnEmptyListingAsAnEmptyList(t *testing.T) {
	t.Parallel()

	resp := readFabrics(t, func(w http.ResponseWriter, r *http.Request) {
		writeFabricsPage(w, "")
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var got fabricsDataSourceModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}
	if got.Fabrics == nil || len(got.Fabrics) != 0 {
		t.Errorf("fabrics = %#v, want an empty, non-null list", got.Fabrics)
	}
}

func TestFabricsDataSourceReportsTheProblemDocument(t *testing.T) {
	t.Parallel()

	resp := readFabrics(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"title":"Internal Server Error","status":500,"code":"E000001","detail":"listing failed","traceId":"0af7651916cd43dd8448eb211c80319c"}`))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	d := resp.Diagnostics.Errors()[0]
	if d.Summary() != "Internal Server Error" {
		t.Errorf("summary = %q, want the problem title", d.Summary())
	}
	for _, want := range []string{"Unable to read i3D.net fabrics", "listing failed", "E000001", "0af7651916cd43dd8448eb211c80319c"} {
		if !strings.Contains(d.Detail(), want) {
			t.Errorf("detail = %q, want it to contain %q", d.Detail(), want)
		}
	}
}

func TestFabricsDataSourceReportsATransportFailure(t *testing.T) {
	t.Parallel()

	resp := readFabrics(t, func(w http.ResponseWriter, r *http.Request) {
		dropConnection(t, w)
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Unable to read i3D.net fabrics" {
		t.Errorf("summary = %q, want the data source message", got)
	}
}

// End to end through the provider: base_url and api_key reach the network
// client, and the listing is served from {base_url}/network/v1/fabrics.
func TestFabricsDataSourceThroughTheProvider(t *testing.T) {
	var badRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/network/v1/fabrics" || r.Header.Get("PRIVATE-TOKEN") != "unit-test-key" {
			badRequests.Add(1)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeFabricsPage(w, "", fabricJSON("rotterdam-a", "Rotterdam A", "Rotterdam"))
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

data "i3dnet_fabrics" "all" {}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_fabrics.all", "fabrics.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_fabrics.all", "fabrics.0.slug", "rotterdam-a"),
					resource.TestCheckResourceAttr("data.i3dnet_fabrics.all", "fabrics.0.display_name", "Rotterdam A"),
					resource.TestCheckResourceAttr("data.i3dnet_fabrics.all", "fabrics.0.metro", "Rotterdam"),
				),
			},
		},
	})

	if n := badRequests.Load(); n != 0 {
		t.Errorf("%d requests missed /network/v1/fabrics or the PRIVATE-TOKEN header", n)
	}
}

// A gateway error whose JSON body does not parse still yields the status
// diagnostic, never a decode error.
func TestFabricsDataSourceReportsAGatewayErrorWithABrokenJSONBody(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"items":[`, ""} {
		t.Run(fmt.Sprintf("body %q", body), func(t *testing.T) {
			t.Parallel()

			resp := readFabrics(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(body))
			})

			if !resp.Diagnostics.HasError() {
				t.Fatal("Read: no error, want a diagnostic")
			}
			d := resp.Diagnostics.Errors()[0]
			if want := "Network API returned HTTP 502 Bad Gateway"; d.Summary() != want {
				t.Errorf("summary = %q, want %q", d.Summary(), want)
			}
			for _, want := range []string{"Unable to read i3D.net fabrics", "Status code: 502"} {
				if !strings.Contains(d.Detail(), want) {
					t.Errorf("detail = %q, want it to contain %q", d.Detail(), want)
				}
			}
			if strings.Contains(d.Detail(), "decode") {
				t.Errorf("detail = %q, want the status, not a decode error", d.Detail())
			}
		})
	}
}

func TestFabricsDataSourceReportsA200ThatDoesNotDecode(t *testing.T) {
	t.Parallel()

	resp := readFabrics(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"slug":`))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	d := resp.Diagnostics.Errors()[0]
	if d.Summary() != "Unable to read i3D.net fabrics" {
		t.Errorf("summary = %q, want the data source message", d.Summary())
	}
	if !strings.Contains(d.Detail(), "200") {
		t.Errorf("detail = %q, want the HTTP status of the undecodable body", d.Detail())
	}
}

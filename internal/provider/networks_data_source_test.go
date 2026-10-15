package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	filterFabric = "filters[fabric][eq]"
	filterState  = "filters[state][eq]"
	filterName   = "filters[name][contains]"
)

func readNetworks(t *testing.T, set map[string]any, handler http.HandlerFunc) *datasource.ReadResponse {
	t.Helper()

	return readDataSource(t, func(api *network_api.ClientWithResponses) datasource.DataSource {
		return &networksDataSource{client: api}
	}, set, handler)
}

// isolatedNetworkJSON is a private isolated network with no name.
func isolatedNetworkJSON(id, fabric string) string {
	return fmt.Sprintf(`{"id":%q,"fabric":%q,"connectivity":{"type":"private","routing":{"enabled":false}},`+
		`"addressing":{"source":"none"},"state":"provisioning","retention":"persistent",`+
		`"links":{"self":{"href":"/networks/%s"}}}`, id, fabric, id)
}

func networksFromState(t *testing.T, resp *datasource.ReadResponse) networksDataSourceModel {
	t.Helper()

	var got networksDataSourceModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}

	return got
}

// Scenario "Filter by fabric and name": the filters and limit are repeated on
// every page, because the cursor carries no filters.
func TestNetworksDataSourceSendsTheFiltersOnEveryPage(t *testing.T) {
	t.Parallel()

	const (
		id1 = "11111111-1111-4111-8111-111111111111"
		id2 = "22222222-2222-4222-8222-222222222222"
		id3 = "33333333-3333-4333-8333-333333333333"
	)

	var (
		mu   sync.Mutex
		reqs []*url.URL
	)
	resp := readNetworks(t, map[string]any{
		"fabric":        "rotterdam-a",
		"state":         "active",
		"name_contains": "db",
	}, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "key" {
			t.Errorf("PRIVATE-TOKEN = %q, want key", got)
		}
		mu.Lock()
		reqs = append(reqs, r.URL)
		n := len(reqs)
		mu.Unlock()

		switch n {
		case 1:
			writeListPage(w, "/networks?limit=100&cursor=o%3D100", routedNetworkJSON(id1, "rotterdam-a", "db-1"))
		case 2:
			writeListPage(w, "/networks?limit=100&cursor=o%3D200", routedNetworkJSON(id2, "rotterdam-a", "db-2"))
		case 3:
			writeListPage(w, "", routedNetworkJSON(id3, "rotterdam-a", "db-3"))
		default:
			t.Errorf("request %d (%s) past the last page", n, r.URL)
			w.WriteHeader(http.StatusTeapot)
		}
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(reqs) != 3 {
		t.Fatalf("made %d requests, want 3", len(reqs))
	}
	wantCursors := []string{"", "o=100", "o=200"}
	for i, u := range reqs {
		q := u.Query()
		if u.Path != "/network/v1/networks" {
			t.Errorf("request %d path = %q, want /network/v1/networks (never the bare href)", i+1, u.Path)
		}
		for key, want := range map[string]string{
			filterFabric: "rotterdam-a",
			filterState:  "active",
			filterName:   "db",
			"limit":      "100",
			"cursor":     wantCursors[i],
		} {
			if got := q.Get(key); got != want {
				t.Errorf("request %d %s = %q, want %q", i+1, key, got, want)
			}
		}
	}

	got := networksFromState(t, resp)
	wantString(t, "fabric", got.Fabric, "rotterdam-a")
	wantString(t, "state", got.State, "active")
	wantString(t, "name_contains", got.NameContains, "db")
	wantIDs := []string{id1, id2, id3}
	if len(got.Networks) != len(wantIDs) {
		t.Fatalf("networks = %d items, want %d (all pages)", len(got.Networks), len(wantIDs))
	}
	for i, id := range wantIDs {
		wantString(t, fmt.Sprintf("networks[%d].id", i), got.Networks[i].ID, id)
		wantString(t, fmt.Sprintf("networks[%d].routing_domain_id", i), got.Networks[i].RoutingDomainID, testRoutingDomainID)
	}
	if !got.Networks[0].VlanID.Equal(types.Int64Value(812)) {
		t.Errorf("networks[0].vlan_id = %v, want 812", got.Networks[0].VlanID)
	}
}

func TestNetworksDataSourceSendsNoFilterWhenNoneIsSet(t *testing.T) {
	t.Parallel()

	const id = "44444444-4444-4444-8444-444444444444"
	resp := readNetworks(t, nil, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for key := range q {
			if strings.HasPrefix(key, "filters[") {
				t.Errorf("request sent filter %s=%q, want no filter", key, q.Get(key))
			}
		}
		if got := q.Get("limit"); got != "100" {
			t.Errorf("limit = %q, want 100", got)
		}
		writeListPage(w, "", isolatedNetworkJSON(id, "tokyo-a"))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	got := networksFromState(t, resp)
	if len(got.Networks) != 1 {
		t.Fatalf("networks = %v, want 1 item", got.Networks)
	}
	wantString(t, "networks[0].id", got.Networks[0].ID, id)
}

// Scenario "No match".
func TestNetworksDataSourceReadsNoMatchAsAnEmptyList(t *testing.T) {
	t.Parallel()

	resp := readNetworks(t, map[string]any{"name_contains": "nothing-matches"}, func(w http.ResponseWriter, r *http.Request) {
		writeListPage(w, "")
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	got := networksFromState(t, resp)
	if got.Networks == nil || len(got.Networks) != 0 {
		t.Errorf("networks = %#v, want an empty, non-null list", got.Networks)
	}
}

func TestNetworksDataSourceReportsAnErrorOnALaterPageWithoutPartialState(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resp := readNetworks(t, map[string]any{"fabric": "rotterdam-a"}, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			writeListPage(w, "/networks?limit=100&cursor=o%3D100",
				isolatedNetworkJSON("55555555-5555-4555-8555-555555555555", "rotterdam-a"))
			return
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"title":"Internal Server Error","status":500,"code":"E000001","detail":"listing failed","traceId":"trace-page-2"}`))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	d := resp.Diagnostics.Errors()[0]
	if d.Summary() != "Internal Server Error" {
		t.Errorf("summary = %q, want the problem title", d.Summary())
	}
	for _, want := range []string{"Unable to read i3D.net networks", "listing failed", "trace-page-2"} {
		if !strings.Contains(d.Detail(), want) {
			t.Errorf("detail = %q, want it to contain %q", d.Detail(), want)
		}
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("state = %v, want no partial state", resp.State.Raw)
	}
	if n := calls.Load(); n != 2 {
		t.Errorf("made %d requests, want 2", n)
	}
}

// End to end through the provider: the filter reaches the request and the
// networks are exposed as list items.
func TestNetworksDataSourceThroughTheProvider(t *testing.T) {
	const id = "66666666-6666-4666-8666-666666666666"

	var badRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/network/v1/networks" || r.Header.Get("PRIVATE-TOKEN") != "unit-test-key" ||
			q.Get(filterFabric) != "rotterdam-a" || q.Get(filterName) != "db" || q.Has(filterState) {
			badRequests.Add(1)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		writeListPage(w, "", routedNetworkJSON(id, "rotterdam-a", "db-main"))
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

data "i3dnet_networks" "db" {
  fabric        = "rotterdam-a"
  name_contains = "db"
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_networks.db", "networks.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_networks.db", "networks.0.id", id),
					resource.TestCheckResourceAttr("data.i3dnet_networks.db", "networks.0.name", "db-main"),
					resource.TestCheckResourceAttr("data.i3dnet_networks.db", "networks.0.vlan_id", "812"),
					resource.TestCheckResourceAttr("data.i3dnet_networks.db", "networks.0.routing_domain_id", testRoutingDomainID),
				),
			},
		},
	})

	if n := badRequests.Load(); n != 0 {
		t.Errorf("%d requests missed /network/v1/networks, the filters or the PRIVATE-TOKEN header", n)
	}
}

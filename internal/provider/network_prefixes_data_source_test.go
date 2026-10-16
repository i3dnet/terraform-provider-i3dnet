package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	filterNetworkID = "filters[networkId][eq]"
	filterFamily    = "filters[family][eq]"
	filterMetro     = "filters[metro][eq]"
)

func readPrefixes(t *testing.T, set map[string]any, handler http.HandlerFunc) *datasource.ReadResponse {
	t.Helper()

	return readDataSource(t, func(api *network_api.ClientWithResponses) datasource.DataSource {
		return &networkPrefixesDataSource{client: api}
	}, set, handler)
}

// networkPrefixJSON is an active IPv4 prefix created through network networkID.
func networkPrefixJSON(id, cidr, networkID string) string {
	return fmt.Sprintf(`{"id":%q,"cidr":%q,"family":4,"state":"active","gatewayIp":"10.0.0.1",`+
		`"routerIps":["10.0.0.2"],"vlanId":812,"fabric":"rotterdam-a","metro":"rotterdam",`+
		`"networkId":%q,"retention":"persistent","contractId":"C-1001",`+
		`"created":"2026-09-01T10:00:00Z","lastUpdated":"2026-09-02T11:30:15Z",`+
		`"links":{"self":{"href":"/prefixes/%s"}}}`, id, cidr, networkID, id)
}

// legacyPrefixJSON is a prefix outside the network model: no networkId, no
// retention, no contract.
func legacyPrefixJSON(id, cidr string) string {
	return fmt.Sprintf(`{"id":%q,"cidr":%q,"family":4,"state":"active","vlanId":812,`+
		`"fabric":"rotterdam-a","metro":"rotterdam","links":{"self":{"href":"/prefixes/%s"}}}`, id, cidr, id)
}

func prefixesFromState(t *testing.T, resp *datasource.ReadResponse) networkPrefixesDataSourceModel {
	t.Helper()

	var got networkPrefixesDataSourceModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}

	return got
}

// Scenario "Prefixes of one network": the filters and limit are repeated on
// every page, because the cursor carries no filters.
func TestNetworkPrefixesDataSourceSendsTheFiltersOnEveryPage(t *testing.T) {
	t.Parallel()

	const (
		id1 = "01984f2e-2b4c-7e31-8a20-000000000001"
		id2 = "01984f2e-2b4c-7e31-8a20-000000000002"
		id3 = "01984f2e-2b4c-7e31-8a20-000000000003"
	)

	var (
		mu   sync.Mutex
		reqs []*url.URL
	)
	resp := readPrefixes(t, map[string]any{
		"network_id": dsNetworkID,
		"fabric":     "rotterdam-a",
		"family":     4,
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
			writeListPage(w, "/prefixes?limit=100&cursor=o%3D100", networkPrefixJSON(id1, "10.0.0.0/24", dsNetworkID))
		case 2:
			writeListPage(w, "/prefixes?limit=100&cursor=o%3D200", legacyPrefixJSON(id2, "10.0.1.0/24"))
		case 3:
			writeListPage(w, "", networkPrefixJSON(id3, "10.0.2.0/24", dsNetworkID))
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
		if u.Path != "/network/v1/prefixes" {
			t.Errorf("request %d path = %q, want /network/v1/prefixes (never the bare href)", i+1, u.Path)
		}
		for key, want := range map[string]string{
			filterNetworkID: dsNetworkID,
			filterFamily:    "4",
			filterFabric:    "rotterdam-a",
			"limit":         "100",
			"cursor":        wantCursors[i],
		} {
			if got := q.Get(key); got != want {
				t.Errorf("request %d %s = %q, want %q", i+1, key, got, want)
			}
		}
		if q.Has(filterMetro) {
			t.Errorf("request %d sent %s, want it absent", i+1, filterMetro)
		}
	}

	got := prefixesFromState(t, resp)
	wantString(t, "network_id", got.NetworkID, dsNetworkID)
	wantString(t, "fabric", got.Fabric, "rotterdam-a")
	wantIDs := []string{id1, id2, id3}
	if len(got.Prefixes) != len(wantIDs) {
		t.Fatalf("prefixes = %d items, want %d (all pages)", len(got.Prefixes), len(wantIDs))
	}
	for i, id := range wantIDs {
		wantString(t, fmt.Sprintf("prefixes[%d].id", i), got.Prefixes[i].ID, id)
	}
	wantString(t, "prefixes[0].network_id", got.Prefixes[0].NetworkID, dsNetworkID)
	wantString(t, "prefixes[0].cidr", got.Prefixes[0].Cidr, "10.0.0.0/24")
	if !got.Prefixes[1].NetworkID.IsNull() {
		t.Errorf("prefixes[1].network_id = %v, want null on a legacy prefix", got.Prefixes[1].NetworkID)
	}
}

func TestNetworkPrefixesDataSourceSendsTheMetroFilter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resp := readPrefixes(t, map[string]any{"metro": "rotterdam"}, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if got := q.Get(filterMetro); got != "rotterdam" {
			t.Errorf("%s = %q, want rotterdam", filterMetro, got)
		}
		for _, key := range []string{filterFabric, filterNetworkID, filterFamily} {
			if q.Has(key) {
				t.Errorf("request sent %s=%q, want it absent", key, q.Get(key))
			}
		}
		if got := q.Get("limit"); got != "100" {
			t.Errorf("limit = %q, want 100", got)
		}
		writeListPage(w, "", legacyPrefixJSON("01984f2e-2b4c-7e31-8a20-000000000004", "10.0.3.0/24"))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d requests, want 1", n)
	}
	if got := prefixesFromState(t, resp); len(got.Prefixes) != 1 {
		t.Errorf("prefixes = %d items, want 1", len(got.Prefixes))
	}
}

func TestNetworkPrefixesDataSourceSendsNoFilterWhenNoneIsSet(t *testing.T) {
	t.Parallel()

	resp := readPrefixes(t, nil, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for key := range q {
			if strings.HasPrefix(key, "filters[") {
				t.Errorf("request sent filter %s=%q, want no filter", key, q.Get(key))
			}
		}
		if got := q.Get("limit"); got != "100" {
			t.Errorf("limit = %q, want 100", got)
		}
		writeListPage(w, "", legacyPrefixJSON("01984f2e-2b4c-7e31-8a20-000000000005", "10.0.4.0/24"))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if got := prefixesFromState(t, resp); len(got.Prefixes) != 1 {
		t.Errorf("prefixes = %d items, want 1", len(got.Prefixes))
	}
}

func TestNetworkPrefixesDataSourceReadsNoMatchAsAnEmptyList(t *testing.T) {
	t.Parallel()

	resp := readPrefixes(t, map[string]any{"network_id": dsNetworkID}, func(w http.ResponseWriter, r *http.Request) {
		writeListPage(w, "")
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	got := prefixesFromState(t, resp)
	if got.Prefixes == nil || len(got.Prefixes) != 0 {
		t.Errorf("prefixes = %#v, want an empty, non-null list", got.Prefixes)
	}
}

func TestNetworkPrefixesDataSourceReportsAnErrorWithoutState(t *testing.T) {
	t.Parallel()

	resp := readPrefixes(t, map[string]any{"fabric": "rotterdam-a"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"title":"Internal Server Error","status":500,"code":"E000001","detail":"listing failed","traceId":"trace-prefixes"}`))
	})

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read: no error, want a diagnostic")
	}
	d := resp.Diagnostics.Errors()[0]
	if d.Summary() != "Internal Server Error" {
		t.Errorf("summary = %q, want the problem title", d.Summary())
	}
	for _, want := range []string{"Unable to read i3D.net network prefixes", "listing failed", "trace-prefixes"} {
		if !strings.Contains(d.Detail(), want) {
			t.Errorf("detail = %q, want it to contain %q", d.Detail(), want)
		}
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("state = %v, want no state on an error", resp.State.Raw)
	}
}

// Scenario "Prefixes of one network", end to end through the provider.
func TestNetworkPrefixesDataSourceThroughTheProvider(t *testing.T) {
	const id = "01984f2e-2b4c-7e31-8a20-000000000006"

	var badRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/network/v1/prefixes" || r.Header.Get("PRIVATE-TOKEN") != "unit-test-key" ||
			q.Get(filterNetworkID) != dsNetworkID || q.Get(filterFamily) != "4" {
			badRequests.Add(1)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		writeListPage(w, "", networkPrefixJSON(id, "10.0.0.0/24", dsNetworkID))
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

data "i3dnet_network_prefixes" "web" {
  network_id = %q
  family     = 4
}
`, srv.URL, dsNetworkID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_network_prefixes.web", "prefixes.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_network_prefixes.web", "prefixes.0.id", id),
					resource.TestCheckResourceAttr("data.i3dnet_network_prefixes.web", "prefixes.0.cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("data.i3dnet_network_prefixes.web", "prefixes.0.family", "4"),
					resource.TestCheckResourceAttr("data.i3dnet_network_prefixes.web", "prefixes.0.network_id", dsNetworkID),
				),
			},
		},
	})

	if n := badRequests.Load(); n != 0 {
		t.Errorf("%d requests missed /network/v1/prefixes, the filters or the PRIVATE-TOKEN header", n)
	}
}

// Scenario "Fabric and metro together", plus the other arguments validated
// before any request.
func TestNetworkPrefixesDataSourceRejectsInvalidArguments(t *testing.T) {
	for name, tc := range map[string]struct {
		args string
		want *regexp.Regexp
	}{
		"fabric and metro together": {
			args: "fabric = \"rotterdam-a\"\n  metro  = \"rotterdam\"",
			want: regexp.MustCompile(`(?s)Invalid Attribute Combination.*cannot be specified when`),
		},
		"family 5": {
			args: "family = 5",
			want: regexp.MustCompile(`(?s)family.*must be one of`),
		},
		"network_id not a version 7 UUID": {
			args: "network_id = \"0b9c2f7e-4d51-4a8e-9f3a-1c2d3e4f5a6b\"",
			want: regexp.MustCompile(`must be a version 7 UUID`),
		},
	} {
		t.Run(name, func(t *testing.T) {
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

data "i3dnet_network_prefixes" "bad" {
  %s
}
`, srv.URL, tc.args),
						ExpectError: tc.want,
					},
				},
			})

			if n := calls.Load(); n != 0 {
				t.Errorf("made %d requests, want none for invalid arguments", n)
			}
		})
	}
}

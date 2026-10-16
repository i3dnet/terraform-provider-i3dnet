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
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// dsLegacyPrefixID is a prefix created before the network model: its
// inventory UUID is version 4, not version 7.
const dsLegacyPrefixID = "0b9c2f7e-4d51-4a8e-9f3a-1c2d3e4f5a6b"

func readIPAddresses(t *testing.T, set map[string]any, handler http.HandlerFunc) *datasource.ReadResponse {
	t.Helper()

	return readDataSource(t, func(api *network_api.ClientWithResponses) datasource.DataSource {
		return &networkIPAddressesDataSource{client: api}
	}, set, handler)
}

// ipAddressJSON is an IPv4 address under prefixID; role "" omits the role
// (a customer address).
func ipAddressJSON(id, address, prefixID, role string) string {
	roleField := ""
	if role != "" {
		roleField = fmt.Sprintf(`"role":%q,`, role)
	}

	return fmt.Sprintf(`{"id":%q,"address":%q,"family":4,"prefixId":%q,%s`+
		`"created":"2026-09-01T10:00:00Z","links":{"self":{"href":"/prefixes/%s/ip-addresses/%s"}}}`,
		id, address, prefixID, roleField, prefixID, id)
}

func ipAddressesFromState(t *testing.T, resp *datasource.ReadResponse) networkIPAddressesDataSourceModel {
	t.Helper()

	var got networkIPAddressesDataSourceModel
	if diags := resp.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatalf("State.Get: %v", diags)
	}

	return got
}

// Scenario "Addresses of one network for DNS": every page of the network
// listing is read with the same limit, and each item carries its prefix_id.
func TestNetworkIPAddressesDataSourceReadsEveryPageOfTheNetwork(t *testing.T) {
	t.Parallel()

	const (
		prefixV4 = "01984f2e-2b4c-7e31-8a20-000000000001"
		ip1      = "01984f2e-3d5e-7f41-9b30-000000000001"
		ip2      = "01984f2e-3d5e-7f41-9b30-000000000002"
		ip3      = "01984f2e-3d5e-7f41-9b30-000000000003"
	)
	base := "/networks/" + dsNetworkID + "/ip-addresses"

	var (
		mu   sync.Mutex
		reqs []*url.URL
	)
	resp := readIPAddresses(t, map[string]any{"network_id": dsNetworkID}, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "key" {
			t.Errorf("PRIVATE-TOKEN = %q, want key", got)
		}
		mu.Lock()
		reqs = append(reqs, r.URL)
		n := len(reqs)
		mu.Unlock()

		switch n {
		case 1:
			writeListPage(w, base+"?cursor=o%3D100", ipAddressJSON(ip1, "10.0.0.1", prefixV4, "gateway"))
		case 2:
			writeListPage(w, base+"?cursor=o%3D200", ipAddressJSON(ip2, "10.0.0.10", prefixV4, ""))
		case 3:
			writeListPage(w, "", ipAddressJSON(ip3, "10.0.1.10", dsLegacyPrefixID, ""))
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
		if want := "/network/v1" + base; u.Path != want {
			t.Errorf("request %d path = %q, want %q (never the bare href)", i+1, u.Path, want)
		}
		if got := u.Query().Get("limit"); got != "100" {
			t.Errorf("request %d limit = %q, want 100", i+1, got)
		}
		if got := u.Query().Get("cursor"); got != wantCursors[i] {
			t.Errorf("request %d cursor = %q, want %q", i+1, got, wantCursors[i])
		}
	}

	got := ipAddressesFromState(t, resp)
	wantString(t, "network_id", got.NetworkID, dsNetworkID)
	if !got.PrefixID.IsNull() {
		t.Errorf("prefix_id = %v, want null (not configured)", got.PrefixID)
	}
	want := []struct{ id, address, prefixID string }{
		{ip1, "10.0.0.1", prefixV4},
		{ip2, "10.0.0.10", prefixV4},
		{ip3, "10.0.1.10", dsLegacyPrefixID},
	}
	if len(got.IPAddresses) != len(want) {
		t.Fatalf("ip_addresses = %d items, want %d (all pages)", len(got.IPAddresses), len(want))
	}
	for i, w := range want {
		wantString(t, fmt.Sprintf("ip_addresses[%d].id", i), got.IPAddresses[i].ID, w.id)
		wantString(t, fmt.Sprintf("ip_addresses[%d].address", i), got.IPAddresses[i].Address, w.address)
		wantString(t, fmt.Sprintf("ip_addresses[%d].prefix_id", i), got.IPAddresses[i].PrefixID, w.prefixID)
	}
	wantString(t, "ip_addresses[0].role", got.IPAddresses[0].Role, "gateway")
}

// The listing counters belong to the usage data source: neither the data
// source nor its items expose them.
func TestNetworkIPAddressesDataSourceDoesNotExposeTheCounters(t *testing.T) {
	t.Parallel()

	var resp datasource.SchemaResponse
	NewNetworkIPAddressesDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	items, ok := resp.Schema.Attributes["ip_addresses"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("ip_addresses is %T, want a list nested attribute", resp.Schema.Attributes["ip_addresses"])
	}
	for _, counter := range []string{"capacity", "allocated", "available"} {
		if _, ok := resp.Schema.Attributes[counter]; ok {
			t.Errorf("data source has attribute %q, want the counters not exposed", counter)
		}
		if _, ok := items.NestedObject.Attributes[counter]; ok {
			t.Errorf("ip_addresses items have attribute %q, want the counters not exposed", counter)
		}
	}
}

// A prefix whose UUID is not version 7 is read through the prefix route.
func TestNetworkIPAddressesDataSourceReadsALegacyPrefix(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resp := readIPAddresses(t, map[string]any{"prefix_id": dsLegacyPrefixID}, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if want := "/network/v1/prefixes/" + dsLegacyPrefixID + "/ip-addresses"; r.URL.Path != want {
			t.Errorf("request path = %q, want %q", r.URL.Path, want)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("limit = %q, want 100", got)
		}
		writeListPage(w, "", ipAddressJSON("01984f2e-3d5e-7f41-9b30-000000000011", "10.0.1.10", dsLegacyPrefixID, ""))
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d requests, want 1", n)
	}

	got := ipAddressesFromState(t, resp)
	wantString(t, "prefix_id", got.PrefixID, dsLegacyPrefixID)
	if len(got.IPAddresses) != 1 {
		t.Fatalf("ip_addresses = %d items, want 1", len(got.IPAddresses))
	}
	wantString(t, "ip_addresses[0].prefix_id", got.IPAddresses[0].PrefixID, dsLegacyPrefixID)
}

func TestNetworkIPAddressesDataSourceReadsNoAddressAsAnEmptyList(t *testing.T) {
	t.Parallel()

	resp := readIPAddresses(t, map[string]any{"network_id": dsNetworkID}, func(w http.ResponseWriter, r *http.Request) {
		writeListPage(w, "")
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	got := ipAddressesFromState(t, resp)
	if got.IPAddresses == nil || len(got.IPAddresses) != 0 {
		t.Errorf("ip_addresses = %#v, want an empty, non-null list", got.IPAddresses)
	}
}

// Scenario "Unknown network", plus a server error: no state in any case.
func TestNetworkIPAddressesDataSourceReportsErrorsWithoutState(t *testing.T) {
	t.Parallel()

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"

	for name, tc := range map[string]struct {
		set         map[string]any
		status      int
		title       string
		wantSummary string
		wantDetail  []string
	}{
		"network not found": {
			set:         map[string]any{"network_id": dsNetworkID},
			status:      http.StatusNotFound,
			title:       "Not Found",
			wantSummary: "network not found on this account",
			wantDetail:  []string{traceID},
		},
		"prefix not found": {
			set:         map[string]any{"prefix_id": dsLegacyPrefixID},
			status:      http.StatusNotFound,
			title:       "Not Found",
			wantSummary: "prefix not found on this account",
			wantDetail:  []string{traceID},
		},
		"server error": {
			set:         map[string]any{"network_id": dsNetworkID},
			status:      http.StatusInternalServerError,
			title:       "Internal Server Error",
			wantSummary: "Internal Server Error",
			wantDetail:  []string{"Unable to read i3D.net IP addresses", "listing failed", "E000001", traceID},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := readIPAddresses(t, tc.set, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"title":%q,"status":%d,"code":"E000001","detail":"listing failed","traceId":%q}`,
					tc.title, tc.status, traceID)
			})

			if !resp.Diagnostics.HasError() {
				t.Fatal("Read: no error, want a diagnostic")
			}
			d := resp.Diagnostics.Errors()[0]
			if d.Summary() != tc.wantSummary {
				t.Errorf("summary = %q, want %q", d.Summary(), tc.wantSummary)
			}
			for _, want := range tc.wantDetail {
				if !strings.Contains(d.Detail(), want) {
					t.Errorf("detail = %q, want it to contain %q", d.Detail(), want)
				}
			}
			if !resp.State.Raw.IsNull() {
				t.Errorf("state = %v, want no state on an error", resp.State.Raw)
			}
		})
	}
}

// Scenario "Addresses of one network for DNS", end to end through the provider.
func TestNetworkIPAddressesDataSourceThroughTheProvider(t *testing.T) {
	const prefixID = "01984f2e-2b4c-7e31-8a20-000000000001"

	var badRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/network/v1/networks/"+dsNetworkID+"/ip-addresses" ||
			r.Header.Get("PRIVATE-TOKEN") != "unit-test-key" || r.URL.Query().Get("limit") != "100" {
			badRequests.Add(1)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		writeListPage(w, "",
			ipAddressJSON("01984f2e-3d5e-7f41-9b30-000000000021", "10.0.0.1", prefixID, "gateway"),
			ipAddressJSON("01984f2e-3d5e-7f41-9b30-000000000022", "10.0.0.10", prefixID, ""),
		)
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

data "i3dnet_network_ip_addresses" "web" {
  network_id = %q
}
`, srv.URL, dsNetworkID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_network_ip_addresses.web", "ip_addresses.#", "2"),
					resource.TestCheckResourceAttr("data.i3dnet_network_ip_addresses.web", "ip_addresses.0.role", "gateway"),
					resource.TestCheckResourceAttr("data.i3dnet_network_ip_addresses.web", "ip_addresses.0.prefix_id", prefixID),
					resource.TestCheckNoResourceAttr("data.i3dnet_network_ip_addresses.web", "ip_addresses.1.role"),
					resource.TestCheckResourceAttr("data.i3dnet_network_ip_addresses.web", "ip_addresses.1.address", "10.0.0.10"),
				),
			},
		},
	})

	if n := badRequests.Load(); n != 0 {
		t.Errorf("%d requests missed GET /network/v1/networks/{id}/ip-addresses, the limit or the PRIVATE-TOKEN header", n)
	}
}

// Scenario "Neither or both arguments", plus the id formats, all rejected
// before any request.
func TestNetworkIPAddressesDataSourceRejectsInvalidArguments(t *testing.T) {
	for name, tc := range map[string]struct {
		args string
		want *regexp.Regexp
	}{
		"neither argument": {
			args: "",
			want: regexp.MustCompile(`(?s)Exactly\s+one\s+of`),
		},
		"both arguments": {
			args: fmt.Sprintf("network_id = %q\n  prefix_id  = %q", dsNetworkID, dsLegacyPrefixID),
			want: regexp.MustCompile(`(?s)Exactly\s+one\s+of`),
		},
		"network_id not a version 7 UUID": {
			args: fmt.Sprintf("network_id = %q", dsLegacyPrefixID),
			want: regexp.MustCompile(`must be a version 7 UUID`),
		},
		"prefix_id not a UUID": {
			args: `prefix_id = "not-a-uuid"`,
			want: regexp.MustCompile(`(?s)Invalid prefix id.*must be a UUID`),
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

data "i3dnet_network_ip_addresses" "bad" {
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

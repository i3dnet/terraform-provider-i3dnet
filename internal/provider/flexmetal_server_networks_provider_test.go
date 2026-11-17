package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// FM-5404: networks on i3dnet_flexmetal_server, through Terraform against a fake One API.

// fakeOneAPI answers the FlexMetal server endpoints the resource uses. A server is delivered at once, with the networks
// its create request asked for: addresses in request order, two default ones (IPv4, IPv6) where ip_allocations was
// left out, and none for [].
type fakeOneAPI struct {
	mu       sync.Mutex
	requests []map[string]any
	servers  map[string]map[string]any
	refuse   *string // a 422 body for every create
}

func newFakeOneAPI(t *testing.T) (*fakeOneAPI, *httptest.Server) {
	t.Helper()
	api := &fakeOneAPI{servers: map[string]map[string]any{}}
	srv := httptest.NewServer(http.HandlerFunc(api.handle))
	t.Cleanup(srv.Close)

	return api, srv
}

func (f *fakeOneAPI) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/flexMetal/servers"):
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.requests = append(f.requests, body)
		if f.refuse != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(*f.refuse))
			return
		}
		uuid := fmt.Sprintf("5f2b1c3d-0000-4000-8000-%012d", len(f.requests))
		f.servers[uuid] = fakeServer(uuid, body)
		created := map[string]any{}
		for k, v := range f.servers[uuid] {
			created[k] = v
		}
		created["status"], created["networks"], created["ipAddresses"] = "created", []any{}, []any{}
		_ = json.NewEncoder(w).Encode([]any{created})
	case strings.Contains(path, "/flexMetal/servers/"):
		uuid := path[strings.LastIndex(path, "/")+1:]
		server, ok := f.servers[uuid]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errorCode": 404, "errorMessage": "Not found"}`))
			return
		}
		if r.Method == http.MethodDelete {
			server["status"] = "released"
		}
		_ = json.NewEncoder(w).Encode([]any{server})
	default:
		w.WriteHeader(http.StatusTeapot)
	}
}

func fakeServer(uuid string, request map[string]any) map[string]any {
	requested, ok := request["networks"].([]any)
	if !ok {
		requested = []any{map[string]any{"networkId": netPublic, "tagged": false}}
	}

	var networks, ipAddresses []any
	for i, n := range requested {
		network := n.(map[string]any)
		entries, hasEntries := network["ipAllocations"].([]any)
		if !hasEntries {
			entries = []any{map[string]any{"family": 4.0}, map[string]any{"family": 6.0}}
		}
		allocations := []any{}
		for j, e := range entries {
			entry := e.(map[string]any)
			family, _ := entry["family"].(float64)
			prefixID, _ := entry["prefixId"].(string)
			if family == 0 {
				family = 4
			}
			if prefixID == "" {
				prefixID = fmt.Sprintf("7e1b8448-7e97-5d24-9fba-%012d", i*10+j)
			}
			address, cidr, gateway := fmt.Sprintf("203.0.%d.%d", 113+i, 10+j), fmt.Sprintf("203.0.%d.0/24", 113+i), any(nil)
			if family == 6 {
				address, cidr = fmt.Sprintf("2a00:1630:%d::%d", i, 10+j), fmt.Sprintf("2a00:1630:%d::/64", i)
			} else if i == 0 {
				gateway = fmt.Sprintf("203.0.%d.1", 113+i)
			}
			allocations = append(allocations, map[string]any{
				"id": fmt.Sprintf("1f9a7c3e-4b2d-4e6f-8a15-%012d", i*10+j), "family": family, "address": address,
				"prefix": map[string]any{"id": prefixID, "cidr": cidr, "gateway": gateway},
			})
			ipAddresses = append(ipAddresses, map[string]any{"ipAddress": address})
		}
		name, vlan := any(fmt.Sprintf("net-%d", i)), any(3000+i)
		if len(allocations) == 0 {
			name, vlan = nil, nil // a layer 2 only network whose details the service didn't send
		}
		networks = append(networks, map[string]any{
			"networkId": network["networkId"], "tagged": network["tagged"], "name": name, "vlanId": vlan,
			"fabric": "rotterdam-a", "connectivity": "public", "ipAllocations": allocations,
		})
	}

	return map[string]any{
		"uuid": uuid, "name": request["name"], "status": "delivered", "statusMessage": "",
		"location":     map[string]any{"id": 6, "name": request["location"]},
		"instanceType": map[string]any{"id": 1, "name": request["instanceType"]},
		"os":           map[string]any{"slug": request["os"].(map[string]any)["slug"]},
		"ipAddresses":  ipAddresses, "networks": networks, "tags": []any{}, "createdAt": 1000, "deliveredAt": 1100,
	}
}

func (f *fakeOneAPI) lastRequestNetworks(t *testing.T) string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := json.Marshal(f.requests[len(f.requests)-1]["networks"])

	return string(raw)
}

func serverConfig(baseURL, networks string) string {
	return fmt.Sprintf(`
provider "i3dnet" {
  api_key  = "unit-test-key"
  base_url = %q
}

resource "i3dnet_flexmetal_server" "web" {
  name          = "web"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
  }
%s
}
`, baseURL, networks)
}

const configuredNetworks = `
  networks = [
    {
      network_id     = "` + netPublic + `"
      ip_allocations = [
        { family = 6 },
        { prefix_id = "` + prefixV4 + `" },
      ]
    },
    {
      network_id     = "` + netBackend + `"
      tagged         = true
      ip_allocations = []
    },
    {
      network_id = "9e4d1b63-2f7a-4c5e-8d39-6a1f0b2c4e43"
      tagged     = true
    },
  ]
`

// Creates the server with networks, reads them back without a diff on the next plan, replaces the server when they
// change, and imports them.
func TestServerNetworksThroughTheProvider(t *testing.T) {
	api, srv := newFakeOneAPI(t)
	const name = "i3dnet_flexmetal_server.web"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: serverConfig(srv.URL, configuredNetworks),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "networks.#", "3"),
					resource.TestCheckResourceAttr(name, "networks.0.network_id", netPublic),
					resource.TestCheckResourceAttr(name, "networks.0.tagged", "false"),
					resource.TestCheckResourceAttr(name, "networks.0.name", "net-0"),
					resource.TestCheckResourceAttr(name, "networks.0.vlan_id", "3000"),
					// {family = 6}: prefix_id from the response; {prefix_id}: family from the response
					resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.0.family", "6"),
					resource.TestCheckResourceAttrSet(name, "networks.0.ip_allocations.0.prefix_id"),
					resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.0.address", "2a00:1630:0::10"),
					resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.1.prefix_id", prefixV4),
					resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.1.family", "4"),
					resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.1.prefix.gateway", "203.0.113.1"),
					// layer 2 only: no addresses, and details the service didn't send are null
					resource.TestCheckResourceAttr(name, "networks.1.tagged", "true"),
					resource.TestCheckResourceAttr(name, "networks.1.ip_allocations.#", "0"),
					resource.TestCheckNoResourceAttr(name, "networks.1.name"),
					// ip_allocations left out: the default addresses, a prefix without gateway
					resource.TestCheckResourceAttr(name, "networks.2.ip_allocations.#", "2"),
					resource.TestCheckNoResourceAttr(name, "networks.2.ip_allocations.0.prefix.gateway"),
				),
			},
			{
				// The plan after apply must be empty (checked by the framework on every step); a change replaces.
				Config: serverConfig(srv.URL, strings.Replace(configuredNetworks, "tagged         = true", "tagged         = false", 1)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(name, "networks.1.tagged", "false"),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				// The resource is identified by uuid; it has no id attribute.
				ImportStateIdFunc:                    importUUID(name),
				ImportStateVerifyIdentifierAttribute: "uuid",
				// Not returned by the GET (documented on import), or not part of the API.
				ImportStateVerifyIgnore: []string{"ssh_key", "post_install_script", "timeouts", "overflow"},
			},
		},
	})

	want := `[{"ipAllocations":[{"family":6},{"prefixId":"` + prefixV4 + `"}],"networkId":"` + netPublic + `","tagged":false},` +
		`{"ipAllocations":[],"networkId":"` + netBackend + `","tagged":false},` +
		`{"networkId":"9e4d1b63-2f7a-4c5e-8d39-6a1f0b2c4e43","tagged":true}]`
	if got := api.lastRequestNetworks(t); got != want {
		t.Errorf("last create request networks =\n%s\nwant\n%s", got, want)
	}
}

// A server created without networks gets them from the response, with no diff on the next plan.
func TestServerWithoutNetworksReadsThemWithoutADiff(t *testing.T) {
	api, srv := newFakeOneAPI(t)
	const name = "i3dnet_flexmetal_server.web"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: serverConfig(srv.URL, ""),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "networks.#", "1"),
				resource.TestCheckResourceAttr(name, "networks.0.network_id", netPublic),
				resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.#", "2"),
			),
		}},
	})

	if got := api.lastRequestNetworks(t); got != "null" {
		t.Errorf("networks must be left out of the request, got %s", got)
	}
}

func TestServerNetworksShowTheOneAPIError(t *testing.T) {
	api, srv := newFakeOneAPI(t)
	refuse := `{"errorCode": 11001, "errorMessage": "Invalid parameter", "errors": [{"property": "networks", ` +
		`"message": "network 0b6f3c2e-6a1d-4f2b-9a51-3c8e2f7d1a10 is in another fabric"}]}`
	api.refuse = &refuse

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      serverConfig(srv.URL, configuredNetworks),
			ExpectError: regexp.MustCompile(`(?s)Message: networks: Network.*another\s+fabric`),
		}},
	})
}

func importUUID(name string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		return state.RootModule().Resources[name].Primary.Attributes["uuid"], nil
	}
}

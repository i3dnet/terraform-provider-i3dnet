package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// fakeCDNAPI is an in-memory CDN API: resources and their settings, stored as
// the raw JSON the client sent, the way the real API echoes them back.
type fakeCDNAPI struct {
	mu sync.Mutex

	resources map[string]map[string]json.RawMessage // uid -> "base" and setting name -> JSON
	nextUID   int
	calls     []string // "METHOD path", in order

	// failPut makes PUT on this setting fail with HTTP 500.
	failPut string
	// staleDeletes makes DELETE of a resource refuse this many times with the
	// API's "still has" error, as when its cleared settings are not visible yet.
	staleDeletes int
	// databaseDown makes GET of a resource answer 404 and the listing 500, as
	// the API does when it cannot read its database.
	databaseDown bool
	// createBody, when set, replaces the body of a successful create.
	createBody string
}

func newFakeCDNAPI(t *testing.T) (*fakeCDNAPI, *httptest.Server) {
	t.Helper()

	api := &fakeCDNAPI{resources: map[string]map[string]json.RawMessage{}}
	srv := httptest.NewServer(http.HandlerFunc(api.handle))
	t.Cleanup(srv.Close)

	return api, srv
}

func (f *fakeCDNAPI) callLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.calls...)
}

func (f *fakeCDNAPI) setting(uid, name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return string(f.resources[uid][name])
}

func (f *fakeCDNAPI) setSetting(uid, name, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.resources[uid][name] = json.RawMessage(value)
}

func writeCDNJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func (f *fakeCDNAPI) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.Header.Get("Authorization") != "Bearer cdn_unit_test" {
		writeCDNJSON(w, http.StatusUnauthorized, `{"code":401,"message":"unauthorized"}`)
		return
	}

	p := strings.TrimPrefix(r.URL.Path, "/api/")
	f.calls = append(f.calls, r.Method+" "+p)
	body, _ := io.ReadAll(r.Body)

	if p == "service/acl/predefined" && r.Method == http.MethodGet {
		writeCDNJSON(w, 200, `[{"key":"office","label":"Office","description":"The office network"}]`)
		return
	}

	if p == "service/resource" && r.Method == http.MethodPost {
		var req struct{ Name, GroupCode, InitiativeCode string }
		_ = json.Unmarshal(body, &req)
		f.nextUID++
		uid := fmt.Sprintf("uid%d", f.nextUID)
		base, _ := json.Marshal(map[string]string{
			"uid": uid, "name": req.Name, "groupCode": req.GroupCode, "initiativeCode": req.InitiativeCode,
		})
		f.resources[uid] = map[string]json.RawMessage{"base": base}
		if f.createBody != "" {
			writeCDNJSON(w, 201, f.createBody)
			return
		}
		writeCDNJSON(w, 201, fmt.Sprintf(`{"uid":%q,"name":%q}`, uid, req.Name))
		return
	}

	if p == "service" && r.Method == http.MethodGet {
		if f.databaseDown {
			writeCDNJSON(w, 500, `{"code":500,"message":"failed to query vhosts"}`)
			return
		}
		list := make([]json.RawMessage, 0, len(f.resources))
		for _, res := range f.resources {
			list = append(list, res["base"])
		}
		out, _ := json.Marshal(list)
		writeCDNJSON(w, 200, string(out))
		return
	}

	parts := strings.Split(strings.TrimPrefix(p, "service/resource/"), "/")
	uid := parts[0]
	res, ok := f.resources[uid]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			if !ok || f.databaseDown {
				writeCDNJSON(w, 404, `{"code":404,"message":"no vhost found with the provided uid"}`)
				return
			}
			writeCDNJSON(w, 200, string(res["base"]))
		case http.MethodPut:
			var req struct{ Name string }
			_ = json.Unmarshal(body, &req)
			var base map[string]string
			_ = json.Unmarshal(res["base"], &base)
			base["name"] = req.Name
			res["base"], _ = json.Marshal(base)
			writeCDNJSON(w, 200, string(res["base"]))
		case http.MethodDelete:
			if !ok {
				writeCDNJSON(w, 400, `{"code":400,"message":"could not get vhost model from db"}`)
				return
			}
			if f.staleDeletes > 0 {
				f.staleDeletes--
				writeCDNJSON(w, 400, `{"code":400,"message":"cannot delete resource because it still has backend entries associated to it"}`)
				return
			}
			for name := range res {
				if name != "base" {
					writeCDNJSON(w, 400, fmt.Sprintf(`{"code":400,"message":"cannot delete resource because it still has %s entries associated to it"}`, name))
					return
				}
			}
			delete(f.resources, uid)
			w.WriteHeader(204)
		}
		return
	}

	setting := parts[1]
	switch r.Method {
	case http.MethodGet:
		v, set := res[setting]
		switch {
		case set:
			writeCDNJSON(w, 200, string(v))
		case setting == "flags":
			writeCDNJSON(w, 200, `{"enabled":false,"force_https":false,"auth_pass_through":false,"use_i3d_static_dns":false,"override_backend_host":false,"override_health_host":true}`)
		case setting == "hostnames":
			writeCDNJSON(w, 200, `[]`)
		case setting == "predefined-allowlist":
			writeCDNJSON(w, 200, `{"key":""}`)
		default:
			writeCDNJSON(w, 200, `{}`)
		}
	case http.MethodPut:
		if setting == f.failPut {
			writeCDNJSON(w, 500, `{"code":500,"message":"failed to save event"}`)
			return
		}
		res[setting] = json.RawMessage(body)
		writeCDNJSON(w, 201, string(body))
	case http.MethodDelete:
		delete(res, setting)
		w.WriteHeader(204)
	}
}

func cdnProviderBlock(url string) string {
	return fmt.Sprintf(`
provider "i3dnet" {
  cdn_api_key  = "cdn_unit_test"
  cdn_base_url = "%s/api"
}
`, url)
}

const cdnResourceConfig = `
resource "i3dnet_cdn_resource" "site" {
  name            = %q
  group_code      = "team"
  initiative_code = "web"

  hostnames = [{ fqdn = "www.example.com", ssl_cn = "wildcard.example.com" }]

  origins = [{
    fqdn        = "origin.example.com"
    healthcheck = { path = "/healthz" }
  }]

  predefined_allowlist = "office"
}
`

func TestCDNResourceLifecycle(t *testing.T) {
	api, srv := newFakeCDNAPI(t)
	cdnDeleteRetryInterval = 10 * time.Millisecond

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			if n := len(api.resources); n != 0 {
				return fmt.Errorf("%d resources left after destroy", n)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "id", "uid1"),
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "cname_target", "static.cdn-v2.i3d.net."),
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "strategy", "HASH"),
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "origins.0.port", "443"),
					resource.TestCheckNoResourceAttr("i3dnet_cdn_resource.site", "origins.0.healthcheck.host"),
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "flags.enabled", "true"),
					func(*terraform.State) error {
						// The default health check host is the origin's URL.
						if got := api.setting("uid1", "backend"); !strings.Contains(got, `"host":"https://origin.example.com"`) {
							return fmt.Errorf("backend = %s, want the default health check host", got)
						}
						// Fail closed: the allowlist is set before anything that publishes the site.
						return checkOrder(api.callLog(),
							"POST service/resource",
							"PUT service/resource/uid1/predefined-allowlist",
							"PUT service/resource/uid1/backend",
							"PUT service/resource/uid1/hostnames",
							"PUT service/resource/uid1/flags",
						)
					},
				),
			},
			// Rename in place.
			{
				Config: cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "renamed"),
				Check:  resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "name", "renamed"),
			},
			// A change made outside Terraform shows up as a difference and is put back.
			{
				PreConfig: func() {
					api.setSetting("uid1", "flags", `{"enabled":false,"force_https":true,"auth_pass_through":false,"override_backend_host":true,"override_health_host":true,"use_i3d_static_dns":false}`)
				},
				Config:             cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "renamed"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "renamed"),
				Check: func(*terraform.State) error {
					if got := api.setting("uid1", "flags"); !strings.Contains(got, `"enabled":true`) {
						return fmt.Errorf("flags = %s, want enabled restored", got)
					}
					return nil
				},
			},
			// Import gives an empty plan.
			{
				ResourceName:            "i3dnet_cdn_resource.site",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			// Moving to another group is refused at plan time.
			{
				Config:      cdnProviderBlock(srv.URL) + strings.Replace(fmt.Sprintf(cdnResourceConfig, "renamed"), `"team"`, `"other"`, 1),
				ExpectError: regexp.MustCompile(`Cannot move a CDN resource`),
			},
			// Removing the allowlist clears it.
			{
				Config: cdnProviderBlock(srv.URL) + strings.Replace(fmt.Sprintf(cdnResourceConfig, "renamed"), `predefined_allowlist = "office"`, "", 1),
				Check: func(*terraform.State) error {
					if got := api.setting("uid1", "predefined-allowlist"); got != "" {
						return fmt.Errorf("predefined allowlist = %s, want cleared", got)
					}
					return nil
				},
			},
			// Destroy waits out a delete the API refuses while it catches up.
			{
				PreConfig: func() { api.staleDeletes = 2 },
				Config:    cdnProviderBlock(srv.URL),
			},
		},
	})
}

// A resource whose configuration fails after creation stays in the state
// (tainted) instead of being lost.
func TestCDNResourcePartialCreateKeepsTheResource(t *testing.T) {
	api, srv := newFakeCDNAPI(t)
	api.failPut = "hostnames"
	cdnDeleteRetryInterval = 10 * time.Millisecond

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
				ExpectError: regexp.MustCompile(`setting the hostnames of resource uid1`),
			},
			{
				// The next apply replaces the tainted resource.
				PreConfig: func() { api.failPut = "" },
				Config:    cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "id", "uid2"),
					func(*terraform.State) error {
						if _, ok := api.resources["uid1"]; ok {
							return fmt.Errorf("the tainted resource uid1 was not deleted")
						}
						return nil
					},
				),
			},
		},
	})
}

// A resource deleted outside Terraform is planned for creation again.
func TestCDNResourceDeletedOutsideTerraform(t *testing.T) {
	api, srv := newFakeCDNAPI(t)
	cdnDeleteRetryInterval = 10 * time.Millisecond

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site")},
			{
				PreConfig:          func() { api.mu.Lock(); delete(api.resources, "uid1"); api.mu.Unlock() },
				Config:             cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// The API answers 404 for a resource when it cannot read its database. That
// must not make Terraform forget a resource that still exists.
func TestCDNResourceKeepsTheResourceWhenA404IsNotConfirmed(t *testing.T) {
	api, srv := newFakeCDNAPI(t)
	cdnDeleteRetryInterval = 10 * time.Millisecond

	setDown := func(down bool) func() {
		return func() { api.mu.Lock(); api.databaseDown = down; api.mu.Unlock() }
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site")},
			{
				// Database down: the listing fails too, so refresh errors.
				PreConfig:   setDown(true),
				Config:      cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
				ExpectError: regexp.MustCompile(`Could not confirm that the resource is gone`),
			},
			{
				// Back up: the same resource, no changes, nothing created.
				PreConfig: setDown(false),
				Config:    cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
				Check:     resource.TestCheckResourceAttr("i3dnet_cdn_resource.site", "id", "uid1"),
			},
		},
	})
}

// A create that succeeds without a usable uid says how to find the resource.
func TestCDNResourceCreateWithoutAUIDGivesRecoveryGuidance(t *testing.T) {
	// " " stands for an empty body: the fake treats "" as "not set".
	for _, body := range []string{`{}`, ` `, `{"uid":`} {
		t.Run(fmt.Sprintf("body %q", body), func(t *testing.T) {
			api, srv := newFakeCDNAPI(t)
			api.createBody = body

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      cdnProviderBlock(srv.URL) + fmt.Sprintf(cdnResourceConfig, "site"),
						ExpectError: regexp.MustCompile(`may have created the resource anyway`),
					},
				},
			})
		})
	}
}

func TestCDNPredefinedAllowlistsDataSource(t *testing.T) {
	_, srv := newFakeCDNAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cdnProviderBlock(srv.URL) + `data "i3dnet_cdn_predefined_allowlists" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_cdn_predefined_allowlists.all", "allowlists.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_cdn_predefined_allowlists.all", "allowlists.0.key", "office"),
				),
			},
		},
	})
}

// checkOrder reports an error unless want appears in calls in this order.
func checkOrder(calls []string, want ...string) error {
	i := 0
	for _, c := range calls {
		if i < len(want) && c == want[i] {
			i++
		}
	}
	if i != len(want) {
		return fmt.Errorf("calls %v do not contain %v in order (stopped at %q)", calls, want, want[i])
	}

	return nil
}

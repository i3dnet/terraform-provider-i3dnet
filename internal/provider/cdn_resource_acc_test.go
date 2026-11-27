package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCDNResource creates, changes, imports and destroys a real CDN
// resource. It needs a CDN API key and the group and initiative the key
// belongs to, and is skipped without them:
//
//	I3D_CDN_API_KEY, I3D_CDN_GROUP_CODE, I3D_CDN_INITIATIVE_CODE
//	I3D_CDN_HOSTNAME, I3D_CDN_SSL_CN   a hostname the key may use, and its certificate
//	I3D_CDN_ORIGIN                     an origin that answers /
//	I3D_CDN_BASE_URL                   optional
//
// The resource is created disabled, so it never serves traffic.
func TestAccCDNResource(t *testing.T) {
	env := map[string]string{}
	for _, name := range []string{"I3D_CDN_API_KEY", "I3D_CDN_GROUP_CODE", "I3D_CDN_INITIATIVE_CODE", "I3D_CDN_HOSTNAME", "I3D_CDN_SSL_CN", "I3D_CDN_ORIGIN"} {
		env[name] = os.Getenv(name)
		if env[name] == "" {
			t.Skipf("%s is not set; skipping the CDN acceptance test", name)
		}
	}

	name := acctest.RandomWithPrefix("tf-acc")

	config := func(name, strategy string) string {
		return fmt.Sprintf(`
provider "i3dnet" {
  cdn_api_key  = %q
  cdn_base_url = %q
}

resource "i3dnet_cdn_resource" "test" {
  name            = %q
  group_code      = %q
  initiative_code = %q
  strategy        = %q

  hostnames = [{ fqdn = %q, ssl_cn = %q }]

  origins = [{
    fqdn        = %q
    healthcheck = { path = "/" }
  }]

  flags = { enabled = false }
}
`, env["I3D_CDN_API_KEY"], os.Getenv("I3D_CDN_BASE_URL"), name, env["I3D_CDN_GROUP_CODE"], env["I3D_CDN_INITIATIVE_CODE"],
			strategy, env["I3D_CDN_HOSTNAME"], env["I3D_CDN_SSL_CN"], env["I3D_CDN_ORIGIN"])
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(name, "HASH"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("i3dnet_cdn_resource.test", "id"),
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.test", "flags.enabled", "false"),
				),
			},
			{
				Config: config(name+"-renamed", "FALLBACK"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.test", "name", name+"-renamed"),
					resource.TestCheckResourceAttr("i3dnet_cdn_resource.test", "strategy", "FALLBACK"),
				),
			},
			{
				ResourceName:            "i3dnet_cdn_resource.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

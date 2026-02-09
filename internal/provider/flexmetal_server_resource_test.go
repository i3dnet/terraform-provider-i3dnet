package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFlexmetalServerResourceWithUpdate(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig(t) + `
resource "i3dnet_flexmetal_server" "my-talos" {
  name          = "talosHostNameAcceptanceTest"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "talos-omni-190"
    kernel_params = [
      {
        key   = "siderolink.api"
        value = "https://siderolink.api/?jointoken=secret"
      },
      {
        key   = "talos.customparam"
        value = "123456"
      }
    ]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify static values are set
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "name", "talosHostNameAcceptanceTest"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "location", "EU: Rotterdam"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "instance_type", "bm7.std.8"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.slug", "talos-omni-190"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.kernel_params.0.key", "siderolink.api"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.kernel_params.0.value", "https://siderolink.api/?jointoken=secret"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.kernel_params.1.key", "talos.customparam"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.kernel_params.1.value", "123456"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "status", "delivered"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_server.my-talos", "uuid"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_server.my-talos", "created_at"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_server.my-talos", "delivered_at"),
					// IPv4 and IPv6 expected, so we expect 2 addresses.
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "ip_addresses.#", "2"),
				),
			},
			{
				Config: providerConfig(t) + `
resource "i3dnet_flexmetal_server" "my-talos" {
  name          = "talosHostNameAcceptanceTest"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "talos-omni-195"
    kernel_params = [
      {
        key   = "siderolink.api"
        value = "https://siderolink.api/?jointoken=secret"
      },
      {
        key   = "talos.customparam_changed"
        value = "654321"
      }
    ]
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.slug", "talos-omni-195"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.kernel_params.1.key", "talos.customparam_changed"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "os.kernel_params.1.value", "654321"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_server.my-talos", "status", "delivered"),
				),
			},
		},
	})
}

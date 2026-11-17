package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFlexmetalServerNetworks delivers a real server on two long-lived networks of the test account (FM-5404): an
// untagged public network with an address from its prefix, and a tagged layer 2 only network. The networks are
// created once in rotterdam-a with the Dedicated Networks API (contract I3DTFTEST) and passed in as
// I3D_TEST_NETWORK_PUBLIC_ID and I3D_TEST_NETWORK_L2_ID; the prefix is looked up the way a customer would.
func TestAccFlexmetalServerNetworks(t *testing.T) {
	// Parallel, so this delivery runs alongside the other acceptance tests instead of adding to the job's 15 minutes.
	t.Parallel()

	if os.Getenv("TF_MAIN") != "true" {
		t.Skip("To run this test, set TF_MAIN=true env var")
	}
	publicNetwork, l2Network := os.Getenv("I3D_TEST_NETWORK_PUBLIC_ID"), os.Getenv("I3D_TEST_NETWORK_L2_ID")
	if publicNetwork == "" || l2Network == "" {
		t.Skip("To run this test, set I3D_TEST_NETWORK_PUBLIC_ID and I3D_TEST_NETWORK_L2_ID")
	}
	const name = "i3dnet_flexmetal_server.networks"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: providerConfig(t, resourceNsFlexmetal) + fmt.Sprintf(`
data "i3dnet_network_prefixes" "public_ipv4" {
  network_id = %q
  family     = 4
}

resource "i3dnet_flexmetal_server" "networks" {
  name          = "terraformNetworksAcceptanceTest"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
  }
  networks = [
    {
      network_id     = %q
      ip_allocations = [{ prefix_id = data.i3dnet_network_prefixes.public_ipv4.prefixes[0].id }]
    },
    {
      network_id     = %q
      tagged         = true
      ip_allocations = []
    },
  ]
}
`, publicNetwork, publicNetwork, l2Network),
			// The plan after apply must be empty, which the framework checks on every step.
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(name, "status", "delivered"),
				resource.TestCheckResourceAttr(name, "networks.#", "2"),
				resource.TestCheckResourceAttr(name, "networks.0.network_id", publicNetwork),
				resource.TestCheckResourceAttr(name, "networks.0.tagged", "false"),
				resource.TestCheckResourceAttrSet(name, "networks.0.vlan_id"),
				resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.#", "1"),
				resource.TestCheckResourceAttr(name, "networks.0.ip_allocations.0.family", "4"),
				resource.TestCheckResourceAttrPair(name, "networks.0.ip_allocations.0.prefix_id",
					"data.i3dnet_network_prefixes.public_ipv4", "prefixes.0.id"),
				resource.TestCheckResourceAttrPair(name, "networks.0.ip_allocations.0.prefix.id",
					"data.i3dnet_network_prefixes.public_ipv4", "prefixes.0.id"),
				resource.TestCheckResourceAttrSet(name, "networks.0.ip_allocations.0.address"),
				resource.TestCheckResourceAttrSet(name, "networks.0.ip_allocations.0.prefix.gateway"),
				resource.TestCheckResourceAttr(name, "networks.1.network_id", l2Network),
				resource.TestCheckResourceAttr(name, "networks.1.tagged", "true"),
				resource.TestCheckResourceAttr(name, "networks.1.ip_allocations.#", "0"),
			),
		}},
	})
}

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFlexmetalVmPool_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
resource "i3dnet_flexmetal_vm_pool" "test" {
  name          = "acc-test-pool"
  location_id   = "EU-NL-01"
  contract_id   = "contract-123"
  type          = "on_demand"
  instance_type = "bm9.hmm.gpu.4rtx4000.64"
  vlan_id       = 100
  subnet = [
    {
      cidr        = "10.0.0.0/24"
      gateway     = "10.0.0.1"
      range_start = "10.0.0.10"
      range_end   = "10.0.0.254"
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "name", "acc-test-pool"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm_pool.test", "id"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm_pool.test", "status"),
				),
			},
		},
	})
}

func TestAccFlexmetalVmPool_update(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
resource "i3dnet_flexmetal_vm_pool" "test" {
  name          = "acc-test-pool-update"
  location_id   = "EU-NL-01"
  contract_id   = "contract-123"
  type          = "on_demand"
  instance_type = "bm9.hmm.gpu.4rtx4000.64"
  vlan_id       = 100
  subnet = [
    {
      cidr        = "10.0.0.0/24"
      gateway     = "10.0.0.1"
      range_start = "10.0.0.10"
      range_end   = "10.0.0.254"
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "name", "acc-test-pool-update"),
				),
			},
			{
				Config: providerConfig(t) + `
resource "i3dnet_flexmetal_vm_pool" "test" {
  name          = "acc-test-pool-renamed"
  location_id   = "EU-NL-01"
  contract_id   = "contract-123"
  type          = "on_demand"
  instance_type = "bm9.hmm.gpu.4rtx4000.64"
  vlan_id       = 100
  subnet = [
    {
      cidr        = "10.0.0.0/24"
      gateway     = "10.0.0.1"
      range_start = "10.0.0.10"
      range_end   = "10.0.0.254"
    }
  ]
  metadata = {
    env = "test"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "name", "acc-test-pool-renamed"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "metadata.env", "test"),
				),
			},
		},
	})
}

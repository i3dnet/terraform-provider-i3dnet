package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const vmPoolConfig = `
resource "i3dnet_flexmetal_vm_pool" "test" {
  name          = "acc-test-vm-pool"
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
`

func TestAccFlexmetalVm_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + vmPoolConfig + `
resource "i3dnet_flexmetal_vm" "test" {
  name    = "acc-test-vm"
  pool_id = i3dnet_flexmetal_vm_pool.test.id
  plan    = "vm-standard-4"
  os = {
    image_id = "ubuntu-2404-lts"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm.test", "name", "acc-test-vm"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm.test", "status", "running"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm.test", "id"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm.test", "ip_address"),
				),
			},
		},
	})
}

func TestAccFlexmetalVm_withGpu(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + vmPoolConfig + `
resource "i3dnet_flexmetal_vm" "gpu" {
  name    = "acc-test-vm-gpu"
  pool_id = i3dnet_flexmetal_vm_pool.test.id
  plan    = "vm-gpu-1rtx4000"
  os = {
    image_id = "ubuntu-2404-lts"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm.gpu", "name", "acc-test-vm-gpu"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm.gpu", "status", "running"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm.gpu", "id"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm.gpu", "ip_address"),
				),
			},
		},
	})
}

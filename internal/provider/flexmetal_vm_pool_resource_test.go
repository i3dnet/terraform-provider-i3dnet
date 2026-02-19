package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// vmPoolEnv holds environment variable values required for VM pool acceptance tests.
type vmPoolEnv struct {
	LocationID   string
	ContractID   string
	InstanceType string
	VlanID       int
	SubnetCIDR   string
	Gateway      string
	RangeStart   string
	RangeEnd     string
}

func vmPoolEnvConfig(t *testing.T) vmPoolEnv {
	t.Helper()
	get := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			t.Fatalf("%s environment variable is required for VM pool acceptance tests", key)
		}
		return v
	}

	vlanStr := get("I3D_VM_VLAN_ID")
	vlanID, err := strconv.Atoi(vlanStr)
	if err != nil {
		t.Fatalf("I3D_VM_VLAN_ID must be an integer, got: %s", vlanStr)
	}

	return vmPoolEnv{
		LocationID:   get("I3D_VM_LOCATION_ID"),
		ContractID:   get("I3D_VM_CONTRACT_ID"),
		InstanceType: get("I3D_VM_INSTANCE_TYPE"),
		VlanID:       vlanID,
		SubnetCIDR:   get("I3D_VM_SUBNET_CIDR"),
		Gateway:      get("I3D_VM_GATEWAY"),
		RangeStart:   get("I3D_VM_RANGE_START"),
		RangeEnd:     get("I3D_VM_RANGE_END"),
	}
}

func vmPoolHCL(name string, env vmPoolEnv) string {
	return fmt.Sprintf(`
resource "i3dnet_flexmetal_vm_pool" "test" {
  name          = %q
  location_id   = %q
  contract_id   = %q
  type          = "on_demand"
  instance_type = %q
  vlan_id       = %d
  subnet = [
    {
      cidr        = %q
      gateway     = %q
      range_start = %q
      range_end   = %q
    }
  ]
}
`, name, env.LocationID, env.ContractID, env.InstanceType, env.VlanID,
		env.SubnetCIDR, env.Gateway, env.RangeStart, env.RangeEnd)
}

func TestAccFlexmetalVmPool_basic(t *testing.T) {
	t.Parallel()
	env := vmPoolEnvConfig(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + vmPoolHCL("acc-test-pool", env),
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
	env := vmPoolEnvConfig(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + vmPoolHCL("acc-test-pool-update", env),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "name", "acc-test-pool-update"),
				),
			},
			{
				Config: providerConfig(t) + vmPoolHCLWithMetadata("acc-test-pool-renamed", env),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "name", "acc-test-pool-renamed"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm_pool.test", "metadata.env", "test"),
				),
			},
		},
	})
}

func vmPoolHCLWithMetadata(name string, env vmPoolEnv) string {
	return fmt.Sprintf(`
resource "i3dnet_flexmetal_vm_pool" "test" {
  name          = %q
  location_id   = %q
  contract_id   = %q
  type          = "on_demand"
  instance_type = %q
  vlan_id       = %d
  subnet = [
    {
      cidr        = %q
      gateway     = %q
      range_start = %q
      range_end   = %q
    }
  ]
  metadata = {
    env = "test"
  }
}
`, name, env.LocationID, env.ContractID, env.InstanceType, env.VlanID,
		env.SubnetCIDR, env.Gateway, env.RangeStart, env.RangeEnd)
}

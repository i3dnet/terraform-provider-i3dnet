package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func vmInstanceEnv(t *testing.T) (plan, imageID string) {
	t.Helper()
	get := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			t.Fatalf("%s environment variable is required for VM instance acceptance tests", key)
		}
		return v
	}
	return get("I3D_VM_PLAN"), get("I3D_VM_IMAGE_ID")
}

func vmInstanceHCL(name, plan, imageID string, env vmPoolEnv) string {
	return vmPoolHCL("acc-test-vm-pool", env) + fmt.Sprintf(`
resource "i3dnet_flexmetal_vm" "test" {
  name    = %q
  pool_id = i3dnet_flexmetal_vm_pool.test.id
  plan    = %q
  os = {
    image_id = %q
  }
}
`, name, plan, imageID)
}

func TestAccFlexmetalVm_basic(t *testing.T) {
	t.Parallel()
	env := vmPoolEnvConfig(t)
	plan, imageID := vmInstanceEnv(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + vmInstanceHCL("acc-test-vm", plan, imageID, env),
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
	env := vmPoolEnvConfig(t)

	gpuPlan := os.Getenv("I3D_VM_GPU_PLAN")
	if gpuPlan == "" {
		t.Skip("I3D_VM_GPU_PLAN not set, skipping GPU test")
	}
	_, imageID := vmInstanceEnv(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + vmInstanceHCL("acc-test-vm-gpu", gpuPlan, imageID, env),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm.test", "name", "acc-test-vm-gpu"),
					resource.TestCheckResourceAttr("i3dnet_flexmetal_vm.test", "status", "running"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm.test", "id"),
					resource.TestCheckResourceAttrSet("i3dnet_flexmetal_vm.test", "ip_address"),
				),
			},
		},
	})
}

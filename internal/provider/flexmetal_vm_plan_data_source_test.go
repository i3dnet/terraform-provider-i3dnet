package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFlexmetalVmPlanDataSource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_flexmetal_vm_plans" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.i3dnet_flexmetal_vm_plans.all", "plans.#"),
				),
			},
		},
	})
}

func TestAccFlexmetalVmPlanDataSource_filterBySlug(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_flexmetal_vm_plans" "filtered" {
  slug = "vm-standard-4"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_flexmetal_vm_plans.filtered", "plans.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_flexmetal_vm_plans.filtered", "plans.0.slug", "vm-standard-4"),
				),
			},
		},
	})
}

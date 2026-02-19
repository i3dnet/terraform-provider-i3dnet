package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFlexmetalVmImageDataSource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_flexmetal_vm_image" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.i3dnet_flexmetal_vm_image.all", "images.#"),
				),
			},
		},
	})
}

func TestAccFlexmetalVmImageDataSource_filterByOsFamily(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_flexmetal_vm_image" "linux" {
  os_family = "linux"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.i3dnet_flexmetal_vm_image.linux", "images.#"),
					// All returned images should have os_family = "linux"
					resource.TestCheckResourceAttr("data.i3dnet_flexmetal_vm_image.linux", "images.0.os_family", "linux"),
				),
			},
		},
	})
}

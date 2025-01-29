package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTagResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig(t) + `
resource "i3dnet_tag" "test" {
  name = "foo"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify static values are set
					resource.TestCheckResourceAttr("i3dnet_tag.test", "id", "foo"),
					resource.TestCheckResourceAttr("i3dnet_tag.test", "name", "foo"),
					resource.TestCheckResourceAttr("i3dnet_tag.test", "resources.count", "0"),
					resource.TestCheckResourceAttr("i3dnet_tag.test", "resources.flex_metal_servers.count", "0"),
				),
			},
			{
				ResourceName:      "i3dnet_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSHKeyDataSource(t *testing.T) {
	// todo: This test requires a key with name `Demo` to be available on the test system. Once we have env variables
	//  passed into tests create this key via API client before running the test

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig(t) + `
data "i3d_ssh_key" "example" {
  name = "Demo"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("data.i3d_ssh_key.example", "name", "Demo"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.i3d_ssh_key.example", "uuid"),
					resource.TestCheckResourceAttrSet("data.i3d_ssh_key.example", "public_key"),
					resource.TestCheckResourceAttrSet("data.i3d_ssh_key.example", "created_at"),
				),
			},
		},
	})
}

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSHKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig(t) + `
resource "i3dnet_ssh_key" "test" {
  name       = "Key From Terraform"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify static values are set
					resource.TestCheckResourceAttr("i3dnet_ssh_key.test", "name", "Key From Terraform"),
					resource.TestCheckResourceAttr("i3dnet_ssh_key.test", "public_key", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("i3dnet_ssh_key.test", "uuid"),
					resource.TestCheckResourceAttrSet("i3dnet_ssh_key.test", "created_at"),
				),
			},
			{
				ResourceName:      "i3dnet_ssh_key.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

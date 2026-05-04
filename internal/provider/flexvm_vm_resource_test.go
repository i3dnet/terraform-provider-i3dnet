package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccFlexvmVMResource(t *testing.T) {
	isMain := os.Getenv("TF_MAIN") == "true"
	if !isMain {
		t.Skip("To run this test, set TF_MAIN=true env var")
	}

	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig(t, resourceNsFlexvm) + `
resource "i3dnet_flexvm_vm" "test" {
  cloud_id           = "019d24e2-98fa-701a-8475-8ac0ff1f4a4a"
  name               = "terraform-gh-workflows-test"
  description        = "Terraform GitHub Workflows test"
  instance_type_name = "vm.4c.8g"
  image_name         = "ubuntu-2404-server-amd64"
  ssh_keys           = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("i3dnet_flexvm_vm.test", "id"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_vm.test", "name",
						"terraform-gh-workflows-test"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_vm.test", "description",
						"Terraform GitHub Workflows test"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_vm.test", "instance_type.name",
						"vm.4c.8g"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_vm.test", "image.name",
						"ubuntu-2404-server-amd64"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_vm.test", "status", "running"),
					resource.TestCheckResourceAttrSet("i3dnet_flexvm_vm.test", "cloud.id"),
					resource.TestCheckResourceAttrSet("i3dnet_flexvm_vm.test", "node.id"),
				),
			},
			// Import testing
			{
				ResourceName:      "i3dnet_flexvm_vm.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"instance_type_name",
					"image_name",
					"ssh_keys",
				},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["i3dnet_flexvm_vm.test"]
					return rs.Primary.Attributes["cloud_id"] + "/" + rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

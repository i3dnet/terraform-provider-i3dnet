package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccFlexvmCloudResource(t *testing.T) {
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
resource "i3dnet_flexvm_cloud" "test" {
  name          = "terraform-gh-workflows-cloud-test"
  description   = "Terraform GitHub Workflows cloud test"
  site          = "frmtl1"
  instance_type = "bm9.hmm.gpu.4rtx4000.64"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("i3dnet_flexvm_cloud.test", "id"),
					resource.TestCheckResourceAttrSet("i3dnet_flexvm_cloud.test", "created_at"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_cloud.test", "name",
						"terraform-gh-workflows-cloud-test"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_cloud.test", "description",
						"Terraform GitHub Workflows cloud test"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_cloud.test", "site", "frmtl1"),
					resource.TestCheckResourceAttr("i3dnet_flexvm_cloud.test", "instance_type",
						"bm9.hmm.gpu.4rtx4000.64"),
				),
			},
			// Import testing
			{
				ResourceName:      "i3dnet_flexvm_cloud.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["i3dnet_flexvm_cloud.test"]
					return rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

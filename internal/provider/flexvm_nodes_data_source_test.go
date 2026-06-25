package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFlexvmNodesDataSource(t *testing.T) {
	isMain := os.Getenv("TF_MAIN") == "true"
	if !isMain {
		t.Skip("To run this test, set TF_MAIN=true env var")
	}

	cloudID := os.Getenv("I3D_FLEXVM_CLOUD_ID")
	if cloudID == "" {
		t.Skip("To run this test, set I3D_FLEXVM_CLOUD_ID to an existing Cloud UUID")
	}

	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t, resourceNsFlexvm) + fmt.Sprintf(`
data "i3dnet_flexvm_nodes" "test" {
  cloud_id = %q
}
`, cloudID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_flexvm_nodes.test", "cloud_id", cloudID),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_nodes.test", "nodes.#"),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_nodes.test", "nodes.0.id"),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_nodes.test", "nodes.0.serial"),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_nodes.test", "nodes.0.status"),
				),
			},
		},
	})
}

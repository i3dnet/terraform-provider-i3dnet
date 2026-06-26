package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFlexvmNodeDataSource(t *testing.T) {
	isMain := os.Getenv("TF_MAIN") == "true"
	if !isMain {
		t.Skip("To run this test, set TF_MAIN=true env var")
	}

	cloudID := os.Getenv("I3D_FLEXVM_CLOUD_ID")
	nodeID := os.Getenv("I3D_FLEXVM_NODE_ID")
	if cloudID == "" || nodeID == "" {
		t.Skip("To run this test, set I3D_FLEXVM_CLOUD_ID and I3D_FLEXVM_NODE_ID to an existing Cloud/node")
	}

	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t, resourceNsFlexvm) + fmt.Sprintf(`
data "i3dnet_flexvm_node" "test" {
  cloud_id = %q
  id       = %q
}
`, cloudID, nodeID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_flexvm_node.test", "cloud_id", cloudID),
					resource.TestCheckResourceAttr("data.i3dnet_flexvm_node.test", "id", nodeID),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_node.test", "name"),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_node.test", "serial"),
					resource.TestCheckResourceAttrSet("data.i3dnet_flexvm_node.test", "status"),
				),
			},
		},
	})
}

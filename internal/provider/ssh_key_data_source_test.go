package provider

import (
	"context"
	"os"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSHKeyDataSource(t *testing.T) {
	t.Parallel()

	createTestSSHKey(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig(t) + `
data "i3dnet_ssh_key" "example" {
  name = "TestApiKey"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("data.i3dnet_ssh_key.example", "name", "TestApiKey"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.i3dnet_ssh_key.example", "uuid"),
					resource.TestCheckResourceAttrSet("data.i3dnet_ssh_key.example", "public_key"),
					resource.TestCheckResourceAttrSet("data.i3dnet_ssh_key.example", "created_at"),
				),
			},
		},
	})
}

// createTestSSHKey will create a test SSH key that will be deleted onm cleanup
func createTestSSHKey(t *testing.T) {
	t.Helper()

	apiclient, err := one_api.NewClient(os.Getenv("I3D_API_KEY"), os.Getenv("I3D_BASE_URL"))
	if err != nil {
		t.Fatalf("error creating API Client: %s", err)
	}

	response, err := apiclient.CreateSSHKey(
		context.Background(),
		one_api.CreateSSHKeyReq{
			Name:      "TestApiKey",
			PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net",
		})
	if err != nil {
		t.Fatalf("error creating test SSH Key: %s", err)
	}

	t.Cleanup(func() {
		err = apiclient.DeleteSSHKey(context.Background(), response.SSHKey.Uuid)
		if err != nil {
			t.Fatalf("error deleting SSH Key: %s", err)
		}
	})
}

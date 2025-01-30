package provider

import (
	"context"
	"os"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTagDataSource(t *testing.T) {
	createTestTags(t, "testTag", "secondTag")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig(t) + `
data "i3dnet_tags" "fooTag" {
  name = "testTag"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "name", "testTag"),

					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "tags.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "tags.0.tag", "testTag"),
					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "tags.0.resources.count", "0"),
					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "tags.0.resources.flex_metal_servers.count", "0"),
				),
			},
		},
	})
}

// createTestTags creates test tags that will be deleted on cleanup
func createTestTags(t *testing.T, names ...string) {
	t.Helper()

	apiclient, err := one_api.NewClient(os.Getenv("I3D_API_KEY"), os.Getenv("I3D_BASE_URL"))
	if err != nil {
		t.Fatalf("error creating API Client: %s", err)
	}

	for _, name := range names {
		_, createErr := apiclient.CreateTag(context.Background(), name)
		if createErr != nil {
			t.Fatalf("error creating tag: %s", err)
		}

		t.Cleanup(func() {
			err = apiclient.DeleteTag(context.Background(), name)
			if err != nil {
				t.Fatalf("error deleting tag: %s", err)
			}
		})
	}

}

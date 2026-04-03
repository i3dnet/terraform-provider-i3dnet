package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTagDataSource(t *testing.T) {
	firstTag := randomTagName("testTag")

	createTestTags(t, firstTag, randomTagName("secondTag"))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig(t, resourceNsFlexmetal) + fmt.Sprintf(`
data "i3dnet_tags" "fooTag" {
  name = "%s"
}
`, firstTag),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "name", firstTag),

					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "tags.#", "1"),
					resource.TestCheckResourceAttr("data.i3dnet_tags.fooTag", "tags.0.tag", firstTag),
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

	apiclient := newOneAPIClient(t, resourceNsFlexmetal)
	for _, name := range names {
		_, createErr := apiclient.CreateTag(context.Background(), name)
		if createErr != nil {
			t.Fatalf("error creating tag: %s", createErr)
		}

		t.Cleanup(func() {
			err := apiclient.DeleteTag(context.Background(), name)
			if err != nil {
				t.Fatalf("error deleting tag: %s", err)
			}
		})
	}

}

package provider

import (
	"context"
	"os"
	"strconv"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLocationDataSource(t *testing.T) {
	apiclient, err := one_api.NewClient(os.Getenv("I3D_API_KEY"), os.Getenv("I3D_BASE_URL"))
	require.NoError(t, err)

	nrOfLocations, err := apiclient.ListLocations(context.Background())
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_locations" "all" {
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Locations from data source should match locations from One API
					resource.TestCheckResourceAttr("data.i3dnet_locations.all", "locations.#", strconv.Itoa(len(nrOfLocations))),
				),
			},
		},
	})
}

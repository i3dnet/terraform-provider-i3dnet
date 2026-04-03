package provider

import (
	"context"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLocationDataSource(t *testing.T) {
	apiclient := newOneAPIClient(t, resourceNsFlexmetal)

	nrOfLocations, err := apiclient.ListLocations(context.Background())
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t, resourceNsFlexmetal) + `
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

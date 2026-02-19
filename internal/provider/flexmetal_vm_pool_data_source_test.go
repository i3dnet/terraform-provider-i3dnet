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

func TestAccFlexmetalVmPoolDataSource(t *testing.T) {
	t.Parallel()

	apiClient, err := one_api.NewClient(os.Getenv("I3D_API_KEY"), os.Getenv("I3D_BASE_URL"))
	require.NoError(t, err)

	allPools, err := apiClient.ListVmPools(context.Background())
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_flexmetal_vm_pools" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.i3dnet_flexmetal_vm_pools.all", "pools.#", strconv.Itoa(len(allPools))),
				),
			},
		},
	})
}

func TestAccFlexmetalVmPoolDataSource_filterByLocation(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(t) + `
data "i3dnet_flexmetal_vm_pools" "by_location" {
  location_id = "EU-NL-01"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.i3dnet_flexmetal_vm_pools.by_location", "pools.#"),
				),
			},
		},
	})
}

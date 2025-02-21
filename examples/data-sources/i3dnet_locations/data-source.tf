# Get all available Bare Metal locations
data "i3dnet_locations" "list" {}

# Convert locations list to a map for iteration
locals {
  locations = { for loc in data.i3dnet_locations.list.locations : loc.id => loc.name }
}

# Create a server for each location
resource "i3dnet_flexmetal_server" "my_talos" {
  for_each = local.locations

  name          = "MyTalosServer-${each.key}"
  location      = each.value
  instance_type = "bm7.std.8"

  os = {
    slug = "talos-omni-190"
    kernel_params = [
      {
        key   = "siderolink.api"
        value = "https://siderolink.api/?jointoken=secret"
      }
    ]
  }
}
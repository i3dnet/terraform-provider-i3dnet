# List the active networks in one fabric whose name contains "db".
data "i3dnet_networks" "db" {
  fabric        = "rotterdam-a"
  state         = "active"
  name_contains = "db"
}

output "db_network_ids" {
  value = [for n in data.i3dnet_networks.db.networks : n.id]
}

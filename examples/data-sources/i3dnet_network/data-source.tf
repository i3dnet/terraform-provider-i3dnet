# Read a network that was created outside Terraform, for example one i3D.net
# set up for your account. Terraform does not manage or destroy it.
data "i3dnet_network" "legacy" {
  id = "3f2b8c1e-7d4a-4e9b-9c2f-1a6d5e8b0c47"
}

output "legacy_vlan_id" {
  value = data.i3dnet_network.legacy.vlan_id
}

output "legacy_routing_domain_id" {
  value = data.i3dnet_network.legacy.routing_domain_id
}

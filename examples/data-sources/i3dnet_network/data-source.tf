# Read a network that was created outside Terraform, for example one i3D.net
# set up for your account. Terraform does not manage or destroy it.
data "i3dnet_network" "legacy" {
  id = "01984f2e-1c3a-7d21-9d10-3542f27adccf"
}

output "legacy_vlan_id" {
  value = data.i3dnet_network.legacy.vlan_id
}

output "legacy_routing_domain_id" {
  value = data.i3dnet_network.legacy.routing_domain_id
}

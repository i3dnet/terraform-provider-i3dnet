# List all nodes within a FlexVM Cloud.
data "i3dnet_flexvm_nodes" "example" {
  cloud_id = "019256ab-1554-73a7-b091-f024b0a724ea"
}

output "node_names" {
  value = [for node in data.i3dnet_flexvm_nodes.example.nodes : node.name]
}

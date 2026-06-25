# Look up a single node within a FlexVM Cloud by its Cloud and node UUID.
data "i3dnet_flexvm_node" "example" {
  cloud_id = "019256ab-1554-73a7-b091-f024b0a724ea"
  id       = "019256ab-1554-73a7-b091-f024b0a724eb"
}

output "node_serial" {
  value = data.i3dnet_flexvm_node.example.serial
}

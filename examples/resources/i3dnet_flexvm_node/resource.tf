data "i3dnet_flexvm_cloud" "my-cloud" {
  id = "019256ab-1554-73a7-b091-f024b0a724ea"
}

# Provision a bare metal Node in an existing FlexVM Cloud. The Node inherits the
# Cloud's instance type and location, so cloud_id is the only required input.
resource "i3dnet_flexvm_node" "my-node" {
  cloud_id = data.i3dnet_flexvm_cloud.my-cloud.id
}

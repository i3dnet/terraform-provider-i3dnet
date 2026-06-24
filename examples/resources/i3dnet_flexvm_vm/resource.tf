# Look up an existing FlexVM Cloud to deploy the VMs into.
data "i3dnet_flexvm_cloud" "my-cloud" {
  id = "019256ab-1554-73a7-b091-f024b0a724ea"
}

resource "i3dnet_flexvm_vm" "my-vm" {
  cloud_id           = data.i3dnet_flexvm_cloud.my-cloud.id
  name               = "test-gaming-vm1"
  description        = "Test Gaming VM 1"
  instance_type_name = "vm.gpu.1rtx4000.15c.248g"
  image_name         = "ubuntu-2404-server-amd64"
  ssh_keys           = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"]
}

# Alternatively, provide cloud-init user-data from a file instead of ssh_keys.
# Exactly one of ssh_keys or user_data_file may be set. The file is read at
# apply time from the current working directory.
resource "i3dnet_flexvm_vm" "my-vm-with-user-data" {
  cloud_id           = data.i3dnet_flexvm_cloud.my-cloud.id
  name               = "test-gaming-vm2"
  description        = "Test Gaming VM 2"
  instance_type_name = "vm.gpu.1rtx4000.15c.248g"
  image_name         = "ubuntu-2404-server-amd64"
  user_data_file     = abspath("${path.module}/cloud-init.yaml")
}

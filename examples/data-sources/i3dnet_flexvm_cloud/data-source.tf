# Look up an existing FlexVM Cloud that is not managed by this configuration.
data "i3dnet_flexvm_cloud" "existing" {
  id = "019256ab-1554-73a7-b091-f024b0a724ea"
}

# Deploy a VM into the existing Cloud. Because the Cloud is referenced through a
# data source (not a resource), `terraform destroy` removes only the VM and
# leaves the Cloud untouched.
resource "i3dnet_flexvm_vm" "my-vm" {
  cloud_id           = data.i3dnet_flexvm_cloud.existing.id
  name               = "development-ubuntu-2404"
  instance_type_name = "vm.4c.8g"
  image_name         = "ubuntu-2404-server-amd64"
  ssh_keys           = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"]
}

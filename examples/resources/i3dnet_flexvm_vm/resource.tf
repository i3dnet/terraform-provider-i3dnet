resource "i3dnet_flexvm_vm" "my-vm" {
  cloud_id           = "019256ab-1554-73a7-b091-f024b0a724ea"
  name               = "test-gaming-vm1"
  description        = "Test Gaming VM 1"
  instance_type_name = "vm.gpu.1rtx4000.15c.248g"
  image_name         = "ubuntu-2404-server-amd64"
  ssh_keys           = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"]
}

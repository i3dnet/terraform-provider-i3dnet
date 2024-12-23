resource "i3d_ssh_key" "my-key" {
  name       = "Key From Terraform"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri andrei.boar@i3d.net"
}

output "my-key" {
  value = i3d_ssh_key.my-key
}
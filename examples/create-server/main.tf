resource "i3d_flexmetal_server" "example" {
  name          = "TerraFlex-Server"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
    kernel_params = [
      {
        key   = "KEY_A"
        value = "VALUE_A"
      }
    ]
  }
  ssh_key             = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHwdgjY0AlmkeLknBpoVmJg/quNSifyBHEK1MREpV4Ri john.doe@i3d.net"]
  post_install_script = "#!/bin/bash\necho \"Hi TerraFlex there!\" > /root/output.txt"
}
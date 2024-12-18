resource "flexmetal_server" "example" {
  name              = "TerraFlex-Server"
  location          = "EU: Rotterdam"
  instance_type     = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
    kernel_params = [
      {
        key   = "KEY_A"
        value = "VALUE_A"
      }
    ]
  }
  ssh_key           = ["603a6c36-04f3-4103-ae2f-798d7dd4f035"]
  post_install_script = "#!/bin/bash\necho \"Hi TerraFlex there!\" > /root/output.txt"
}
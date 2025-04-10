# Create a Flexmetal Server
resource "i3dnet_flexmetal_server" "my-server" {
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
  ssh_key             = ["<YOUR-PUBLIC-SSH-KEY>"]
  post_install_script = "#!/bin/bash\necho \"Hi TerraFlex there!\" > /root/output.txt"
}

# Create a partitioned Flexmetal Server On-Demand
resource "i3dnet_flexmetal_server" "my-partitioned-server" {
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
    partitions = [
      {
        "target" : "/boot",
        "filesystem" : "ext2",
        "size" : 4096
      },
      {
        "target" : "/",
        "filesystem" : "ext4",
        "size" : -1
      },
      {
        "target"     = "/custom",
        "filesystem" = "ext4",
        "size"       = 10240
      }
    ]
  }
  ssh_key             = ["<YOUR-PUBLIC-SSH-KEY>"]
  post_install_script = "#!/bin/bash\necho \"Hi TerraFlex there!\" > /root/output.txt"
}

# Create a Talos OS 1.9.0 server On-Demand with custom timeout for create resource of 30 minutes. Default timeout is 45 minutes.
resource "i3dnet_flexmetal_server" "my-talos" {
  name          = "talosHostName"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "talos-omni-190"
    kernel_params = [
      {
        key   = "siderolink.api"
        value = "https://siderolink.api/?jointoken=secret"
      },
      {
        key   = "talos.customparam"
        value = "123456"
      }
    ]
  }
  timeouts = {
    create = "30m" // duration is specified as Terraform string (e.g., "30m", "1h")
  }
}

# Create a Talos OS 1.9.0 server under committed capacity contract `CONTRACT-123`
resource "i3dnet_flexmetal_server" "my-talos" {
  name          = "MyTalosServer"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "talos-omni-190"
    kernel_params = [
      {
        key   = "siderolink.api"
        value = "https://siderolink.api/?jointoken=secret"
      }
    ]
  }
  contract_id = "CONTRACT-123"
}

# Create a Talos OS 1.9.0 server under committed capacity contract `CONTRACT-123`
# You can add overflow = true to request server On-Demand if committed capacity is full
resource "i3dnet_flexmetal_server" "my-talos" {
  name          = "MyTalosServer"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "talos-omni-190"
    kernel_params = [
      {
        key   = "siderolink.api"
        value = "https://siderolink.api/?jointoken=secret"
      }
    ]
  }
  contract_id = "CONTRACT-123"
  overflow    = true
}
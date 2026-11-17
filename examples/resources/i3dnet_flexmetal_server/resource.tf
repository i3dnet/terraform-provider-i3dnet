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

# Create a Custom iPXE server
resource "i3dnet_flexmetal_server" "my-custom-ipxe" {
  name          = "MyCustomIpxeServer"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug          = "custom-ipxe"
    ipxeScriptUrl = "https://example.org/custom-ipxe.cfg"
  }
}
# Attach a server to networks you created with the Dedicated Networks API (https://docs.i3d.net/connectivity/dedicated-networks-api).
# The networks and prefixes must already exist; refer to them with data sources. Changing `networks` replaces the server.
data "i3dnet_network" "prod" {
  id = "01928f3e-7c4a-7b1e-9d2f-3a5b6c7d8e9f"
}

data "i3dnet_network" "backend" {
  id = "01984f2e-1c3a-7d21-9d10-3542f27adccf"
}

data "i3dnet_network_prefixes" "prod_ipv4" {
  network_id = data.i3dnet_network.prod.id
  family     = 4
}

resource "i3dnet_flexmetal_server" "my-networked-server" {
  name          = "MyNetworkedServer"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
  }
  ssh_key = ["<YOUR-PUBLIC-SSH-KEY>"]
  networks = [
    {
      # Untagged, with an IPv6 address and an IPv4 address from a chosen prefix
      network_id = data.i3dnet_network.prod.id
      ip_allocations = [
        { family = 6 },
        { prefix_id = data.i3dnet_network_prefixes.prod_ipv4.prefixes[0].id },
      ]
    },
    {
      # A tagged 802.1Q VLAN without addresses (layer 2 only)
      network_id     = data.i3dnet_network.backend.id
      tagged         = true
      ip_allocations = []
    },
  ]
}

output "my_networked_server_ipv4" {
  value = i3dnet_flexmetal_server.my-networked-server.networks[0].ip_allocations[1].address
}

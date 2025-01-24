terraform {
  required_providers {
    i3dnet = {
      source = "registry.terraform.io/i3D-net/i3dnet"
    }
  }
}

# Set the variable value in *.tfvars file
# or using -var="i3dnet_api_key=..." CLI option
variable "i3dnet_api_key" {}

# Configure the Provider
provider "i3dnet" {
  api_key = var.i3dnet_api_key
}

# Create your SSH key
resource "i3dnet_ssh_key" "ssh" {
  name       = "MyPublicKey"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIER64HsjCSspx/JMhHELr8LgYwW/PdFrfj7Kr6UM76WS john.doe@email.com"
}


# Create an Ubuntu Server
resource "i3dnet_flexmetal_server" "my-server" {
  name          = "TerraFlex-Server"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
  }
  ssh_key = [i3dnet_ssh_key.ssh.public_key]
}
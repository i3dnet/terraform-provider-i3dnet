terraform {
  required_providers {
    i3dnet = {
      source = "registry.terraform.io/i3dnet/i3dnet"
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

# Look up an existing FlexVM Cloud
data "i3dnet_flexvm_cloud" "my-cloud" {
  id = "019256ab-1554-73a7-b091-f024b0a724ea"
}

# Deploy a FlexVM VM into the existing Cloud, referencing the Cloud from
# its data source
resource "i3dnet_flexvm_vm" "my-vm" {
  cloud_id           = data.i3dnet_flexvm_cloud.my-cloud.id
  name               = "development-ubuntu-2404"
  instance_type_name = "vm.4c.8g"
  image_name         = "ubuntu-2404-server-amd64"
  ssh_keys           = [i3dnet_ssh_key.ssh.public_key]
}

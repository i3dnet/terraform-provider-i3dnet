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

# Create a flexmetal server
resource "i3dnet_flexmetal_server" "my-server" {
  # ...
}
terraform {
  required_providers {
    i3d = {
      source = "registry.terraform.io/i3D-net/i3d"
    }
  }
}

# Set the variable value in *.tfvars file
# or using -var="i3d_api_key=..." CLI option
variable "i3d_api_key" {}

# Configure the i3D Provider
provider "i3d" {
  api_key = var.i3d_api_key
}

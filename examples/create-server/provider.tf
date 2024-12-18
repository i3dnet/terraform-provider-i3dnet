terraform {
  required_providers {
    flexmetal = {
      source  = "terraform.i3d.net/i3d-net/flexmetal"
      version = ">= 0.1"
    }
  }
}

provider "flexmetal" {}
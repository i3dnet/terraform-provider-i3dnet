terraform {
  required_providers {
    i3dnet = {
      source = "registry.terraform.io/i3D-net/i3dnet"
    }
  }
}

# base_url is optional. If you omit it, the i3D.net prod API url will be used
provider "i3dnet" {
  api_key  = "your-api-key"
  base_url = "http://localhost:8081"
}

terraform {
  required_providers {
    i3d = {
      source = "registry.terraform.io/i3D-net/i3d"
    }
  }
}

# base_url is optional. If you omit it, the i3D prod API url will be used
provider "i3d" {
  api_key  = "your-api-key"
  base_url = "http://localhost:8081"
}

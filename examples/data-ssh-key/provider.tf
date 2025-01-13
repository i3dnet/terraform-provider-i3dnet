terraform {
  required_providers {
    i3d = {
      source = "registry.terraform.io/i3D-net/i3d"
    }
  }
}

// Make sure you set `export FLEXMETAL_API_KEY=yourAPIKey` or add the `api_key` attribute
provider "i3d" {
  base_url = "http://localhost:8081"
}
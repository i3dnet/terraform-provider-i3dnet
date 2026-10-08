# List the fabrics where networks can be created.
data "i3dnet_fabrics" "all" {}

# Pick the fabric slugs of one metro area.
locals {
  rotterdam_fabrics = [for f in data.i3dnet_fabrics.all.fabrics : f.slug if f.metro == "Rotterdam"]
}

output "rotterdam_fabrics" {
  value = local.rotterdam_fabrics
}

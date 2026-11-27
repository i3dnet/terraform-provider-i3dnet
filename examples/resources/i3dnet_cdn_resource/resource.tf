provider "i3dnet" {
  cdn_api_key = var.cdn_api_key
}

# The hostname www.example.com, served by the CDN from one origin.
resource "i3dnet_cdn_resource" "www" {
  name            = "www"
  group_code      = "my-group"
  initiative_code = "my-initiative"

  hostnames = [{
    fqdn   = "www.example.com"
    ssl_cn = "wildcard.example.com"
  }]

  origins = [{
    fqdn        = "origin.example.com"
    healthcheck = { path = "/healthz" }
  }]

  flags = {
    auth_pass_through = true
  }
}

# Point the hostname at the CDN with a CNAME record, in whichever DNS
# provider manages the zone.
output "cname_target" {
  value = i3dnet_cdn_resource.www.cname_target
}

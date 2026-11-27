data "i3dnet_cdn_predefined_allowlists" "all" {}

output "allowlist_keys" {
  value = [for a in data.i3dnet_cdn_predefined_allowlists.all.allowlists : a.key]
}

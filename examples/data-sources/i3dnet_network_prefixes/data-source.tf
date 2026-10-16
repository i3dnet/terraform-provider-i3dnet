# List the IPv4 prefixes of one network, for example to build a firewall allowlist.
data "i3dnet_network_prefixes" "web_ipv4" {
  network_id = "01928f3e-7c4a-7b1e-9d2f-3a5b6c7d8e9f"
  family     = 4
}

output "web_ipv4_cidrs" {
  value = [for p in data.i3dnet_network_prefixes.web_ipv4.prefixes : p.cidr if p.cidr != null]
}

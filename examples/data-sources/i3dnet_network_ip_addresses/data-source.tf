# List the addresses of your servers on one network, for example to create DNS A records.
data "i3dnet_network_ip_addresses" "web" {
  network_id = "01928f3e-7c4a-7b1e-9d2f-3a5b6c7d8e9f"
}

output "web_server_ipv4_addresses" {
  value = [for ip in data.i3dnet_network_ip_addresses.web.ip_addresses : ip.address if ip.role == null && ip.family == 4]
}

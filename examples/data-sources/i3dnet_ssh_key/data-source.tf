# Get resource by name
data "i3dnet_ssh_key" "example" {
  name = "Demo"
}

output "my-key" {
  value = data.i3dnet_ssh_key.example
}

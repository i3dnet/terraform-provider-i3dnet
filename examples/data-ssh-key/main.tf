data "i3d_ssh_key" "example" {
  name = "Demo"
}

output "my-key" {
  value = data.i3d_ssh_key.example
}

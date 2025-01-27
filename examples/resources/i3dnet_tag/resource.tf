# Create a new Tag
resource "i3dnet_tag" "foo" {
  name = "foo"
}

# Create a new Server with the foo tag
resource "i3dnet_flexmetal_server" "my-server" {
  name          = "TerraFlex-Server"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
  }
  ssh_key = ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIER64HsjCSspx/JMhHELr8LgYwW/PdFrfj7Kr6UM76WS andrei.boar@i3d.net"]
  tags    = [i3dnet_tag.foo.id]
}
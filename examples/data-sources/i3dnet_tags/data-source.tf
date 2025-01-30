# Get all your i3D.net tags
data "i3dnet_tags" "allTags" {

}

# Get tag with name `foo`
data "i3dnet_tags" "fooTag" {
  name = "foo"
}
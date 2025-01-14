# Create a new SSH key
resource "i3d_ssh_key" "my-key" {
  name       = "Key From Terraform"
  public_key = "<YOUR-PUBLIC-SSH-KEY>"
}
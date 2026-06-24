resource "i3dnet_flexvm_cloud" "my-cloud" {
  name          = "my-private-cloud"
  description   = "Cloud for the odyssey project"
  site          = "frmtl1"
  instance_type = "bm9.hmm.gpu.4rtx4000.64"
}

# Implementation Notes

## FlexMetal VM Extensions - Known Gaps

All items below have been resolved. This file is kept for reference.

| Item | Severity | Resolution |
|---|---|---|
| `vm_capacity.go` not created | Low | Created with `GetVmCapacity` and unit test |
| `os` missing `RequiresReplace` | High | Fixed in `resource_flexmetal_vm/model.go` |
| `name` wrong `RequiresReplace` | Medium | Removed from `resource_flexmetal_vm/model.go` |
| `metadata` spurious `UseStateForUnknown` | Low | Removed from `resource_flexmetal_vm_pool/model.go` |
| `vmPoolRespToModel` null metadata bug | Medium | Removed the `else if` branch that overwrote null with `{}` |
| Pool/plan/image data source names not plural | Low | Renamed to `i3dnet_flexmetal_vm_plans`, `i3dnet_flexmetal_vm_pools`, `i3dnet_flexmetal_vm_images` |
| Hard-coded acceptance test values | High | Replaced with env vars: `I3D_VM_LOCATION_ID`, `I3D_VM_CONTRACT_ID`, `I3D_VM_INSTANCE_TYPE`, `I3D_VM_VLAN_ID`, `I3D_VM_SUBNET_CIDR`, `I3D_VM_GATEWAY`, `I3D_VM_RANGE_START`, `I3D_VM_RANGE_END`, `I3D_VM_PLAN`, `I3D_VM_IMAGE_ID`, `I3D_VM_GPU_PLAN` (optional, skips if unset) |

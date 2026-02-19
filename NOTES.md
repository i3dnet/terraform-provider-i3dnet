# Implementation Notes

## FlexMetal VM Extensions - Known Gaps

The following issues were identified after the initial implementation of the FlexMetal VM resources (branch `flexvm`).

### Not implemented

**`internal/one_api/vm_capacity.go`**
The plan included a `VmCapacity` struct and `GetVmCapacity` method (path `vm/pools/{id}/capacity`). No file was created and no test covers it. Low priority since no resource consumes it yet, but it is part of the original spec.

---

### Schema bugs

**`os` block missing `RequiresReplace` on `i3dnet_flexmetal_vm`**
File: `internal/provider/resource_flexmetal_vm/model.go:66`
The `os` `SingleNestedAttribute` has no `PlanModifiers`. Changing `os.image_id` will not trigger a replace -- Terraform will call `Update` instead, which only sends `tags` and silently ignores the OS change.
Fix: add `objectplanmodifier.RequiresReplace()` (from `resource/schema/objectplanmodifier`) to the `os` attribute.

**`name` incorrectly marked `RequiresReplace` on `i3dnet_flexmetal_vm`**
File: `internal/provider/resource_flexmetal_vm/model.go:48`
`name` has `stringplanmodifier.RequiresReplace()` but the spec lists `name` as Required only, without ForceNew. If the API allows renaming a VM this forces unnecessary destroy/recreate.

**`metadata` has `UseStateForUnknown()` but is not `Computed`**
File: `internal/provider/resource_flexmetal_vm_pool/model.go:116`
`metadata` is `Optional` only, not `Computed`, so `mapplanmodifier.UseStateForUnknown()` has no effect. Remove it.

---

### Potential runtime bug

**`vmPoolRespToModel` overwrites null `metadata` with empty map**
File: `internal/provider/flexmetal_vm_pool_resource.go:183`
When the API returns an empty metadata map and the user did not set `metadata` in config, the code writes `{}` to state. The plan value would be `null`, causing a "planned value does not match actual value" error on the next `terraform plan`. The `else if` branch should be removed; leave `data.Metadata` unchanged when the API returns nothing and the field was not configured.

---

### Data source name discrepancy

The VM pool data source was registered as `i3dnet_flexmetal_vm_pools` (plural) but the plan specified `i3dnet_flexmetal_vm_pool` (singular). The doc and tests are internally consistent with the plural name, but it differs from the plan spec and from the resource name `i3dnet_flexmetal_vm_pool`. Needs a deliberate decision before the API is public.

---

### Acceptance test hard-coded values

The acceptance tests use values that do not exist in any real environment:
- `location_id = "EU-NL-01"`
- `contract_id = "contract-123"`
- `plan = "vm-standard-4"`
- `image_id = "ubuntu-2404-lts"`
- slug `"vm-standard-4"` in `TestAccFlexmetalVmPlanDataSource_filterBySlug`

These must be replaced with values sourced from environment variables (same pattern as `I3D_API_KEY`) before the tests can pass against a live API.

---

### Summary

| Item | Severity | Location |
|---|---|---|
| `vm_capacity.go` not created | Low | `internal/one_api/` |
| `os` missing `RequiresReplace` | High | `resource_flexmetal_vm/model.go:66` |
| `name` wrong `RequiresReplace` | Medium | `resource_flexmetal_vm/model.go:48` |
| `metadata` spurious `UseStateForUnknown` | Low | `resource_flexmetal_vm_pool/model.go:116` |
| `vmPoolRespToModel` null metadata bug | Medium | `flexmetal_vm_pool_resource.go:183` |
| Pool data source name plural vs singular | Low | `flexmetal_vm_pool_data_source.go:33` |
| Hard-coded acceptance test values | High | all `*_resource_test.go` and `*_data_source_test.go` files |

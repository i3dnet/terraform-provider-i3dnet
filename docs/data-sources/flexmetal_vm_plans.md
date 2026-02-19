---
page_title: "i3dnet_flexmetal_vm_plans Data Source - i3dnet"
subcategory: ""
description: |-
  Lists available FlexMetal VM plans, optionally filtered by slug or GPU count.
---

# i3dnet_flexmetal_vm_plans (Data Source)

Lists available FlexMetal VM plans, optionally filtered by slug or GPU count.

## Example Usage

```terraform
# List all plans
data "i3dnet_flexmetal_vm_plans" "all" {}

# Find a specific plan by slug
data "i3dnet_flexmetal_vm_plans" "standard" {
  slug = "vm-standard-4"
}

# Find all GPU plans
data "i3dnet_flexmetal_vm_plans" "gpu" {
  gpu_count = 1
}

output "gpu_plan_slugs" {
  value = [for p in data.i3dnet_flexmetal_vm_plans.gpu.plans : p.slug]
}
```

## Schema

### Optional

- `gpu_count` (Number) Filter plans by GPU count.
- `slug` (String) Filter plans by slug.

### Read-Only

- `plans` (Attributes List) List of VM plans. (see [below for nested schema](#nestedatt--plans))

<a id="nestedatt--plans"></a>
### Nested Schema for `plans`

Read-Only:

- `cpu` (Number) Number of vCPUs.
- `gpu_count` (Number) Number of GPUs.
- `gpu_model` (String) GPU model name.
- `memory_gb` (Number) Memory in gigabytes.
- `name` (String) Human-readable plan name.
- `slug` (String) Plan slug identifier.

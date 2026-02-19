---
page_title: "i3dnet_flexmetal_vm_pools Data Source - i3dnet"
subcategory: ""
description: |-
  Lists FlexMetal VM pools, optionally filtered by name, location, or status.
---

# i3dnet_flexmetal_vm_pools (Data Source)

Lists FlexMetal VM pools, optionally filtered by name, location, or status.

## Example Usage

```terraform
# List all pools
data "i3dnet_flexmetal_vm_pools" "all" {}

# Filter by location
data "i3dnet_flexmetal_vm_pools" "nl" {
  location_id = "EU-NL-01"
}

# Filter by status
data "i3dnet_flexmetal_vm_pools" "active" {
  status = "active"
}

output "pool_ids" {
  value = [for p in data.i3dnet_flexmetal_vm_pools.all.pools : p.id]
}
```

## Schema

### Optional

- `location_id` (String) Filter pools by location ID.
- `name` (String) Filter pools by name.
- `status` (String) Filter pools by status.

### Read-Only

- `pools` (Attributes List) List of VM pools. (see [below for nested schema](#nestedatt--pools))

<a id="nestedatt--pools"></a>
### Nested Schema for `pools`

Read-Only:

- `contract_id` (String) Contract ID.
- `id` (String) Pool identifier.
- `instance_type` (String) Instance type.
- `location_id` (String) Location ID.
- `name` (String) Pool name.
- `status` (String) Pool status.
- `subnet` (Attributes List) Subnet configuration. (see [below for nested schema](#nestedatt--pools--subnet))
- `type` (String) Pool type.
- `vlan_id` (Number) VLAN ID.

<a id="nestedatt--pools--subnet"></a>
### Nested Schema for `pools.subnet`

Read-Only:

- `cidr` (String) CIDR block.
- `gateway` (String) Gateway IP.
- `range_end` (String) End of IP range.
- `range_start` (String) Start of IP range.

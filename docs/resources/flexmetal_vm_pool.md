---
page_title: "i3dnet_flexmetal_vm_pool Resource - i3dnet"
subcategory: ""
description: |-
  FlexMetal VM Pool resource. A pool groups VM instances and defines the network configuration.
---

# i3dnet_flexmetal_vm_pool (Resource)

FlexMetal VM Pool resource. A pool groups VM instances and defines the network configuration.

## Example Usage

```terraform
resource "i3dnet_flexmetal_vm_pool" "my-pool" {
  name          = "my-vm-pool"
  location_id   = "EU-NL-01"
  contract_id   = "CONTRACT-123"
  type          = "on_demand"
  instance_type = "bm9.hmm.gpu.4rtx4000.64"
  vlan_id       = 100
  subnet = [
    {
      cidr        = "10.0.0.0/24"
      gateway     = "10.0.0.1"
      range_start = "10.0.0.10"
      range_end   = "10.0.0.254"
    }
  ]
  metadata = {
    env  = "production"
    team = "platform"
  }
}
```

## Schema

### Required

- `contract_id` (String) Contract ID associated with this pool.
- `instance_type` (String) Bare-metal instance type backing this pool.
- `location_id` (String) Location where the pool is deployed.
- `name` (String) Name of the VM pool.
- `subnet` (Attributes List) Subnet configuration for the pool. (see [below for nested schema](#nestedatt--subnet))
- `type` (String) Pool type (e.g. on_demand).
- `vlan_id` (Number) VLAN ID for the pool network.

### Optional

- `metadata` (Map of String) Arbitrary key-value metadata for the pool.

### Read-Only

- `id` (String) Unique identifier of the VM pool.
- `status` (String) Current status of the pool.

<a id="nestedatt--subnet"></a>
### Nested Schema for `subnet`

Required:

- `cidr` (String) CIDR block of the subnet.
- `gateway` (String) Gateway IP address.
- `range_end` (String) End of the IP range.
- `range_start` (String) Start of the IP range.

## Import

Import is supported using the following syntax:

```shell
terraform import i3dnet_flexmetal_vm_pool.my-pool <pool-id>
```

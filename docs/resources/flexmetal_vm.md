---
page_title: "i3dnet_flexmetal_vm Resource - i3dnet"
subcategory: ""
description: |-
  FlexMetal VM instance resource.
---

# i3dnet_flexmetal_vm (Resource)

FlexMetal VM instance resource. Provisions a virtual machine within a FlexMetal VM pool.

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
}

resource "i3dnet_flexmetal_vm" "my-vm" {
  name    = "my-vm"
  pool_id = i3dnet_flexmetal_vm_pool.my-pool.id
  plan    = "vm-standard-4"
  os = {
    image_id = "ubuntu-2404-lts"
  }
  tags = ["env:production"]
}

# GPU VM with custom timeout
resource "i3dnet_flexmetal_vm" "gpu-vm" {
  name    = "gpu-vm"
  pool_id = i3dnet_flexmetal_vm_pool.my-pool.id
  plan    = "vm-gpu-1rtx4000"
  os = {
    image_id = "ubuntu-2404-lts"
  }
  user_data = "#!/bin/bash\napt-get install -y nvidia-driver-535"
  timeouts = {
    create = "20m"
    delete = "15m"
  }
}
```

## Schema

### Required

- `name` (String) Name of the VM instance.
- `os` (Attributes) Operating system configuration. (see [below for nested schema](#nestedatt--os))
- `plan` (String) VM plan (size/type).
- `pool_id` (String) ID of the VM pool this instance belongs to.

### Optional

- `tags` (List of String) List of tags associated with the VM.
- `timeouts` (Attributes) (see [below for nested schema](#nestedatt--timeouts))
- `user_data` (String) Cloud-init user data.

### Read-Only

- `gateway` (String) Default gateway for the VM.
- `id` (String) Unique identifier of the VM instance.
- `ip_address` (String) IPv4 address assigned to the VM.
- `ip_address_v6` (String) IPv6 address assigned to the VM.
- `netmask` (String) Netmask for the VM network.
- `provisioned_at` (String) Timestamp when the VM was provisioned.
- `status` (String) Current status of the VM instance.
- `vlan_id` (Number) VLAN ID the VM is connected to.

<a id="nestedatt--os"></a>
### Nested Schema for `os`

Required:

- `image_id` (String) Image identifier to use for the VM.

<a id="nestedatt--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Default: 15m.
- `delete` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Default: 10m.

## Import

Import is supported using the following syntax:

```shell
terraform import i3dnet_flexmetal_vm.my-vm <vm-id>
```

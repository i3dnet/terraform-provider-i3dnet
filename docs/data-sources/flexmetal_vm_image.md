---
page_title: "i3dnet_flexmetal_vm_image Data Source - i3dnet"
subcategory: ""
description: |-
  Lists available FlexMetal VM images, optionally filtered by slug or OS family.
---

# i3dnet_flexmetal_vm_image (Data Source)

Lists available FlexMetal VM images, optionally filtered by slug or OS family.

## Example Usage

```terraform
# List all images
data "i3dnet_flexmetal_vm_image" "all" {}

# List Linux images
data "i3dnet_flexmetal_vm_image" "linux" {
  os_family = "linux"
}

# Find a specific image by slug
data "i3dnet_flexmetal_vm_image" "ubuntu" {
  slug = "ubuntu-2404-lts"
}

output "linux_image_slugs" {
  value = [for img in data.i3dnet_flexmetal_vm_image.linux.images : img.slug]
}
```

## Schema

### Optional

- `os_family` (String) Filter images by OS family (e.g. linux, windows).
- `slug` (String) Filter images by slug.

### Read-Only

- `images` (Attributes List) List of VM images. (see [below for nested schema](#nestedatt--images))

<a id="nestedatt--images"></a>
### Nested Schema for `images`

Read-Only:

- `id` (String) Image identifier.
- `name` (String) Image name.
- `os_family` (String) OS family.
- `slug` (String) Image slug.
- `version` (String) Image version.

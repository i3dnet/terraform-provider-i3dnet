# flexMetal Terraform Provider

The `flexMetal Terraform Provider` allows you to manage FlexMetal resources using Terraform.

## Getting Started

To get started with the `flexMetal Terraform Provider`, follow the steps below.

### Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) installed on your machine.
- FlexMetal API key.
- tfplugingen-openapi (
  `go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@latest`)
- tfplugingen-framework (
  `go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@latest`)

### Installation

1. Clone the repository:

    ```sh
    git clone git@github.com:i3D-net/terraform-provider-i3d.git
    cd flexmetal-terraform-provider
    ```

2. Build the provider:

    ```sh
    go build -o terraform-provider-flexmetal
    ```

3. FOR DEV: Locally install provider and validate that it works:

Terraform installs providers and verifies their versions and checksums when you run `terraform init`.
Terraform will download your providers from either the provider registry or a local registry. However, while building
your provider you will want to test Terraform configuration against a local development build of the provider. The
development build will not have an associated version number or an official set of checksums listed in a provider
registry.

Terraform allows you to use local provider builds by setting a `dev_overrides` block in a configuration file called
`.terraformrc`. This block overrides all other configured installation methods.

Terraform searches for the `.terraformrc` file in your home directory and applies any configuration settings you set.

Windows users follow instructions
from [here](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider#prepare-terraform-for-local-provider-install).

Instructions for MAC/Linux:

First, find the `GOBIN` path where Go installs your binaries. Your path may vary depending on how your Go environment
variables are configured.

```sh
$ go env GOBIN
/Users/<Username>/go/bin
```

Create a new file called `.terraformrc` in your home directory (~), then add the `dev_overrides` block below. Change
the <PATH> to the value returned from the `go env GOBIN` command above. If the `GOBIN` go environment variable is not
set,
use the default path, `/Users/<Username>/go/bin`.

```terraform
provider_installation {

  dev_overrides {
    "terraform.i3d.net/i3d-net/flexmetal" = "/Users/<Username>/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Verify it works:

```sh
cd examples/provider-install-verification/
terraform plan
```

You should get the similar output to validate override is in effect:

```sh
│ Warning: Provider development overrides are in effect
│ 
│ The following provider development overrides are set in the CLI configuration:
│  - terraform.i3d.net/i3d-net/flexmetal in /home/andrei/go/bin
│ 
│ The behavior may therefore not match any released version of the provider and applying changes may cause the state to become incompatible with published releases.
╵

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

```

4. FOR REAL (probably): Move the provider binary to your Terraform plugins directory:

    ```sh
    mkdir -p ~/.terraform.d/plugins
    mv terraform-provider-flexmetal ~/.terraform.d/plugins/
    ```

### Generate from open API

A part of this provider is generated using the tf plugin openAPI-generator (see
this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)) and the TF framework
Generator (see this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator)).

How to generate the resources & the config:

1. download the OpenAPI Spec (https://www.i3d.net/docs/api/v3/getjson) and overwrite the
   `./generator_data/openAPISpec.json`.
2. Create/modify the `./generator_data/GeneratorConfig.yaml`
3. Generate the `./generator_data/provider_code_spec.json` file:

  ```bash
   tfplugingen-openapi generate \
    --config ./generator_data/GeneratorConfig.yaml \
    --output ./generator_data/provider_code_spec.json \
    ./generator_data/openAPISpec.json
   ```

4. Generate the skeleton of the provider using the previously generated files:
    ```bash
    tfplugingen-framework generate all \
    --input ./generator_data/provider_code_spec.json \
    --output internal/provider
    ```

You can now customize `internal/provider/server_resource.go` to add the good logic.

### Configuration (for dev)

Create a Terraform project directory, e.g. `~/fm_tf`

Create a `main.tf` file in your Terraform directory and add the following:

```hcl
resource "flexmetal_server" "example" {
  name          = "example-server"
  location      = "EU: Rotterdam"
  instance_type = "bm7.std.8"
  os = {
    slug = "ubuntu-2404-lts"
    kernel_params = [
      {
        key   = "KEY_A"
        value = "VALUE_A"
      }
    ]
  }
  ssh_key = ["ssh-rsa AAA..."]
  post_install_script = "#!/bin/bash\necho \"Hi there!\" > /root/output.txt"
}
```

Fix the ssh_key entry

Create a `provider.tf` file in your Terraform directory and add the following:

```hcl
terraform {
  required_providers {
    flexmetal = {
      source  = "terraform.i3d.net/i3d-net/flexmetal"
      version = ">= 0.1"
    }
  }
}

provider "flexmetal" {}
```

Create an `outputs.tf` file in your Terraform directory and add the following:

```hcl
output "inventory" {
  sensitive = false
  value     = [
    for s in flexmetal_server.example : {
      "name" : s.name,
      "uuid" : s.uuid,
      "ip" : s.ip_addresses[0].ip_address,
    }
  ]
}
```

### Usage

Initialize flexmetal API key:

```bash
export FLEXMETAL_API_KEY=<your API key>
```

Initialize Terraform:

```bash
terraform init
```

Apply the configuration:

```bash
terraform apply
```

This will order 1 FlexMetal server.

This is the minimal configuration to have something working. You can of course expand a lot on this. Request multiple
servers in 1 go, or have multiple configurations, or add variables instead of hardcoding the configuration, etc.

To release the server:

```bash
terraform destroy
```

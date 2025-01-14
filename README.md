# i3D Terraform Provider

The `i3D Terraform Provider` allows you to manage i3D resources using Terraform.

## Getting Started

To get started with the `i3D Terraform Provider`, follow the steps below.

### Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) installed on your machine.
- FlexMetal API key [https://one.i3d.net/Account/API-Keys](https://one.i3d.net/Account/API-Keys)
- tfplugingen-openapi (
  `go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@latest`)
- tfplugingen-framework (
  `go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@latest`)

### Installation

1. Clone the repository:

    ```sh
    git clone git@github.com:i3D-net/terraform-provider-i3d.git
    cd terraform-provider-i3d
    ```

2. Build the provider:

    ```sh
    go build -o terraform-provider-i3d
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
the `<PATH>` to the value returned from the `go env GOBIN` command above. If the `GOBIN` go environment variable is not
set, use the default path, `/Users/<Username>/go/bin`.

```terraform
provider_installation {

  dev_overrides {
    "registry.terraform.io/i3D-net/i3d" = "/Users/<Username>/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

If you have not yet compiled the plugin, you can do that now. From the project root, do:

```sh
go install .
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
│  - i3d-net/i3d in /Users/<Username>/go/bin
│ 
│ The behavior may therefore not match any released version of the provider and applying changes may cause the state to become incompatible with published releases.
╵

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

```

Now you are able to modify and interact with your local build of the provider. Just make sure to run `go install .`
anytime you apply some changes to your provider code.

For more examples on usages see [examples](./examples) directory.

### Generate from OpenAPI specification

A part of this provider is generated using the tf plugin openAPI-generator (see
this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)) and the TF framework
Generator (see this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator)).

How to generate the resources & the config:

1. Download the OpenAPI Spec (https://www.i3d.net/docs/api/v3/getjson) and overwrite the
   `./generator_data/openAPISpec.json`.
2. Create/modify the `./generator_data/GeneratorConfig.yaml`
3. Generate the `./generator_data/provider_code_spec.json` file:

  ```bash
   tfplugingen-openapi generate \
    --config ./generator_data/GeneratorConfig.yaml \
    --output ./generator_data/provider_code_spec.json \
    ./generator_data/openAPISpec.json
   ```

4. Generate the skeleton of the provider using the Provider Code Specification from `provider_code_spec.json`:
    ```bash
    tfplugingen-framework generate all \
    --input ./generator_data/provider_code_spec.json \
    --output internal/provider
    ```

You can now customize `internal/provider/server_resource.go` to add the good logic.

Documentation
for [generate command](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator#generate-command).

### Configuration (for dev)

Create a Terraform project directory, e.g. `~/fm_tf`

Create a `main.tf` file in your Terraform directory and add the following:

```hcl
resource "i3d_flexmetal_server" "example" {
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
    i3d = {
      source = "registry.terraform.io/i3D-net/i3d"
    }
  }
}

provider "i3d" {}
```

Create an `outputs.tf` file in your Terraform directory and add the following:

```hcl
output "inventory" {
  sensitive = false
  value     = i3d_flexmetal_server.example.name
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

## Running acceptance tests

Rebuild the provider before running acceptance tests.

Acceptance tests run against a real working environment. To run them you must have these environment variables set:
`I3D_API_KEY`, `I3D_BASE_URL` and `TF_ACC`.

You can omit `I3D_BASE_URL` in which case the default `https://api.i3d.net` production URL is used.

Then, you can run them using this command:

```shell
I3D_API_KEY=yourapiKey I3D_BASE_URL=targetBaseURL TF_ACC=1 go test -count=1 -v ./...
```

You can also run tests using [Task](https://taskfile.dev/). Make sure you have all your variables set in the `.env`
file.

It is preferable to run one acceptance test at a time. In order to run a specific acceptance test, use the `TESTARGS`
environment variable. For example, the following command will run `TestAccSSHKeyResource` acceptance test only:

```shell
TESTARGS='-run=TestAccSSHKeyResource' task testacc
```

Run all acceptance tests:

``shell
task testacc
``

## Generating documentation

Documentation for provider is generated inside `docs` directory
using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) tool. The generation uses schema
descriptions
and [conventionally placed files](https://github.com/hashicorp/terraform-plugin-docs?tab=readme-ov-file#conventional-paths)
to produce provider documentation that is compatible with the Terraform
Registry.

You can run `task docs` command to generate documentation everytime you update the schema of your provider and
resources.

```shell
task docs
```

`task docs` will format your examples and run `tfplugindocs generate` command. For usage
see [tools.go](./tools/tools.go).

Use [Doc Preview Tool](https://registry.terraform.io/tools/doc-preview) to preview how provider docs will render on the
Terraform Registry.

Do not manually edit files inside `docs` directory because they will be overwritten on re-generation.

If you want to extend the templates of `tfplugindocs` you can extend them inside [./templates](./templates) directory.
For more information check the Readme inside that directory. You can view the default templates in the
`tfplugindocs` [source code](https://github.com/hashicorp/terraform-plugin-docs/blob/a9c737d5accfd312e40b5d54fe2241405606697c/internal/provider/template.go#L272).

[Provider documentation](https://developer.hashicorp.com/terraform/registry/providers/docs)
[Schema and configuration for provider documentation](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-documentation-generation#add-configuration-examples)

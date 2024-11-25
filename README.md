# flexMetal Terraform Provider

The `flexMetal Terraform Provider` allows you to manage FlexMetal resources using Terraform.

## Getting Started

To get started with the `flexMetal Terraform Provider`, follow the steps below.

### Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) installed on your machine.
- FlexMetal API key.
- tfplugingen-openapi (`go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@latest`)
- tfplugingen-framework (`go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@latest`)

### Installation

1. Clone the repository:

    ```sh
    git clone https://github.com/yourusername/flexmetal-terraform-provider.git
    cd flexmetal-terraform-provider
    ```

2. Build the provider:

    ```sh
    go build -o terraform-provider-flexmetal
    ```

3. Move the provider binary to your Terraform plugins directory:

    ```sh
    mkdir -p ~/.terraform.d/plugins
    mv terraform-provider-flexmetal ~/.terraform.d/plugins/
    ```

### Generate from open API

A part of this provider is generated using the tf plugin openAPI-generator (see this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)) and the TF framework Generator (see this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator)).

How to generate the resources & the config:

1. download the OpenAPI Spec (https://www.i3d.net/docs/api/v3/getjson) and overwrite  the `./generator_data/openAPISpec.json`.
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

### Configuration

Create a `main.tf` file in your Terraform configuration directory and add the following:

```hcl
provider "flexmetal" {}

resource "flexmetal_server" "example" {
  name              = "example-server"
  location          = "EU: Rotterdam"
  instance_type     = "bm7.std.8"
  os = {
    slug = "ubuntu-2204-lts"
    kernel_params = [
      {
        key   = "KEY_A"
        value = "VALUE_A"
      }
    ]
  }
  ssh_key           = ["ssh-rsa AAA..."]
  post_install_script = "#!/bin/bash\necho \"Hi there!\" > /root/output.txt"
}

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

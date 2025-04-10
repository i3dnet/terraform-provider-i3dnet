# Contributing

If you wish to work on the provider, you'll first need:

- [Terraform](https://www.terraform.io/downloads.html) installed on your machine.
- i3D.net API key [https://one.i3d.net/Account/API-Keys](https://one.i3d.net/Account/API-Keys)
- tfplugingen-openapi (
  `go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@latest`)
- tfplugingen-framework (
  `go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@latest`)

### Installation

1. Clone the repository:

    ```sh
    git clone https://github.com/i3D-net/terraform-provider-i3dnet
    cd terraform-provider-i3dnet
    ```

2. Locally install provider and validate that it works:

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

### Instructions for MAC/Linux:

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
    "registry.terraform.io/i3D-net/i3dnet" = "/Users/<Username>/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

If you have not yet compiled the plugin, you can do that now. From the project root, do:

```sh
go install
```

You should get the similar output to validate override is in effect:

```sh
│ Warning: Provider development overrides are in effect
│ 
│ The following provider development overrides are set in the CLI configuration:
│  - i3d-net/i3dnet in /Users/<Username>/go/bin
│ 
│ The behavior may therefore not match any released version of the provider and applying changes may cause the state to become incompatible with published releases.
╵

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

```

Now you are able to modify and interact with your local build of the provider. Just make sure to run `go install`
anytime you apply some changes to your provider code.

### Generate from OpenAPI specification

A part of this provider is generated using the tf plugin openAPI-generator (see
this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/openapi-generator)) and the TF framework
Generator (see this [doc](https://developer.hashicorp.com/terraform/plugin/code-generation/framework-generator)).

How to generate the resources & the config:

1. Download the OpenAPI Spec (https://www.i3d.net/docs/api/v3/getjson) and overwrite the
   `./generator_data/openAPISpec.json`. Make sure to replace `\/` with `/`.
2. In the `openApiSpec.json`, under  `paths > /v3/flexMetal/servers > post > responses > 200` change to:

```json
{
  "description": "OK",
  "content": {
    "application/json": {
      "schema": {
        "$ref": "#/components/schemas/FlexMetalServer"
      }
    }
  }
}
```

This change is required because the generation tool will not properly create the `i3dnet_flexmetal_server` resource due
to the fact that we don't have a RESTFUL API. To be more specific, following fields are lost during mapping process:
created_at, delivered_at, ip_addresses, released_at, status, status_message.
Checkout [how resources are mapped](https://github.com/hashicorp/terraform-plugin-codegen-openapi/blob/main/DESIGN.md)
if interested.

3. Create/modify the `./generator_data/GeneratorConfig.yaml`
4. Generate the `./generator_data/provider_code_spec.json` file:

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

## Running against local environment
To use local environment you can pass `base_url` to the provider configuration
```
# Configure the Provider
provider "i3dnet" {
    api_key = "<YOUR-API-KEY>"
    base_url = "http://localhost:8081"
}
```

## Debugging and Logging

If you'd like to see more detailed logs for debugging, you can set the `TF_LOG` environment variable to `DEBUG` or
`TRACE`.

``` console
export TF_LOG=DEBUG
export TF_LOG=TRACE
```

After setting the log level, you can run `terraform plan` or `terraform apply` again to see more detailed output. Find
out more [here](https://developer.hashicorp.com/terraform/internals/debugging).

## Running acceptance tests

Rebuild the provider before running acceptance tests.

Acceptance tests run against a real working environment. To run them you must have these environment variables set:
`I3D_API_KEY`, `I3D_BASE_URL` and `TF_ACC`. For a full list of environment variable used by our provider check
`.env.dist`.

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

Run only Flexmetal Server Resource test:

```shell
TESTARGS='-run=TestAccFlexmetalServerResource' task testacc
```

Note: Creating servers is a long-running operation.

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

## Updating documentation on Terraform Registry

This `Release` GitHub Action will release new versions of the provider whenever you tag a commit on the main branch.

Terraform provider versions must follow the [Semantic Versioning](https://semver.org/) standard (vMAJOR.MINOR.PATCH).

//go:build generate

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)

// Generate the Network API client from the vendored public contract. It runs
// before the documentation, which compiles the provider.
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=../internal/network_api/oapi-codegen.yaml ../internal/network_api/openapi.yaml

// Format Terraform code for use in documentation.
// If you do not have Terraform installed, you can remove the formatting command, but it is suggested
// to ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ../examples/

// Generate documentation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name i3dnet

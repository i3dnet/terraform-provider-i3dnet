package provider

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-i3d/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"i3d": providerserver.NewProtocol6WithError(New()()),
	}
)

// providerConfig is a shared configuration to combine with the actual
// test configuration so the i3d One API client is properly configured.
func providerConfig(t *testing.T) string {
	const (
		providerConfigTemplate = `
provider "i3d" {
  api_key = "%s"
  base_url = "%s"
}
`
	)
	apiKey := os.Getenv("I3D_API_KEY")
	if apiKey == "" {
		t.Fatalf("I3D_API_KEY key is required to run acceptance tests.")
	}

	baseURL := os.Getenv("I3D_BASE_URL")
	if baseURL == "" {
		baseURL = one_api.DefaultBaseURL
	}

	return fmt.Sprintf(providerConfigTemplate, apiKey, baseURL)
}

package provider

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"
)

const (
	resourceNsFlexmetal = "flexmetal"
	resourceNsFlexvm    = "flexvm"
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"i3dnet": providerserver.NewProtocol6WithError(New()),
	}
)

// providerConfig is a shared configuration to combine with the actual
// test configuration so the i3D.net One API client is properly configured.
func providerConfig(t *testing.T, resourceNs string) string {
	t.Helper()

	const (
		providerConfigTemplate = `
provider "i3dnet" {
  api_key = "%s"
  base_url = "%s"
}
`
	)

	apiKey, baseURL := oneAPIVars(t, resourceNs)

	return fmt.Sprintf(providerConfigTemplate, apiKey, baseURL)
}

func newOneAPIClient(t *testing.T, resourceNs string) *one_api.Client {
	t.Helper()

	apiKey, baseURL := oneAPIVars(t, resourceNs)

	client, err := one_api.NewClient(apiKey, baseURL)
	require.NoError(t, err)

	return client
}

func oneAPIVars(t *testing.T, resourceNs string) (apiKey, baseURL string) {
	t.Helper()

	if resourceNs != resourceNsFlexmetal && resourceNs != resourceNsFlexvm {
		t.Fatalf(`resourceNs should be either %q or %q, got %q`,
			resourceNsFlexmetal, resourceNsFlexvm, resourceNs)
	}

	apiKeyEnv := "I3D_FLEXMETAL_API_KEY"
	if resourceNs == resourceNsFlexvm {
		apiKeyEnv = "I3D_FLEXVM_API_KEY"
	}

	apiKey = os.Getenv(apiKeyEnv)
	if apiKey == "" {
		t.Fatalf("%s env is required to run acceptance tests.", apiKeyEnv)
	}

	baseURL = os.Getenv("I3D_BASE_URL")
	if baseURL == "" {
		baseURL = one_api.DefaultBaseURL
	}

	return apiKey, baseURL
}

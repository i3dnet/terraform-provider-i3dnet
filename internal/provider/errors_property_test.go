package provider

import (
	"strings"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// FM-5404: a 422 shows which property One API refused, e.g. networks, with its message.
func TestAddErrorResponseToDiagsShowsTheProperty(t *testing.T) {
	resp := &one_api.ErrorResponse{StatusCode: 422, ErrorCode: 11001, ErrorMessage: "Invalid parameter"}
	resp.Errors = append(resp.Errors, struct {
		Property string `json:"property"`
		Message  string `json:"message"`
	}{Property: "networks", Message: "network x is in another fabric"})
	var diags diag.Diagnostics

	AddErrorResponseToDiags("Error creating server", resp, &diags)

	if detail := diags[0].Detail(); !strings.Contains(detail, "Message: networks: Network x is in another fabric") {
		t.Fatalf("detail = %q, want the property and its message", detail)
	}
}

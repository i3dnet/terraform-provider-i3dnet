package provider

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"terraform-provider-i3dnet/internal/network_api"
	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func AddErrorResponseToDiags(message string, resp *one_api.ErrorResponse, diags *diag.Diagnostics) {
	summary := message

	details := fmt.Sprintf("Status code: %d\n", resp.StatusCode)
	details += fmt.Sprintf("Code: %d\n", resp.ErrorCode)
	if len(resp.Errors) == 0 { // for duplicated tags, the error is not in resp.Errors but resp.ErrorMessage
		details += fmt.Sprintf("Message: %s\n", firstUpper(resp.ErrorMessage))
	}

	for _, v := range resp.Errors {
		details += fmt.Sprintf("Message: %s\n", strings.TrimRight(firstUpper(v.Message), ", "))
	}

	diags.AddError(summary, strings.TrimRight(details, "\n"))
}

// addNetworkAPIErrorToDiags reports a Network API failure. An API error
// response becomes summary = problem title, detail = problem detail, code and
// traceId; any other error is reported under message.
func addNetworkAPIErrorToDiags(message string, err error, diags *diag.Diagnostics) {
	var apiErr *network_api.APIError
	if errors.As(err, &apiErr) {
		diags.AddError(apiErr.Summary(), message+"\n"+apiErr.DiagnosticDetail())
		return
	}

	diags.AddError(message, err.Error())
}

// firstUpper returns a string with the first character as upper case.
func firstUpper(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

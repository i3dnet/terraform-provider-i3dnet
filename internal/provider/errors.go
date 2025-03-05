package provider

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func AddErrorResponseToDiags(message string, resp *one_api.ErrorResponse, diags *diag.Diagnostics) {
	summary := message
	var details string

	if len(resp.Errors) == 0 { // for duplicated tags, the error is not in resp.Errors but resp.ErrorMessage
		details += fmt.Sprintf("Message: %s", firstUpper(resp.ErrorMessage))
	}

	for k, v := range resp.Errors {
		details += fmt.Sprintf("Message: %s", strings.TrimRight(firstUpper(v.Message), ", "))
		if k != len(resp.Errors)-1 {
			details += "\n"
		}
	}
	diags.AddError(summary, details)
}

// firstUpper returns a string with the first character as upper case.
func firstUpper(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

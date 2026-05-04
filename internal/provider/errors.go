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

// firstUpper returns a string with the first character as upper case.
func firstUpper(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

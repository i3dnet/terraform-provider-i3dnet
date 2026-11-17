package network_api

import (
	"encoding/json"
	"testing"

	"github.com/oapi-codegen/nullable"
)

// A merge patch must tell "leave unchanged" (omitted) from "clear" (null) and "set" (a value), which the generated
// PATCH bodies only can with nullable-type (oapi-codegen.yaml).
func TestPrefixPatchRequestKeepsOmittedNullAndValueApart(t *testing.T) {
	for name, tc := range map[string]struct {
		request PrefixPatchRequest
		want    string
	}{
		"omitted": {PrefixPatchRequest{}, `{}`},
		"null":    {PrefixPatchRequest{CustomerDescription: nullable.NewNullNullable[string]()}, `{"customerDescription":null}`},
		"value":   {PrefixPatchRequest{CustomerDescription: nullable.NewNullableWithValue("edge pool")}, `{"customerDescription":"edge pool"}`},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("body = %s, want %s", got, tc.want)
			}
		})
	}
}

package network_api

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCheckResponseAcceptsEvery2xx(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		if err := CheckResponse(status, []byte(`{"title":"ignored","code":"E0"}`)); err != nil {
			t.Errorf("CheckResponse(%d) = %v, want nil", status, err)
		}
	}
}

func TestCheckResponseMapsAProblemDocument(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"type": "about:blank",
		"title": "Unprocessable Entity",
		"status": 422,
		"code": "E020001",
		"detail": "a public network requires explicit addressing.prefixes[]",
		"traceId": "0af7651916cd43dd8448eb211c80319c"
	}`)

	err := CheckResponse(http.StatusUnprocessableEntity, body)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CheckResponse: %v, want an *APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
	if got := apiErr.Summary(); got != "Unprocessable Entity" {
		t.Errorf("Summary() = %q, want the problem title", got)
	}

	detail := apiErr.DiagnosticDetail()
	for _, want := range []string{
		"a public network requires explicit addressing.prefixes[]",
		"E020001",
		"0af7651916cd43dd8448eb211c80319c",
		"quote this traceId to i3D.net support",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("DiagnosticDetail() = %q, want it to contain %q", detail, want)
		}
	}
	if strings.Contains(detail, "api_key") {
		t.Errorf("DiagnosticDetail() = %q, the api_key hint belongs to 401 only", detail)
	}
	if !strings.Contains(apiErr.Error(), "E020001") {
		t.Errorf("Error() = %q, want it to carry the code", apiErr.Error())
	}
}

// A problem document without detail or traceId still yields a diagnostic,
// without dangling labels.
func TestCheckResponseMapsAProblemDocumentWithOnlyTitleAndCode(t *testing.T) {
	t.Parallel()

	err := CheckResponse(http.StatusConflict, []byte(`{"title":"Conflict","status":409,"code":"E020003"}`))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CheckResponse: %v, want an *APIError", err)
	}
	if apiErr.Summary() != "Conflict" {
		t.Errorf("Summary() = %q, want Conflict", apiErr.Summary())
	}
	detail := apiErr.DiagnosticDetail()
	if !strings.Contains(detail, "E020003") {
		t.Errorf("DiagnosticDetail() = %q, want the code", detail)
	}
	if strings.Contains(detail, "Trace ID") {
		t.Errorf("DiagnosticDetail() = %q, want no Trace ID line without a traceId", detail)
	}
}

func TestCheckResponseKeepsANonProblemBody(t *testing.T) {
	t.Parallel()

	html := "<html><body><h1>502 Bad Gateway</h1></body></html>"

	err := CheckResponse(http.StatusBadGateway, []byte(html))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CheckResponse: %v, want an *APIError", err)
	}
	if !strings.Contains(apiErr.Summary(), "502") {
		t.Errorf("Summary() = %q, want the HTTP status", apiErr.Summary())
	}
	detail := apiErr.DiagnosticDetail()
	if !strings.Contains(detail, "502") {
		t.Errorf("DiagnosticDetail() = %q, want the status code", detail)
	}
	if !strings.Contains(detail, html) {
		t.Errorf("DiagnosticDetail() = %q, want the raw body", detail)
	}
}

func TestCheckResponseTruncatesANonProblemBodyToOneKiB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "ASCII body",
			body: strings.Repeat("a", 5000),
		},
		{
			// 1023 ASCII bytes then a 3-byte rune: the 1 KiB cut lands inside it.
			name: "cut inside a multi-byte character",
			body: strings.Repeat("a", 1023) + "€" + strings.Repeat("b", 2000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckResponse(http.StatusBadGateway, []byte(tt.body))

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("CheckResponse: %v, want an *APIError", err)
			}
			if len(apiErr.RawBody) > 1024 {
				t.Errorf("RawBody is %d bytes, want at most 1024", len(apiErr.RawBody))
			}
			if len(apiErr.RawBody) < 1000 {
				t.Errorf("RawBody is %d bytes, want the first KiB kept", len(apiErr.RawBody))
			}
			if !strings.HasPrefix(tt.body, apiErr.RawBody) {
				t.Error("RawBody is not a prefix of the body")
			}
			if !utf8.ValidString(apiErr.RawBody) {
				t.Error("RawBody is not valid UTF-8")
			}
			if !utf8.ValidString(apiErr.DiagnosticDetail()) {
				t.Error("DiagnosticDetail() is not valid UTF-8")
			}
		})
	}
}

// JSON that is not a problem document (no title, no code) is shown raw.
func TestCheckResponseTreatsJSONWithoutTitleOrCodeAsRaw(t *testing.T) {
	t.Parallel()

	body := `{"message":"upstream timeout"}`

	err := CheckResponse(http.StatusGatewayTimeout, []byte(body))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CheckResponse: %v, want an *APIError", err)
	}
	if apiErr.RawBody != body {
		t.Errorf("RawBody = %q, want %q", apiErr.RawBody, body)
	}
	if !strings.Contains(apiErr.Summary(), "504") {
		t.Errorf("Summary() = %q, want the HTTP status", apiErr.Summary())
	}
}

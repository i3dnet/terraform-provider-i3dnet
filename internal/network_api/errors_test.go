package network_api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

// A short body is kept whole, but invalid UTF-8 never reaches the diagnostic.
func TestCheckResponseDropsInvalidUTF8FromAShortNonProblemBody(t *testing.T) {
	t.Parallel()

	body := []byte("bad gateway \xff\xfe from \xc3upstream")

	err := CheckResponse(http.StatusBadGateway, body)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CheckResponse: %v, want an *APIError", err)
	}
	if !utf8.ValidString(apiErr.RawBody) {
		t.Errorf("RawBody = %q, want valid UTF-8", apiErr.RawBody)
	}
	if !utf8.ValidString(apiErr.DiagnosticDetail()) {
		t.Errorf("DiagnosticDetail() = %q, want valid UTF-8", apiErr.DiagnosticDetail())
	}
	if apiErr.RawBody != "bad gateway  from upstream" {
		t.Errorf("RawBody = %q, want the valid text kept", apiErr.RawBody)
	}
}

// closeTracker records whether Do closed the response body.
type closeTracker struct {
	io.Reader
	closed bool
}

func (c *closeTracker) Close() error {
	c.closed = true
	return nil
}

func newResponse(status int, contentType, body string) (*http.Response, *closeTracker) {
	tracker := &closeTracker{Reader: strings.NewReader(body)}
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	return &http.Response{StatusCode: status, Header: header, Body: tracker}, tracker
}

func TestDoPassesATransportErrorThroughUnchanged(t *testing.T) {
	t.Parallel()

	transportErr := fmt.Errorf("failed to do HTTP request: %w: %w", ErrNoResponse, errors.New("connection reset"))

	got, err := Do[FabricListResponse](nil, transportErr)
	// Identity, not errors.Is: the error must pass through unchanged.
	if err != transportErr {
		t.Errorf("Do: %v, want the transport error itself", err)
	}
	if got.Items != nil {
		t.Errorf("Do: %+v, want the zero value", got)
	}
}

func TestDoReportsANon2xxAsAnAPIErrorWhateverTheBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
	}{
		{name: "502 JSON malformed body", status: http.StatusBadGateway, contentType: "application/json", body: `{"items":[`},
		{name: "502 JSON empty body", status: http.StatusBadGateway, contentType: "application/json", body: ""},
		{name: "500 JSON body of another shape", status: http.StatusInternalServerError, contentType: "application/json", body: `{"items":"nope"}`},
		{name: "404 no content type", status: http.StatusNotFound, contentType: "", body: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, tracker := newResponse(tt.status, tt.contentType, tt.body)

			_, err := Do[FabricListResponse](resp, nil)

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("Do: %v, want an *APIError, not a decode error", err)
			}
			if apiErr.StatusCode != tt.status {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.status)
			}
			if apiErr.RawBody != tt.body {
				t.Errorf("RawBody = %q, want %q", apiErr.RawBody, tt.body)
			}
			if errors.Is(err, ErrNoResponse) {
				t.Errorf("Do: %v, a non-2xx is a known response, not ErrNoResponse", err)
			}
			if !tracker.closed {
				t.Error("response body not closed")
			}
		})
	}
}

func TestDoProblemDocumentKeepsItsFields(t *testing.T) {
	t.Parallel()

	resp, _ := newResponse(http.StatusConflict, "application/problem+json",
		`{"title":"Conflict","status":409,"code":"E020003","traceId":"trace-1"}`)

	_, err := Do[NetworkResponse](resp, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Do: %v, want an *APIError", err)
	}
	if apiErr.Code != "E020003" || apiErr.TraceID != "trace-1" || apiErr.StatusCode != http.StatusConflict {
		t.Errorf("APIError = %+v, want the problem document fields", apiErr)
	}
}

// The caller never saw the result of a 2xx it cannot decode, so a create must
// treat it as an unknown outcome.
func TestDoReportsAn2xxThatDoesNotDecodeAsErrNoResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "200 malformed JSON", status: http.StatusOK, body: `{"items":[`},
		{name: "200 empty body", status: http.StatusOK, body: ""},
		{name: "200 wrong shape", status: http.StatusOK, body: `{"items":"nope"}`},
		{name: "202 HTML body", status: http.StatusAccepted, body: "<html>accepted</html>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp, tracker := newResponse(tt.status, "application/json", tt.body)

			_, err := Do[FabricListResponse](resp, nil)
			if !errors.Is(err, ErrNoResponse) {
				t.Fatalf("Do: %v, want it to wrap ErrNoResponse", err)
			}
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				t.Errorf("Do: %v, a 2xx is not an *APIError", err)
			}
			if !strings.Contains(err.Error(), fmt.Sprint(tt.status)) {
				t.Errorf("Do: %v, want the status in the message", err)
			}
			if !tracker.closed {
				t.Error("response body not closed")
			}
		})
	}
}

func TestDoDecodesAValid2xxBody(t *testing.T) {
	t.Parallel()

	resp, tracker := newResponse(http.StatusOK, "application/json",
		`{"items":[{"slug":"rotterdam-a","displayName":"Rotterdam A","metro":"Rotterdam"}],"links":{"self":{"href":"/fabrics?limit=100"},"next":{"href":"/fabrics?cursor=abc"}}}`)

	got, err := Do[FabricListResponse](resp, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Slug != "rotterdam-a" || got.Items[0].Metro != "Rotterdam" {
		t.Errorf("Items = %+v, want the decoded fabric", got.Items)
	}
	if got.Links.Next == nil || got.Links.Next.Href != "/fabrics?cursor=abc" {
		t.Errorf("Links.Next = %+v, want the decoded next link", got.Links.Next)
	}
	if !tracker.closed {
		t.Error("response body not closed")
	}
}

// A body cut short after the status line, read through the client's doer, is
// ErrNoResponse: the caller never saw the result.
func TestDoReportsABodyReadFailureAsErrNoResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":`))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("Hijack: %v", err)
			return
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)

	api, err := NewAPI("key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}

	_, err = Do[FabricListResponse](api.ListFabrics(context.Background(), &ListFabricsParams{}))
	if !errors.Is(err, ErrNoResponse) {
		t.Fatalf("Do: %v, want it to wrap ErrNoResponse", err)
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("Do: %v, want no *APIError for a body that never arrived", err)
	}
}

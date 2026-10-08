package network_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// maxRawBodyBytes caps how much of a body that is not a problem document is
// kept for the diagnostic.
const maxRawBodyBytes = 1024

// APIError is a non-2xx Network API response. Problem documents
// (application/problem+json) fill Title, Detail, Code and TraceID; any other
// body is kept in RawBody, truncated to 1 KiB.
type APIError struct {
	StatusCode int
	Title      string
	Detail     string
	Code       string
	TraceID    string
	RawBody    string
}

// CheckResponse returns nil for a 2xx status and an *APIError otherwise.
// body is the already-read response body (the generated responses' Body).
func CheckResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	apiErr := &APIError{StatusCode: statusCode}

	var problem ProblemDetails
	if err := json.Unmarshal(body, &problem); err == nil && (problem.Title != "" || problem.Code != "") {
		apiErr.Title = problem.Title
		apiErr.Code = problem.Code
		apiErr.TraceID = problem.TraceId
		if problem.Detail != nil {
			apiErr.Detail = *problem.Detail
		}

		return apiErr
	}

	raw := string(body)
	if len(raw) > maxRawBodyBytes {
		// A cut inside a multi-byte character would leave invalid UTF-8.
		raw = strings.ToValidUTF8(raw[:maxRawBodyBytes], "")
	}
	apiErr.RawBody = raw

	return apiErr
}

func (e *APIError) Error() string {
	return e.Summary() + ": " + e.DiagnosticDetail()
}

// Summary is the diagnostic summary: the problem title, or the HTTP status
// when the body was not a problem document.
func (e *APIError) Summary() string {
	if e.Title != "" {
		return e.Title
	}

	return fmt.Sprintf("Network API returned HTTP %d %s", e.StatusCode, http.StatusText(e.StatusCode))
}

// DiagnosticDetail is the diagnostic detail: the problem detail, code and
// traceId with a support hint, or the status and raw body.
func (e *APIError) DiagnosticDetail() string {
	var b strings.Builder

	if e.StatusCode == http.StatusUnauthorized {
		b.WriteString("The Network API rejected the credentials: check the provider api_key.\n")
	}

	if e.Title == "" && e.Code == "" {
		fmt.Fprintf(&b, "Status code: %d\nBody: %s", e.StatusCode, e.RawBody)
		return b.String()
	}

	if e.Detail != "" {
		b.WriteString(e.Detail + "\n")
	}
	if e.Code != "" {
		fmt.Fprintf(&b, "Code: %s\n", e.Code)
	}
	if e.TraceID != "" {
		fmt.Fprintf(&b, "Trace ID: %s (quote this traceId to i3D.net support)\n", e.TraceID)
	}

	return strings.TrimRight(b.String(), "\n")
}

package network_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// body is the already-read response body.
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

	if len(body) > maxRawBodyBytes {
		body = body[:maxRawBodyBytes]
	}
	// Drops invalid UTF-8 from the body and from a cut inside a multi-byte character.
	apiErr.RawBody = strings.ToValidUTF8(string(body), "")

	return apiErr
}

// Do completes a call to a generated raw method (ClientInterface), e.g.
// Do[FabricListResponse](c.ListFabrics(ctx, params)). It reads the body,
// returns the CheckResponse error for a non-2xx status and only then decodes
// the 2xx JSON body into T. A 2xx body that does not decode is reported as
// ErrNoResponse, so a create treats its outcome as unknown.
func Do[T any](resp *http.Response, err error) (T, error) {
	var out T
	if err != nil {
		return out, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if !errors.Is(err, ErrNoResponse) {
			err = fmt.Errorf("%w: %w", ErrNoResponse, err)
		}
		return out, err
	}

	if err := CheckResponse(resp.StatusCode, body); err != nil {
		return out, err
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("could not decode HTTP %d response body: %w: %w", resp.StatusCode, ErrNoResponse, err)
	}

	return out, nil
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

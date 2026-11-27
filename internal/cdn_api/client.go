// Package cdn_api is a client for the i3D.net CDN API, which manages CDN
// resources: a set of public hostnames served by the CDN edges from one or more
// origins.
//
// The client is hand-written rather than generated: the API's published
// Swagger document describes the strategy, location and dns_record enums as
// integers, while the API reads and writes them as their names.
package cdn_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// DefaultBaseURL is the CDN API base URL used when cdn_base_url is not set.
	DefaultBaseURL = "https://api.cdn-v2.i3d.net/api"

	// defaultRequestTimeout bounds requests whose context carries no deadline.
	defaultRequestTimeout = 5 * time.Minute
)

// ErrNoResponse marks a request that never produced an HTTP response: a dial,
// TLS, write or read failure, or an expired context. The API may still have
// accepted it, so a create must treat its outcome as unknown.
var ErrNoResponse = errors.New("request did not complete")

// APIError is a non-2xx response from the CDN API.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	// Message is the API's error text, or the raw body when it is not the
	// API's error shape.
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("%s %s: HTTP %d", e.Method, e.Path, e.StatusCode)
	}

	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
}

// IsNotFound reports whether err is a 404 from the API.
func IsNotFound(err error) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// Client calls the CDN API with a CDN API key.
type Client struct {
	apiKey         string
	baseURL        *url.URL
	transport      http.RoundTripper
	requestTimeout time.Duration
}

// NewClient returns a client for rawBaseURL (DefaultBaseURL when empty) that
// sends apiKey as a bearer token.
func NewClient(apiKey, rawBaseURL string) (*Client, error) {
	if rawBaseURL == "" {
		rawBaseURL = DefaultBaseURL
	}

	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse CDN base url %s: %w", rawBaseURL, err)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("CDN base url %q must be an absolute URL, for example %s", rawBaseURL, DefaultBaseURL)
	}

	return &Client{
		apiKey:         apiKey,
		baseURL:        baseURL,
		requestTimeout: defaultRequestTimeout,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 2 * time.Minute,
		},
	}, nil
}

// do sends one request. in, when not nil, is the JSON body; out, when not nil,
// receives the decoded 2xx body. Any 2xx is success: the API answers PUTs with
// 200 on some paths and 201 on others.
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("could not encode request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL.JoinPath(path).String(), body)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{Transport: &loggingRoundTripper{next: c.transport, ctx: ctx}}
	if _, ok := ctx.Deadline(); !ok {
		client.Timeout = c.requestTimeout
	}

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("%s %s: %w", method, path, err)
		}

		return fmt.Errorf("%s %s: %w: %w", method, path, ErrNoResponse, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s: reading response: %w: %w", method, path, ErrNoResponse, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       path,
			Message:    errorMessage(respBody),
		}
	}

	if out == nil || len(bytes.TrimSpace(respBody)) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("%s %s: could not decode response: %w", method, path, err)
	}

	return nil
}

// errorMessage extracts the text of the API's error body, {"code":..,"message":..}
// or {"error":..}, falling back to the raw body.
func errorMessage(body []byte) string {
	var e struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if json.Unmarshal(body, &e) == nil {
		if e.Message != "" {
			return e.Message
		}
		if e.Error != "" {
			return e.Error
		}
	}

	const limit = 512
	s := string(bytes.TrimSpace(body))
	if len(s) > limit {
		s = s[:limit] + "..."
	}

	return s
}

// loggingRoundTripper logs requests and responses at TF_LOG=DEBUG, with the
// API key redacted.
type loggingRoundTripper struct {
	next http.RoundTripper
	ctx  context.Context
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	original := req.Header
	redacted := original.Clone()
	if redacted.Get("Authorization") != "" {
		redacted.Set("Authorization", "***")
	}
	req.Header = redacted
	dump, err := httputil.DumpRequestOut(req, true)
	req.Header = original
	if err != nil {
		return nil, fmt.Errorf("error dumping request: %w", err)
	}

	tflog.Debug(l.ctx, "Sending CDN API request", map[string]any{"body": string(dump), "method": req.Method})

	resp, err := l.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("error dumping response: %w", err)
	}

	tflog.Debug(l.ctx, "Getting CDN API response", map[string]any{"body": string(respDump), "duration": time.Since(start)})

	return resp, nil
}

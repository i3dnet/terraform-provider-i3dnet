package one_api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	apiKey    string
	baseURL   *url.URL
	transport http.RoundTripper
	// requestTimeout caps a request whose context carries no deadline.
	requestTimeout time.Duration
}

const (
	DefaultBaseURL = "https://api.i3d.net"
	apiVersion     = "v3"

	// defaultRequestTimeout bounds requests that arrive without a deadline of
	// their own: Read, Update, Delete and the data sources. It has to be
	// generous, since it covers reading the response body as well.
	defaultRequestTimeout = 5 * time.Minute
)

// ErrRequestNotCompleted marks a request that never produced an HTTP response:
// a dial, TLS, write or read failure, or an expired context. The API may still
// have accepted it, so callers that create resources must reconcile instead of
// assuming failure.
var ErrRequestNotCompleted = errors.New("request did not complete")

func NewClient(apiKey string, rawBaseURL string) (*Client, error) {
	if rawBaseURL == "" {
		rawBaseURL = DefaultBaseURL
	}

	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse base url %s: %w", rawBaseURL, err)
	}

	baseURL = baseURL.JoinPath(apiVersion)

	return &Client{
		apiKey:         apiKey,
		baseURL:        baseURL,
		requestTimeout: defaultRequestTimeout,
		// No http.Client.Timeout: a fixed whole-request cap would override the
		// context deadline callers derive from their Terraform timeouts, and a
		// slow create POST would then fail even though the server gets built.
		// These bound the parts that can hang without progress instead.
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
			ResponseHeaderTimeout: 3 * time.Minute,
		},
	}, nil
}

func (c *Client) callAPI(ctx context.Context, method, endpoint, path string, body []byte, queryParams map[string]string) (*http.Response, error) {
	return c.callAPIWithHeaders(ctx, method, endpoint, path, body, queryParams, nil)
}

// callAPIWithHeaders behaves like callAPI but also sets the provided request
// headers in addition to the default ones. It is useful for headers such as
// the RANGED-DATA pagination header.
func (c *Client) callAPIWithHeaders(ctx context.Context, method, endpoint, path string, body []byte,
	queryParams, headers map[string]string) (*http.Response, error) {
	client := &http.Client{
		Transport: &loggingRoundTripper{next: c.transport, ctx: ctx},
	}

	// A deadline on the context always wins: capping the whole request at a
	// fixed duration is what used to doom slow create requests. Without one,
	// fall back to a generous cap so a stalled body read cannot hang forever.
	if _, ok := ctx.Deadline(); !ok {
		client.Timeout = c.requestTimeout
	}

	apiURL := c.baseURL
	if endpoint != "" {
		apiURL = apiURL.JoinPath(endpoint)
	}
	if path != "" {
		apiURL = apiURL.JoinPath(path)
	}

	query := apiURL.Query()
	for k, v := range queryParams {
		query.Set(k, v)
	}
	apiURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, apiURL.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.apiKey)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		// A non-nil response alongside the error only happens when
		// CheckRedirect fails, and net/http has closed its body already. A
		// response did arrive, so this is not an incomplete request.
		if resp != nil {
			return nil, fmt.Errorf("failed to do HTTP request: %w", err)
		}

		return nil, fmt.Errorf("failed to do HTTP request: %w: %w", ErrRequestNotCompleted, err)
	}

	return resp, nil
}

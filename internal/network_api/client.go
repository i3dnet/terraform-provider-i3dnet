package network_api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	// DefaultBaseURL is the provider's default base_url, shared with the One API client.
	DefaultBaseURL = "https://api.i3d.net"

	// defaultRequestTimeout bounds requests whose context carries no deadline,
	// as in one_api. It covers reading the response body as well.
	defaultRequestTimeout = 5 * time.Minute
)

// apiPath is where the API gateway serves the public Network API, relative to base_url.
var apiPath = []string{"network", "v1"}

// ErrNoResponse marks a request that never produced an HTTP response: a dial,
// TLS, write or read failure, or an expired context. The API may still have
// accepted it, so a create must treat its outcome as unknown.
var ErrNoResponse = errors.New("request did not complete")

// NewAPI builds the generated client for {rawBaseURL}/network/v1 that
// authenticates every request with the PRIVATE-TOKEN header. An empty
// rawBaseURL means DefaultBaseURL. The generated code owns the names Client
// and NewClient.
func NewAPI(apiKey string, rawBaseURL string) (*ClientWithResponses, error) {
	if rawBaseURL == "" {
		rawBaseURL = DefaultBaseURL
	}

	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse base url %s: %w", rawBaseURL, err)
	}

	server := baseURL.JoinPath(apiPath...).String()

	client, err := NewClientWithResponses(server,
		WithHTTPClient(newDoer()),
		WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("PRIVATE-TOKEN", apiKey)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create network api client: %w", err)
	}

	return client, nil
}

// doer sends requests with the transport timeouts of one_api and classifies
// every failure that leaves the caller without a full response as ErrNoResponse.
type doer struct {
	// withDeadline serves requests whose context has a deadline. It has no
	// whole-request timeout: the context deadline, derived from the Terraform
	// timeouts, always wins.
	withDeadline *http.Client
	// withoutDeadline caps a request whose context carries no deadline.
	withoutDeadline *http.Client
}

func newDoer() *doer {
	// Same settings as one_api.NewClient: these bound the parts of a request
	// that can hang without progress instead of the whole request.
	transport := &http.Transport{
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
	}

	return &doer{
		withDeadline:    &http.Client{Transport: transport},
		withoutDeadline: &http.Client{Transport: transport, Timeout: defaultRequestTimeout},
	}
}

func (d *doer) Do(req *http.Request) (*http.Response, error) {
	client := d.withDeadline
	if _, ok := req.Context().Deadline(); !ok {
		client = d.withoutDeadline
	}

	resp, err := client.Do(req)
	if err != nil {
		// A non-nil response alongside the error only happens when
		// CheckRedirect fails, and net/http has closed its body already. A
		// response did arrive, so this is not an incomplete request.
		if resp != nil {
			return nil, fmt.Errorf("failed to do HTTP request: %w", err)
		}

		return nil, fmt.Errorf("failed to do HTTP request: %w: %w", ErrNoResponse, err)
	}

	resp.Body = &classifyingBody{ReadCloser: resp.Body}

	return resp, nil
}

// classifyingBody marks a failure while reading the response body as
// ErrNoResponse: the status line arrived, but the caller never saw the result.
type classifyingBody struct {
	io.ReadCloser
}

func (b *classifyingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("failed to read HTTP response body: %w: %w", ErrNoResponse, err)
	}

	return n, err
}

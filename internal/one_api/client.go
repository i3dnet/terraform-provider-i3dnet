package one_api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	apiKey  string
	baseURL *url.URL
}

const (
	DefaultBaseURL = "https://api.i3d.net"
	apiVersion     = "v3"
)

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
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

func (c *Client) callAPI(ctx context.Context, method, endpoint, path string, body []byte, queryParams map[string]string) (*http.Response, error) {
	client := &http.Client{
		Transport: &loggingRoundTripper{next: http.DefaultTransport, ctx: ctx},
		Timeout:   90 * time.Second,
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do HTTP request: %w", err)
	}

	return resp, nil
}

package one_api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var FlexmetalUrl = "https://api.i3d.net" + "/v3/flexMetal"

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

func (c *Client) callAPI(method, endpoint, path string, body []byte) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	apiURL := c.baseURL
	if endpoint != "" {
		apiURL = apiURL.JoinPath(endpoint)
	}
	if path != "" {
		apiURL = apiURL.JoinPath(path)
	}

	req, err := http.NewRequest(method, apiURL.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do HTTP request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wrong status code received: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// CallFlexMetalAPI is a placeholder for the API call logic to the FlexMetal API.
func (c *Client) CallFlexMetalAPI(method string, path string, body []byte) ([]byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	var responseBody []byte
	// Create an HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Build the full URL using the server's endpoint and the provided path
	url := fmt.Sprintf("%s/%s", FlexmetalUrl, path)

	// Create the request
	var req *http.Request
	var err error
	if len(body) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		diags.AddError("API Request Error", fmt.Sprintf("Failed to create HTTP request: %s", err))
		return responseBody, diags
	}

	// Set the required headers (API Key and Content-Type)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", c.apiKey)

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		diags.AddError("API Request Error", fmt.Sprintf("Failed to execute API request: %s", err))
		return []byte{}, diags
	}
	defer resp.Body.Close()

	// Read the response body
	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		diags.AddError("API Response Error", fmt.Sprintf("Failed to read response body: %s", err))
		return responseBody, diags
	}

	// Check for non-2xx response codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		diags.AddError("API Response Error", fmt.Sprintf("API responded with non-2xx status: %d, reason: %s, sent body: %s", resp.StatusCode, responseBody, body))
		return responseBody, diags
	}

	return responseBody, diags
}

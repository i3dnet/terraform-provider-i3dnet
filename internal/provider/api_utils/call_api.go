package api_utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var FlexmetalUrl = "https://api.i3d.net" + "/v3/flexMetal"

type Client struct {
	apiKey  string
	baseURL string
}

func NewClient(apiKey string, baseURL string) *Client {
	var apiBaseURL = "https://api.i3d.net"
	if baseURL != "" {
		apiBaseURL = baseURL
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: apiBaseURL + "/v3",
	}
}

type CreateSSHKey struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type SshKeyResp struct {
	Uuid      string `json:"uuid"`
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
	CreatedAt int64  `json:"createdAt"`
}

func (c *Client) CreateSSHKey(req CreateSSHKey) (*SshKeyResp, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.callAPI(http.MethodPost, "sshKey", "", body)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	var sshKeyResp []SshKeyResp
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, err
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &sshKeyResp[0], nil
}

func (c *Client) GetSSHKey(id string) (*SshKeyResp, error) {
	resp, err := c.callAPI(http.MethodGet, "sshKey", id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	var sshKeyResp []SshKeyResp
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("error decoding respones: %w", err)
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &sshKeyResp[0], nil
}

func (c *Client) DeleteSSHKey(id string) error {
	resp, err := c.callAPI(http.MethodDelete, "sshKey", id, nil)
	if err != nil {
		return fmt.Errorf("error calling API: %w", err)
	}
	defer resp.Close()

	return nil
}

func (c *Client) callAPI(method, endpoint, path string, body []byte) (io.ReadCloser, error) {
	// Create an HTTP client
	client := &http.Client{Timeout: 10 * time.Second}

	// Build the full URL using the server's endpoint and the provided path
	url := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	if path != "" {
		url += "/" + path
	}

	var req *http.Request
	var err error
	if len(body) > 0 {
		fmt.Println(method, url, string(body))

		fmt.Printf("Request payload: %s\n", string(body)) // Add debug log to check payload

		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
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
func CallFlexMetalAPI(method string, path string, body []byte) ([]byte, diag.Diagnostics) {
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
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("FLEXMETAL_API_KEY"))

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

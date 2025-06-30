package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const sshKeyEndpoint = "sshKey"

type CreateSSHKeyReq struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
}

type SSHKey struct {
	Uuid      string `json:"uuid"`
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
	CreatedAt int64  `json:"createdAt"`
}

// SSHKeyResponse contains a SSHKey in case of a 200 response
// or an ErrorResponse
type SSHKeyResponse struct {
	ErrorResponse *ErrorResponse
	SSHKey        *SSHKey
}

func (c *Client) CreateSSHKey(ctx context.Context, req CreateSSHKeyReq) (*SSHKeyResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.callAPI(ctx, http.MethodPost, sshKeyEndpoint, "", body, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling create ssh key api: %w", err)
	}
	defer resp.Body.Close()

	var response SSHKeyResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var sshKeyResp []SSHKey
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.SSHKey = &sshKeyResp[0]
	return &response, nil
}

func (c *Client) GetSSHKey(ctx context.Context, id string) (*SSHKeyResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, sshKeyEndpoint, id, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling get ssh key api: %w", err)
	}
	defer resp.Body.Close()

	var response SSHKeyResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var sshKeyResp []SSHKey
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.SSHKey = &sshKeyResp[0]
	return &response, nil
}

func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, sshKeyEndpoint, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling list ssh key api: %w", err)
	}
	defer resp.Body.Close()

	var sshKeyResp []SSHKey
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return sshKeyResp, nil
}

func (c *Client) DeleteSSHKey(ctx context.Context, id string) error {
	resp, err := c.callAPI(ctx, http.MethodDelete, sshKeyEndpoint, id, nil, nil)
	if err != nil {
		return fmt.Errorf("error calling delete ssh key API: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

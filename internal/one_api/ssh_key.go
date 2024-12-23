package one_api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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

func (c *Client) CreateSSHKey(req CreateSSHKeyReq) (*SSHKey, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.callAPI(http.MethodPost, "sshKey", "", body)
	if err != nil {
		return nil, fmt.Errorf("error on calling create ssh key api: %w", err)
	}
	defer resp.Close()

	var sshKeyResp []SSHKey
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &sshKeyResp[0], nil
}

func (c *Client) GetSSHKey(id string) (*SSHKey, error) {
	resp, err := c.callAPI(http.MethodGet, "sshKey", id, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling get ssh key api: %w", err)
	}
	defer resp.Close()

	var sshKeyResp []SSHKey
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(sshKeyResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &sshKeyResp[0], nil
}

func (c *Client) DeleteSSHKey(id string) error {
	resp, err := c.callAPI(http.MethodDelete, "sshKey", id, nil)
	if err != nil {
		return fmt.Errorf("error calling delete ssh key API: %w", err)
	}
	defer resp.Close()

	return nil
}

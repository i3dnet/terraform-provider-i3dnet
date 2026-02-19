package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VmInstanceOS struct {
	ImageID string `json:"imageId"`
}

type VmInstance struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	PoolID        string       `json:"poolId"`
	Plan          string       `json:"plan"`
	OS            VmInstanceOS `json:"os"`
	UserData      string       `json:"userData,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
	Status        string       `json:"status"`
	IPAddress     string       `json:"ipAddress,omitempty"`
	IPAddressV6   string       `json:"ipAddressV6,omitempty"`
	Gateway       string       `json:"gateway,omitempty"`
	Netmask       string       `json:"netmask,omitempty"`
	VlanID        int64        `json:"vlanId,omitempty"`
	ProvisionedAt string       `json:"provisionedAt,omitempty"`
}

type CreateVmInstanceRequest struct {
	Name     string       `json:"name"`
	PoolID   string       `json:"poolId"`
	Plan     string       `json:"plan"`
	OS       VmInstanceOS `json:"os"`
	UserData string       `json:"userData,omitempty"`
	Tags     []string     `json:"tags,omitempty"`
}

type UpdateVmInstanceRequest struct {
	Tags []string `json:"tags"`
}

type VmInstanceResponse struct {
	ErrorResponse *ErrorResponse
	Instance      *VmInstance
}

func (c *Client) ListVmInstances(ctx context.Context) ([]VmInstance, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, "vm/instances", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling list vm instances API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", decodeErrResponse(resp).ErrorMessage)
	}

	var instances []VmInstance
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return instances, nil
}

func (c *Client) CreateVmInstance(ctx context.Context, req CreateVmInstanceRequest) (*VmInstanceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, flexMetalEndpoint, "vm/instances", body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling create vm instance API: %w", err)
	}
	defer resp.Body.Close()

	var response VmInstanceResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var instance VmInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Instance = &instance
	return &response, nil
}

func (c *Client) GetVmInstance(ctx context.Context, id string) (*VmInstanceResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("vm/instances/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling get vm instance API: %w", err)
	}
	defer resp.Body.Close()

	var response VmInstanceResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var instance VmInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Instance = &instance
	return &response, nil
}

func (c *Client) UpdateVmInstance(ctx context.Context, id string, req UpdateVmInstanceRequest) (*VmInstanceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPatch, flexMetalEndpoint, fmt.Sprintf("vm/instances/%s", id), body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling update vm instance API: %w", err)
	}
	defer resp.Body.Close()

	var response VmInstanceResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var instance VmInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Instance = &instance
	return &response, nil
}

func (c *Client) DeleteVmInstance(ctx context.Context, id string) (*VmInstanceResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("vm/instances/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete vm instance API: %w", err)
	}
	defer resp.Body.Close()

	var response VmInstanceResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	return &response, nil
}

func (c *Client) SendVmCommand(ctx context.Context, id, command string) (*VmInstanceResponse, error) {
	body, err := json.Marshal(map[string]string{"command": command})
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, flexMetalEndpoint, fmt.Sprintf("vm/instances/%s/command", id), body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling vm command API: %w", err)
	}
	defer resp.Body.Close()

	var response VmInstanceResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var instance VmInstance
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Instance = &instance
	return &response, nil
}

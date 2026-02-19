package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VmPoolSubnet struct {
	CIDR       string `json:"cidr"`
	Gateway    string `json:"gateway"`
	RangeStart string `json:"rangeStart"`
	RangeEnd   string `json:"rangeEnd"`
}

type VmPool struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	LocationID   string            `json:"locationId"`
	ContractID   string            `json:"contractId"`
	Type         string            `json:"type"`
	InstanceType string            `json:"instanceType"`
	VlanID       int64             `json:"vlanId"`
	Subnet       []VmPoolSubnet    `json:"subnet"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Status       string            `json:"status"`
}

type CreateVmPoolRequest struct {
	Name         string            `json:"name"`
	LocationID   string            `json:"locationId"`
	ContractID   string            `json:"contractId"`
	Type         string            `json:"type"`
	InstanceType string            `json:"instanceType"`
	VlanID       int64             `json:"vlanId"`
	Subnet       []VmPoolSubnet    `json:"subnet"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type UpdateVmPoolRequest struct {
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type VmPoolResponse struct {
	ErrorResponse *ErrorResponse
	Pool          *VmPool
}

func (c *Client) ListVmPools(ctx context.Context) ([]VmPool, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, "vm/pools", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling list vm pools API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", decodeErrResponse(resp).ErrorMessage)
	}

	var pools []VmPool
	if err := json.NewDecoder(resp.Body).Decode(&pools); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return pools, nil
}

func (c *Client) CreateVmPool(ctx context.Context, req CreateVmPoolRequest) (*VmPoolResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, flexMetalEndpoint, "vm/pools", body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling create vm pool API: %w", err)
	}
	defer resp.Body.Close()

	var response VmPoolResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var pool VmPool
	if err := json.NewDecoder(resp.Body).Decode(&pool); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Pool = &pool
	return &response, nil
}

func (c *Client) GetVmPool(ctx context.Context, id string) (*VmPoolResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("vm/pools/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling get vm pool API: %w", err)
	}
	defer resp.Body.Close()

	var response VmPoolResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var pool VmPool
	if err := json.NewDecoder(resp.Body).Decode(&pool); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Pool = &pool
	return &response, nil
}

func (c *Client) UpdateVmPool(ctx context.Context, id string, req UpdateVmPoolRequest) (*VmPoolResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPatch, flexMetalEndpoint, fmt.Sprintf("vm/pools/%s", id), body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling update vm pool API: %w", err)
	}
	defer resp.Body.Close()

	var response VmPoolResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var pool VmPool
	if err := json.NewDecoder(resp.Body).Decode(&pool); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Pool = &pool
	return &response, nil
}

func (c *Client) DeleteVmPool(ctx context.Context, id string) (*VmPoolResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("vm/pools/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete vm pool API: %w", err)
	}
	defer resp.Body.Close()

	var response VmPoolResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	return &response, nil
}

package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const flexVMEndpoint = "flexVM"

type FlexvmCreateVMRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	InstanceTypeName string   `json:"instance_type_name,omitempty"`
	ImageName        string   `json:"image_name,omitempty"`
	SSHKeys          []string `json:"ssh_keys"`
}

type FlexvmVM struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	InstanceType FlexvmInstanceType `json:"instance_type"`
	Image        FlexvmImage        `json:"image"`
	Status       string             `json:"status"`
	IPs          []FlexvmIPAddress  `json:"ips"`
	Cloud        FlexvmCloud        `json:"cloud"`
	Node         *FlexvmNode        `json:"node"`
	CreatedAt    string             `json:"createdAt"`
	DeletedAt    string             `json:"deleted_at"`
}

type FlexvmInstanceType struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	VCPU   int    `json:"vcpu"`
	Memory int    `json:"memory"`
	Disk   int    `json:"disk"`
}

type FlexvmImage struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	OS     string `json:"os"`
	OSType string `json:"os_type"`
}

type FlexvmIPAddress struct {
	Address string `json:"address"`
	Public  bool   `json:"public"`
}

type FlexvmCloud struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Site        string `json:"site"`
}

type FlexvmNode struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	InstanceType string      `json:"instance_type"`
	Serial       string      `json:"serial"`
	Cloud        FlexvmCloud `json:"cloud"`
}

type FlexvmVMResponse struct {
	ErrorResponse *ErrorResponse
	VM            *FlexvmVM
}

func (c *Client) FlexvmCreateVM(ctx context.Context, cloudUUID string, req FlexvmCreateVMRequest) (*FlexvmVMResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, flexVMEndpoint, fmt.Sprintf("clouds/%s/vms", cloudUUID), body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm create vm API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmVMResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var vm FlexvmVM
	if err := json.NewDecoder(resp.Body).Decode(&vm); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.VM = &vm
	return &response, nil
}

func (c *Client) FlexvmGetVM(ctx context.Context, cloudUUID, vmUUID string) (*FlexvmVMResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexVMEndpoint, fmt.Sprintf("clouds/%s/vms/%s", cloudUUID, vmUUID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm get vm API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmVMResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var vm FlexvmVM
	if err := json.NewDecoder(resp.Body).Decode(&vm); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.VM = &vm
	return &response, nil
}

func (c *Client) FlexvmDeleteVM(ctx context.Context, cloudUUID, vmUUID string) (*FlexvmVMResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexVMEndpoint, fmt.Sprintf("clouds/%s/vms/%s", cloudUUID, vmUUID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexvm API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmVMResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	return &response, nil
}

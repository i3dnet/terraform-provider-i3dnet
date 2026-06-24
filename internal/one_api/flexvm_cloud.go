package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// FlexvmCloudCreateRequest is the body sent when creating a FlexVM Cloud. Only
// the writable fields are included; id and created_at are assigned by the API.
type FlexvmCloudCreateRequest struct {
	Name         string `json:"name"`
	Site         string `json:"site"`
	InstanceType string `json:"instance_type"`
	Description  string `json:"description,omitempty"`
}

// FlexvmCloudObj is the full FlexVM Cloud representation returned by the API.
type FlexvmCloudObj struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Site         string `json:"site"`
	InstanceType string `json:"instance_type"`
	Description  string `json:"description"`
	CreatedAt    string `json:"created_at"`
}

type FlexvmCloudResponse struct {
	ErrorResponse *ErrorResponse
	Cloud         *FlexvmCloudObj
}

func (c *Client) FlexvmCreateCloud(ctx context.Context, req FlexvmCloudCreateRequest) (*FlexvmCloudResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, flexVMEndpoint, "clouds", body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm create cloud API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmCloudResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var cloud FlexvmCloudObj
	if err := json.NewDecoder(resp.Body).Decode(&cloud); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Cloud = &cloud
	return &response, nil
}

func (c *Client) FlexvmGetCloud(ctx context.Context, cloudID string) (*FlexvmCloudResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexVMEndpoint, fmt.Sprintf("clouds/%s", cloudID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm get cloud API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmCloudResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var cloud FlexvmCloudObj
	if err := json.NewDecoder(resp.Body).Decode(&cloud); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Cloud = &cloud
	return &response, nil
}

func (c *Client) FlexvmDeleteCloud(ctx context.Context, cloudID string) (*FlexvmCloudResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexVMEndpoint, fmt.Sprintf("clouds/%s", cloudID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm delete cloud API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmCloudResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	return &response, nil
}

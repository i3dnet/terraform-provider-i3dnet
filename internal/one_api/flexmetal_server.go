package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const flexMetalEndpoint = "flexMetal"

type CreateServerReq struct {
	Name              string   `json:"name"`
	Location          string   `json:"location"`
	InstanceType      string   `json:"instanceType"`
	OS                OS       `json:"os"`
	Tags              []string `json:"tags"`
	SSHkey            []string `json:"sshKey,omitempty"`
	PostInstallScript string   `json:"postInstallScript"`
}

type OS struct {
	Slug         string        `json:"slug"`
	KernelParams []KernelParam `json:"kernelParams"`
	Partitions   []Partition   `json:"partitions"`
}

type KernelParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Partition struct {
	Target     string `json:"target"`
	Filesystem string `json:"filesystem"`
	Size       int64  `json:"size"`
}

func (c *Client) CreateServer(ctx context.Context, req CreateServerReq) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, "flexMetal", "servers", body)
	if err != nil {
		return nil, fmt.Errorf("error on calling create flexmetal server api: %w", err)
	}
	defer resp.Close()

	respBody, err := io.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return respBody, nil
}

func (c *Client) GetServer(ctx context.Context, id string) ([]byte, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("servers/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling get flexmetal server api: %w", err)
	}
	defer resp.Close()

	respBody, err := io.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return respBody, nil
}

func (c *Client) DeleteServer(ctx context.Context, id string) ([]byte, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("servers/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Close()

	respBody, err := io.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return respBody, nil
}

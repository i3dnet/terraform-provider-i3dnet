package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	ContractID        string   `json:"contractId,omitempty"`
	Overflow          bool     `json:"overflow"`
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

type Server struct {
	Uuid          string `json:"uuid"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage"`
	Location      struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"location"`
	InstanceType struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"instanceType"`
	Os struct {
		Slug string `json:"slug"`
	} `json:"os"`
	IpAddresses []struct {
		IpAddress string `json:"ipAddress"`
	} `json:"ipAddresses"`
	Tags        []string `json:"tags"`
	CreatedAt   int64    `json:"createdAt"`
	DeliveredAt int64    `json:"deliveredAt"`
	ReleasedAt  int64    `json:"releasedAt"`
	ContractID  string   `json:"contractId"`
}

type ErrorResponse struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Errors       []struct {
		Property string `json:"property"`
		Message  string `json:"message"`
	} `json:"errors"`
}

// ServerResponse can contain Server in case of a 200 response
// or an ErrorResponse
type ServerResponse struct {
	ErrorResponse *ErrorResponse
	Server        *Server
}

func (c *Client) CreateServer(ctx context.Context, req CreateServerReq) (*ServerResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, "flexMetal", "servers", body)
	if err != nil {
		return nil, fmt.Errorf("error on calling create flexmetal server api: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var serverResp []Server
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&serverResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(serverResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Server = &serverResp[0]

	return &response, nil
}

func (c *Client) GetServer(ctx context.Context, id string) (*ServerResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("servers/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling get flexmetal server api: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse

	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var serverResp []Server
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&serverResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(serverResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Server = &serverResp[0]

	return &response, nil
}

func (c *Client) DeleteServer(ctx context.Context, id string) (*ServerResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("servers/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse

	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var serverResp []Server
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&serverResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(serverResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Server = &serverResp[0]

	return &response, nil
}

func (c *Client) AddTagToServer(ctx context.Context, serverID, tag string) (*ServerResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodPost, flexMetalEndpoint, fmt.Sprintf("servers/%s/tag/%s", serverID, tag), nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var serverResp []Server
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&serverResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(serverResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Server = &serverResp[0]

	return &response, nil
}

func (c *Client) DeleteTagFromServer(ctx context.Context, serverID, tag string) (*ServerResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("servers/%s/tag/%s", serverID, tag), nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var serverResp []Server
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&serverResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(serverResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Server = &serverResp[0]

	return &response, nil
}

func decodeErrResponse(ctx context.Context, resp *http.Response) *ErrorResponse {
	var errResponse ErrorResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&errResponse); err != nil {
		tflog.Error(ctx, fmt.Sprintf("could not unmarshal resp %v", err))
		return nil
	}

	return &errResponse
}

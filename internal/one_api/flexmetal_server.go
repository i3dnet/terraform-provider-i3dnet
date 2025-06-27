package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	ContractID        string   `json:"contractId,omitempty"`
	Overflow          bool     `json:"overflow"`
}

type PatchServerReq struct {
	Name              string   `json:"name"`
	Os                OS       `json:"os"`
	SSHKey            []string `json:"sshKey,omitempty"`
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

type OperationStatus struct {
	UUID       string        `json:"uuid"`
	ServerUUID string        `json:"serverUuid"`
	Payload    []interface{} `json:"payload"`
	State      string        `json:"state"`
	CreatedAt  string        `json:"createdAt"`
	UpdatedAt  string        `json:"updatedAt"`
}

type OperationStatusResponse struct {
	ErrorResponse *ErrorResponse
	Command       *Command
}

type Command struct {
	UUID       string        `json:"uuid"`
	ServerUUID string        `json:"server_uuid"`
	Payload    []interface{} `json:"payload"`
	State      string        `json:"state"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
}

type Paginator struct {
	From         int     `json:"from"`
	To           int     `json:"to"`
	Total        int     `json:"total"`
	CurrentPage  int     `json:"current_page"`
	LastPage     int     `json:"last_page"`
	FirstPageURL string  `json:"first_page_url"`
	PrevPageURL  *string `json:"prev_page_url"`
	NextPageURL  *string `json:"next_page_url"`
	LastPageURL  string  `json:"last_page_url"`
	Path         string  `json:"path"`
	PerPage      int     `json:"per_page"`
}

func (c *Client) CreateServer(ctx context.Context, req CreateServerReq) (*ServerResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPost, "flexMetal", "servers", body, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling create flexmetal server api: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
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
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("servers/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling get flexmetal server api: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse

	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
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
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("servers/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse

	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
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

func (c *Client) ReinstallOs(ctx context.Context, serverID string, req PatchServerReq) (*ServerResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := c.callAPI(ctx, http.MethodPatch, flexMetalEndpoint, fmt.Sprintf("servers/%s", serverID), body, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling reinstall flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
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

func (c *Client) GetOperationStatus(ctx context.Context, serverID string) (*OperationStatusResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("servers/%s/commands", serverID), nil, map[string]string{"type": "update-server"})
	if err != nil {
		return nil, fmt.Errorf("error calling get flexmetal server operation API: %w", err)
	}
	defer resp.Body.Close()

	var response OperationStatusResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var commands []Command
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&commands)
	if err != nil {
		panic(err)
	}

	var firstStatus Command
	if len(commands) > 0 {
		firstStatus = commands[0]
		tflog.Debug(ctx, fmt.Sprintf("First operation status:\n%+v\n", firstStatus))
	}

	response.Command = &firstStatus
	return &response, nil
}

func (c *Client) AddTagToServer(ctx context.Context, serverID, tag string) (*ServerResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodPost, flexMetalEndpoint, fmt.Sprintf("servers/%s/tag/%s", serverID, tag), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
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
	resp, err := c.callAPI(ctx, http.MethodDelete, flexMetalEndpoint, fmt.Sprintf("servers/%s/tag/%s", serverID, tag), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete flexmetal server API: %w", err)
	}
	defer resp.Body.Close()

	var response ServerResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
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

func decodeErrResponse(resp *http.Response) *ErrorResponse {
	var errResponse ErrorResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&errResponse); err != nil {
		// if we cannot decode, it means response format is different, so we compute custom ErrorResponse with status code
		errResponse = ErrorResponse{
			ErrorMessage: fmt.Sprintf("Received %d status code.", resp.StatusCode),
		}
	}

	return &errResponse
}

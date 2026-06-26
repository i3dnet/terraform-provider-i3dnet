package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// flexvmNodesPageSize is the page size requested via the RANGED-DATA header when
// listing Cloud nodes.
const flexvmNodesPageSize = 100

// flexvmNodesMaxPages caps the number of pages fetched as a safety net
// against a server that ignores the RANGED-DATA header.
const flexvmNodesMaxPages = 10

// FlexvmNodeObj is the FlexVM Cloud Node representation returned by the API.
type FlexvmNodeObj struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Serial string          `json:"serial"`
	Status string          `json:"status"`
	Cloud  *FlexvmCloudObj `json:"cloud"`
}

// CloudID returns the UUID of the Cloud the node belongs to, or "" if absent.
func (n FlexvmNodeObj) CloudID() string {
	if n.Cloud == nil {
		return ""
	}
	return n.Cloud.ID
}

type FlexvmNodeResponse struct {
	ErrorResponse *ErrorResponse
	Node          *FlexvmNodeObj
}

type FlexvmNodeListResponse struct {
	ErrorResponse *ErrorResponse
	Nodes         []FlexvmNodeObj
}

func (c *Client) FlexvmGetNode(ctx context.Context, cloudID, nodeID string) (*FlexvmNodeResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexVMEndpoint, fmt.Sprintf("clouds/%s/nodes/%s", cloudID, nodeID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm get node API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmNodeResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var node FlexvmNodeObj
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Node = &node
	return &response, nil
}

// FlexvmCreateNode adds a new bare metal Node to the given Cloud. The Node uses
// the Cloud's instance type and location, so no request body is required.
//
// On success, a FlexvmNodeResponse is returned and the creation / provisioning process
// continues on the background until the node's status turns to 'running' or 'failed' (in case
// it failed).
func (c *Client) FlexvmCreateNode(ctx context.Context, cloudID string) (*FlexvmNodeResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodPost, flexVMEndpoint, fmt.Sprintf("clouds/%s/nodes", cloudID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm create node API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmNodeResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	var node FlexvmNodeObj
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	response.Node = &node
	return &response, nil
}

// FlexvmDeleteNode removes a Node from the given Cloud. The API responds with
// 202 Accepted and continues the removal asynchronously.
func (c *Client) FlexvmDeleteNode(ctx context.Context, cloudID, nodeID string) (*FlexvmNodeResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodDelete, flexVMEndpoint, fmt.Sprintf("clouds/%s/nodes/%s", cloudID, nodeID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling flexvm delete node API: %w", err)
	}
	defer resp.Body.Close()

	var response FlexvmNodeResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(resp)
		return &response, nil
	}

	return &response, nil
}

// FlexvmListNodes returns every node in the given Cloud, paging through the
// RANGED-DATA header until all of them are retrieved.
func (c *Client) FlexvmListNodes(ctx context.Context, cloudID string) (*FlexvmNodeListResponse, error) {
	var response FlexvmNodeListResponse
	path := fmt.Sprintf("clouds/%s/nodes", cloudID)

	for start, page := 0, 0; page < flexvmNodesMaxPages; start, page = start+flexvmNodesPageSize, page+1 {
		nodes, errResp, err := c.flexvmListNodesPage(ctx, path, start)
		if err != nil {
			return nil, err
		}
		if errResp != nil {
			response.ErrorResponse = errResp
			return &response, nil
		}

		response.Nodes = append(response.Nodes, nodes...)

		// A page smaller than the requested size means we reached the end.
		if len(nodes) < flexvmNodesPageSize {
			break
		}
	}

	return &response, nil
}

// flexvmListNodesPage fetches a single page of Cloud nodes starting at the given
// offset, using the RANGED-DATA header. It returns the decoded nodes, or an
// *ErrorResponse when the API responds with a status >= 400.
func (c *Client) flexvmListNodesPage(ctx context.Context, path string, start int) ([]FlexvmNodeObj, *ErrorResponse, error) {
	headers := map[string]string{
		"RANGED-DATA": fmt.Sprintf("start=%d,results=%d", start, flexvmNodesPageSize),
	}

	resp, err := c.callAPIWithHeaders(ctx, http.MethodGet, flexVMEndpoint, path, nil, nil, headers)
	if err != nil {
		return nil, nil, fmt.Errorf("error calling flexvm list nodes API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, decodeErrResponse(resp), nil
	}

	var nodes []FlexvmNodeObj
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, nil, fmt.Errorf("error decoding response: %w", err)
	}

	return nodes, nil, nil
}

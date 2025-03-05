package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Tag struct {
	Tag       string `json:"tag"`
	Resources struct {
		Count            int64 `json:"count"`
		FlexMetalServers struct {
			Count int64 `json:"count"`
		} `json:"flexMetalServers"`
	} `json:"resources"`
}

const tagEndpoint = "flexMetal/tags"

// TagResponse can contain Tag in case of a 200 response
// or an ErrorResponse
type TagResponse struct {
	ErrorResponse *ErrorResponse
	Tag           *Tag
}

func (c *Client) CreateTag(ctx context.Context, name string) (*TagResponse, error) {
	req := Tag{Tag: name}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.callAPI(ctx, http.MethodPost, tagEndpoint, "", body)
	if err != nil {
		return nil, fmt.Errorf("error on calling create tag api: %w", err)
	}
	defer resp.Body.Close()

	var response TagResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var tagResp []Tag
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Tag = &tagResp[0]
	return &response, nil
}

func (c *Client) UpdateTag(ctx context.Context, oldName, newName string) (*TagResponse, error) {
	req := Tag{Tag: newName}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.callAPI(ctx, http.MethodPut, tagEndpoint, oldName, body)
	if err != nil {
		return nil, fmt.Errorf("error calling update tag API: %w", err)
	}
	defer resp.Body.Close()

	var response TagResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var tagResp []Tag
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Tag = &tagResp[0]
	return &response, nil
}

func (c *Client) DeleteTag(ctx context.Context, name string) error {
	resp, err := c.callAPI(ctx, http.MethodDelete, tagEndpoint, name, nil)
	if err != nil {
		return fmt.Errorf("error calling delete tag API: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) GetTag(ctx context.Context, name string) (*TagResponse, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, tagEndpoint, name, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete tag API: %w", err)
	}
	defer resp.Body.Close()

	var response TagResponse
	if resp.StatusCode >= 400 {
		response.ErrorResponse = decodeErrResponse(ctx, resp)
		return &response, nil
	}

	var tagResp []Tag
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	response.Tag = &tagResp[0]
	return &response, nil
}

func (c *Client) ListTags(ctx context.Context, name string) ([]Tag, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, tagEndpoint, "", nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling list tags api: %w", err)
	}
	defer resp.Body.Close()

	var tagResp []Tag
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	if name != "" {
		tagResp = filterByName(name, tagResp)
	}

	return tagResp, nil
}

func filterByName(name string, tags []Tag) []Tag {
	var filtered []Tag
	for _, v := range tags {
		if v.Tag != name {
			continue
		}
		filtered = append(filtered, v)
	}
	return filtered
}

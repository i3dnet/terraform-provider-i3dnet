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

func (c *Client) CreateTag(ctx context.Context, name string) (*Tag, error) {
	req := Tag{Tag: name}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.callAPI(ctx, http.MethodPost, tagEndpoint, "", body)
	if err != nil {
		return nil, fmt.Errorf("error on calling create tag api: %w", err)
	}
	defer resp.Close()

	var tagResp []Tag
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &tagResp[0], nil
}

func (c *Client) UpdateTag(ctx context.Context, oldName, newName string) (*Tag, error) {
	req := Tag{Tag: newName}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.callAPI(ctx, http.MethodPut, tagEndpoint, oldName, body)
	if err != nil {
		return nil, fmt.Errorf("error calling update tag API: %w", err)
	}
	defer resp.Close()

	var tagResp []Tag
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &tagResp[0], nil
}

func (c *Client) DeleteTag(ctx context.Context, name string) error {
	resp, err := c.callAPI(ctx, http.MethodDelete, tagEndpoint, name, nil)
	if err != nil {
		return fmt.Errorf("error calling delete tag API: %w", err)
	}
	defer resp.Close()

	return nil
}

func (c *Client) GetTag(ctx context.Context, name string) (*Tag, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, tagEndpoint, name, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling delete tag API: %w", err)
	}
	defer resp.Close()

	var tagResp []Tag
	dec := json.NewDecoder(resp)
	if err := dec.Decode(&tagResp); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if len(tagResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return &tagResp[0], nil
}

func (c *Client) ListTags(ctx context.Context, name string) ([]Tag, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, tagEndpoint, "", nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling list tags api: %w", err)
	}
	defer resp.Close()

	var tagResp []Tag
	dec := json.NewDecoder(resp)
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

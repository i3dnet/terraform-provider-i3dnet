package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VmImage struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	OsFamily string `json:"osFamily"`
	Version  string `json:"version,omitempty"`
}

type VmImageFilters struct {
	Slug     string
	OsFamily string
}

func (c *Client) ListVmImages(ctx context.Context, filters VmImageFilters) ([]VmImage, error) {
	queryParams := map[string]string{}
	if filters.Slug != "" {
		queryParams["slug"] = filters.Slug
	}
	if filters.OsFamily != "" {
		queryParams["osFamily"] = filters.OsFamily
	}

	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, "vm/images", nil, queryParams)
	if err != nil {
		return nil, fmt.Errorf("error calling list vm images API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", decodeErrResponse(resp).ErrorMessage)
	}

	var images []VmImage
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return images, nil
}

func (c *Client) GetVmImage(ctx context.Context, id string) (*VmImage, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("vm/images/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling get vm image API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", decodeErrResponse(resp).ErrorMessage)
	}

	var image VmImage
	if err := json.NewDecoder(resp.Body).Decode(&image); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &image, nil
}

package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VmCapacity struct {
	PoolID    string `json:"poolId"`
	Available int    `json:"available"`
	Total     int    `json:"total"`
}

func (c *Client) GetVmCapacity(ctx context.Context, poolID string) (*VmCapacity, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, fmt.Sprintf("vm/pools/%s/capacity", poolID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling get vm capacity API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", decodeErrResponse(resp).ErrorMessage)
	}

	var capacity VmCapacity
	if err := json.NewDecoder(resp.Body).Decode(&capacity); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &capacity, nil
}

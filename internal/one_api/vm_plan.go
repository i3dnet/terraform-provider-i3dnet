package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type VmPlan struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	CPU      int    `json:"cpu"`
	MemoryGB int    `json:"memoryGb"`
	GpuCount int    `json:"gpuCount"`
	GpuModel string `json:"gpuModel,omitempty"`
}

func (c *Client) ListVmPlans(ctx context.Context) ([]VmPlan, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, flexMetalEndpoint, "vm/plans", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error calling list vm plans API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", decodeErrResponse(resp).ErrorMessage)
	}

	var plans []VmPlan
	if err := json.NewDecoder(resp.Body).Decode(&plans); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return plans, nil
}

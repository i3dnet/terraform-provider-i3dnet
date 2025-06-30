package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Location struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	ShortName        string `json:"shortName"`
	DisplayName      string `json:"displayName"`
	CountryId        int    `json:"countryId"`
	CountryName      string `json:"countryName"`
	CountryShortName string `json:"countryShortName"`
}

const locationsEndpoint = "flexMetal/location"

func (c *Client) ListLocations(ctx context.Context) ([]Location, error) {
	resp, err := c.callAPI(ctx, http.MethodGet, locationsEndpoint, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error on calling list locations api: %w", err)
	}
	defer resp.Body.Close()

	var locationsResp []Location
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&locationsResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(locationsResp) == 0 {
		return nil, fmt.Errorf("unexpected empty response")
	}

	return locationsResp, nil
}

package provider

import (
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Scenario "Infrastructure roles": a role is passed through unchanged, known or not.
func TestIPAddressModelFromResponse(t *testing.T) {
	t.Parallel()

	const (
		ipID     = "01984f2e-3d5e-7f41-9b30-000000000001"
		prefixID = "0b9c2f7e-4d51-4a8e-9f3a-1c2d3e4f5a6b"
	)
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	ip := func(role *string, created *time.Time) network_api.IPAddressResponse {
		return network_api.IPAddressResponse{
			Id:       mustUUID(t, ipID),
			Address:  "203.0.113.1",
			Family:   network_api.IPAddressResponseFamilyN4,
			PrefixId: mustUUID(t, prefixID),
			Role:     role,
			Created:  created,
		}
	}
	str := func(s string) *string { return &s }

	tests := []struct {
		name        string
		in          network_api.IPAddressResponse
		wantRole    types.String
		wantCreated types.String
	}{
		{"infrastructure role", ip(str("gateway"), &created), types.StringValue("gateway"), types.StringValue("2026-09-01T10:00:00Z")},
		{"no role is a customer address", ip(nil, &created), types.StringNull(), types.StringValue("2026-09-01T10:00:00Z")},
		{"unknown role passed through", ip(str("anycast"), &created), types.StringValue("anycast"), types.StringValue("2026-09-01T10:00:00Z")},
		{"created absent", ip(nil, nil), types.StringNull(), types.StringNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := ipAddressModelFromResponse(tt.in)
			wantString(t, "id", m.ID, ipID)
			wantString(t, "address", m.Address, "203.0.113.1")
			wantString(t, "prefix_id", m.PrefixID, prefixID)
			if !m.Family.Equal(types.Int64Value(4)) {
				t.Errorf("family = %v, want 4", m.Family)
			}
			if !m.Role.Equal(tt.wantRole) {
				t.Errorf("role = %v, want %v", m.Role, tt.wantRole)
			}
			if !m.CreatedAt.Equal(tt.wantCreated) {
				t.Errorf("created_at = %v, want %v", m.CreatedAt, tt.wantCreated)
			}
		})
	}
}

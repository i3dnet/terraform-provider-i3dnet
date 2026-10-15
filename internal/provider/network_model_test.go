package provider

import (
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func mustUUID(t *testing.T, s string) openapi_types.UUID {
	t.Helper()

	var id openapi_types.UUID
	if err := id.UnmarshalText([]byte(s)); err != nil {
		t.Fatalf("UnmarshalText(%q): %v", s, err)
	}

	return id
}

func TestNetworkModelFromResponse(t *testing.T) {
	t.Parallel()

	const (
		networkID = "0b9c2f7e-4d51-4a8e-9f3a-1c2d3e4f5a6b"
		domainID  = "7f1e2d3c-4b5a-4968-8776-655443322110"
	)
	name := "db-backend"
	vlan := 1234
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 9, 2, 11, 30, 15, 0, time.FixedZone("CEST", 2*3600))
	domain := mustUUID(t, domainID)

	base := func(connectivity network_api.NetworkConnectivityType) network_api.NetworkResponse {
		return network_api.NetworkResponse{
			Id:           mustUUID(t, networkID),
			Fabric:       "rotterdam-a",
			Connectivity: network_api.NetworkConnectivityResponse{Type: connectivity},
			Addressing:   network_api.NetworkAddressingResponse{Source: network_api.NetworkAddressingResponseSourceProvider},
			State:        network_api.NetworkStateActive,
			Retention:    network_api.Persistent,
			Name:         &name,
			VlanId:       &vlan,
			CreatedAt:    &created,
			UpdatedAt:    &updated,
		}
	}

	tests := []struct {
		name   string
		in     func() network_api.NetworkResponse
		assert func(t *testing.T, m networkModel)
	}{
		{
			name: "public network has no routing",
			in:   func() network_api.NetworkResponse { return base(network_api.NetworkConnectivityTypePublic) },
			assert: func(t *testing.T, m networkModel) {
				wantString(t, "connectivity", m.Connectivity, "public")
				if !m.RoutingEnabled.IsNull() {
					t.Errorf("routing_enabled = %v, want null on a public network", m.RoutingEnabled)
				}
				if !m.RoutingDomainID.IsNull() {
					t.Errorf("routing_domain_id = %v, want null on a public network", m.RoutingDomainID)
				}
			},
		},
		{
			name: "private isolated network",
			in: func() network_api.NetworkResponse {
				n := base(network_api.NetworkConnectivityTypePrivate)
				n.Connectivity.Routing = &network_api.NetworkRouting{Enabled: false}
				return n
			},
			assert: func(t *testing.T, m networkModel) {
				wantString(t, "connectivity", m.Connectivity, "private")
				if !m.RoutingEnabled.Equal(types.BoolValue(false)) {
					t.Errorf("routing_enabled = %v, want false", m.RoutingEnabled)
				}
				if !m.RoutingDomainID.IsNull() {
					t.Errorf("routing_domain_id = %v, want null on an isolated network", m.RoutingDomainID)
				}
			},
		},
		{
			name: "private routed network",
			in: func() network_api.NetworkResponse {
				n := base(network_api.NetworkConnectivityTypePrivate)
				n.Connectivity.Routing = &network_api.NetworkRouting{Enabled: true, RoutingDomainId: &domain}
				return n
			},
			assert: func(t *testing.T, m networkModel) {
				if !m.RoutingEnabled.Equal(types.BoolValue(true)) {
					t.Errorf("routing_enabled = %v, want true", m.RoutingEnabled)
				}
				wantString(t, "routing_domain_id", m.RoutingDomainID, domainID)
			},
		},
		{
			name: "unnamed network",
			in: func() network_api.NetworkResponse {
				n := base(network_api.NetworkConnectivityTypePrivate)
				n.Name = nil
				return n
			},
			assert: func(t *testing.T, m networkModel) {
				if !m.Name.IsNull() {
					t.Errorf("name = %v, want null", m.Name)
				}
			},
		},
		{
			name: "provisioning network has no VLAN yet",
			in: func() network_api.NetworkResponse {
				n := base(network_api.NetworkConnectivityTypePrivate)
				n.State = network_api.NetworkStateProvisioning
				n.VlanId = nil
				return n
			},
			assert: func(t *testing.T, m networkModel) {
				wantString(t, "state", m.State, "provisioning")
				if !m.VlanID.IsNull() {
					t.Errorf("vlan_id = %v, want null while provisioning", m.VlanID)
				}
			},
		},
		{
			name: "every attribute mapped and timestamps in RFC 3339",
			in:   func() network_api.NetworkResponse { return base(network_api.NetworkConnectivityTypePrivate) },
			assert: func(t *testing.T, m networkModel) {
				wantString(t, "id", m.ID, networkID)
				wantString(t, "fabric", m.Fabric, "rotterdam-a")
				wantString(t, "name", m.Name, "db-backend")
				wantString(t, "addressing_source", m.AddressingSource, "provider")
				wantString(t, "state", m.State, "active")
				wantString(t, "retention", m.Retention, "persistent")
				if !m.VlanID.Equal(types.Int64Value(1234)) {
					t.Errorf("vlan_id = %v, want 1234", m.VlanID)
				}
				wantString(t, "created_at", m.CreatedAt, "2026-09-01T10:00:00Z")
				wantString(t, "updated_at", m.UpdatedAt, "2026-09-02T11:30:15+02:00")
			},
		},
		{
			name: "every absent optional field is null",
			in: func() network_api.NetworkResponse {
				return network_api.NetworkResponse{
					Id:           mustUUID(t, networkID),
					Fabric:       "rotterdam-a",
					Connectivity: network_api.NetworkConnectivityResponse{Type: network_api.NetworkConnectivityTypePublic},
					Addressing:   network_api.NetworkAddressingResponse{Source: network_api.NetworkAddressingResponseSourceNone},
					State:        network_api.NetworkStateProvisioning,
					Retention:    network_api.Ephemeral,
				}
			},
			assert: func(t *testing.T, m networkModel) {
				for attr, v := range map[string]interface{ IsNull() bool }{
					"name":              m.Name,
					"vlan_id":           m.VlanID,
					"routing_enabled":   m.RoutingEnabled,
					"routing_domain_id": m.RoutingDomainID,
					"created_at":        m.CreatedAt,
					"updated_at":        m.UpdatedAt,
				} {
					if !v.IsNull() {
						t.Errorf("%s = %v, want null (never \"\" or 0)", attr, v)
					}
				}
				wantString(t, "addressing_source", m.AddressingSource, "none")
				wantString(t, "retention", m.Retention, "ephemeral")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.assert(t, networkModelFromResponse(tt.in()))
		})
	}
}

func wantString(t *testing.T, attr string, got types.String, want string) {
	t.Helper()

	if !got.Equal(types.StringValue(want)) {
		t.Errorf("%s = %v, want %q", attr, got, want)
	}
}

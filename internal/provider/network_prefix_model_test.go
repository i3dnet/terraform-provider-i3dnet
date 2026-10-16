package provider

import (
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPrefixModelFromResponse(t *testing.T) {
	t.Parallel()

	const (
		prefixID  = "01984f2e-2b4c-7e31-8a20-4653f38bdde0"
		networkID = "01984f2e-1c3a-7d21-9d10-3542f27adccf"
	)
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 9, 2, 11, 30, 15, 0, time.FixedZone("CEST", 2*3600))

	full := func() network_api.PrefixResponse {
		gateway := "203.0.113.1"
		routers := []string{"203.0.113.2", "203.0.113.3"}
		vlan := 812
		fabric := "rotterdam-a"
		metro := "rotterdam"
		network := mustUUID(t, networkID)
		retention := network_api.Persistent
		contract := "C-1001"
		description := "web frontends"

		return network_api.PrefixResponse{
			Id:                  mustUUID(t, prefixID),
			Cidr:                "203.0.113.0/28",
			Family:              network_api.PrefixResponseFamilyN4,
			State:               "active",
			GatewayIp:           &gateway,
			RouterIps:           &routers,
			VlanId:              &vlan,
			Fabric:              &fabric,
			Metro:               &metro,
			NetworkId:           &network,
			Retention:           &retention,
			ContractId:          &contract,
			CustomerDescription: &description,
			Created:             &created,
			LastUpdated:         &updated,
		}
	}
	// bare carries only the fields the contract always sends. It covers a
	// provisioning prefix with an empty cidr, a prefix outside the network
	// model and omitted router_ips.
	bare := func() network_api.PrefixResponse {
		return network_api.PrefixResponse{
			Id:     mustUUID(t, prefixID),
			Cidr:   "",
			Family: network_api.PrefixResponseFamilyN6,
			State:  "provisioning",
		}
	}

	tests := []struct {
		name   string
		in     func() network_api.PrefixResponse
		assert func(t *testing.T, m prefixModel)
	}{
		{
			name: "every attribute mapped and timestamps in RFC 3339",
			in:   full,
			assert: func(t *testing.T, m prefixModel) {
				wantString(t, "id", m.ID, prefixID)
				wantString(t, "cidr", m.Cidr, "203.0.113.0/28")
				if !m.Family.Equal(types.Int64Value(4)) {
					t.Errorf("family = %v, want 4", m.Family)
				}
				wantString(t, "state", m.State, "active")
				wantString(t, "gateway_ip", m.GatewayIP, "203.0.113.1")
				if want := types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("203.0.113.2"), types.StringValue("203.0.113.3"),
				}); !m.RouterIPs.Equal(want) {
					t.Errorf("router_ips = %v, want %v", m.RouterIPs, want)
				}
				if !m.VlanID.Equal(types.Int64Value(812)) {
					t.Errorf("vlan_id = %v, want 812", m.VlanID)
				}
				wantString(t, "fabric", m.Fabric, "rotterdam-a")
				wantString(t, "metro", m.Metro, "rotterdam")
				wantString(t, "network_id", m.NetworkID, networkID)
				wantString(t, "retention", m.Retention, "persistent")
				wantString(t, "contract_id", m.ContractID, "C-1001")
				wantString(t, "customer_description", m.CustomerDescription, "web frontends")
				wantString(t, "created_at", m.CreatedAt, "2026-09-01T10:00:00Z")
				wantString(t, "updated_at", m.UpdatedAt, "2026-09-02T11:30:15+02:00")
			},
		},
		{
			name: "router_ips empty",
			in: func() network_api.PrefixResponse {
				p := full()
				p.RouterIps = &[]string{}
				return p
			},
			assert: func(t *testing.T, m prefixModel) {
				if !m.RouterIPs.IsNull() {
					t.Errorf("router_ips = %v, want null (never an empty list)", m.RouterIPs)
				}
			},
		},
		{
			name: "every absent optional field is null",
			in:   bare,
			assert: func(t *testing.T, m prefixModel) {
				for attr, v := range map[string]interface{ IsNull() bool }{
					"cidr":                 m.Cidr,
					"gateway_ip":           m.GatewayIP,
					"router_ips":           m.RouterIPs,
					"vlan_id":              m.VlanID,
					"fabric":               m.Fabric,
					"metro":                m.Metro,
					"network_id":           m.NetworkID,
					"retention":            m.Retention,
					"contract_id":          m.ContractID,
					"customer_description": m.CustomerDescription,
					"created_at":           m.CreatedAt,
					"updated_at":           m.UpdatedAt,
				} {
					if !v.IsNull() {
						t.Errorf("%s = %v, want null (never \"\", 0 or an empty list)", attr, v)
					}
				}
				wantString(t, "id", m.ID, prefixID)
				wantString(t, "state", m.State, "provisioning")
				if !m.Family.Equal(types.Int64Value(6)) {
					t.Errorf("family = %v, want 6", m.Family)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.assert(t, prefixModelFromResponse(tt.in()))
		})
	}
}

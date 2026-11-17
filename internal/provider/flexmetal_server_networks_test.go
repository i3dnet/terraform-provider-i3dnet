package provider

import (
	"context"
	"encoding/json"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_server"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FM-5404: networks on i3dnet_flexmetal_server.

const (
	netPublic  = "0b6f3c2e-6a1d-4f2b-9a51-3c8e2f7d1a10"
	netBackend = "5d2a9e47-1c3b-4e8f-b6a2-7f0d4c9e3b21"
	prefixV4   = "a3c71e5f-8b24-4d6a-9e13-2f5b8c0d7e32"
	prefixV6   = "7b2e5d1a-3c4f-4a8e-b9d6-0e1f2a3b4c76"
)

func TestNetworksToRequestLeavesOutNetworksThatAreNotSet(t *testing.T) {
	ctx := context.Background()

	if got := networksToRequest(ctx, types.ListNull(networksElementType(ctx))); got != nil {
		t.Errorf("null networks = %v, want nil", got)
	}
	if got := networksToRequest(ctx, types.ListUnknown(networksElementType(ctx))); got != nil {
		t.Errorf("unknown networks = %v, want nil", got)
	}
}

func TestNetworksToRequestSendsTheConfiguredNetworks(t *testing.T) {
	ctx := context.Background()
	networks := networksList(ctx,
		// ip_allocations omitted: the default addresses
		planNetwork(ctx, netPublic, false, types.ListUnknown(ipAllocationsElementType(ctx))),
		// [] is layer 2 only; {family} and {prefix_id} as configured
		planNetwork(ctx, netBackend, true, ipAllocationsList(ctx)),
		planNetwork(ctx, netBackend, true, ipAllocationsList(ctx,
			planAllocation(ctx, types.Int64Value(6), types.StringUnknown()),
			planAllocation(ctx, types.Int64Unknown(), types.StringValue(prefixV4)),
		)),
	)

	got, _ := json.Marshal(networksToRequest(ctx, networks))
	want := `[{"networkId":"` + netPublic + `","tagged":false},` +
		`{"networkId":"` + netBackend + `","tagged":true,"ipAllocations":[]},` +
		`{"networkId":"` + netBackend + `","tagged":true,"ipAllocations":[{"family":6},{"prefixId":"` + prefixV4 + `"}]}]`
	if string(got) != want {
		t.Fatalf("request networks =\n%s\nwant\n%s", got, want)
	}
}

func TestNetworksToRequestSendsAnEmptyList(t *testing.T) {
	ctx := context.Background()

	got := networksToRequest(ctx, networksList(ctx))
	if got == nil || len(*got) != 0 {
		t.Fatalf("networks = [] must be sent as [], got %v", got)
	}
}

func TestNetworksRespToPlanFillsEverythingWithoutPriorState(t *testing.T) {
	ctx := context.Background()

	got := networksRespToPlan(ctx, responseNetworks(), types.ListNull(networksElementType(ctx)))

	networks := elements(t, got)
	if len(networks) != 2 {
		t.Fatalf("networks = %d, want 2", len(networks))
	}
	public, backend := networks[0], networks[1]
	assertString(t, "network_id", public.NetworkId, netPublic)
	assertString(t, "name", public.Name, "public")
	if public.VlanId.ValueInt64() != 2602 || public.Tagged.ValueBool() {
		t.Errorf("public network = %+v", public)
	}
	allocations := allocationElements(t, public.IpAllocations)
	if len(allocations) != 2 {
		t.Fatalf("public allocations = %d, want 2", len(allocations))
	}
	assertString(t, "address", allocations[0].Address, "203.0.113.10")
	if allocations[0].Family.ValueInt64() != 4 {
		t.Errorf("family = %v, want 4", allocations[0].Family)
	}
	assertString(t, "prefix_id", allocations[0].PrefixId, prefixV4)
	prefix := prefixOf(t, allocations[1])
	if !prefix.Gateway.IsNull() {
		t.Errorf("a prefix without gateway must have a null gateway, got %v", prefix.Gateway)
	}

	if !backend.Tagged.ValueBool() || !backend.Name.IsNull() || !backend.VlanId.IsNull() ||
		!backend.Fabric.IsNull() || !backend.Connectivity.IsNull() {
		t.Errorf("backend network = %+v", backend)
	}
	if len(allocationElements(t, backend.IpAllocations)) != 0 {
		t.Errorf("a layer 2 only network has no allocations")
	}
}

// The configured values stay verbatim; the other half of {family}/{prefix_id} comes from the response.
func TestNetworksRespToPlanKeepsTheConfiguredValues(t *testing.T) {
	ctx := context.Background()
	prior := networksList(ctx,
		planNetwork(ctx, "0B6F3C2E-6A1D-4F2B-9A51-3C8E2F7D1A10", false, ipAllocationsList(ctx,
			planAllocation(ctx, types.Int64Unknown(), types.StringValue(prefixV4)),
			planAllocation(ctx, types.Int64Value(6), types.StringUnknown()),
		)),
		planNetwork(ctx, netBackend, true, ipAllocationsList(ctx)),
	)
	resp := responseNetworks()
	resp[0].IpAllocations[1] = one_api.ServerNetworkAllocation{ID: "a6", Family: 6, Address: "2a00:1::10",
		Prefix: one_api.ServerNetworkPrefix{ID: prefixV6, Cidr: "2a00:1::/64"}}

	networks := elements(t, networksRespToPlan(ctx, resp, prior))

	assertString(t, "network_id", networks[0].NetworkId, "0B6F3C2E-6A1D-4F2B-9A51-3C8E2F7D1A10")
	allocations := allocationElements(t, networks[0].IpAllocations)
	assertString(t, "configured prefix_id", allocations[0].PrefixId, prefixV4)
	if allocations[0].Family.ValueInt64() != 4 {
		t.Errorf("family from the response = %v, want 4", allocations[0].Family)
	}
	if allocations[1].Family.ValueInt64() != 6 {
		t.Errorf("configured family = %v, want 6", allocations[1].Family)
	}
	assertString(t, "prefix_id from the response", allocations[1].PrefixId, prefixV6)
	assertString(t, "address", allocations[1].Address, "2a00:1::10")
}

func TestNetworksRespToPlanFillsOmittedAllocationsFromTheResponse(t *testing.T) {
	ctx := context.Background()
	prior := networksList(ctx,
		planNetwork(ctx, netPublic, false, types.ListUnknown(ipAllocationsElementType(ctx))),
		planNetwork(ctx, netBackend, true, ipAllocationsList(ctx)),
	)

	networks := elements(t, networksRespToPlan(ctx, responseNetworks(), prior))

	if got := len(allocationElements(t, networks[0].IpAllocations)); got != 2 {
		t.Errorf("default allocations = %d, want 2 from the response", got)
	}
}

// Right after the create request the server has no addresses yet: what was configured is kept, and the rest stays
// null rather than making the next plan replace the server.
func TestNetworksRespToPlanKeepsTheConfigWhileTheResponseHasLess(t *testing.T) {
	ctx := context.Background()
	prior := networksList(ctx,
		planNetwork(ctx, netPublic, false, ipAllocationsList(ctx,
			planAllocation(ctx, types.Int64Value(4), types.StringUnknown()),
		)),
		planNetwork(ctx, netBackend, true, ipAllocationsList(ctx)),
	)

	networks := elements(t, networksRespToPlan(ctx, nil, prior))

	if len(networks) != 2 {
		t.Fatalf("networks = %d, want the 2 configured", len(networks))
	}
	assertString(t, "network_id", networks[0].NetworkId, netPublic)
	if !networks[0].Name.IsNull() || !networks[0].VlanId.IsNull() {
		t.Errorf("unknown details must be null, got %+v", networks[0])
	}
	allocations := allocationElements(t, networks[0].IpAllocations)
	if len(allocations) != 1 || allocations[0].Family.ValueInt64() != 4 || !allocations[0].Address.IsNull() ||
		!allocations[0].PrefixId.IsNull() || !allocations[0].Prefix.IsNull() {
		t.Errorf("allocation = %+v", allocations)
	}
}

func responseNetworks() []one_api.ServerNetwork {
	name, fabric, connectivity, gateway := "public", "ams-01", "public", "203.0.113.1"
	vlan := int64(2602)

	return []one_api.ServerNetwork{
		{
			NetworkID: netPublic, Name: &name, VlanID: &vlan, Fabric: &fabric, Connectivity: &connectivity,
			IpAllocations: []one_api.ServerNetworkAllocation{
				{ID: "a1", Family: 4, Address: "203.0.113.10",
					Prefix: one_api.ServerNetworkPrefix{ID: prefixV4, Cidr: "203.0.113.0/26", Gateway: &gateway}},
				{ID: "a2", Family: 4, Address: "10.0.0.5",
					Prefix: one_api.ServerNetworkPrefix{ID: "p2", Cidr: "10.0.0.0/24"}},
			},
		},
		{NetworkID: netBackend, Tagged: true, IpAllocations: []one_api.ServerNetworkAllocation{}},
	}
}

func planNetwork(ctx context.Context, networkID string, tagged bool, allocations types.List) attr.Value {
	return resource_flexmetal_server.NewNetworksValueMust(
		resource_flexmetal_server.NetworksValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"network_id":     types.StringValue(networkID),
			"tagged":         types.BoolValue(tagged),
			"name":           types.StringUnknown(),
			"vlan_id":        types.Int64Unknown(),
			"fabric":         types.StringUnknown(),
			"connectivity":   types.StringUnknown(),
			"ip_allocations": allocations,
		},
	)
}

func planAllocation(ctx context.Context, family types.Int64, prefixID types.String) attr.Value {
	return resource_flexmetal_server.NewIpAllocationsValueMust(
		resource_flexmetal_server.IpAllocationsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"family":    family,
			"prefix_id": prefixID,
			"id":        types.StringUnknown(),
			"address":   types.StringUnknown(),
			"prefix":    types.ObjectUnknown(resource_flexmetal_server.PrefixValue{}.AttributeTypes(ctx)),
		},
	)
}

func networksList(ctx context.Context, values ...attr.Value) types.List {
	return types.ListValueMust(networksElementType(ctx), values)
}

func ipAllocationsList(ctx context.Context, values ...attr.Value) types.List {
	return types.ListValueMust(ipAllocationsElementType(ctx), values)
}

func elements(t *testing.T, list types.List) []resource_flexmetal_server.NetworksValue {
	t.Helper()
	var out []resource_flexmetal_server.NetworksValue
	for _, v := range list.Elements() {
		out = append(out, v.(resource_flexmetal_server.NetworksValue))
	}

	return out
}

func allocationElements(t *testing.T, list types.List) []resource_flexmetal_server.IpAllocationsValue {
	t.Helper()
	if list.IsNull() || list.IsUnknown() {
		t.Fatalf("ip_allocations is null or unknown")
	}
	var out []resource_flexmetal_server.IpAllocationsValue
	for _, v := range list.Elements() {
		out = append(out, v.(resource_flexmetal_server.IpAllocationsValue))
	}

	return out
}

func prefixOf(t *testing.T, allocation resource_flexmetal_server.IpAllocationsValue) resource_flexmetal_server.PrefixValue {
	t.Helper()
	attrs := allocation.Prefix.Attributes()

	return resource_flexmetal_server.PrefixValue{
		Id:      attrs["id"].(types.String),
		Cidr:    attrs["cidr"].(types.String),
		Gateway: attrs["gateway"].(types.String),
	}
}

func assertString(t *testing.T, name string, got types.String, want string) {
	t.Helper()
	if got.ValueString() != want {
		t.Errorf("%s = %q, want %q", name, got.ValueString(), want)
	}
}

// On refresh, state is what One API reports, so drift shows: added and removed networks, and changed values.
func TestNetworksRefreshFollowsTheResponse(t *testing.T) {
	ctx := context.Background()
	prior := networksRespToPlan(ctx, responseNetworks(), types.ListNull(networksElementType(ctx)))

	// one network fewer, and the remaining one now tagged, with another prefix
	resp := responseNetworks()[:1]
	resp[0].Tagged = true
	resp[0].IpAllocations = resp[0].IpAllocations[:1]
	resp[0].IpAllocations[0].Prefix.ID = prefixV6
	resp[0].IpAllocations[0].Family = 6

	networks := elements(t, networksRefresh(ctx, resp, prior))

	if len(networks) != 1 || !networks[0].Tagged.ValueBool() {
		t.Fatalf("networks = %+v, want the one tagged network the response has", networks)
	}
	allocations := allocationElements(t, networks[0].IpAllocations)
	if len(allocations) != 1 || allocations[0].Family.ValueInt64() != 6 {
		t.Fatalf("allocations = %+v, want the one IPv6 allocation the response has", allocations)
	}
	assertString(t, "prefix_id", allocations[0].PrefixId, prefixV6)

	added := append(responseNetworks(), responseNetworks()[1])
	if got := len(elements(t, networksRefresh(ctx, added, prior))); got != 3 {
		t.Errorf("networks = %d, want the 3 the response has", got)
	}
}

// The same ID in another case is not drift: the configured spelling stays, so the next plan has no diff.
func TestNetworksRefreshKeepsTheSpellingOfTheSameIDs(t *testing.T) {
	ctx := context.Background()
	upperNetwork, upperPrefix := "0B6F3C2E-6A1D-4F2B-9A51-3C8E2F7D1A10", "A3C71E5F-8B24-4D6A-9E13-2F5B8C0D7E32"
	prior := networksList(ctx,
		planNetwork(ctx, upperNetwork, false, ipAllocationsList(ctx,
			planAllocation(ctx, types.Int64Unknown(), types.StringValue(upperPrefix)),
		)),
	)

	networks := elements(t, networksRefresh(ctx, responseNetworks()[:1], prior))

	assertString(t, "network_id", networks[0].NetworkId, upperNetwork)
	allocations := allocationElements(t, networks[0].IpAllocations)
	assertString(t, "prefix_id", allocations[0].PrefixId, upperPrefix)
	assertString(t, "prefix.id from the response", prefixOf(t, allocations[0]).Id, prefixV4)
}

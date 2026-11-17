package one_api

import (
	"encoding/json"
	"testing"
)

// FM-5404: networks in the create request and the server response.

func TestCreateServerReqWithoutNetworksLeavesThemOut(t *testing.T) {
	body := marshal(t, CreateServerReq{Name: "web"})

	if _, ok := body["networks"]; ok {
		t.Fatalf("networks must be left out when not set, got %v", body["networks"])
	}
}

func TestCreateServerReqSendsAnEmptyNetworksList(t *testing.T) {
	// One API refuses it with a clear message, rather than the provider silently using the default network.
	body := marshal(t, CreateServerReq{Name: "web", Networks: &[]NetworkReq{}})

	if got, ok := body["networks"].([]any); !ok || len(got) != 0 {
		t.Fatalf("networks = %v, want []", body["networks"])
	}
}

func TestCreateServerReqKeepsIpAllocationsOmittedApartFromEmpty(t *testing.T) {
	body := marshal(t, CreateServerReq{Networks: &[]NetworkReq{
		{NetworkID: "net-default"},
		{NetworkID: "net-l2", Tagged: true, IpAllocations: &[]IpAllocationReq{}},
	}})
	networks := body["networks"].([]any)

	if _, ok := networks[0].(map[string]any)["ipAllocations"]; ok {
		t.Errorf("ipAllocations must be left out for the default allocation")
	}
	if got, ok := networks[1].(map[string]any)["ipAllocations"].([]any); !ok || len(got) != 0 {
		t.Errorf("ipAllocations = %v, want [] for layer 2 only", networks[1].(map[string]any)["ipAllocations"])
	}
}

func TestCreateServerReqSendsFamilyOrPrefixId(t *testing.T) {
	family := int64(6)
	prefixID := "a3c71e5f-8b24-4d6a-9e13-2f5b8c0d7e32"
	body := marshal(t, CreateServerReq{Networks: &[]NetworkReq{{
		NetworkID:     "net",
		IpAllocations: &[]IpAllocationReq{{Family: &family}, {PrefixID: &prefixID}},
	}}})

	got, _ := json.Marshal(body["networks"].([]any)[0].(map[string]any)["ipAllocations"])
	want := `[{"family":6},{"prefixId":"a3c71e5f-8b24-4d6a-9e13-2f5b8c0d7e32"}]`
	if string(got) != want {
		t.Fatalf("ipAllocations = %s, want %s", got, want)
	}
}

func TestCreateServerReqSendsTagged(t *testing.T) {
	body := marshal(t, CreateServerReq{Networks: &[]NetworkReq{{NetworkID: "a"}, {NetworkID: "b", Tagged: true}}})
	networks := body["networks"].([]any)

	if networks[0].(map[string]any)["tagged"] != false || networks[1].(map[string]any)["tagged"] != true {
		t.Fatalf("tagged = %v, %v; want false, true", networks[0], networks[1])
	}
}

func TestServerReadsNetworksWithNullsAndLayer2Only(t *testing.T) {
	var server Server
	err := json.Unmarshal([]byte(`{"uuid": "u", "networks": [
		{"networkId": "n1", "tagged": false, "name": "public", "vlanId": 2602, "fabric": "ams-01", "connectivity": "public",
		 "ipAllocations": [{"id": "a1", "family": 4, "address": "203.0.113.10",
		                    "prefix": {"id": "p1", "cidr": "203.0.113.0/26", "gateway": "203.0.113.1"}},
		                   {"id": "a2", "family": 4, "address": "10.0.0.5",
		                    "prefix": {"id": "p2", "cidr": "10.0.0.0/24", "gateway": null}}]},
		{"networkId": "n2", "tagged": true, "name": null, "vlanId": null, "fabric": null, "connectivity": null,
		 "ipAllocations": []}
	]}`), &server)
	if err != nil {
		t.Fatal(err)
	}

	if len(server.Networks) != 2 {
		t.Fatalf("networks = %d, want 2", len(server.Networks))
	}
	first, second := server.Networks[0], server.Networks[1]
	if first.NetworkID != "n1" || first.Tagged || *first.Name != "public" || *first.VlanID != 2602 ||
		*first.Fabric != "ams-01" || *first.Connectivity != "public" {
		t.Errorf("first network = %+v", first)
	}
	if first.IpAllocations[0].Address != "203.0.113.10" || *first.IpAllocations[0].Prefix.Gateway != "203.0.113.1" {
		t.Errorf("first allocation = %+v", first.IpAllocations[0])
	}
	if first.IpAllocations[1].Prefix.Gateway != nil {
		t.Errorf("a prefix without gateway must have a nil gateway")
	}
	if !second.Tagged || second.Name != nil || second.VlanID != nil || second.Fabric != nil || second.Connectivity != nil {
		t.Errorf("second network = %+v", second)
	}
	if second.IpAllocations == nil || len(second.IpAllocations) != 0 {
		t.Errorf("a layer 2 only network has an empty ipAllocations, got %v", second.IpAllocations)
	}
}

func marshal(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}

	return body
}

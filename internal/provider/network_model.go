package provider

import (
	"time"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// networkModel holds the API-backed attributes of a network, shared by the
// i3dnet_network and i3dnet_networks data sources.
type networkModel struct {
	ID               types.String `tfsdk:"id"`
	Fabric           types.String `tfsdk:"fabric"`
	Connectivity     types.String `tfsdk:"connectivity"`
	Name             types.String `tfsdk:"name"`
	AddressingSource types.String `tfsdk:"addressing_source"`
	State            types.String `tfsdk:"state"`
	VlanID           types.Int64  `tfsdk:"vlan_id"`
	Retention        types.String `tfsdk:"retention"`
	RoutingEnabled   types.Bool   `tfsdk:"routing_enabled"`
	RoutingDomainID  types.String `tfsdk:"routing_domain_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// networkModelFromResponse maps a network as the API returns it. Fields the
// API omits (name, vlanId while provisioning, routing on public networks,
// timestamps) become null, never an empty string or 0.
func networkModelFromResponse(n network_api.NetworkResponse) networkModel {
	m := networkModel{
		ID:               types.StringValue(n.Id.String()),
		Fabric:           types.StringValue(n.Fabric),
		Connectivity:     types.StringValue(string(n.Connectivity.Type)),
		Name:             types.StringPointerValue(n.Name),
		AddressingSource: types.StringValue(string(n.Addressing.Source)),
		State:            types.StringValue(string(n.State)),
		VlanID:           types.Int64Null(),
		Retention:        types.StringValue(string(n.Retention)),
		RoutingEnabled:   types.BoolNull(),
		RoutingDomainID:  types.StringNull(),
		CreatedAt:        timeValue(n.CreatedAt),
		UpdatedAt:        timeValue(n.UpdatedAt),
	}

	if n.VlanId != nil {
		m.VlanID = types.Int64Value(int64(*n.VlanId))
	}

	if routing := n.Connectivity.Routing; routing != nil {
		m.RoutingEnabled = types.BoolValue(routing.Enabled)
		if routing.RoutingDomainId != nil {
			m.RoutingDomainID = types.StringValue(routing.RoutingDomainId.String())
		}
	}

	return m
}

// timeValue formats t as RFC 3339, null when t is absent.
func timeValue(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}

	return types.StringValue(t.Format(time.RFC3339))
}

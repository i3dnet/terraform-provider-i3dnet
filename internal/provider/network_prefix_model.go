package provider

import (
	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// prefixModel holds the API-backed attributes of a prefix, as the items of
// the i3dnet_network_prefixes data source.
type prefixModel struct {
	ID                  types.String `tfsdk:"id"`
	Cidr                types.String `tfsdk:"cidr"`
	Family              types.Int64  `tfsdk:"family"`
	State               types.String `tfsdk:"state"`
	GatewayIP           types.String `tfsdk:"gateway_ip"`
	RouterIPs           types.List   `tfsdk:"router_ips"`
	VlanID              types.Int64  `tfsdk:"vlan_id"`
	Fabric              types.String `tfsdk:"fabric"`
	Metro               types.String `tfsdk:"metro"`
	NetworkID           types.String `tfsdk:"network_id"`
	Retention           types.String `tfsdk:"retention"`
	ContractID          types.String `tfsdk:"contract_id"`
	CustomerDescription types.String `tfsdk:"customer_description"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// prefixModelFromResponse maps a prefix as the API returns it. Fields the API
// omits become null, never an empty string, 0 or an empty list: network_id on
// a prefix outside the network model, router_ips when no routers are
// recorded, and cidr while the prefix is still provisioning (sent as "").
func prefixModelFromResponse(p network_api.PrefixResponse) prefixModel {
	m := prefixModel{
		ID:                  types.StringValue(p.Id.String()),
		Cidr:                types.StringNull(),
		Family:              types.Int64Value(int64(p.Family)),
		State:               types.StringValue(p.State),
		GatewayIP:           types.StringPointerValue(p.GatewayIp),
		RouterIPs:           types.ListNull(types.StringType),
		VlanID:              types.Int64Null(),
		Fabric:              types.StringPointerValue(p.Fabric),
		Metro:               types.StringPointerValue(p.Metro),
		NetworkID:           types.StringNull(),
		Retention:           types.StringNull(),
		ContractID:          types.StringPointerValue(p.ContractId),
		CustomerDescription: types.StringPointerValue(p.CustomerDescription),
		CreatedAt:           timeValue(p.Created),
		UpdatedAt:           timeValue(p.LastUpdated),
	}

	if p.Cidr != "" {
		m.Cidr = types.StringValue(p.Cidr)
	}

	if p.RouterIps != nil && len(*p.RouterIps) > 0 {
		routerIPs := make([]attr.Value, 0, len(*p.RouterIps))
		for _, ip := range *p.RouterIps {
			routerIPs = append(routerIPs, types.StringValue(ip))
		}
		m.RouterIPs = types.ListValueMust(types.StringType, routerIPs)
	}

	if p.VlanId != nil {
		m.VlanID = types.Int64Value(int64(*p.VlanId))
	}

	if p.NetworkId != nil {
		m.NetworkID = types.StringValue(p.NetworkId.String())
	}

	if p.Retention != nil {
		m.Retention = types.StringValue(string(*p.Retention))
	}

	return m
}

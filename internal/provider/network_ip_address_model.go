package provider

import (
	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ipAddressModel holds an IP address as the items of the
// i3dnet_network_ip_addresses data source.
type ipAddressModel struct {
	ID        types.String `tfsdk:"id"`
	Address   types.String `tfsdk:"address"`
	Family    types.Int64  `tfsdk:"family"`
	PrefixID  types.String `tfsdk:"prefix_id"`
	Role      types.String `tfsdk:"role"`
	CreatedAt types.String `tfsdk:"created_at"`
}

// ipAddressModelFromResponse maps an IP address as the API returns it. role
// is null for an address without an infrastructure role and any other value
// is kept unchanged, including values the provider does not know. created_at
// is null when the API omits it.
func ipAddressModelFromResponse(ip network_api.IPAddressResponse) ipAddressModel {
	return ipAddressModel{
		ID:        types.StringValue(ip.Id.String()),
		Address:   types.StringValue(ip.Address),
		Family:    types.Int64Value(int64(ip.Family)),
		PrefixID:  types.StringValue(ip.PrefixId.String()),
		Role:      types.StringPointerValue(ip.Role),
		CreatedAt: timeValue(ip.Created),
	}
}

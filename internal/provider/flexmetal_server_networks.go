package provider

import (
	"context"
	"strings"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_server"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const networksDescription = "The networks to attach the server to, in this order: untagged or as tagged (802.1Q) " +
	"VLANs, with addresses from the prefixes you choose, or without addresses (`ip_allocations = []`). Leave it out " +
	"for the default network and addresses. The networks and prefixes must already exist: create them with the " +
	"[Dedicated Networks API](https://docs.i3d.net/connectivity/dedicated-networks-api) and refer to them with the " +
	"`i3dnet_network`, `i3dnet_networks` and `i3dnet_network_prefixes` data sources. Changing `networks` replaces the " +
	"server. A network created just before the apply can still be provisioning; the delivery waits for it, so raise " +
	"`timeouts.create` in that case."

// networksSchema adjusts the generated networks attribute (FM-5404): tagged defaults to false, every computed value
// keeps its state value (otherwise it would show as "known after apply" on every plan and, with replace-on-change,
// replace the server on every apply), and any change replaces the server, as the server PATCH can't change networks.
func networksSchema(generated schema.ListNestedAttribute) schema.ListNestedAttribute {
	networks := generated.NestedObject.Attributes

	tagged := networks["tagged"].(schema.BoolAttribute)
	tagged.Default = booldefault.StaticBool(false)
	networks["tagged"] = tagged

	for _, name := range []string{"name", "fabric", "connectivity"} {
		networks[name] = withStringUseState(networks[name].(schema.StringAttribute))
	}
	vlanID := networks["vlan_id"].(schema.Int64Attribute)
	vlanID.PlanModifiers = append(vlanID.PlanModifiers, int64planmodifier.UseStateForUnknown())
	networks["vlan_id"] = vlanID

	allocations := networks["ip_allocations"].(schema.ListNestedAttribute)
	allocations.PlanModifiers = append(allocations.PlanModifiers, listplanmodifier.UseStateForUnknown())
	allocations.Description = "The addresses the server gets in this network, in this order. Leave it out for the " +
		"default addresses; `[]` means layer 2 only. Each entry sets either `family` (4 or 6) or `prefix_id`; the " +
		"other is filled in from the response."
	allocations.MarkdownDescription = allocations.Description
	allocationAttributes := allocations.NestedObject.Attributes
	family := allocationAttributes["family"].(schema.Int64Attribute)
	family.PlanModifiers = append(family.PlanModifiers, int64planmodifier.UseStateForUnknown())
	allocationAttributes["family"] = family
	for _, name := range []string{"prefix_id", "id", "address"} {
		allocationAttributes[name] = withStringUseState(allocationAttributes[name].(schema.StringAttribute))
	}
	prefix := allocationAttributes["prefix"].(schema.SingleNestedAttribute)
	prefix.PlanModifiers = append(prefix.PlanModifiers, objectplanmodifier.UseStateForUnknown())
	allocationAttributes["prefix"] = prefix
	networks["ip_allocations"] = allocations

	// UseStateForUnknown first, so a value Terraform marks unknown is restored before the replace check compares it.
	generated.PlanModifiers = append(generated.PlanModifiers,
		listplanmodifier.UseStateForUnknown(), listplanmodifier.RequiresReplace())
	generated.Description = networksDescription
	generated.MarkdownDescription = networksDescription

	return generated
}

func withStringUseState(attribute schema.StringAttribute) schema.StringAttribute {
	attribute.PlanModifiers = append([]planmodifier.String{}, attribute.PlanModifiers...)
	attribute.PlanModifiers = append(attribute.PlanModifiers, stringplanmodifier.UseStateForUnknown())

	return attribute
}

// FM-5404: networks on i3dnet_flexmetal_server. Networks and their allocations come back in request order, so plan
// and response are matched by position.

func networksElementType(ctx context.Context) attr.Type {
	return resource_flexmetal_server.NetworksValue{}.Type(ctx)
}

func ipAllocationsElementType(ctx context.Context) attr.Type {
	return resource_flexmetal_server.IpAllocationsValue{}.Type(ctx)
}

func known(v attr.Value) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// networksToRequest turns the planned networks into the create request. Only what the config sets is sent: networks
// left out are nil (One API picks the default network), ip_allocations left out are nil (the default addresses in
// that network), and an empty list stays [] (layer 2 only, or no networks at all, which One API refuses).
func networksToRequest(ctx context.Context, networks types.List) *[]one_api.NetworkReq {
	if !known(networks) {
		return nil
	}

	request := []one_api.NetworkReq{}
	for _, element := range networks.Elements() {
		network := element.(resource_flexmetal_server.NetworksValue)
		networkReq := one_api.NetworkReq{
			NetworkID: network.NetworkId.ValueString(),
			Tagged:    network.Tagged.ValueBool(),
		}

		if known(network.IpAllocations) {
			allocations := []one_api.IpAllocationReq{}
			for _, a := range network.IpAllocations.Elements() {
				allocation := a.(resource_flexmetal_server.IpAllocationsValue)
				allocationReq := one_api.IpAllocationReq{}
				if known(allocation.Family) {
					family := allocation.Family.ValueInt64()
					allocationReq.Family = &family
				}
				if known(allocation.PrefixId) {
					prefixID := allocation.PrefixId.ValueString()
					allocationReq.PrefixID = &prefixID
				}
				allocations = append(allocations, allocationReq)
			}
			networkReq.IpAllocations = &allocations
		}

		request = append(request, networkReq)
	}

	return &request
}

// networksRespToPlan builds the networks for state from the response, keeping what the config set verbatim:
// network_id, tagged, and a configured family or prefix_id. Without prior networks (import, a server created without
// them) everything comes from the response. Values the response doesn't have yet, e.g. right after the create request,
// are null, so a state saved early still matches the config on the next plan.
func networksRespToPlan(ctx context.Context, response []one_api.ServerNetwork, prior types.List) types.List {
	if !known(prior) {
		values := make([]attr.Value, 0, len(response))
		for i := range response {
			values = append(values, networkValue(ctx, &response[i], nil))
		}

		return types.ListValueMust(networksElementType(ctx), values)
	}

	values := make([]attr.Value, 0, len(prior.Elements()))
	for i, element := range prior.Elements() {
		configured := element.(resource_flexmetal_server.NetworksValue)
		var network *one_api.ServerNetwork
		if i < len(response) {
			network = &response[i]
		}
		values = append(values, networkValue(ctx, network, &configured))
	}

	return types.ListValueMust(networksElementType(ctx), values)
}

func networkValue(
	ctx context.Context,
	network *one_api.ServerNetwork,
	configured *resource_flexmetal_server.NetworksValue,
) attr.Value {
	networkID, tagged := types.StringNull(), types.BoolNull()
	name, fabric, connectivity, vlanID := types.StringNull(), types.StringNull(), types.StringNull(), types.Int64Null()
	var allocations []one_api.ServerNetworkAllocation
	if network != nil {
		networkID, tagged = types.StringValue(network.NetworkID), types.BoolValue(network.Tagged)
		name, fabric = types.StringPointerValue(network.Name), types.StringPointerValue(network.Fabric)
		connectivity, vlanID = types.StringPointerValue(network.Connectivity), types.Int64PointerValue(network.VlanID)
		allocations = network.IpAllocations
	}

	priorAllocations := types.ListNull(ipAllocationsElementType(ctx))
	if configured != nil {
		if known(configured.NetworkId) {
			networkID = configured.NetworkId
		}
		if known(configured.Tagged) {
			tagged = configured.Tagged
		}
		priorAllocations = configured.IpAllocations
	}

	return resource_flexmetal_server.NewNetworksValueMust(
		resource_flexmetal_server.NetworksValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"network_id":     networkID,
			"tagged":         tagged,
			"name":           name,
			"vlan_id":        vlanID,
			"fabric":         fabric,
			"connectivity":   connectivity,
			"ip_allocations": allocationsValue(ctx, allocations, network != nil, priorAllocations),
		},
	)
}

// allocationsValue is like networksRespToPlan, one level down. hasResponse tells an absent network (null when the
// config left ip_allocations out) from one without addresses ([]).
func allocationsValue(
	ctx context.Context,
	response []one_api.ServerNetworkAllocation,
	hasResponse bool,
	prior types.List,
) types.List {
	if !known(prior) {
		if !hasResponse {
			return types.ListNull(ipAllocationsElementType(ctx))
		}
		values := make([]attr.Value, 0, len(response))
		for i := range response {
			values = append(values, allocationValue(ctx, &response[i], nil))
		}

		return types.ListValueMust(ipAllocationsElementType(ctx), values)
	}

	values := make([]attr.Value, 0, len(prior.Elements()))
	for i, element := range prior.Elements() {
		configured := element.(resource_flexmetal_server.IpAllocationsValue)
		var allocation *one_api.ServerNetworkAllocation
		if i < len(response) {
			allocation = &response[i]
		}
		values = append(values, allocationValue(ctx, allocation, &configured))
	}

	return types.ListValueMust(ipAllocationsElementType(ctx), values)
}

func allocationValue(
	ctx context.Context,
	allocation *one_api.ServerNetworkAllocation,
	configured *resource_flexmetal_server.IpAllocationsValue,
) attr.Value {
	prefixTypes := resource_flexmetal_server.PrefixValue{}.AttributeTypes(ctx)
	family, prefixID, id, address := types.Int64Null(), types.StringNull(), types.StringNull(), types.StringNull()
	prefix := types.ObjectNull(prefixTypes)
	if allocation != nil {
		family, prefixID = types.Int64Value(allocation.Family), types.StringValue(allocation.Prefix.ID)
		id, address = types.StringValue(allocation.ID), types.StringValue(allocation.Address)
		prefix = types.ObjectValueMust(prefixTypes, map[string]attr.Value{
			"id":      types.StringValue(allocation.Prefix.ID),
			"cidr":    types.StringValue(allocation.Prefix.Cidr),
			"gateway": types.StringPointerValue(allocation.Prefix.Gateway),
		})
	}

	if configured != nil {
		if known(configured.Family) {
			family = configured.Family
		}
		if known(configured.PrefixId) {
			prefixID = configured.PrefixId
		}
	}

	return resource_flexmetal_server.NewIpAllocationsValueMust(
		resource_flexmetal_server.IpAllocationsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"family":    family,
			"prefix_id": prefixID,
			"id":        id,
			"address":   address,
			"prefix":    prefix,
		},
	)
}

// expectsAddresses tells whether the requested networks give the server any address: the default network does, and
// so does a network with default (left out) or explicit allocations. Only all-layer-2 networks give none.
func expectsAddresses(networks types.List) bool {
	if !known(networks) {
		return true
	}
	for _, element := range networks.Elements() {
		allocations := element.(resource_flexmetal_server.NetworksValue).IpAllocations
		if !known(allocations) || len(allocations.Elements()) > 0 {
			return true
		}
	}

	return false
}

// networksRefresh is the state on refresh, update and import: what One API reports, so drift shows (networks or
// addresses added or removed, a changed tagged, family or prefix). Only the spelling of network_id and prefix_id
// in prior is kept where it's the same ID in another case, so a configured spelling doesn't show as a diff.
func networksRefresh(ctx context.Context, response []one_api.ServerNetwork, prior types.List) types.List {
	var priorNetworks []resource_flexmetal_server.NetworksValue
	if known(prior) {
		for _, element := range prior.Elements() {
			priorNetworks = append(priorNetworks, element.(resource_flexmetal_server.NetworksValue))
		}
	}

	values := make([]attr.Value, 0, len(response))
	for i := range response {
		network := response[i]
		var priorAllocations []resource_flexmetal_server.IpAllocationsValue
		if i < len(priorNetworks) {
			network.NetworkID = sameSpelling(priorNetworks[i].NetworkId, network.NetworkID)
			if known(priorNetworks[i].IpAllocations) {
				for _, a := range priorNetworks[i].IpAllocations.Elements() {
					priorAllocations = append(priorAllocations, a.(resource_flexmetal_server.IpAllocationsValue))
				}
			}
		}

		allocations := make([]attr.Value, 0, len(network.IpAllocations))
		for j := range network.IpAllocations {
			allocation := allocationValue(ctx, &network.IpAllocations[j], nil).(resource_flexmetal_server.IpAllocationsValue)
			if j < len(priorAllocations) {
				allocation.PrefixId = types.StringValue(sameSpelling(priorAllocations[j].PrefixId, allocation.PrefixId.ValueString()))
			}
			allocations = append(allocations, allocation)
		}

		value := networkValue(ctx, &network, nil).(resource_flexmetal_server.NetworksValue)
		value.IpAllocations = types.ListValueMust(ipAllocationsElementType(ctx), allocations)
		values = append(values, value)
	}

	return types.ListValueMust(networksElementType(ctx), values)
}

// sameSpelling returns prior's spelling when it is the same ID as returned, ignoring case, else returned.
func sameSpelling(prior types.String, returned string) string {
	if known(prior) && strings.EqualFold(prior.ValueString(), returned) {
		return prior.ValueString()
	}

	return returned
}

package datasource_flexmetal_vm_pool

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VmPoolSubnetValue struct {
	CIDR       types.String `tfsdk:"cidr"`
	Gateway    types.String `tfsdk:"gateway"`
	RangeStart types.String `tfsdk:"range_start"`
	RangeEnd   types.String `tfsdk:"range_end"`
}

type VmPoolValue struct {
	ID           types.String        `tfsdk:"id"`
	Name         types.String        `tfsdk:"name"`
	LocationID   types.String        `tfsdk:"location_id"`
	ContractID   types.String        `tfsdk:"contract_id"`
	Type         types.String        `tfsdk:"type"`
	InstanceType types.String        `tfsdk:"instance_type"`
	VlanID       types.Int64         `tfsdk:"vlan_id"`
	Subnet       []VmPoolSubnetValue `tfsdk:"subnet"`
	Status       types.String        `tfsdk:"status"`
}

type VmPoolsData struct {
	Name       types.String  `tfsdk:"name"`
	LocationID types.String  `tfsdk:"location_id"`
	Status     types.String  `tfsdk:"status"`
	Pools      []VmPoolValue `tfsdk:"pools"`
}

func VmPoolDataSourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Lists FlexMetal VM pools, optionally filtered by name, location, or status.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Filter pools by name.",
			},
			"location_id": schema.StringAttribute{
				Optional:    true,
				Description: "Filter pools by location ID.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Description: "Filter pools by status.",
			},
			"pools": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of VM pools.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Pool identifier.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Pool name.",
						},
						"location_id": schema.StringAttribute{
							Computed:    true,
							Description: "Location ID.",
						},
						"contract_id": schema.StringAttribute{
							Computed:    true,
							Description: "Contract ID.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Pool type.",
						},
						"instance_type": schema.StringAttribute{
							Computed:    true,
							Description: "Instance type.",
						},
						"vlan_id": schema.Int64Attribute{
							Computed:    true,
							Description: "VLAN ID.",
						},
						"subnet": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Subnet configuration.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"cidr": schema.StringAttribute{
										Computed:    true,
										Description: "CIDR block.",
									},
									"gateway": schema.StringAttribute{
										Computed:    true,
										Description: "Gateway IP.",
									},
									"range_start": schema.StringAttribute{
										Computed:    true,
										Description: "Start of IP range.",
									},
									"range_end": schema.StringAttribute{
										Computed:    true,
										Description: "End of IP range.",
									},
								},
							},
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Pool status.",
						},
					},
				},
			},
		},
	}
}

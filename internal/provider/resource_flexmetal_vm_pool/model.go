package resource_flexmetal_vm_pool

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SubnetValue struct {
	CIDR       types.String `tfsdk:"cidr"`
	Gateway    types.String `tfsdk:"gateway"`
	RangeStart types.String `tfsdk:"range_start"`
	RangeEnd   types.String `tfsdk:"range_end"`
}

type FlexmetalVmPoolModel struct {
	ID           types.String  `tfsdk:"id"`
	Name         types.String  `tfsdk:"name"`
	LocationID   types.String  `tfsdk:"location_id"`
	ContractID   types.String  `tfsdk:"contract_id"`
	Type         types.String  `tfsdk:"type"`
	InstanceType types.String  `tfsdk:"instance_type"`
	VlanID       types.Int64   `tfsdk:"vlan_id"`
	Subnet       []SubnetValue `tfsdk:"subnet"`
	Metadata     types.Map     `tfsdk:"metadata"`
	Status       types.String  `tfsdk:"status"`
}

func FlexmetalVmPoolResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "FlexMetal VM Pool resource. A pool groups VM instances and defines the network configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the VM pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the VM pool.",
			},
			"location_id": schema.StringAttribute{
				Required:    true,
				Description: "Location where the pool is deployed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"contract_id": schema.StringAttribute{
				Required:    true,
				Description: "Contract ID associated with this pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Pool type (e.g. on_demand).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_type": schema.StringAttribute{
				Required:    true,
				Description: "Bare-metal instance type backing this pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vlan_id": schema.Int64Attribute{
				Required:    true,
				Description: "VLAN ID for the pool network.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"subnet": schema.ListNestedAttribute{
				Required:    true,
				Description: "Subnet configuration for the pool.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr": schema.StringAttribute{
							Required:    true,
							Description: "CIDR block of the subnet.",
						},
						"gateway": schema.StringAttribute{
							Required:    true,
							Description: "Gateway IP address.",
						},
						"range_start": schema.StringAttribute{
							Required:    true,
							Description: "Start of the IP range.",
						},
						"range_end": schema.StringAttribute{
							Required:    true,
							Description: "End of the IP range.",
						},
					},
				},
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary key-value metadata for the pool.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current status of the pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

package resource_flexmetal_vm

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OsValue struct {
	ImageID types.String `tfsdk:"image_id"`
}

type FlexmetalVmModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	PoolID        types.String `tfsdk:"pool_id"`
	Plan          types.String `tfsdk:"plan"`
	Os            OsValue      `tfsdk:"os"`
	UserData      types.String `tfsdk:"user_data"`
	Tags          types.List   `tfsdk:"tags"`
	Status        types.String `tfsdk:"status"`
	IPAddress     types.String `tfsdk:"ip_address"`
	IPAddressV6   types.String `tfsdk:"ip_address_v6"`
	Gateway       types.String `tfsdk:"gateway"`
	Netmask       types.String `tfsdk:"netmask"`
	VlanID        types.Int64  `tfsdk:"vlan_id"`
	ProvisionedAt types.String `tfsdk:"provisioned_at"`
}

func FlexmetalVmResourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "FlexMetal VM instance resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the VM instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the VM instance.",
			},
			"pool_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the VM pool this instance belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"plan": schema.StringAttribute{
				Required:    true,
				Description: "VM plan (size/type).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"os": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Operating system configuration.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"image_id": schema.StringAttribute{
						Required:    true,
						Description: "Image identifier to use for the VM.",
					},
				},
			},
			"user_data": schema.StringAttribute{
				Optional:    true,
				Description: "Cloud-init user data.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of tags associated with the VM.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current status of the VM instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address": schema.StringAttribute{
				Computed:    true,
				Description: "IPv4 address assigned to the VM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address_v6": schema.StringAttribute{
				Computed:    true,
				Description: "IPv6 address assigned to the VM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway": schema.StringAttribute{
				Computed:    true,
				Description: "Default gateway for the VM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"netmask": schema.StringAttribute{
				Computed:    true,
				Description: "Netmask for the VM network.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vlan_id": schema.Int64Attribute{
				Computed:    true,
				Description: "VLAN ID the VM is connected to.",
			},
			"provisioned_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the VM was provisioned.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

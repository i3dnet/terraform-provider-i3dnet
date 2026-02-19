package datasource_flexmetal_vm_plan

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VmPlanValue struct {
	Slug     types.String `tfsdk:"slug"`
	Name     types.String `tfsdk:"name"`
	CPU      types.Int64  `tfsdk:"cpu"`
	MemoryGB types.Int64  `tfsdk:"memory_gb"`
	GpuCount types.Int64  `tfsdk:"gpu_count"`
	GpuModel types.String `tfsdk:"gpu_model"`
}

type VmPlansData struct {
	Slug     types.String  `tfsdk:"slug"`
	GpuCount types.Int64   `tfsdk:"gpu_count"`
	Plans    []VmPlanValue `tfsdk:"plans"`
}

func VmPlanDataSourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Lists available FlexMetal VM plans, optionally filtered by slug or GPU count.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Optional:    true,
				Description: "Filter plans by slug.",
			},
			"gpu_count": schema.Int64Attribute{
				Optional:    true,
				Description: "Filter plans by GPU count.",
			},
			"plans": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of VM plans.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug": schema.StringAttribute{
							Computed:    true,
							Description: "Plan slug identifier.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Human-readable plan name.",
						},
						"cpu": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of vCPUs.",
						},
						"memory_gb": schema.Int64Attribute{
							Computed:    true,
							Description: "Memory in gigabytes.",
						},
						"gpu_count": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of GPUs.",
						},
						"gpu_model": schema.StringAttribute{
							Computed:    true,
							Description: "GPU model name.",
						},
					},
				},
			},
		},
	}
}

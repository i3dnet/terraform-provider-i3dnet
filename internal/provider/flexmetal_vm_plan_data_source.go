package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/datasource_flexmetal_vm_plan"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*vmPlanDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vmPlanDataSource)(nil)
)

func NewVmPlanDataSource() datasource.DataSource {
	return &vmPlanDataSource{}
}

type vmPlanDataSource struct {
	client *one_api.Client
}

func (d *vmPlanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_vm_plan"
}

func (d *vmPlanDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_flexmetal_vm_plan.VmPlanDataSourceSchema(ctx)
}

func (d *vmPlanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*one_api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *one_api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *vmPlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_flexmetal_vm_plan.VmPlansData

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plans, err := d.client.ListVmPlans(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list VM plans", err.Error())
		return
	}

	var planValues []datasource_flexmetal_vm_plan.VmPlanValue
	for _, p := range plans {
		if !data.Slug.IsNull() && p.Slug != data.Slug.ValueString() {
			continue
		}
		if !data.GpuCount.IsNull() && int64(p.GpuCount) != data.GpuCount.ValueInt64() {
			continue
		}
		planValues = append(planValues, datasource_flexmetal_vm_plan.VmPlanValue{
			Slug:     types.StringValue(p.Slug),
			Name:     types.StringValue(p.Name),
			CPU:      types.Int64Value(int64(p.CPU)),
			MemoryGB: types.Int64Value(int64(p.MemoryGB)),
			GpuCount: types.Int64Value(int64(p.GpuCount)),
			GpuModel: types.StringValue(p.GpuModel),
		})
	}

	data.Plans = planValues
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

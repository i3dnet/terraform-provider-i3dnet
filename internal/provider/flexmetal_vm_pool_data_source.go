package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/datasource_flexmetal_vm_pool"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*vmPoolDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vmPoolDataSource)(nil)
)

func NewVmPoolDataSource() datasource.DataSource {
	return &vmPoolDataSource{}
}

type vmPoolDataSource struct {
	client *one_api.Client
}

func (d *vmPoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_vm_pools"
}

func (d *vmPoolDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_flexmetal_vm_pool.VmPoolDataSourceSchema(ctx)
}

func (d *vmPoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vmPoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_flexmetal_vm_pool.VmPoolsData

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pools, err := d.client.ListVmPools(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list VM pools", err.Error())
		return
	}

	var poolValues []datasource_flexmetal_vm_pool.VmPoolValue
	for _, p := range pools {
		if !data.Name.IsNull() && p.Name != data.Name.ValueString() {
			continue
		}
		if !data.LocationID.IsNull() && p.LocationID != data.LocationID.ValueString() {
			continue
		}
		if !data.Status.IsNull() && p.Status != data.Status.ValueString() {
			continue
		}

		subnets := make([]datasource_flexmetal_vm_pool.VmPoolSubnetValue, len(p.Subnet))
		for i, s := range p.Subnet {
			subnets[i] = datasource_flexmetal_vm_pool.VmPoolSubnetValue{
				CIDR:       types.StringValue(s.CIDR),
				Gateway:    types.StringValue(s.Gateway),
				RangeStart: types.StringValue(s.RangeStart),
				RangeEnd:   types.StringValue(s.RangeEnd),
			}
		}

		poolValues = append(poolValues, datasource_flexmetal_vm_pool.VmPoolValue{
			ID:           types.StringValue(p.ID),
			Name:         types.StringValue(p.Name),
			LocationID:   types.StringValue(p.LocationID),
			ContractID:   types.StringValue(p.ContractID),
			Type:         types.StringValue(p.Type),
			InstanceType: types.StringValue(p.InstanceType),
			VlanID:       types.Int64Value(p.VlanID),
			Subnet:       subnets,
			Status:       types.StringValue(p.Status),
		})
	}

	data.Pools = poolValues
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_vm_pool"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*vmPoolResource)(nil)
var _ resource.ResourceWithConfigure = (*vmPoolResource)(nil)
var _ resource.ResourceWithImportState = (*vmPoolResource)(nil)

func NewVmPoolResource() resource.Resource {
	return &vmPoolResource{}
}

type vmPoolResource struct {
	client *one_api.Client
}

func (r *vmPoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_vm_pool"
}

func (r *vmPoolResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_flexmetal_vm_pool.FlexmetalVmPoolResourceSchema(ctx)
}

func (r *vmPoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*one_api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *one_api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *vmPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_flexmetal_vm_pool.FlexmetalVmPoolModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := one_api.CreateVmPoolRequest{
		Name:         data.Name.ValueString(),
		LocationID:   data.LocationID.ValueString(),
		ContractID:   data.ContractID.ValueString(),
		Type:         data.Type.ValueString(),
		InstanceType: data.InstanceType.ValueString(),
		VlanID:       data.VlanID.ValueInt64(),
		Subnet:       subnetsToAPI(data.Subnet),
		Metadata:     mapFromTF(ctx, data.Metadata),
	}

	poolResp, err := r.client.CreateVmPool(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating VM pool", err.Error())
		return
	}
	if poolResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating VM pool", poolResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmPoolRespToModel(ctx, poolResp.Pool, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_flexmetal_vm_pool.FlexmetalVmPoolModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolResp, err := r.client.GetVmPool(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading VM pool", err.Error())
		return
	}
	if poolResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error reading VM pool", poolResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmPoolRespToModel(ctx, poolResp.Pool, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_flexmetal_vm_pool.FlexmetalVmPoolModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := one_api.UpdateVmPoolRequest{
		Name:     data.Name.ValueString(),
		Metadata: mapFromTF(ctx, data.Metadata),
	}

	poolResp, err := r.client.UpdateVmPool(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating VM pool", err.Error())
		return
	}
	if poolResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error updating VM pool", poolResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmPoolRespToModel(ctx, poolResp.Pool, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_flexmetal_vm_pool.FlexmetalVmPoolModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolResp, err := r.client.DeleteVmPool(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting VM pool", err.Error())
		return
	}
	if poolResp != nil && poolResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error deleting VM pool", poolResp.ErrorResponse, &resp.Diagnostics)
	}
}

func (r *vmPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func vmPoolRespToModel(ctx context.Context, pool *one_api.VmPool, data *resource_flexmetal_vm_pool.FlexmetalVmPoolModel) {
	data.ID = types.StringValue(pool.ID)
	data.Name = types.StringValue(pool.Name)
	data.LocationID = types.StringValue(pool.LocationID)
	data.ContractID = types.StringValue(pool.ContractID)
	data.Type = types.StringValue(pool.Type)
	data.InstanceType = types.StringValue(pool.InstanceType)
	data.VlanID = types.Int64Value(pool.VlanID)
	data.Status = types.StringValue(pool.Status)

	subnets := make([]resource_flexmetal_vm_pool.SubnetValue, len(pool.Subnet))
	for i, s := range pool.Subnet {
		subnets[i] = resource_flexmetal_vm_pool.SubnetValue{
			CIDR:       types.StringValue(s.CIDR),
			Gateway:    types.StringValue(s.Gateway),
			RangeStart: types.StringValue(s.RangeStart),
			RangeEnd:   types.StringValue(s.RangeEnd),
		}
	}
	data.Subnet = subnets

	if len(pool.Metadata) > 0 {
		metadataMap, diags := types.MapValueFrom(ctx, types.StringType, pool.Metadata)
		if !diags.HasError() {
			data.Metadata = metadataMap
		}
	}
}

func subnetsToAPI(subnets []resource_flexmetal_vm_pool.SubnetValue) []one_api.VmPoolSubnet {
	result := make([]one_api.VmPoolSubnet, len(subnets))
	for i, s := range subnets {
		result[i] = one_api.VmPoolSubnet{
			CIDR:       s.CIDR.ValueString(),
			Gateway:    s.Gateway.ValueString(),
			RangeStart: s.RangeStart.ValueString(),
			RangeEnd:   s.RangeEnd.ValueString(),
		}
	}
	return result
}

func mapFromTF(ctx context.Context, m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	result := make(map[string]string)
	m.ElementsAs(ctx, &result, false)
	return result
}

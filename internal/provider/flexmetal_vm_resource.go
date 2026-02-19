package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_vm"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*vmResource)(nil)
var _ resource.ResourceWithConfigure = (*vmResource)(nil)
var _ resource.ResourceWithImportState = (*vmResource)(nil)

func NewVmResource() resource.Resource {
	return &vmResource{}
}

type vmResource struct {
	client *one_api.Client
}

type vmModel struct {
	resource_flexmetal_vm.FlexmetalVmModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_vm"
}

func (r *vmResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_flexmetal_vm.FlexmetalVmResourceSchema(ctx)
	s.Attributes["timeouts"] = timeouts.Attributes(ctx, timeouts.Opts{
		Create: true,
		Delete: true,
	})
	resp.Schema = s
}

func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data vmModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, 15*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	var tags []string
	for _, tag := range data.Tags.Elements() {
		tags = append(tags, strings.Trim(tag.String(), "\""))
	}

	apiReq := one_api.CreateVmInstanceRequest{
		Name:     data.Name.ValueString(),
		PoolID:   data.PoolID.ValueString(),
		Plan:     data.Plan.ValueString(),
		OS:       one_api.VmInstanceOS{ImageID: data.Os.ImageID.ValueString()},
		UserData: data.UserData.ValueString(),
		Tags:     tags,
	}

	instanceResp, err := r.client.CreateVmInstance(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating VM instance", err.Error())
		return
	}
	if instanceResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating VM instance", instanceResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmRespToModel(instanceResp.Instance, &data.FlexmetalVmModel)

	// Save state early to prevent dangling VMs on timeout
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID := data.ID.ValueString()
	lastStatus := data.Status.ValueString()

	err = r.waitForVmStatus(ctx, vmID, []string{"running", "error"}, 15*time.Second, func(i *one_api.VmInstance) {
		lastStatus = i.Status
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for VM to be ready",
			fmt.Sprintf("Error: %v\nLast status: %s\nVM id: %s", err, lastStatus, vmID),
		)
		return
	}

	if lastStatus == "error" {
		resp.Diagnostics.AddError(
			"VM creation failed",
			fmt.Sprintf("VM reached error status. VM id: %s", vmID),
		)
		return
	}

	getResp, err := r.client.GetVmInstance(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting VM instance", err.Error())
		return
	}
	if getResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error getting VM instance", getResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmRespToModel(getResp.Instance, &data.FlexmetalVmModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data vmModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instanceResp, err := r.client.GetVmInstance(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading VM instance", err.Error())
		return
	}
	if instanceResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error reading VM instance", instanceResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmRespToModel(instanceResp.Instance, &data.FlexmetalVmModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data vmModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	for _, tag := range data.Tags.Elements() {
		tags = append(tags, strings.Trim(tag.String(), "\""))
	}

	updateReq := one_api.UpdateVmInstanceRequest{Tags: tags}

	instanceResp, err := r.client.UpdateVmInstance(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating VM instance", err.Error())
		return
	}
	if instanceResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error updating VM instance", instanceResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	vmRespToModel(instanceResp.Instance, &data.FlexmetalVmModel)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data vmModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	instanceResp, err := r.client.DeleteVmInstance(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting VM instance", err.Error())
		return
	}
	if instanceResp != nil && instanceResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error deleting VM instance", instanceResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	lastStatus := data.Status.ValueString()
	err = r.waitForVmStatus(ctx, data.ID.ValueString(), []string{"destroyed"}, 15*time.Second, func(i *one_api.VmInstance) {
		lastStatus = i.Status
	})
	if err != nil {
		// 404 after delete is expected -- treat it as success
		tflog.Info(ctx, "VM delete wait ended", map[string]interface{}{"err": err.Error(), "last_status": lastStatus})
	}
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// waitForVmStatus polls GetVmInstance every interval until status is one of desiredStatuses or context expires.
// Returns an error if the API returns 404 (VM gone) or context is done.
func (r *vmResource) waitForVmStatus(ctx context.Context, vmID string, desiredStatuses []string, interval time.Duration, onInstance func(i *one_api.VmInstance)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			instanceResp, err := r.client.GetVmInstance(ctx, vmID)
			if err != nil {
				tflog.Error(ctx, "error getting VM instance", map[string]interface{}{"id": vmID, "err": err})
				return err
			}

			if instanceResp.ErrorResponse != nil {
				tflog.Error(ctx, "API error getting VM instance", map[string]interface{}{"msg": instanceResp.ErrorResponse.ErrorMessage})
				continue
			}

			if instanceResp.Instance != nil && onInstance != nil {
				onInstance(instanceResp.Instance)
			}

			if slices.Contains(desiredStatuses, instanceResp.Instance.Status) {
				tflog.Info(ctx, fmt.Sprintf("VM reached desired status: %s", instanceResp.Instance.Status))
				return nil
			}
		}
	}
}

func vmRespToModel(instance *one_api.VmInstance, data *resource_flexmetal_vm.FlexmetalVmModel) {
	data.ID = types.StringValue(instance.ID)
	data.Name = types.StringValue(instance.Name)
	data.PoolID = types.StringValue(instance.PoolID)
	data.Plan = types.StringValue(instance.Plan)
	data.Os = resource_flexmetal_vm.OsValue{
		ImageID: types.StringValue(instance.OS.ImageID),
	}
	data.UserData = types.StringValue(instance.UserData)
	data.Status = types.StringValue(instance.Status)
	data.IPAddress = types.StringValue(instance.IPAddress)
	data.IPAddressV6 = types.StringValue(instance.IPAddressV6)
	data.Gateway = types.StringValue(instance.Gateway)
	data.Netmask = types.StringValue(instance.Netmask)
	data.VlanID = types.Int64Value(instance.VlanID)
	data.ProvisionedAt = types.StringValue(instance.ProvisionedAt)

	if len(instance.Tags) > 0 {
		var tagVals []attr.Value
		for _, t := range instance.Tags {
			tagVals = append(tagVals, types.StringValue(t))
		}
		data.Tags = types.ListValueMust(types.StringType, tagVals)
	} else if data.Tags.IsNull() || data.Tags.IsUnknown() {
		data.Tags = types.ListValueMust(types.StringType, []attr.Value{})
	}
}

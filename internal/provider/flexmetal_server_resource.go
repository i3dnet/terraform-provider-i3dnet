package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/modifiers"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_server"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var waitForReadyTimeout = 30 * time.Minute
var waitForReleasedTimeout = 5 * time.Minute

var _ resource.Resource = (*serverResource)(nil)
var _ resource.ResourceWithConfigure = (*serverResource)(nil)

func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	client *one_api.Client
}

func (r *serverResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*one_api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api_utils.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *serverResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_server"
}

func (r *serverResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_flexmetal_server.FlexmetalServerResourceSchema(ctx)

	generatedSchema.MarkdownDescription = "FlexMetal servers are physical servers that can be requested and released at will.\n\n" +
		"A How to Guide is available at this URL : https://www.i3d.net/docs/one/Compute/FlexMetal/api/"

	// make post_install_script, os.kernel_params and os.partitions as optional:true and computed:false
	// because they are not included in the GET response body
	generatedSchema.Attributes["post_install_script"] = schema.StringAttribute{
		Optional:            true,
		Computed:            false,
		Description:         generatedSchema.Attributes["post_install_script"].GetDescription(),
		MarkdownDescription: generatedSchema.Attributes["post_install_script"].GetMarkdownDescription(),
	}

	generatedOSAttribute := generatedSchema.Attributes["os"].(schema.SingleNestedAttribute)

	osAttributes := generatedOSAttribute.GetAttributes()

	kernelParams := osAttributes["kernel_params"].(schema.ListNestedAttribute)
	kernelParams.Optional = true
	kernelParams.Computed = false

	partitions := osAttributes["partitions"].(schema.ListNestedAttribute)
	partitions.Optional = true
	partitions.Computed = false

	generatedSchema.Attributes["os"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"kernel_params": kernelParams,
			"partitions":    partitions,
			"slug": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the OS. Available operating systems can be obtained from [/v3/operatingsystem](https://www.i3d.net/docs/api/v3/all#/OperatingSystem/getOperatingsystems). Use the `slug` field from the response.",
				MarkdownDescription: "Identifier of the OS. Available operating systems can be obtained from [/v3/operatingsystem](https://www.i3d.net/docs/api/v3/all#/OperatingSystem/getOperatingsystems). Use the `slug` field from the response.",
			},
		},
		CustomType:          generatedOSAttribute.CustomType,
		Required:            generatedOSAttribute.Required,
		Description:         generatedOSAttribute.GetDescription(),
		MarkdownDescription: generatedOSAttribute.GetMarkdownDescription(),
	}

	// For certain OS(talos, windows) ssh_key is not needed
	generatedSchema.Attributes["ssh_key"] = schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Description:         "A list of SSH keys. You can either supply SSH key UUIDs from stored objects in [/v3/sshKey](https://www.i3d.net/docs/api/v3/all#/SSHKey/getSshKeys) or provide public keys directly. SSH keys are installed for the root user.",
		MarkdownDescription: "A list of SSH keys. You can either supply SSH key UUIDs from stored objects in [/v3/sshKey](https://www.i3d.net/docs/api/v3/all#/SSHKey/getSshKeys) or provide public keys directly. SSH keys are installed for the root user.",
	}

	// Add extra info to docs
	generatedSchema.Attributes["instance_type"] = schema.StringAttribute{
		Required:            true,
		Description:         "Server instance type. Available instance types can be obtained from [/v3/flexMetal/location/{locationId}}/instanceTypes](https://www.i3d.net/docs/api/v3/all#/FlexMetalServer/getFlexMetalLocationInstanceTypes). Use the `name` field from the response.",
		MarkdownDescription: "Server instance type. Available instance types can be obtained from [/v3/flexMetal/location/{locationId}}/instanceTypes](https://www.i3d.net/docs/api/v3/all#/FlexMetalServer/getFlexMetalLocationInstanceTypes). Use the `name` field from the response.",
	}
	generatedSchema.Attributes["location"] = schema.StringAttribute{
		Required:            true,
		Description:         "Server location. Available locations can be obtained from [/v3/flexMetal/location](https://www.i3d.net/docs/api/v3/all#/FlexMetalServer/getFlexMetalLocations). Use the `name` field from the response.",
		MarkdownDescription: "Server location. Available locations can be obtained from [/v3/flexMetal/location](https://www.i3d.net/docs/api/v3/all#/FlexMetalServer/getFlexMetalLocations). Use the `name` field from the response.",
	}

	modifiers.UpdateComputed(generatedSchema, []string{"tags", "overflow", "contract_id"}, false)

	modifiers.ApplyRequireReplace(generatedSchema, []string{"instance_type", "name", "location", "post_install_script", "ssh_key", "os"})
	modifiers.ApplyUseStateForUnknown(generatedSchema, []string{"uuid", "status", "status_message", "ip_addresses", "released_at", "created_at", "delivered_at", "overflow"})

	resp.Schema = generatedSchema
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	var kernelParams []one_api.KernelParam
	for _, kernelParam := range data.Os.KernelParams.Elements() {
		kernelParam1 := kernelParam.(resource_flexmetal_server.KernelParamsValue)
		kernelParams = append(kernelParams, one_api.KernelParam{
			Key:   kernelParam1.Key.ValueString(),
			Value: kernelParam1.Value.ValueString(),
		})
	}
	var tags []string
	for _, tag := range data.Tags.Elements() {
		tags = append(tags, strings.Replace(tag.String(), "\"", "", -1))
	}

	var sskKeys []string
	for _, sshKey := range data.SshKey.Elements() {
		sskKeys = append(sskKeys, strings.Replace(sshKey.String(), "\"", "", -1))
	}

	var partitions []one_api.Partition
	for _, v := range data.Os.Partitions.Elements() {
		part := v.(resource_flexmetal_server.PartitionsValue)

		partitions = append(partitions, one_api.Partition{
			Target:     part.Target.ValueString(),
			Filesystem: part.Filesystem.ValueString(),
			Size:       part.Size.ValueInt64(),
		})
	}

	createServerReq := one_api.CreateServerReq{
		Name:         data.Name.ValueString(),
		Location:     data.Location.ValueString(),
		InstanceType: data.InstanceType.ValueString(),
		OS: one_api.OS{
			Slug:         data.Os.Slug.ValueString(),
			KernelParams: kernelParams,
			Partitions:   partitions,
		},
		Tags:              tags,
		SSHkey:            sskKeys,
		PostInstallScript: data.PostInstallScript.ValueString(),
		ContractID:        data.ContractId.ValueString(),
		Overflow:          data.Overflow.ValueBool(),
	}

	serverResp, err := r.client.CreateServer(ctx, createServerReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating server", "Unexpected error: "+err.Error())
		return
	}
	if serverResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating server", serverResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	serverRespToPlan(ctx, serverResp.Server, &data)

	// Add resource to TF state earlier to prevent dangling servers
	// Example: timeout reached, but server is delivered later on
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err, lastStatus := r.waitForStatus(ctx, data.Uuid.ValueString(), []string{"delivered", "failed"}, waitForReadyTimeout, 1*time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for server to be ready", fmt.Sprintf("Error: %v\nLast status: %s", err, lastStatus))
		return
	}

	if lastStatus == "failed" {
		resp.Diagnostics.AddError("Server creation failed", fmt.Sprintf("Status message: %s", data.StatusMessage.ValueString()))
		return
	}

	// server is delivered, get its details to save them to state
	getServerResp, err := r.client.GetServer(ctx, data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error getting server", "Unexpected error: "+err.Error())
		return
	}
	if getServerResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error getting server", serverResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	if len(getServerResp.Server.IpAddresses) == 0 {
		resp.Diagnostics.AddError("Server creation failed", fmt.Sprintf("Server %s has no ipAddresses attached", getServerResp.Server.Uuid))
		return
	}

	serverRespToPlan(ctx, getServerResp.Server, &data)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func serverRespToPlan(ctx context.Context, server *one_api.Server, data *resource_flexmetal_server.FlexmetalServerModel) {
	data.Uuid = types.StringValue(server.Uuid)
	data.CreatedAt = types.Int64Value(server.CreatedAt)
	data.ReleasedAt = types.Int64Value(server.ReleasedAt)
	data.DeliveredAt = types.Int64Value(server.DeliveredAt)
	data.Status = types.StringValue(server.Status)
	data.StatusMessage = types.StringValue(server.StatusMessage)

	if server.ContractID != "" {
		data.ContractId = types.StringValue(server.ContractID)
	}

	data.IpAddresses = basetypes.NewListValueMust(
		resource_flexmetal_server.IpAddressesValue{}.Type(context.Background()),
		[]attr.Value{},
	)
	if len(server.IpAddresses) > 0 {
		var values []attr.Value
		for _, ip := range server.IpAddresses {
			ipAddressValue := resource_flexmetal_server.NewIpAddressesValueMust(
				map[string]attr.Type{
					"ip_address": basetypes.StringType{},
				},
				map[string]attr.Value{
					"ip_address": basetypes.NewStringValue(ip.IpAddress),
				})
			values = append(values, ipAddressValue)
		}
		data.IpAddresses = basetypes.NewListValueMust(
			resource_flexmetal_server.IpAddressesValue{}.Type(context.Background()),
			values,
		)
	}

	if len(server.Tags) > 0 {
		var values []attr.Value
		for _, tag := range server.Tags {
			values = append(values, types.StringValue(tag))
		}
		data.Tags = basetypes.NewListValueMust(types.StringType, values)
	}
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serverResp, err := r.client.GetServer(ctx, data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading server",
			"Could not read server by id "+data.Uuid.ValueString()+": "+err.Error(),
		)
		return
	}

	if serverResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error reading server", serverResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	serverRespToPlan(ctx, serverResp.Server, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resource_flexmetal_server.FlexmetalServerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var planTags []string
	for _, v := range plan.Tags.Elements() {
		planTags = append(planTags, v.(types.String).ValueString())
	}

	var stateTags []string
	for _, v := range state.Tags.Elements() {
		stateTags = append(stateTags, v.(types.String).ValueString())
	}

	var newTags []string
	for _, v := range planTags {
		if !slices.Contains(stateTags, v) {
			newTags = append(newTags, v)
		}
	}

	var removedTags []string
	for _, v := range stateTags {
		if !slices.Contains(planTags, v) {
			removedTags = append(removedTags, v)
		}
	}

	for _, tag := range newTags {
		serverResp, err := r.client.AddTagToServer(ctx, plan.Uuid.ValueString(), tag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding tag to server",
				"Unexpected error: "+err.Error(),
			)
			return
		}
		if serverResp.ErrorResponse != nil {
			AddErrorResponseToDiags("Error adding tag to server", serverResp.ErrorResponse, &resp.Diagnostics)
			return
		}
	}

	for _, tag := range removedTags {
		serverResp, err := r.client.DeleteTagFromServer(ctx, plan.Uuid.ValueString(), tag)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting tag from server",
				"Could not delete tag from server, unexpected error: "+err.Error(),
			)
			return
		}
		if serverResp.ErrorResponse != nil {
			AddErrorResponseToDiags("Error deleting tag from server", serverResp.ErrorResponse, &resp.Diagnostics)
			return
		}
	}

	s, err := r.client.GetServer(ctx, plan.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting server",
			"Could not get server, unexpected error: "+err.Error(),
		)
		return
	}

	if s.ErrorResponse != nil {
		AddErrorResponseToDiags("Error reading server", s.ErrorResponse, &resp.Diagnostics)
		return
	}

	serverRespToPlan(ctx, s.Server, &plan)

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serverResp, err := r.client.DeleteServer(ctx, data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting server",
			"Could not delete server by id: "+err.Error(),
		)
		return
	}

	if serverResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error deleting server", serverResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	err, lastStatus := r.waitForStatus(ctx, data.Uuid.ValueString(), []string{"released"}, waitForReleasedTimeout, 1*time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Server deletion failed", fmt.Sprintf("Last status: %q", lastStatus))
		return
	}
}

// waitForStatus performs a GET server request every interval until server status reaches desiredStatuses or timeouts
// it returns an error and last known status
func (r *serverResource) waitForStatus(ctx context.Context, serverID string, desiredStatuses []string, timeout, interval time.Duration) (err error, lastStatus string) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err(), lastStatus
		case <-deadline:
			return fmt.Errorf("timeout reached while waiting for server status"), lastStatus
		case <-ticker.C:
			serverResponse, err := r.client.GetServer(ctx, serverID)
			if err != nil {
				tflog.Error(ctx, "error getting server by id", map[string]interface{}{"id": serverID})
				continue
			}

			if serverResponse.ErrorResponse != nil {
				tflog.Error(ctx, "error response on get server", map[string]interface{}{"errorMsg": serverResponse.ErrorResponse.ErrorMessage})
				continue
			}

			lastStatus = serverResponse.Server.Status
			if slices.Contains(desiredStatuses, lastStatus) {
				tflog.Info(ctx, fmt.Sprintf("server reached desired status: %s", lastStatus))
				return nil, lastStatus
			}
		}
	}
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

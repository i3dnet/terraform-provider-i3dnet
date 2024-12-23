package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3d/internal/provider/api_utils"
	"terraform-provider-i3d/internal/provider/resource_ssh_key"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = (*sshKeyResource)(nil)
	_ resource.ResourceWithConfigure = (*sshKeyResource)(nil)
)

func NewSshKeyResource() resource.Resource {
	return &sshKeyResource{}
}

type sshKeyResource struct {
	client *api_utils.Client
}

func (r *sshKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api_utils.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api_utils.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *sshKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_ssh_key.SshKeyResourceSchema(ctx)
}

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan resource_ssh_key.SshKeyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	reqBody := api_utils.CreateSSHKey{
		Name:      plan.Name.ValueString(),
		PublicKey: plan.PublicKey.ValueString(),
	}

	// Create new SSH key
	sshResp, err := r.client.CreateSSHKey(reqBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ssh key",
			"Could not create ssh key, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.PublicKey = types.StringValue(sshResp.PublicKey)
	plan.Name = types.StringValue(sshResp.Name)
	plan.CreatedAt = types.Int64Value(sshResp.CreatedAt)
	plan.Uuid = types.StringValue(sshResp.Uuid)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_ssh_key.SshKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	sshResp, err := r.client.GetSSHKey(data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading i3D ssh key",
			"Could not read ssh key id "+data.Uuid.ValueString()+": "+err.Error(),
		)
		return
	}

	data.PublicKey = types.StringValue(sshResp.PublicKey)
	data.Name = types.StringValue(sshResp.Name)
	data.CreatedAt = types.Int64Value(sshResp.CreatedAt)
	data.Uuid = types.StringValue(sshResp.Uuid)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_ssh_key.SshKeyModel

	// Not implemented: We don't have an UPDATE endpoint for ssh keys

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update API call logic

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_ssh_key.SshKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSSHKey(data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting i3d SSHKey",
			"Could not delete ssh key, unexpected error: "+err.Error(),
		)
		return
	}
}

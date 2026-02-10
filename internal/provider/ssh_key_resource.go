package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_ssh_key"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*sshKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*sshKeyResource)(nil)
	_ resource.ResourceWithImportState = (*sshKeyResource)(nil)
)

func NewSshKeyResource() resource.Resource {
	return &sshKeyResource{}
}

type sshKeyResource struct {
	client *one_api.Client
}

func (r *sshKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sshKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_ssh_key.SshKeyResourceSchema(ctx)

	generatedSchema.MarkdownDescription = "Provides an i3D.net SSH key resource to allow you to manage SSH keys for server access. " +
		"Keys created with this resource can be referenced in your `i3dnet_flexmetal_server` resource configuration via its value."

	// currently all fields of SSHKey are marked as required in our public open API SPEC
	// https://docs.i3d.net/api/api_general#get-v3-sshkey
	// until we fix OpenAPI Spec we need to mark `created_at` and `uuid` as computed, not required
	generatedSchema.Attributes["created_at"] = schema.Int64Attribute{
		Computed:            true,
		Description:         generatedSchema.Attributes["created_at"].GetDescription(),
		MarkdownDescription: generatedSchema.Attributes["created_at"].GetMarkdownDescription(),
	}
	generatedSchema.Attributes["uuid"] = schema.StringAttribute{
		Computed:            true,
		Description:         generatedSchema.Attributes["uuid"].GetDescription(),
		MarkdownDescription: generatedSchema.Attributes["uuid"].GetMarkdownDescription(),
	}

	// add id to allow for terraform import
	generatedSchema.Attributes["id"] = schema.StringAttribute{
		MarkdownDescription: "The ID of this resource. Same as `uuid`.",
		Computed:            true,
	}

	resp.Schema = generatedSchema
}

// SshKeyModel overrides resource_ssh_key.SshKeyModel due to the fact that .gen files should not be edited
// and currently the Provider Spec Generator doesn't allow for adding new fields via GeneratorConfig.yaml.
// Use this struct whenever resource_ssh_key.SshKeyModel is required.
type SshKeyModel struct {
	resource_ssh_key.SshKeyModel
	ID types.String `tfsdk:"id"`
}

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan SshKeyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	reqBody := one_api.CreateSSHKeyReq{
		Name:      plan.Name.ValueString(),
		PublicKey: plan.PublicKey.ValueString(),
	}

	// Create new SSH key
	sshResp, err := r.client.CreateSSHKey(ctx, reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ssh key", "Unexpected error: "+err.Error())
		return
	}

	if sshResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating ssh key", sshResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.PublicKey = types.StringValue(sshResp.SSHKey.PublicKey)
	plan.Name = types.StringValue(sshResp.SSHKey.Name)
	plan.CreatedAt = types.Int64Value(sshResp.SSHKey.CreatedAt)
	plan.Uuid = types.StringValue(sshResp.SSHKey.Uuid)
	plan.ID = types.StringValue(sshResp.SSHKey.Uuid)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SshKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	sshResp, err := r.client.GetSSHKey(ctx, data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ssh key",
			"Could not read ssh key id "+data.Uuid.ValueString()+": "+err.Error(),
		)
		return
	}

	if sshResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error reading ssh key", sshResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	data.PublicKey = types.StringValue(sshResp.SSHKey.PublicKey)
	data.Name = types.StringValue(sshResp.SSHKey.Name)
	data.CreatedAt = types.Int64Value(sshResp.SSHKey.CreatedAt)
	data.Uuid = types.StringValue(sshResp.SSHKey.Uuid)
	data.ID = types.StringValue(sshResp.SSHKey.Uuid)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *sshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan SshKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Currently there is no API endpoint to update by id, so we re-create a new ssh key

	// Delete old key
	err := r.client.DeleteSSHKey(ctx, state.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not replace SSHKey",
			"Unexpected error: "+err.Error(),
		)
		return
	}

	// Create new SSH key
	reqBody := one_api.CreateSSHKeyReq{
		Name:      plan.Name.ValueString(),
		PublicKey: plan.PublicKey.ValueString(),
	}

	sshResp, err := r.client.CreateSSHKey(ctx, reqBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ssh key", "Unexpected error: "+err.Error())
		return
	}
	if sshResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating ssh key", sshResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.PublicKey = types.StringValue(sshResp.SSHKey.PublicKey)
	plan.Name = types.StringValue(sshResp.SSHKey.Name)
	plan.CreatedAt = types.Int64Value(sshResp.SSHKey.CreatedAt)
	plan.Uuid = types.StringValue(sshResp.SSHKey.Uuid)
	plan.ID = types.StringValue(sshResp.SSHKey.Uuid)

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SshKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSSHKey(ctx, data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting SSHKey",
			"Unexpected error: "+err.Error(),
		)
		return
	}
}

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*flexvmNodeResource)(nil)
	_ resource.ResourceWithConfigure   = (*flexvmNodeResource)(nil)
	_ resource.ResourceWithImportState = (*flexvmNodeResource)(nil)
)

func NewFlexvmNodeResource() resource.Resource {
	return &flexvmNodeResource{}
}

type flexvmNodeResource struct {
	client *one_api.Client
}

type FlexvmNodeModel struct {
	CloudID  types.String   `tfsdk:"cloud_id"`
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Serial   types.String   `tfsdk:"serial"`
	Status   types.String   `tfsdk:"status"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *flexvmNodeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *flexvmNodeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexvm_node"
}

func (r *flexvmNodeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a bare metal Node within an i3D.net FlexVM private cloud. A Node uses the " +
			"Cloud's instance type and location, so the only required input is the Cloud UUID; all other " +
			"attributes are assigned by the platform. Creating a Node provisions bare metal hardware, which can " +
			"take a while; the resource waits until the Node reaches the `running` status.",
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the Cloud in which to create the Node.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Node UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Node name.",
			},
			"serial": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Node serial number.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the Node. One of: `created`, `requested`, `bootstrapping`, `running`, `failed`, `deleting`, `deleted`.",
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *flexvmNodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FlexvmNodeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, 30*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	cloudID := data.CloudID.ValueString()

	nodeResp, err := r.client.FlexvmCreateNode(ctx, cloudID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating FlexVM Cloud Node",
			fmt.Sprintf("Unexpected error: %v", err),
		)
		return
	}
	if nodeResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating FlexVM Cloud Node", nodeResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	flexvmNodeRespToState(nodeResp.Node, &data)

	// Save state early to prevent dangling Nodes on timeout.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodeID := data.ID.ValueString()

	// Provisioning continues in the background; wait until the Node comes up.
	// waitForCreated (via getNode) keeps state in sync with the latest status.
	r.waitForCreated(ctx, cloudID, nodeID, &resp.State, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Status.ValueString() != "running" {
		// The Node did not come up; its last observed status is already in state.
		resp.Diagnostics.AddError(
			"FlexVM Cloud Node creation failed, node is not running",
			fmt.Sprintf("Node did not reach 'running' status.\nNode id: %s", nodeID),
		)
		return
	}
}

func (r *flexvmNodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FlexvmNodeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.getNode(ctx, data.CloudID.ValueString(), data.ID.ValueString(), true, &resp.State, &data, &resp.Diagnostics)
}

func (r *flexvmNodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update API exists and the only configurable attribute (cloud_id) has
	// RequiresReplace, so Terraform destroys and recreates instead of calling Update.
}

func (r *flexvmNodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FlexvmNodeModel
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

	cloudID := data.CloudID.ValueString()
	nodeID := data.ID.ValueString()
	logFields := map[string]any{
		"cloud_id": cloudID,
		"node_id":  nodeID,
	}

	// Refresh the Node to determine its current status before deleting.
	terminal, failed := r.getNode(ctx, cloudID, nodeID, false, &resp.State, &data, &resp.Diagnostics)
	if failed {
		return
	}
	if terminal {
		tflog.Debug(ctx, "FlexVM Cloud Node is in a terminal state; removing from state without calling delete", logFields)
		return
	}

	if data.Status.ValueString() != "running" {
		resp.Diagnostics.AddError(
			"FlexVM Cloud Node cannot be deleted",
			fmt.Sprintf("Node must be in 'running' or 'failed' status to be deleted, but is in '%s' status.\nNode id: %s", data.Status.ValueString(), nodeID),
		)
		return
	}

	nodeResp, err := r.client.FlexvmDeleteNode(ctx, cloudID, nodeID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting FlexVM Cloud Node",
			"Could not delete FlexVM Cloud Node: "+err.Error(),
		)
		return
	}
	if nodeResp.ErrorResponse != nil {
		// Any non-2XX response is treated as a deletion failure.
		AddErrorResponseToDiags("Error deleting FlexVM Cloud Node", nodeResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	// Deletion was accepted; poll until the Node becomes terminal.
	r.waitForDeleted(ctx, cloudID, nodeID, &resp.State, &data, &resp.Diagnostics)
}

func (r *flexvmNodeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID format: cloud_id/node_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cloud_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// waitForCreated polls the Node until it finishes provisioning or if it
// becomes terminal. See getNode for more on a terminal state. getNode keeps
// data in sync with the latest status, so callers should inspect data.Status.
func (r *flexvmNodeResource) waitForCreated(ctx context.Context, cloudID, nodeID string, state *tfsdk.State, data *FlexvmNodeModel, diags *diag.Diagnostics) {
	err := r.poll(ctx, 10*time.Second, 30*time.Second, func() bool {
		terminal, failed := r.getNode(ctx, cloudID, nodeID, true, state, data, diags)
		if failed {
			return false
		}
		status := data.Status.ValueString()
		// "failed" is non-terminal here (allowFailed), so stop on it explicitly.
		return terminal || status == "running" || status == "failed"
	})
	if err != nil {
		diags.AddError(
			"Error waiting for FlexVM Cloud Node to be ready",
			fmt.Sprintf("Error: %v\nLast status: %s\nNode id: %s", err, data.Status.ValueString(), nodeID),
		)
	}
}

// waitForDeleted polls the Node until it reaches a terminal state. See
// getNode for more on a terminal state.
func (r *flexvmNodeResource) waitForDeleted(ctx context.Context, cloudID, nodeID string, state *tfsdk.State, data *FlexvmNodeModel, diags *diag.Diagnostics) {
	err := r.poll(ctx, 500*time.Millisecond, 10*time.Second, func() bool {
		terminal, failed := r.getNode(ctx, cloudID, nodeID, false, state, data, diags)
		if failed {
			return false
		}
		return terminal
	})
	if err != nil {
		diags.AddError(
			"FlexVM Cloud Node deletion failed",
			fmt.Sprintf("Error: %v\nLast status: %s\nNode id: %s", err, data.Status.ValueString(), nodeID),
		)
	}
}

// poll invokes check on an interval until it reports done (or returns an error).
// The first call happens after initialWait, subsequent calls after interval.
// The wait is bounded by ctx.
func (r *flexvmNodeResource) poll(ctx context.Context, initialWait, interval time.Duration, check func() bool) error {
	timer := time.NewTimer(initialWait)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			done := check()
			if done {
				return nil
			}

			timer.Reset(interval)
		}
	}
}

// getNode fetches the Node and synchronises Terraform state with it: a Node that
// is not in a terminal state is written to state, while a terminal Node — it no
// longer exists (404) or has reached the "failed" or "deleted" status — is
// removed from state, as it should no longer be tracked.
//
// When allowFailed is set, a Node in the "failed" status is not treated as
// terminal: it is written to state and returned with terminal=false so the
// caller can keep tracking it (and/or report the failure) instead of dropping it.
//
// Any failure to read the Node (a transport-level error or a non-404 API error)
// is recorded on diags and reported via the failed return flag; callers should
// stop on failed. State is left untouched in that case. When the call succeeds
// and the Node is not terminal, the latest Node details are written to data, so
// callers can inspect data (e.g. data.Status) instead of a returned object.
func (r *flexvmNodeResource) getNode(ctx context.Context, cloudID, nodeID string, allowFailed bool, state *tfsdk.State, data *FlexvmNodeModel, diags *diag.Diagnostics) (terminal bool, failed bool) {
	nodeResp, err := r.client.FlexvmGetNode(ctx, cloudID, nodeID)
	if err != nil {
		diags.AddError(
			"Error reading FlexVM Cloud Node",
			"Could not read FlexVM Cloud Node id "+nodeID+": "+err.Error(),
		)
		return false, true
	}

	if nodeResp.ErrorResponse != nil {
		if nodeResp.ErrorResponse.StatusCode == http.StatusNotFound {
			state.RemoveResource(ctx)
			return true, false
		}
		AddErrorResponseToDiags("Error reading FlexVM Cloud Node", nodeResp.ErrorResponse, diags)
		return false, true
	}

	if nodeResp.Node == nil {
		diags.AddError(
			"Error reading FlexVM Cloud Node",
			"call to FlexvmGetNode: no node was set in the response",
		)
		return false, true
	}

	switch nodeResp.Node.Status {
	case "failed":
		if !allowFailed {
			state.RemoveResource(ctx)
			return true, false
		}
	case "deleted":
		state.RemoveResource(ctx)
		return true, false
	}

	// Not terminal: refresh state with the latest Node details.
	flexvmNodeRespToState(nodeResp.Node, data)
	diags.Append(state.Set(ctx, data)...)
	return false, false
}

// flexvmNodeRespToState maps an API Node object onto the resource state model.
func flexvmNodeRespToState(node *one_api.FlexvmNodeObj, data *FlexvmNodeModel) {
	data.ID = types.StringValue(node.ID)
	data.Name = types.StringValue(node.Name)
	data.Serial = types.StringValue(node.Serial)
	data.Status = types.StringValue(node.Status)
	if node.CloudID() != "" {
		data.CloudID = types.StringValue(node.CloudID())
	}
}

package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = (*flexvmVMResource)(nil)
	_ resource.ResourceWithConfigure        = (*flexvmVMResource)(nil)
	_ resource.ResourceWithImportState      = (*flexvmVMResource)(nil)
	_ resource.ResourceWithConfigValidators = (*flexvmVMResource)(nil)
)

// flexvmUserDataMaxLen is the maximum length (in characters) the API accepts
// for the user-data "data" field. For base64-encoded content this applies to
// the base64-encoded string.
const flexvmUserDataMaxLen = 64000

func NewFlexvmVMResource() resource.Resource {
	return &flexvmVMResource{}
}

type flexvmVMResource struct {
	client *one_api.Client
}

type FlexvmVMModel struct {
	CloudID          types.String   `tfsdk:"cloud_id"`
	Name             types.String   `tfsdk:"name"`
	Description      types.String   `tfsdk:"description"`
	InstanceTypeName types.String   `tfsdk:"instance_type_name"`
	ImageName        types.String   `tfsdk:"image_name"`
	SSHKeys          types.List     `tfsdk:"ssh_keys"`
	UserDataFile     types.String   `tfsdk:"user_data_file"`
	ID               types.String   `tfsdk:"id"`
	Status           types.String   `tfsdk:"status"`
	IPs              types.List     `tfsdk:"ips"`
	InstanceType     types.Object   `tfsdk:"instance_type"`
	Image            types.Object   `tfsdk:"image"`
	Cloud            types.Object   `tfsdk:"cloud"`
	Node             types.Object   `tfsdk:"node"`
	CreatedAt        types.String   `tfsdk:"created_at"`
	DeletedAt        types.String   `tfsdk:"deleted_at"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

var ipsObjectAttrTypes = map[string]attr.Type{
	"address": types.StringType,
	"public":  types.BoolType,
}

var instanceTypeObjectAttrTypes = map[string]attr.Type{
	"id":     types.StringType,
	"name":   types.StringType,
	"vcpu":   types.Int64Type,
	"memory": types.Int64Type,
	"disk":   types.Int64Type,
}

var imageObjectAttrTypes = map[string]attr.Type{
	"id":      types.StringType,
	"name":    types.StringType,
	"os":      types.StringType,
	"os_type": types.StringType,
}

var cloudObjectAttrTypes = map[string]attr.Type{
	"id":          types.StringType,
	"name":        types.StringType,
	"description": types.StringType,
	"site":        types.StringType,
}

var nodeObjectAttrTypes = map[string]attr.Type{
	"id":            types.StringType,
	"name":          types.StringType,
	"instance_type": types.StringType,
	"serial":        types.StringType,
}

func (r *flexvmVMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *flexvmVMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexvm_vm"
}

func (r *flexvmVMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine within an i3D.net FlexvmVM private cloud.",
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the cloud in which to create the VM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "VM name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "An optional free-form description of your VM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_type_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The instance type name to base the VM on.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The image name to create the VM from.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ssh_keys": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of public SSH keys. Exactly one of `ssh_keys` or `user_data_file` must be set.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"user_data_file": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Path to a file whose contents are passed to the VM as cloud-init " +
					"user-data on first boot. Exactly one of `ssh_keys` or `user_data_file` must be set; " +
					"when `user_data_file` is used, configure SSH access through the user-data itself.\n\n" +
					"Relative paths are resolved against the directory in which Terraform is run (its working " +
					"directory), so prefer an absolute path or wrap it with `abspath(\"${path.module}/...\")`.\n\n" +
					"If the file content is not valid UTF-8 (e.g. gzip-compressed cloud-init) it is automatically " +
					"base64-encoded before being sent. Only the file path is tracked in state: editing the file's " +
					"content without changing the path will not trigger a replacement. See " +
					"https://docs.cloud-init.io/en/latest/reference/examples.html#yaml-examples for examples of a " +
					"configuration file.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The status of the VM.",
			},
			"ips": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "A list of IP address objects that belong to the VM.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "An IP address, can be v4 or v6, public or private.",
						},
						"public": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Indicates whether the IP address is a public or private one.",
						},
					},
				},
			},
			"instance_type": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The instance type the VM is based on.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "FlexvmVM Instance Type UUID.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The name of the instance type.",
					},
					"vcpu": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "The number of vCPU resources.",
					},
					"memory": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "The amount of memory in MB.",
					},
					"disk": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "The size of the OS disk in GB.",
					},
				},
			},
			"image": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The image the VM is based on.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "FlexvmVM Image UUID.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Image name.",
					},
					"os": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The name of the OS that the image represents.",
					},
					"os_type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The OS type. Can be \"linux\" or \"windows\".",
					},
				},
			},
			"cloud": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud object within which the VM is deployed.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud UUID.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud name.",
					},
					"description": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud description.",
					},
					"site": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The i3D site (location) in which the Cloud is located.",
					},
				},
			},
			"node": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Node object on which the VM is deployed.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud Node UUID.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud Node name.",
					},
					"instance_type": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud Node FlexMetal instance type.",
					},
					"serial": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Cloud Node serial number.",
					},
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM creation timestamp.",
			},
			"deleted_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM deletion timestamp.",
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *flexvmVMResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("ssh_keys"),
			path.MatchRoot("user_data_file"),
		),
	}
}

func (r *flexvmVMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FlexvmVMModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := one_api.FlexvmCreateVMRequest{
		Name:             data.Name.ValueString(),
		Description:      data.Description.ValueString(),
		InstanceTypeName: data.InstanceTypeName.ValueString(),
		ImageName:        data.ImageName.ValueString(),
	}

	// Exactly one of ssh_keys / user_data_file is set (enforced by ConfigValidators).
	if !data.UserDataFile.IsNull() {
		userData, err := buildUserDataFromFile(data.UserDataFile.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading user_data_file",
				err.Error(),
			)
			return
		}
		createReq.UserData = userData
	} else {
		var sshKeys []string
		resp.Diagnostics.Append(data.SSHKeys.ElementsAs(ctx, &sshKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SSHKeys = sshKeys
	}

	createTimeout, diags := data.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	vmResp, err := r.client.FlexvmCreateVM(ctx, data.CloudID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating FlexvmVM",
			fmt.Sprintf("Unexpected error: %v", err),
		)
		return
	}
	if vmResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating FlexvmVM", vmResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	flexvmVMRespToState(vmResp.VM, &data)

	// Save state early to prevent dangling VMs on timeout
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID := data.ID.ValueString()
	cloudID := data.CloudID.ValueString()
	lastStatus := data.Status.ValueString()
	var lastVM *one_api.FlexvmVM

	err = r.waitForCondition(ctx, cloudID, vmID, 10*time.Second, 5*time.Second, func(vmResp *one_api.FlexvmVMResponse) (bool, error) {
		if vmResp.ErrorResponse != nil {
			return false, fmt.Errorf("call to FlexvmGetVM error response: %s", vmResp.ErrorResponse.ErrorMessage)
		}
		if vmResp.VM == nil {
			return false, errors.New("call to FlexvmGetVM: no vm was set in the response")
		}
		lastVM = vmResp.VM
		lastStatus = vmResp.VM.Status
		return lastStatus == "running" || lastStatus == "failed", nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for FlexvmVM to be ready",
			fmt.Sprintf("Error: %v\nLast status: %s\nVM id: %s", err, lastStatus, vmID),
		)
		return
	}

	if lastStatus == "failed" {
		// Persist the latest VM details so state reflects the actual "failed" status.
		flexvmVMRespToState(lastVM, &data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		resp.Diagnostics.AddError(
			"FlexvmVM creation failed, vm is not running",
			fmt.Sprintf("VM reached 'failed' status.\nVM id: %s", vmID),
		)
		return
	}

	// VM is running, get final details
	getResp, err := r.client.FlexvmGetVM(ctx, cloudID, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Error getting FlexvmVM", "Unexpected error: "+err.Error())
		return
	}
	if getResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error getting FlexvmVM", getResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	flexvmVMRespToState(getResp.VM, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *flexvmVMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FlexvmVMModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmResp, err := r.client.FlexvmGetVM(ctx, data.CloudID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading FlexvmVM",
			"Could not read FlexvmVM id "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if vmResp.ErrorResponse != nil {
		if vmResp.ErrorResponse.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		AddErrorResponseToDiags("Error reading FlexvmVM", vmResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	flexvmVMRespToState(vmResp.VM, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *flexvmVMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update API exists. All mutable attributes have RequiresReplace,
	// so Terraform will destroy and recreate instead of calling Update.
}

func (r *flexvmVMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FlexvmVMModel
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
	vmID := data.ID.ValueString()
	lastStatus := data.Status.ValueString()

	vmResp, err := r.client.FlexvmDeleteVM(ctx, cloudID, vmID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting FlexvmVM",
			"Could not delete FlexvmVM: "+err.Error(),
		)
		return
	}

	logFields := map[string]any{"cloud_id": cloudID, "vm_id": vmID}

	if vmResp.ErrorResponse != nil {
		errResp := vmResp.ErrorResponse
		switch {
		case errResp.StatusCode == http.StatusConflict:
			// VM is already being deleted; poll until it reaches "deleted" (or 404).
			tflog.Debug(ctx, "FlexvmVM is already deleting; polling for the 'deleted' state", logFields)

		case errResp.StatusCode == http.StatusUnprocessableEntity && errResp.ErrorCode == one_api.FlexvmErrCodeVMTerminal:
			// VM is already in a terminal state ("failed" or "deleted"); treat as deleted.
			tflog.Debug(ctx, "FlexvmVM is already in a terminal state; treating it as deleted", logFields)
			return

		case errResp.StatusCode == http.StatusUnprocessableEntity && errResp.ErrorCode == one_api.FlexvmErrCodeVMInTransition:
			// VM is in a transitional state; wait until it stabilizes, then retry delete.
			tflog.Debug(ctx, "FlexvmVM is in a transitional state; polling for a stable state before retrying delete", logFields)

			reachedTerminal, err := r.waitForFlexvmStable(ctx, cloudID, vmID, &lastStatus)
			if err != nil {
				resp.Diagnostics.AddError(
					"FlexvmVM deletion failed",
					fmt.Sprintf("Error waiting for VM to stabilise: %v\nLast status: %s\nVM id: %s", err, lastStatus, vmID),
				)
				return
			}
			if reachedTerminal {
				// Observed "failed", "deleted" or 404 while waiting; nothing more to do.
				return
			}

			retryResp, err := r.client.FlexvmDeleteVM(ctx, cloudID, vmID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error deleting FlexvmVM",
					"Could not delete FlexvmVM on retry: "+err.Error(),
				)
				return
			}
			if retryResp.ErrorResponse != nil {
				retryErr := retryResp.ErrorResponse
				if retryErr.StatusCode == http.StatusUnprocessableEntity && retryErr.ErrorCode == one_api.FlexvmErrCodeVMTerminal {
					tflog.Debug(ctx, "FlexvmVM became terminal between stabilisation and retry; treating as deleted", logFields)
					return
				}
				AddErrorResponseToDiags("Error deleting FlexvmVM on retry", retryErr, &resp.Diagnostics)
				return
			}

		default:
			AddErrorResponseToDiags("Error deleting FlexvmVM", errResp, &resp.Diagnostics)
			return
		}
	}

	if err := r.waitForFlexvmDeleted(ctx, cloudID, vmID, &lastStatus); err != nil {
		resp.Diagnostics.AddError(
			"FlexvmVM deletion failed",
			fmt.Sprintf("Error: %v\nLast status: %s\nVM id: %s", err, lastStatus, vmID),
		)
		return
	}
}

// waitForFlexvmDeleted polls the VM until it reports a 404 or status "deleted".
// "failed" is intentionally not treated as deleted here: a VM only counts as
// deleted from "failed" when the API tells us so explicitly via the
// FlexvmErrCodeVMTerminal error on the DELETE call.
func (r *flexvmVMResource) waitForFlexvmDeleted(ctx context.Context, cloudID, vmID string, lastStatus *string) error {
	return r.waitForCondition(ctx, cloudID, vmID, 500*time.Millisecond, 5*time.Second, func(vmResp *one_api.FlexvmVMResponse) (bool, error) {
		if vmResp.ErrorResponse != nil {
			if vmResp.ErrorResponse.StatusCode == http.StatusNotFound {
				return true, nil
			}
			return false, fmt.Errorf("call to FlexvmGetVM error response: %s", vmResp.ErrorResponse.ErrorMessage)
		}
		if vmResp.VM == nil {
			return false, errors.New("call to FlexvmGetVM: no vm was set in the response")
		}
		*lastStatus = vmResp.VM.Status
		return *lastStatus == "deleted", nil
	})
}

// waitForFlexvmStable polls the VM until it reaches a state in which delete can
// be retried ("running" or "stopped") or a terminal state ("failed", "deleted",
// or 404). It returns reachedTerminal=true when the VM is already gone or has
// failed, so the caller can skip the retry-delete step.
func (r *flexvmVMResource) waitForFlexvmStable(ctx context.Context, cloudID, vmID string, lastStatus *string) (bool, error) {
	var reachedTerminal bool
	err := r.waitForCondition(ctx, cloudID, vmID, 500*time.Millisecond, 5*time.Second, func(vmResp *one_api.FlexvmVMResponse) (bool, error) {
		if vmResp.ErrorResponse != nil {
			if vmResp.ErrorResponse.StatusCode == http.StatusNotFound {
				reachedTerminal = true
				return true, nil
			}
			return false, fmt.Errorf("call to FlexvmGetVM error response: %s", vmResp.ErrorResponse.ErrorMessage)
		}
		if vmResp.VM == nil {
			return false, errors.New("call to FlexvmGetVM: no vm was set in the response")
		}
		*lastStatus = vmResp.VM.Status
		switch *lastStatus {
		case "running", "stopped":
			return true, nil
		case "failed", "deleted":
			reachedTerminal = true
			return true, nil
		}
		return false, nil
	})
	return reachedTerminal, err
}

func (r *flexvmVMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID format: cloud_id/vm_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cloud_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *flexvmVMResource) waitForCondition(ctx context.Context, cloudID, vmID string,
	initialPollInterval, pollInterval time.Duration, check func(vmResp *one_api.FlexvmVMResponse) (bool, error)) error {
	timer := time.NewTimer(initialPollInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			vmResp, err := r.client.FlexvmGetVM(ctx, cloudID, vmID)
			if err != nil {
				return fmt.Errorf("call to FlexvmGetVM: %w", err)
			}

			done, err := check(vmResp)
			if err != nil {
				return err
			}
			if done {
				return nil
			}

			timer.Reset(pollInterval)
		}
	}
}

// buildUserDataFromFile reads the file at the given path and turns it into a
// FlexvmUserData request. Relative paths are resolved against the process
// working directory (where Terraform is run). Content that is not valid UTF-8
// is base64-encoded and flagged with is_base64=true.
func buildUserDataFromFile(filePath string) (*one_api.FlexvmUserData, error) {
	resolvedPath := filePath
	if !filepath.IsAbs(resolvedPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not determine working directory to resolve %q: %w", filePath, err)
		}
		resolvedPath = filepath.Join(cwd, resolvedPath)
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %q: %w", resolvedPath, err)
	}

	return buildUserData(content)
}

// buildUserData applies the UTF-8/base64 rule to raw file content and validates
// the resulting payload length against the API limit. It is separated from file
// I/O so the encoding logic can be unit-tested.
func buildUserData(content []byte) (*one_api.FlexvmUserData, error) {
	if len(content) == 0 {
		return nil, errors.New("user-data file is empty")
	}

	userData := &one_api.FlexvmUserData{}
	if utf8.Valid(content) {
		userData.Data = string(content)
	} else {
		userData.Data = base64.StdEncoding.EncodeToString(content)
		userData.IsBase64 = true
	}

	if len(userData.Data) > flexvmUserDataMaxLen {
		return nil, fmt.Errorf(
			"user-data is too large: %d characters, maximum is %d (base64-encoded content counts after encoding)",
			len(userData.Data), flexvmUserDataMaxLen,
		)
	}

	return userData, nil
}

func flexvmVMRespToState(vm *one_api.FlexvmVM, data *FlexvmVMModel) {
	data.ID = types.StringValue(vm.ID)
	data.Name = types.StringValue(vm.Name)
	data.Description = types.StringValue(vm.Description)
	data.Status = types.StringValue(vm.Status)
	data.CreatedAt = types.StringValue(vm.CreatedAt)
	data.DeletedAt = types.StringValue(vm.DeletedAt)

	// IPs
	var ipValues []attr.Value
	for _, ip := range vm.IPs {
		ipObj, _ := types.ObjectValue(ipsObjectAttrTypes, map[string]attr.Value{
			"address": types.StringValue(ip.Address),
			"public":  types.BoolValue(ip.Public),
		})
		ipValues = append(ipValues, ipObj)
	}
	if ipValues == nil {
		ipValues = []attr.Value{}
	}
	data.IPs = types.ListValueMust(types.ObjectType{AttrTypes: ipsObjectAttrTypes}, ipValues)

	// Instance Type
	data.InstanceType, _ = types.ObjectValue(instanceTypeObjectAttrTypes, map[string]attr.Value{
		"id":     types.StringValue(vm.InstanceType.ID),
		"name":   types.StringValue(vm.InstanceType.Name),
		"vcpu":   types.Int64Value(int64(vm.InstanceType.VCPU)),
		"memory": types.Int64Value(int64(vm.InstanceType.Memory)),
		"disk":   types.Int64Value(int64(vm.InstanceType.Disk)),
	})

	// Image
	data.Image, _ = types.ObjectValue(imageObjectAttrTypes, map[string]attr.Value{
		"id":      types.StringValue(vm.Image.ID),
		"name":    types.StringValue(vm.Image.Name),
		"os":      types.StringValue(vm.Image.OS),
		"os_type": types.StringValue(vm.Image.OSType),
	})

	// Cloud
	data.Cloud, _ = types.ObjectValue(cloudObjectAttrTypes, map[string]attr.Value{
		"id":          types.StringValue(vm.Cloud.ID),
		"name":        types.StringValue(vm.Cloud.Name),
		"description": types.StringValue(vm.Cloud.Description),
		"site":        types.StringValue(vm.Cloud.Site),
	})

	// Node
	if vm.Node != nil {
		data.Node, _ = types.ObjectValue(nodeObjectAttrTypes, map[string]attr.Value{
			"id":            types.StringValue(vm.Node.ID),
			"name":          types.StringValue(vm.Node.Name),
			"instance_type": types.StringValue(vm.Node.InstanceType),
			"serial":        types.StringValue(vm.Node.Serial),
		})
	} else {
		data.Node = types.ObjectNull(nodeObjectAttrTypes)
	}
}

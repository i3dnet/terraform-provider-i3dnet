package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*flexvmCloudResource)(nil)
	_ resource.ResourceWithConfigure   = (*flexvmCloudResource)(nil)
	_ resource.ResourceWithImportState = (*flexvmCloudResource)(nil)
)

func NewFlexvmCloudResource() resource.Resource {
	return &flexvmCloudResource{}
}

type flexvmCloudResource struct {
	client *one_api.Client
}

type FlexvmCloudModel struct {
	Name         types.String   `tfsdk:"name"`
	Site         types.String   `tfsdk:"site"`
	InstanceType types.String   `tfsdk:"instance_type"`
	Description  types.String   `tfsdk:"description"`
	ID           types.String   `tfsdk:"id"`
	CreatedAt    types.String   `tfsdk:"created_at"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func (r *flexvmCloudResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *flexvmCloudResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexvm_cloud"
}

func (r *flexvmCloudResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an i3D.net FlexVM private cloud. A Cloud groups VMs onto dedicated " +
			"FlexMetal nodes of a single instance type within one site. There is no update API: changing " +
			"any attribute forces the Cloud to be destroyed and recreated.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Cloud name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"site": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The i3D site (location) in which the Cloud is located. One of: `frmtl1`, `camtr6`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The FlexMetal instance type shared by every node in the Cloud.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "An optional free-form description of your Cloud.",
				PlanModifiers: []planmodifier.String{
					// Empty strings "" are also treated as null, which means that
					// adding a `description = ""` field to your resource
					// will not trigger a re-creation of the cloud.
					emptyStringAsNull{},
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the Cloud was created (RFC3339).",
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *flexvmCloudResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FlexvmCloudModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	createReq := one_api.FlexvmCloudCreateRequest{
		Name:         data.Name.ValueString(),
		Site:         data.Site.ValueString(),
		InstanceType: data.InstanceType.ValueString(),
		Description:  data.Description.ValueString(),
	}

	cloudResp, err := r.client.FlexvmCreateCloud(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating FlexVM Cloud",
			fmt.Sprintf("Unexpected error: %v", err),
		)
		return
	}
	if cloudResp.ErrorResponse != nil {
		AddErrorResponseToDiags("Error creating FlexVM Cloud", cloudResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	flexvmCloudRespToState(cloudResp.Cloud, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *flexvmCloudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FlexvmCloudModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cloudResp, err := r.client.FlexvmGetCloud(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading FlexVM Cloud",
			"Could not read FlexVM Cloud id "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if cloudResp.ErrorResponse != nil {
		if cloudResp.ErrorResponse.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		AddErrorResponseToDiags("Error reading FlexVM Cloud", cloudResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	flexvmCloudRespToState(cloudResp.Cloud, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *flexvmCloudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update API exists. All attributes have RequiresReplace, so Terraform
	// will destroy and recreate instead of calling Update.
}

func (r *flexvmCloudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FlexvmCloudModel
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

	cloudResp, err := r.client.FlexvmDeleteCloud(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting FlexVM Cloud",
			"Could not delete FlexVM Cloud: "+err.Error(),
		)
		return
	}

	if cloudResp.ErrorResponse != nil {
		// Already gone; nothing left to do.
		if cloudResp.ErrorResponse.StatusCode == http.StatusNotFound {
			return
		}
		AddErrorResponseToDiags("Error deleting FlexVM Cloud", cloudResp.ErrorResponse, &resp.Diagnostics)
		return
	}
}

func (r *flexvmCloudResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func flexvmCloudRespToState(cloud *one_api.FlexvmCloudObj, data *FlexvmCloudModel) {
	data.ID = types.StringValue(cloud.ID)
	data.Name = types.StringValue(cloud.Name)
	data.Site = types.StringValue(cloud.Site)
	data.InstanceType = types.StringValue(cloud.InstanceType)
	data.CreatedAt = types.StringValue(cloud.CreatedAt)

	// description is optional; keep it null when the API returns an empty value
	// so it round-trips cleanly against an unset configuration.
	if cloud.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(cloud.Description)
	}
}

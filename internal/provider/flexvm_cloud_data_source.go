package provider

import (
	"context"
	"fmt"
	"net/http"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*flexvmCloudDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*flexvmCloudDataSource)(nil)
)

func NewFlexvmCloudDataSource() datasource.DataSource {
	return &flexvmCloudDataSource{}
}

// flexvmCloudDataSource reads an existing FlexVM Cloud by its UUID. It is useful
// for referencing an existing Cloud that is not managed by this Terraform configuration.
type flexvmCloudDataSource struct {
	client *one_api.Client
}

type flexvmCloudDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Site         types.String `tfsdk:"site"`
	InstanceType types.String `tfsdk:"instance_type"`
	Description  types.String `tfsdk:"description"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (d *flexvmCloudDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
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

func (d *flexvmCloudDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexvm_cloud"
}

func (d *flexvmCloudDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about an existing i3D.net FlexVM Cloud by its UUID. This is useful " +
			"for referencing an existing Cloud that is not managed by this Terraform configuration. An error is " +
			"raised if no Cloud exists for the provided id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Cloud UUID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud name.",
			},
			"site": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The i3D site (location) in which the Cloud is located.",
			},
			"instance_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The FlexMetal instance type shared by every node in the Cloud.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Free-form description of the Cloud.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the Cloud was created (RFC3339).",
			},
		},
	}
}

func (d *flexvmCloudDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data flexvmCloudDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cloudResp, err := d.client.FlexvmGetCloud(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading FlexVM Cloud",
			"Could not read FlexVM Cloud id "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if cloudResp.ErrorResponse != nil {
		if cloudResp.ErrorResponse.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError(
				"FlexVM Cloud not found",
				fmt.Sprintf("No FlexVM Cloud found for id %s", data.ID.ValueString()),
			)
			return
		}
		AddErrorResponseToDiags("Error reading FlexVM Cloud", cloudResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	cloud := cloudResp.Cloud
	data.ID = types.StringValue(cloud.ID)
	data.Name = types.StringValue(cloud.Name)
	data.Site = types.StringValue(cloud.Site)
	data.InstanceType = types.StringValue(cloud.InstanceType)
	data.Description = types.StringValue(cloud.Description)
	data.CreatedAt = types.StringValue(cloud.CreatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

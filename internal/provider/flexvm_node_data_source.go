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
	_ datasource.DataSource              = (*flexvmNodeDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*flexvmNodeDataSource)(nil)
)

func NewFlexvmNodeDataSource() datasource.DataSource {
	return &flexvmNodeDataSource{}
}

// flexvmNodeDataSource reads a single FlexVM Cloud Node by its Cloud and node UUID.
type flexvmNodeDataSource struct {
	client *one_api.Client
}

type flexvmNodeDataSourceModel struct {
	CloudID types.String `tfsdk:"cloud_id"`
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Serial  types.String `tfsdk:"serial"`
}

func (d *flexvmNodeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *flexvmNodeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexvm_node"
}

func (d *flexvmNodeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a single node within an i3D.net FlexVM Cloud, by Cloud UUID and " +
			"node UUID.\n\nAn error is raised if no node exists for the provided ids.",
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the Cloud the node belongs to.",
			},
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Cloud Node UUID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Node name.",
			},
			"serial": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cloud Node serial number.",
			},
		},
	}
}

func (d *flexvmNodeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data flexvmNodeDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodeResp, err := d.client.FlexvmGetNode(ctx, data.CloudID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading FlexVM Cloud Node",
			"Could not read FlexVM Cloud Node id "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	if nodeResp.ErrorResponse != nil {
		if nodeResp.ErrorResponse.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError(
				"FlexVM Cloud Node not found",
				fmt.Sprintf("No FlexVM Cloud Node found for cloud_id %s and id %s", data.CloudID.ValueString(), data.ID.ValueString()),
			)
			return
		}
		AddErrorResponseToDiags("Error reading FlexVM Cloud Node", nodeResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	node := nodeResp.Node
	data.ID = types.StringValue(node.ID)
	data.Name = types.StringValue(node.Name)
	data.Serial = types.StringValue(node.Serial)
	if node.CloudID() != "" {
		data.CloudID = types.StringValue(node.CloudID())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

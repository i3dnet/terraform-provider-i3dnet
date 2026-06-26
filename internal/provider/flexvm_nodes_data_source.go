package provider

import (
	"context"
	"fmt"
	"net/http"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*flexvmNodesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*flexvmNodesDataSource)(nil)
)

func NewFlexvmNodesDataSource() datasource.DataSource {
	return &flexvmNodesDataSource{}
}

// flexvmNodesDataSource lists all nodes within an i3D.net FlexVM Cloud.
type flexvmNodesDataSource struct {
	client *one_api.Client
}

type flexvmNodesDataSourceModel struct {
	CloudID types.String `tfsdk:"cloud_id"`
	Nodes   types.List   `tfsdk:"nodes"`
}

var flexvmNodeObjectAttrTypes = map[string]attr.Type{
	"id":     types.StringType,
	"name":   types.StringType,
	"serial": types.StringType,
	"status": types.StringType,
}

func (d *flexvmNodesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *flexvmNodesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexvm_nodes"
}

func (d *flexvmNodesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get all nodes within an i3D.net FlexVM Cloud.",
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the Cloud whose nodes to list.",
			},
			"nodes": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The nodes that belong to the Cloud.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
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
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The status of the Node. One of: `created`, `requested`, `bootstrapping`, `running`, `failed`, `deleting`, `deleted`.",
						},
					},
				},
			},
		},
	}
}

func (d *flexvmNodesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data flexvmNodesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nodesResp, err := d.client.FlexvmListNodes(ctx, data.CloudID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing FlexVM Cloud Nodes",
			"Could not list FlexVM Cloud Nodes for cloud_id "+data.CloudID.ValueString()+": "+err.Error(),
		)
		return
	}

	if nodesResp.ErrorResponse != nil {
		if nodesResp.ErrorResponse.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError(
				"FlexVM Cloud not found",
				fmt.Sprintf("No FlexVM Cloud found for id %s", data.CloudID.ValueString()),
			)
			return
		}
		AddErrorResponseToDiags("Error listing FlexVM Cloud Nodes", nodesResp.ErrorResponse, &resp.Diagnostics)
		return
	}

	nodeValues := make([]attr.Value, 0, len(nodesResp.Nodes))
	for _, node := range nodesResp.Nodes {
		obj, diags := types.ObjectValue(flexvmNodeObjectAttrTypes, map[string]attr.Value{
			"id":     types.StringValue(node.ID),
			"name":   types.StringValue(node.Name),
			"serial": types.StringValue(node.Serial),
			"status": types.StringValue(node.Status),
		})
		resp.Diagnostics.Append(diags...)
		nodeValues = append(nodeValues, obj)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	nodesList, diags := types.ListValue(types.ObjectType{AttrTypes: flexvmNodeObjectAttrTypes}, nodeValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Nodes = nodesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

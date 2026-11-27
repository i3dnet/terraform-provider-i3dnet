package provider

import (
	"context"

	"terraform-provider-i3dnet/internal/cdn_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*cdnPredefinedAllowlistsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cdnPredefinedAllowlistsDataSource)(nil)
)

func NewCDNPredefinedAllowlistsDataSource() datasource.DataSource {
	return &cdnPredefinedAllowlistsDataSource{}
}

type cdnPredefinedAllowlistsDataSource struct {
	client *cdn_api.Client
}

type cdnPredefinedAllowlistsModel struct {
	Allowlists []cdnPredefinedAllowlistModel `tfsdk:"allowlists"`
}

type cdnPredefinedAllowlistModel struct {
	Key         types.String `tfsdk:"key"`
	Label       types.String `tfsdk:"label"`
	Description types.String `tfsdk:"description"`
}

func (d *cdnPredefinedAllowlistsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = cdnClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *cdnPredefinedAllowlistsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_predefined_allowlists"
}

func (d *cdnPredefinedAllowlistsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the predefined allowlists a CDN resource can use in its `predefined_allowlist` attribute.",
		Attributes: map[string]schema.Attribute{
			"allowlists": schema.ListNestedAttribute{
				MarkdownDescription: "The predefined allowlists.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "Key to set in `predefined_allowlist`.",
							Computed:            true,
						},
						"label": schema.StringAttribute{
							MarkdownDescription: "Short name of the allowlist.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Who the allowlist lets in.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *cdnPredefinedAllowlistsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	lists, err := d.client.ListPredefinedAllowLists(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading CDN predefined allowlists", err.Error())
		return
	}

	state := cdnPredefinedAllowlistsModel{Allowlists: []cdnPredefinedAllowlistModel{}}
	for _, l := range lists {
		state.Allowlists = append(state.Allowlists, cdnPredefinedAllowlistModel{
			Key:         types.StringValue(l.Key),
			Label:       types.StringValue(l.Label),
			Description: types.StringValue(l.Description),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

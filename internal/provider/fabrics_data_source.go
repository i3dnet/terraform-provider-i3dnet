package provider

import (
	"context"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*fabricsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*fabricsDataSource)(nil)
)

func NewFabricsDataSource() datasource.DataSource {
	return &fabricsDataSource{}
}

// fabricsDataSource lists the fabrics (i3D.net locations) where networks can
// be created.
type fabricsDataSource struct {
	client *network_api.ClientWithResponses
}

type fabricsDataSourceModel struct {
	Fabrics []fabricModel `tfsdk:"fabrics"`
}

type fabricModel struct {
	Slug        types.String `tfsdk:"slug"`
	DisplayName types.String `tfsdk:"display_name"`
	Metro       types.String `tfsdk:"metro"`
}

func (d *fabricsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = networkClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *fabricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fabrics"
}

func (d *fabricsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the fabrics (i3D.net locations) where networks can be created. Use a " +
			"`slug` from this list wherever a network takes a `fabric`, rather than hard-coding it.",
		Attributes: map[string]schema.Attribute{
			"fabrics": schema.ListNestedAttribute{
				MarkdownDescription: "List of available fabrics.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug": schema.StringAttribute{
							MarkdownDescription: "Fabric identifier, the value to use in `fabric` arguments.",
							Computed:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "Human-readable fabric name.",
							Computed:            true,
						},
						"metro": schema.StringAttribute{
							MarkdownDescription: "Metro area the fabric belongs to.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *fabricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	fabrics, err := d.client.ListAllFabrics(ctx)
	if err != nil {
		addNetworkAPIErrorToDiags("Unable to read i3D.net fabrics", err, &resp.Diagnostics)
		return
	}

	data := fabricsDataSourceModel{Fabrics: make([]fabricModel, 0, len(fabrics))}
	for _, f := range fabrics {
		data.Fabrics = append(data.Fabrics, fabricModel{
			Slug:        types.StringValue(f.Slug),
			DisplayName: types.StringValue(f.DisplayName),
			Metro:       types.StringValue(f.Metro),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

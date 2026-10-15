package provider

import (
	"context"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*networksDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networksDataSource)(nil)
)

func NewNetworksDataSource() datasource.DataSource {
	return &networksDataSource{}
}

// networksDataSource lists the networks of the account, optionally filtered.
type networksDataSource struct {
	client *network_api.ClientWithResponses
}

type networksDataSourceModel struct {
	Fabric       types.String   `tfsdk:"fabric"`
	State        types.String   `tfsdk:"state"`
	NameContains types.String   `tfsdk:"name_contains"`
	Networks     []networkModel `tfsdk:"networks"`
}

func (d *networksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = networkClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *networksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *networksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	itemAttributes := networkDataSourceAttributes()
	itemAttributes["id"] = schema.StringAttribute{
		MarkdownDescription: "ID (UUID) of the network.",
		Computed:            true,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the networks on your account, including networks created outside Terraform. " +
			"All filters are optional and combine: a network is returned only if it matches every filter that is " +
			"set. Terraform only reads the networks: it does not manage, change or destroy them.",
		Attributes: map[string]schema.Attribute{
			"fabric": schema.StringAttribute{
				MarkdownDescription: "Return only networks in this fabric, a `slug` from `i3dnet_fabrics`.",
				Optional:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Return only networks in this state: `provisioning`, `active` or `failed`.",
				Optional:            true,
			},
			"name_contains": schema.StringAttribute{
				MarkdownDescription: "Return only networks whose name contains this text (not case-sensitive). " +
					"Networks without a name never match.",
				Optional: true,
			},
			"networks": schema.ListNestedAttribute{
				MarkdownDescription: "Networks that match the filters. An empty list when none match.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: itemAttributes,
				},
			},
		},
	}
}

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := network_api.ListNetworksParams{
		FiltersFabricEq:     data.Fabric.ValueStringPointer(),
		FiltersNameContains: data.NameContains.ValueStringPointer(),
	}
	if state := data.State.ValueStringPointer(); state != nil {
		filter := network_api.ListNetworksParamsFiltersStateEq(*state)
		params.FiltersStateEq = &filter
	}

	networks, err := d.client.ListAllNetworks(ctx, params)
	if err != nil {
		addNetworkAPIErrorToDiags("Unable to read i3D.net networks", err, &resp.Diagnostics)
		return
	}

	data.Networks = make([]networkModel, 0, len(networks))
	for _, n := range networks {
		data.Networks = append(data.Networks, networkModelFromResponse(n))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

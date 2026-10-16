package provider

import (
	"context"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*networkPrefixesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networkPrefixesDataSource)(nil)
)

func NewNetworkPrefixesDataSource() datasource.DataSource {
	return &networkPrefixesDataSource{}
}

// networkPrefixesDataSource lists the prefixes of the account, optionally
// filtered.
type networkPrefixesDataSource struct {
	client *network_api.ClientWithResponses
}

type networkPrefixesDataSourceModel struct {
	NetworkID types.String  `tfsdk:"network_id"`
	Family    types.Int64   `tfsdk:"family"`
	Fabric    types.String  `tfsdk:"fabric"`
	Metro     types.String  `tfsdk:"metro"`
	Prefixes  []prefixModel `tfsdk:"prefixes"`
}

func (d *networkPrefixesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = networkClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *networkPrefixesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_prefixes"
}

func (d *networkPrefixesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the IP prefixes (address ranges) on your account, including prefixes created " +
			"outside Terraform. Prefixes created through a network's initial prefixes appear here too. All " +
			"filters are optional and combine: a prefix is returned only if it matches every filter that is " +
			"set. Terraform only reads the prefixes: it does not manage, change or destroy them.",
		Attributes: map[string]schema.Attribute{
			"network_id": schema.StringAttribute{
				MarkdownDescription: "Return only prefixes on the network with this ID (UUID). This can include " +
					"older prefixes on the same VLAN that were not created through the network: those have a " +
					"null `network_id`.",
				Optional: true,
				Validators: []validator.String{
					uuidV7Validator{},
				},
			},
			"family": schema.Int64Attribute{
				MarkdownDescription: "Return only prefixes of this IP family: `4` for IPv4 or `6` for IPv6.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(4, 6),
				},
			},
			"fabric": schema.StringAttribute{
				MarkdownDescription: "Return only prefixes in this fabric, a `slug` from `i3dnet_fabrics`. Cannot " +
					"be used together with `metro`.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("metro")),
				},
			},
			"metro": schema.StringAttribute{
				MarkdownDescription: "Return only prefixes in this metro area, a `metro` from `i3dnet_fabrics`. " +
					"Cannot be used together with `fabric`.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("fabric")),
				},
			},
			"prefixes": schema.ListNestedAttribute{
				MarkdownDescription: "Prefixes that match the filters. An empty list when none match.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: prefixDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *networkPrefixesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkPrefixesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := network_api.ListPrefixesParams{
		FiltersFabricEq: data.Fabric.ValueStringPointer(),
		FiltersMetroEq:  data.Metro.ValueStringPointer(),
	}
	if networkID := data.NetworkID.ValueStringPointer(); networkID != nil {
		id, err := uuid.Parse(*networkID)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("network_id"), "Invalid network id", err.Error())
			return
		}
		params.FiltersNetworkIdEq = &id
	}
	if family := data.Family.ValueInt64Pointer(); family != nil {
		filter := network_api.ListPrefixesParamsFiltersFamilyEq(*family)
		params.FiltersFamilyEq = &filter
	}

	prefixes, err := d.client.ListAllPrefixes(ctx, params)
	if err != nil {
		addNetworkAPIErrorToDiags("Unable to read i3D.net network prefixes", err, &resp.Diagnostics)
		return
	}

	data.Prefixes = make([]prefixModel, 0, len(prefixes))
	for _, p := range prefixes {
		data.Prefixes = append(data.Prefixes, prefixModelFromResponse(p))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// prefixDataSourceAttributes returns the computed attributes of a prefix, the
// items of i3dnet_network_prefixes.
func prefixDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "ID (UUID) of the prefix.",
			Computed:            true,
		},
		"cidr": schema.StringAttribute{
			MarkdownDescription: "The prefix in CIDR notation, for example `203.0.113.0/28`. Null while the " +
				"prefix is still being provisioned: i3D.net has not picked the addresses yet.",
			Computed: true,
		},
		"family": schema.Int64Attribute{
			MarkdownDescription: "IP family: `4` for IPv4 or `6` for IPv6.",
			Computed:            true,
		},
		"state": schema.StringAttribute{
			MarkdownDescription: "State of the prefix, for example `provisioning`, `active` or `failed`. More " +
				"values can appear, such as `reserved`.",
			Computed: true,
		},
		"gateway_ip": schema.StringAttribute{
			MarkdownDescription: "Gateway IP address of the prefix. Null when the prefix has no gateway, for " +
				"example on a private prefix.",
			Computed: true,
		},
		"router_ips": schema.ListAttribute{
			MarkdownDescription: "IP addresses of the i3D.net routers that serve the prefix. Null when no " +
				"routers are recorded.",
			ElementType: types.StringType,
			Computed:    true,
		},
		"vlan_id": schema.Int64Attribute{
			MarkdownDescription: "VLAN ID the prefix is on. Null when the prefix has no VLAN.",
			Computed:            true,
		},
		"fabric": schema.StringAttribute{
			MarkdownDescription: "Fabric (i3D.net location) of the prefix, a `slug` from `i3dnet_fabrics`. Null " +
				"when the prefix has no fabric.",
			Computed: true,
		},
		"metro": schema.StringAttribute{
			MarkdownDescription: "Metro area of the prefix's fabric. Null when `fabric` is null.",
			Computed:            true,
		},
		"network_id": schema.StringAttribute{
			MarkdownDescription: "ID of the network the prefix was created through. Null for prefixes that are " +
				"not part of a network.",
			Computed: true,
		},
		"retention": schema.StringAttribute{
			MarkdownDescription: "`persistent` (stays until it is deleted) or `ephemeral` (i3D.net set it up for " +
				"your account and removes it again together with what it was set up for). Null for prefixes " +
				"created before retention existed.",
			Computed: true,
		},
		"contract_id": schema.StringAttribute{
			MarkdownDescription: "Contract the prefix is billed under. Null when none is recorded, for example " +
				"on an `ephemeral` prefix.",
			Computed: true,
		},
		"customer_description": schema.StringAttribute{
			MarkdownDescription: "Your description of the prefix. Null when never set.",
			Computed:            true,
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "When the prefix was created (RFC 3339). Null when not known.",
			Computed:            true,
		},
		"updated_at": schema.StringAttribute{
			MarkdownDescription: "When the prefix last changed (RFC 3339). Null when not known.",
			Computed:            true,
		},
	}
}

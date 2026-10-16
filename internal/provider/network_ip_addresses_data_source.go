package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = (*networkIPAddressesDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*networkIPAddressesDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*networkIPAddressesDataSource)(nil)
)

func NewNetworkIPAddressesDataSource() datasource.DataSource {
	return &networkIPAddressesDataSource{}
}

// networkIPAddressesDataSource lists the IP addresses of one network or one
// prefix. It never claims or releases addresses.
type networkIPAddressesDataSource struct {
	client *network_api.ClientWithResponses
}

type networkIPAddressesDataSourceModel struct {
	NetworkID   types.String     `tfsdk:"network_id"`
	PrefixID    types.String     `tfsdk:"prefix_id"`
	IPAddresses []ipAddressModel `tfsdk:"ip_addresses"`
}

func (d *networkIPAddressesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = networkClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *networkIPAddressesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_ip_addresses"
}

func (d *networkIPAddressesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the IP addresses in use on one network or on one prefix: the addresses " +
			"given to your servers and the addresses i3D.net uses itself, such as gateway and router " +
			"addresses. Set exactly one of `network_id` or `prefix_id`. Free addresses are not listed.\n\n" +
			"The addresses are read-only: i3D.net allocates them when a server is provisioned and releases " +
			"them when the server is torn down. Terraform only reads them.\n\nThe read fails if the network " +
			"or prefix does not exist on your account.",
		Attributes: map[string]schema.Attribute{
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the network. Lists the addresses under all of the network's " +
					"prefixes, IPv4 and IPv6. Cannot be used together with `prefix_id`.",
				Optional: true,
				Validators: []validator.String{
					uuidV7Validator{},
				},
			},
			"prefix_id": schema.StringAttribute{
				MarkdownDescription: "ID (UUID) of the prefix. Lists the addresses under this prefix only. Also " +
					"works for older prefixes that are not part of a network. Cannot be used together with " +
					"`network_id`.",
				Optional: true,
			},
			"ip_addresses": schema.ListNestedAttribute{
				MarkdownDescription: "IP addresses in use. An empty list when there are none.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: ipAddressDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *networkIPAddressesDataSource) ConfigValidators(ctx context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("network_id"),
			path.MatchRoot("prefix_id"),
		),
	}
}

func (d *networkIPAddressesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data networkIPAddressesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Exactly one of network_id / prefix_id is set (enforced by ConfigValidators).
	var (
		addresses []network_api.IPAddressResponse
		notFound  string
		err       error
	)
	if networkID := data.NetworkID.ValueStringPointer(); networkID != nil {
		id, parseErr := uuid.Parse(*networkID)
		if parseErr != nil {
			resp.Diagnostics.AddAttributeError(path.Root("network_id"), "Invalid network id", parseErr.Error())
			return
		}
		notFound = "network not found on this account"
		addresses, err = d.client.ListAllNetworkIPAddresses(ctx, id)
	} else {
		id, parseErr := uuid.Parse(data.PrefixID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddAttributeError(path.Root("prefix_id"), "Invalid prefix id",
				fmt.Sprintf("%q must be a UUID: %s.", data.PrefixID.ValueString(), parseErr))
			return
		}
		notFound = "prefix not found on this account"
		addresses, err = d.client.ListAllPrefixIPAddresses(ctx, id)
	}
	if err != nil {
		var apiErr *network_api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError(notFound, apiErr.DiagnosticDetail())
			return
		}

		addNetworkAPIErrorToDiags("Unable to read i3D.net IP addresses", err, &resp.Diagnostics)
		return
	}

	data.IPAddresses = make([]ipAddressModel, 0, len(addresses))
	for _, ip := range addresses {
		data.IPAddresses = append(data.IPAddresses, ipAddressModelFromResponse(ip))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ipAddressDataSourceAttributes returns the computed attributes of an IP
// address, the items of i3dnet_network_ip_addresses.
func ipAddressDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "ID (UUID) of the IP address.",
			Computed:            true,
		},
		"address": schema.StringAttribute{
			MarkdownDescription: "The IP address, without prefix length, for example `203.0.113.10`.",
			Computed:            true,
		},
		"family": schema.Int64Attribute{
			MarkdownDescription: "IP family: `4` for IPv4 or `6` for IPv6.",
			Computed:            true,
		},
		"prefix_id": schema.StringAttribute{
			MarkdownDescription: "ID of the prefix the address belongs to.",
			Computed:            true,
		},
		"role": schema.StringAttribute{
			MarkdownDescription: "What i3D.net uses the address for, for example `gateway`, `router-ip` or " +
				"`reserved` (kept free for i3D.net). More values can appear. Null for an address given to one " +
				"of your servers: filter on `role == null` to get your own addresses.",
			Computed: true,
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "When the address was allocated (RFC 3339). Null when not known.",
			Computed:            true,
		},
	}
}

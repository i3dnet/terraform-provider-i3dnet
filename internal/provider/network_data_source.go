package provider

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var (
	_ datasource.DataSource              = (*networkDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networkDataSource)(nil)
)

// uuidPattern matches a UUID in its canonical 8-4-4-12 hex form.
var uuidPattern = regexp.MustCompile(`^(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

// networkDataSource reads one network by id without managing it.
type networkDataSource struct {
	client *network_api.ClientWithResponses
}

func (d *networkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = networkClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *networkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *networkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := networkDataSourceAttributes()
	attributes["id"] = schema.StringAttribute{
		MarkdownDescription: "ID (UUID) of the network to read.",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.RegexMatches(uuidPattern, "must be a UUID"),
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads one network by its ID. Use it for networks created outside Terraform, for " +
			"example networks i3D.net set up for your account. Terraform only reads the network: it does not " +
			"manage, change or destroy it.\n\nThe read fails if no network with this ID exists on your account.",
		Attributes: attributes,
	}
}

func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config networkModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var id openapi_types.UUID
	if err := id.UnmarshalText([]byte(config.ID.ValueString())); err != nil {
		resp.Diagnostics.AddError("Invalid network id", err.Error())
		return
	}

	network, err := network_api.Do[network_api.NetworkResponse](d.client.GetNetworkById(ctx, id))
	if err != nil {
		var apiErr *network_api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("network not found on this account", apiErr.DiagnosticDetail())
			return
		}

		addNetworkAPIErrorToDiags("Unable to read i3D.net network", err, &resp.Diagnostics)
		return
	}

	data := networkModelFromResponse(network)
	// Keep the id exactly as configured: Terraform rejects a read that changes
	// a configured value, which the API's lower-case form would do for an
	// upper-case id.
	data.ID = config.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// networkDataSourceAttributes returns the computed attributes of a network,
// shared by i3dnet_network and the items of i3dnet_networks. The caller adds
// "id".
func networkDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"fabric": schema.StringAttribute{
			MarkdownDescription: "Fabric (i3D.net location) of the network, a `slug` from `i3dnet_fabrics`.",
			Computed:            true,
		},
		"connectivity": schema.StringAttribute{
			MarkdownDescription: "`public` (connected to the internet) or `private` (traffic between your own machines only).",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the network. Null when the network has no name.",
			Computed:            true,
		},
		"addressing_source": schema.StringAttribute{
			MarkdownDescription: "Where the network's addresses come from: `provider` (i3D.net supplies them) or " +
				"`none` (the network has no i3D.net addresses).",
			Computed: true,
		},
		"state": schema.StringAttribute{
			MarkdownDescription: "State of the network: `provisioning`, `active` or `failed`.",
			Computed:            true,
		},
		"vlan_id": schema.Int64Attribute{
			MarkdownDescription: "VLAN ID of the network, to configure on servers attached to it. Null while the " +
				"network is still being provisioned.",
			Computed: true,
		},
		"retention": schema.StringAttribute{
			MarkdownDescription: "`persistent` (stays until it is deleted) or `ephemeral` (i3D.net set it up for " +
				"your account and removes it again together with what it was set up for).",
			Computed: true,
		},
		"routing_enabled": schema.BoolAttribute{
			MarkdownDescription: "Private networks only: `true` when the network has a gateway and routes " +
				"traffic to your other networks in the same routing domain, `false` when it is isolated. Null on " +
				"public networks.",
			Computed: true,
		},
		"routing_domain_id": schema.StringAttribute{
			MarkdownDescription: "ID of the routing domain the network belongs to. Networks with the same value " +
				"route traffic to each other. Set only when `routing_enabled` is `true`, null otherwise.",
			Computed: true,
		},
		"created_at": schema.StringAttribute{
			MarkdownDescription: "When the network was created (RFC 3339).",
			Computed:            true,
		},
		"updated_at": schema.StringAttribute{
			MarkdownDescription: "When the network last changed (RFC 3339).",
			Computed:            true,
		},
	}
}

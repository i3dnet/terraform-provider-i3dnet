package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ datasource.DataSource              = (*networkDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networkDataSource)(nil)
)

// uuidV7Validator accepts only version 7 UUIDs: every network id is one,
// whether the network was created through the API or set up by i3D.net.
type uuidV7Validator struct{}

func (uuidV7Validator) Description(context.Context) string {
	return "must be a version 7 UUID"
}

func (v uuidV7Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v uuidV7Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	u, err := uuid.Parse(req.ConfigValue.ValueString())
	if err != nil || u.Version() != 7 || u.Variant() != uuid.RFC4122 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid network id",
			fmt.Sprintf("%q %s.", req.ConfigValue.ValueString(), v.Description(ctx)))
	}
}

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
			uuidV7Validator{},
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

	id, err := uuid.Parse(config.ID.ValueString())
	if err != nil {
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

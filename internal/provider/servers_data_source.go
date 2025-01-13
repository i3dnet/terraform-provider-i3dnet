package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-i3d/internal/one_api"
	"terraform-provider-i3d/internal/provider/datasource_servers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ datasource.DataSource = (*serversDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*serversDataSource)(nil)

func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

type serversDataSource struct {
	client *one_api.Client
}

func (d *serversDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serversDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (d *serversDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_servers.ServersDataSourceSchema(ctx)
}

func (d *serversDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_servers.ServersModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	respBody, diags := d.client.CallFlexMetalAPI("GET", "servers", nil)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	// Example data value setting
	diags = ParseServersResponseBodyDataSource(ctx, respBody, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ParseServersResponseBodyDataSource is a helper function to parse the response body from the FlexMetal API
func ParseServersResponseBodyDataSource(ctx context.Context, responseBody []byte, servers *datasource_servers.ServersModel) diag.Diagnostics {
	var diags diag.Diagnostics
	// parse the response body
	unmarshalledData := []map[string]any{}

	err := json.Unmarshal(responseBody, &unmarshalledData)
	if err != nil {
		diags.AddError("API Response Error", fmt.Sprintf("Failed to parse response body: %s, body content: %s", err, responseBody))
		return diags
	}

	for _, answer := range unmarshalledData {

		server := datasource_servers.ServersValue{}
		server.Uuid = basetypes.NewStringValue(answer["uuid"].(string))
		server.Status = basetypes.NewStringValue(answer["status"].(string))
		server.StatusMessage = basetypes.NewStringValue(answer["statusMessage"].(string))
		// wipe the list
		server.IpAddresses = basetypes.NewListUnknown(datasource_servers.IpAddressesType{})
		if answer["ipAddresses"] != nil {
			for _, ip := range answer["ipAddresses"].([]interface{}) {

				ipAddress := datasource_servers.NewIpAddressesValueMust(
					map[string]attr.Type{
						"ip_address": basetypes.StringType{},
					},
					map[string]attr.Value{
						"ip_address": basetypes.NewStringValue(ip.(map[string]any)["ipAddress"].(string)),
					})
				values := append(server.IpAddresses.Elements(), ipAddress)
				server.IpAddresses = basetypes.NewListValueMust(
					ipAddress.Type(ctx),
					values,
				)

			}
		}
		server.Tags, _ = basetypes.NewListValueFrom(ctx, basetypes.StringType{}, answer["tags"].(string))
		if answer["createdAt"] != nil {
			server.CreatedAt = basetypes.NewInt64Value(int64(answer["createdAt"].(float64)))
		} else {
			server.CreatedAt = basetypes.NewInt64Value(0)
		}
		if answer["deliveredAt"] != nil {
			server.DeliveredAt = basetypes.NewInt64Value(int64(answer["deliveredAt"].(float64)))
		} else {
			server.DeliveredAt = basetypes.NewInt64Value(0)
		}
		if answer["releasedAt"] != nil {
			server.ReleasedAt = basetypes.NewInt64Value(int64(answer["releasedAt"].(float64)))
		} else {
			server.ReleasedAt = basetypes.NewInt64Value(0)
		}

		values := append(servers.Servers.Elements(), server)
		servers.Servers = basetypes.NewSetValueMust(servers.Servers.Type(context.Background()), values)
	}

	return diags
}

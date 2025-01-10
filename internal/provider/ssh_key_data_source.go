package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3d/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*sshKeyDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*sshKeyDataSource)(nil)

func NewSshKeyDataSource() datasource.DataSource {
	return &sshKeyDataSource{}
}

// sshKeyDataSource gets information of a ssh key by name.
// This data source provides the name, created_at, public_key and uuid as configured on i3d.
// An error is triggered if the provided ssh key name does not exist.
type sshKeyDataSource struct {
	client *one_api.Client
}

func (d *sshKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type sshKeyDataSourceModel struct {
	CreatedAt types.Int64  `tfsdk:"created_at"`
	Name      types.String `tfsdk:"name"`
	PublicKey types.String `tfsdk:"public_key"`
	Uuid      types.String `tfsdk:"uuid"`
}

func (d *sshKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (d *sshKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a ssh key. This data source provides the name, created_at, public_key and uuid as configured on your i3d account. This is useful if the ssh key in question is not managed by Terraform or you need to utilize any of the keys data.\n\nAn error is triggered if the provided ssh key name does not exist.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "SSH key name.",
				MarkdownDescription: "SSH key name.",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				Description:         "SSH key UUID as specified in RFC 4122.",
				MarkdownDescription: "SSH key UUID as specified in RFC 4122.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "SSH key createdAt.",
				MarkdownDescription: "SSH key createdAt.",
			},
			"public_key": schema.StringAttribute{
				Computed:            true,
				Description:         "Public SSH key contents.",
				MarkdownDescription: "Public SSH key contents.",
			},
		},
	}
}

func (d *sshKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sshKeyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	sshKeys, err := d.client.ListSSHKeys()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read i3d ssh keys",
			err.Error(),
		)
		return
	}

	var foundKey *one_api.SSHKey
	for _, v := range sshKeys {
		if v.Name == data.Name.ValueString() {
			foundKey = &v
		}
	}
	if foundKey == nil {
		resp.Diagnostics.AddError(
			"No ssh key found",
			fmt.Sprintf("No ssh key found for name %s", data.Name),
		)
		return
	}

	data.PublicKey = types.StringValue(foundKey.PublicKey)
	data.Name = types.StringValue(foundKey.Name)
	data.CreatedAt = types.Int64Value(foundKey.CreatedAt)
	data.Uuid = types.StringValue(foundKey.Uuid)

	// Save data into Terraform data
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

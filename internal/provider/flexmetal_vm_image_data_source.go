package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/datasource_flexmetal_vm_image"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*vmImageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vmImageDataSource)(nil)
)

func NewVmImageDataSource() datasource.DataSource {
	return &vmImageDataSource{}
}

type vmImageDataSource struct {
	client *one_api.Client
}

func (d *vmImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_vm_image"
}

func (d *vmImageDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_flexmetal_vm_image.VmImageDataSourceSchema(ctx)
}

func (d *vmImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vmImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_flexmetal_vm_image.VmImagesData

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := one_api.VmImageFilters{
		Slug:     data.Slug.ValueString(),
		OsFamily: data.OsFamily.ValueString(),
	}

	images, err := d.client.ListVmImages(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list VM images", err.Error())
		return
	}

	var imageValues []datasource_flexmetal_vm_image.VmImageValue
	for _, img := range images {
		imageValues = append(imageValues, datasource_flexmetal_vm_image.VmImageValue{
			ID:       types.StringValue(img.ID),
			Slug:     types.StringValue(img.Slug),
			Name:     types.StringValue(img.Name),
			OsFamily: types.StringValue(img.OsFamily),
			Version:  types.StringValue(img.Version),
		})
	}

	data.Images = imageValues
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

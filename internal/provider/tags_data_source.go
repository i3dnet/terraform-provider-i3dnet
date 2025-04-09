package provider

import (
	"context"
	"fmt"
	"maps"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/datasource_tags"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ datasource.DataSource              = (*tagsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*tagsDataSource)(nil)
)

func NewTagsDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

type tagsDataSource struct {
	client *one_api.Client
}

func (d *tagsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = providerData.Client
}

func (d *tagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	generatedSchema := datasource_tags.TagsDataSourceSchema(ctx)
	overrideSchema := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional:            true,
			Description:         "Filter by name",
			MarkdownDescription: "Filter by name",
		},
	}

	maps.Insert(generatedSchema.Attributes, maps.All(overrideSchema))

	generatedSchema.Description = "Returns a list of tags in your i3D.net account, with the ability to filter them by name. " +
		"If no name is specified, all tags will be returned."
	generatedSchema.MarkdownDescription = generatedSchema.Description

	resp.Schema = generatedSchema
}

type TagsModel struct {
	Name types.String `tfsdk:"name"`
	Tags types.Set    `tfsdk:"tags"`
}

func (d *tagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TagsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	tagsData, err := d.client.ListTags(ctx, data.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read i3D.net tags",
			err.Error(),
		)
		return
	}

	var elements []attr.Value

	for _, tagResp := range tagsData {
		rs, diags := resources(ctx, tagResp.Resources.Count, tagResp.Resources.FlexMetalServers.Count)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		tagElem, diags := datasource_tags.NewTagsValue(
			datasource_tags.TagsValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"tag":       types.StringValue(tagResp.Tag),
				"resources": rs,
			},
		)

		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		elements = append(elements, tagElem)
	}

	data.Tags = types.SetValueMust(datasource_tags.TagsType{
		ObjectType: types.ObjectType{
			AttrTypes: datasource_tags.TagsValue{}.AttributeTypes(ctx),
		},
	}, elements)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func resources(ctx context.Context, resourcesCount int64, serversCount int64) (basetypes.ObjectValue, diag.Diagnostics) {
	elementTypes := map[string]attr.Type{
		"count": basetypes.Int64Type{},
		"flex_metal_servers": basetypes.ObjectType{
			AttrTypes: datasource_tags.FlexMetalServersValue{}.AttributeTypes(ctx),
		},
	}

	servers, diags := flexmetalServers(serversCount)
	if diags.HasError() {
		return basetypes.ObjectValue{}, diags
	}

	elements := map[string]attr.Value{
		"count":              types.Int64Value(resourcesCount),
		"flex_metal_servers": servers,
	}
	return types.ObjectValue(elementTypes, elements)
}

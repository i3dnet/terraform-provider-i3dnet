package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-i3d/internal/one_api"
	"terraform-provider-i3d/internal/provider/datasource_tags"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ datasource.DataSource = (*tagsDataSource)(nil)

func NewTagsDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

type tagsDataSource struct{}

func (d *tagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_tags.TagsDataSourceSchema(ctx)
}

func (d *tagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasource_tags.TagsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	respBody, diags := one_api.CallFlexMetalAPI("GET", "tags", nil)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	// Example data value setting
	diags = ParseTagsResponseBodyDataSource(ctx, respBody, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ParseTagsResponseBodyDataSource is a helper function to parse the response body from the FlexMetal API
func ParseTagsResponseBodyDataSource(ctx context.Context, responseBody []byte, tags *datasource_tags.TagsModel) diag.Diagnostics {
	var diags diag.Diagnostics
	// parse the response body
	unmarshalledData := []map[string]any{}

	err := json.Unmarshal(responseBody, &unmarshalledData)
	if err != nil {
		diags.AddError("API Response Error", fmt.Sprintf("Failed to parse response body: %s, body content: %s", err, responseBody))
		return diags
	}

	for _, answer := range unmarshalledData {

		tag := datasource_tags.TagsValue{}
		tag.Tag = basetypes.NewStringValue(answer["tag"].(string))

		values := append(tags.Tags.Elements(), tag)
		tags.Tags = basetypes.NewSetValueMust(tags.Tags.Type(context.Background()), values)
	}

	return diags
}

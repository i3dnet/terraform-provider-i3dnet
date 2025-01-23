package provider

import (
	"context"
	"fmt"
	"maps"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_tag"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = (*tagResource)(nil)
var _ resource.ResourceWithConfigure = (*tagResource)(nil)
var _ resource.ResourceWithImportState = (*tagResource)(nil)

func NewTagResource() resource.Resource {
	return &tagResource{}
}

type tagResource struct {
	client *one_api.Client
}

func (r *tagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *tagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*one_api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api_utils.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *tagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *tagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_tag.TagResourceSchema(ctx)

	resources := generatedSchema.Attributes["resources"].(schema.SingleNestedAttribute)

	// override to add id used in update
	overrideSchema := map[string]schema.Attribute{
		// add id for import
		"id": schema.StringAttribute{
			Computed: true,
		},
		// replace tag with name
		"name": generatedSchema.Attributes["tag"],
		// make resources computed instead of required
		"resources": schema.SingleNestedAttribute{
			Attributes: map[string]schema.Attribute{
				"count":              resources.Attributes["count"],
				"flex_metal_servers": resources.Attributes["flex_metal_servers"],
			},
			CustomType:          resources.CustomType,
			Computed:            true,
			Description:         resources.GetDescription(),
			MarkdownDescription: resources.GetMarkdownDescription(),
		},
	}

	maps.Insert(generatedSchema.Attributes, maps.All(overrideSchema))
	// tag is replaced by name, so we no longer need it in the schema
	delete(generatedSchema.Attributes, "tag")

	resp.Schema = generatedSchema
}

type TagModel struct {
	Resources resource_tag.ResourcesValue `tfsdk:"resources"`
	Name      types.String                `tfsdk:"name"`
	ID        types.String                `tfsdk:"id"`
}

func (r *tagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TagModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create API call logic
	tagResp, err := r.client.CreateTag(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating tag",
			"Could not create tag, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(tagRespToPlan(ctx, tagResp, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func tagRespToPlan(ctx context.Context, tagResp *one_api.Tag, data *TagModel) diag.Diagnostics {
	var diags diag.Diagnostics
	flexmetalServers, flexDiags := flexmetalServers(tagResp.Resources.FlexMetalServers.Count)

	diags.Append(flexDiags...)

	if diags.HasError() {
		return diags
	}

	data.Name = types.StringValue(tagResp.Tag)
	data.ID = types.StringValue(tagResp.Tag) // id is same as tag name
	data.Resources = resource_tag.NewResourcesValueMust(
		resource_tag.ResourcesValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"count":              types.Int64Value(tagResp.Resources.FlexMetalServers.Count),
			"flex_metal_servers": flexmetalServers,
		},
	)

	return diags
}

func flexmetalServers(serversCount int64) (basetypes.ObjectValue, diag.Diagnostics) {
	elementTypes := map[string]attr.Type{
		"count": types.Int64Type,
	}
	elements := map[string]attr.Value{
		"count": types.Int64Value(serversCount),
	}
	return types.ObjectValue(elementTypes, elements)
}

func (r *tagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TagModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	tagResp, err := r.client.GetTag(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting tag",
			"Could not create tag, unexpected error: "+err.Error(),
		)
		return
	}

	tagRespToPlan(ctx, tagResp, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TagModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// name attribute was changed
	oldName, newName := state.Name.ValueString(), plan.Name.ValueString()

	tagResp, err := r.client.UpdateTag(ctx, oldName, newName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating tag",
			"Could not update tag, unexpected error: "+err.Error(),
		)
		return
	}

	tagRespToPlan(ctx, tagResp, &plan)

	// Save updated plan into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *tagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TagModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	err := r.client.DeleteTag(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Tag",
			"Could not delete tag, unexpected error: "+err.Error(),
		)
		return
	}
}

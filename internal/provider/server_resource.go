package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_server"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var timeOut = 30 * time.Minute

var _ resource.Resource = (*serverResource)(nil)
var _ resource.ResourceWithConfigure = (*serverResource)(nil)

func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	client *one_api.Client
}

func (r *serverResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serverResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flexmetal_server"
}

func (r *serverResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := resource_flexmetal_server.FlexmetalServerResourceSchema(ctx)

	generatedSchema.MarkdownDescription = "FlexMetal servers are physical servers that can be requested and released at will."

	// make post_install_script, os.kernel_params and os.partitions as optional:true and computed:false
	// because they are not included in the GET response body
	generatedSchema.Attributes["post_install_script"] = schema.StringAttribute{
		Optional:            true,
		Computed:            false,
		Description:         generatedSchema.Attributes["post_install_script"].GetDescription(),
		MarkdownDescription: generatedSchema.Attributes["post_install_script"].GetMarkdownDescription(),
	}

	generatedOSAttribute := generatedSchema.Attributes["os"].(schema.SingleNestedAttribute)

	osAttributes := generatedOSAttribute.GetAttributes()

	kernelParams := osAttributes["kernel_params"].(schema.ListNestedAttribute)
	kernelParams.Optional = true
	kernelParams.Computed = false

	partitions := osAttributes["partitions"].(schema.ListNestedAttribute)
	partitions.Optional = true
	partitions.Computed = false

	generatedSchema.Attributes["os"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"kernel_params": kernelParams,
			"partitions":    partitions,
			"slug":          osAttributes["slug"],
		},
		CustomType:          generatedOSAttribute.CustomType,
		Required:            generatedOSAttribute.Required,
		Description:         generatedOSAttribute.GetDescription(),
		MarkdownDescription: generatedOSAttribute.GetMarkdownDescription(),
	}

	// For certain OS(talos, windows) ssh_key is not needed
	generatedSchema.Attributes["ssh_key"] = schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Description:         generatedSchema.Attributes["ssh_key"].GetDescription(),
		MarkdownDescription: generatedSchema.Attributes["ssh_key"].GetMarkdownDescription(),
	}

	resp.Schema = generatedSchema
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	kernelParams := []map[string]string{}
	for _, kernelParam := range data.Os.KernelParams.Elements() {
		kernelParam1 := kernelParam.(resource_flexmetal_server.KernelParamsValue)
		kernelParams = append(kernelParams, map[string]string{
			"key":   kernelParam1.Key.ValueString(),
			"value": kernelParam1.Value.ValueString(),
		})
	}
	tags := []string{}
	for _, tag := range data.Tags.Elements() {
		tags = append(tags, strings.Replace(tag.String(), "\"", "", -1))
	}

	sskKeys := []string{}

	for _, sshKey := range data.SshKey.Elements() {
		sskKeys = append(sskKeys, strings.Replace(sshKey.String(), "\"", "", -1))
	}

	type partitionReq struct {
		Target     string `json:"target"`
		Filesystem string `json:"filesystem"`
		Size       int64  `json:"size"`
	}

	var partitions []partitionReq
	for _, v := range data.Os.Partitions.Elements() {
		part := v.(resource_flexmetal_server.PartitionsValue)

		partitions = append(partitions, partitionReq{
			Target:     part.Target.ValueString(),
			Filesystem: part.Filesystem.ValueString(),
			Size:       part.Size.ValueInt64(),
		})
	}

	// Build the body for the API call
	postData := map[string]any{
		"name":         data.Name.ValueString(),
		"location":     data.Location.ValueString(),
		"instanceType": data.InstanceType.ValueString(),
		"os": map[string]any{
			"slug":         data.Os.Slug.ValueString(),
			"kernelParams": kernelParams,
			"partitions":   partitions,
		},
		"tags":              tags,
		"sshKey":            sskKeys,
		"postInstallScript": data.PostInstallScript.ValueString(),
	}
	postBody, _ := json.Marshal(postData)

	respBody, diags := r.client.CallFlexMetalAPI("POST", "servers", postBody)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(ParseResponseBody(ctx, respBody, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the actual time
	startTime := time.Now()

	// Waiting for the server to be ready
	for data.Status.ValueString() != "delivered" && data.Status.ValueString() != "failed" {
		respBody, diags = r.client.CallFlexMetalAPI("GET", fmt.Sprintf("servers/%s", data.Uuid.ValueString()), nil)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resp.Diagnostics.Append(ParseResponseBody(ctx, respBody, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if time.Since(startTime) > timeOut {
			resp.Diagnostics.AddError("Server creation timeout", "Server creation timeout")
			return
		}
		time.Sleep(10 * time.Second)
	}

	if data.Status.ValueString() == "failed" {
		resp.Diagnostics.AddError("Server creation failed", fmt.Sprintf("Status message: %s", data.StatusMessage.ValueString()))
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respBody, diags := r.client.CallFlexMetalAPI("GET", fmt.Sprintf("servers/%s", data.Uuid.ValueString()), nil)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(ParseResponseBody(ctx, respBody, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_flexmetal_server.FlexmetalServerModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respBody, diags := r.client.CallFlexMetalAPI("DELETE", fmt.Sprintf("servers/%s", data.Uuid.ValueString()), nil)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Delete API call logic

	resp.Diagnostics.Append(ParseResponseBody(ctx, respBody, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Status.ValueString() != "releasing" {
		resp.Diagnostics.AddError("Server deletion failed", fmt.Sprintf("Status message: %s", data.StatusMessage.ValueString()))
		return
	}
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// ParseResponseBody is a helper function to parse the response body from the FlexMetal API
func ParseResponseBody(ctx context.Context, responseBody []byte, server *resource_flexmetal_server.FlexmetalServerModel) diag.Diagnostics {
	var diags diag.Diagnostics
	// parse the response body
	unmarshalledData := []map[string]any{}

	err := json.Unmarshal(responseBody, &unmarshalledData)
	if err != nil {
		diags.AddError("API Response Error", fmt.Sprintf("Failed to parse response body: %s, body content: %s", err, responseBody))
		return diags
	}

	for _, answer := range unmarshalledData {
		server.Uuid = basetypes.NewStringValue(answer["uuid"].(string))
		server.Status = basetypes.NewStringValue(answer["status"].(string))
		server.StatusMessage = basetypes.NewStringValue(answer["statusMessage"].(string))
		// wipe the list
		server.IpAddresses = basetypes.NewListUnknown(resource_flexmetal_server.IpAddressesType{})
		if answer["ipAddresses"] != nil {
			for _, ip := range answer["ipAddresses"].([]interface{}) {

				ipAddress := resource_flexmetal_server.NewIpAddressesValueMust(
					map[string]attr.Type{
						"ip_address": basetypes.StringType{},
					},
					map[string]attr.Value{
						"ip_address": basetypes.NewStringValue(ip.(map[string]any)["ipAddress"].(string)),
					})
				values := append(server.IpAddresses.Elements(), ipAddress)
				server.IpAddresses = basetypes.NewListValueMust(
					ipAddress.Type(context.Background()),
					values,
				)

			}
		}
		server.Tags = basetypes.NewListNull(basetypes.StringType{})
		if answer["tags"] != nil {
			values := []attr.Value{}
			for _, tag := range answer["tags"].([]interface{}) {
				values = append(values, basetypes.NewStringValue(tag.(string)))
			}
			values = append(server.Tags.Elements(), values...)
			server.Tags = basetypes.NewListValueMust(
				basetypes.StringType{},
				values,
			)
		}
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

	}

	return diags
}

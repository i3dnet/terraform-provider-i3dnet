package provider

import (
	"context"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/resource_flexmetal_server"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// importedServer is the GET response shape: it carries name, location,
// instanceType and os.slug, but never sshKey, postInstallScript or the os
// sub-attributes.
func importedServer() *one_api.Server {
	s := &one_api.Server{
		Uuid:          "5f2b1c3d-0000-4000-8000-000000000001",
		Name:          "wk-0054",
		Status:        "delivered",
		StatusMessage: "Server delivered",
		Tags:          []string{"tf"},
		CreatedAt:     1000,
		DeliveredAt:   1100,
	}
	s.Location.Name = "EU: Rotterdam"
	s.InstanceType.Name = "bm7.std.8"
	s.Os.Slug = "ubuntu-2404-lts"
	return s
}

// On import the prior state is empty, so every attribute the API returns has
// to come from the response.
func TestServerRespToPlanPopulatesAttributesOnImport(t *testing.T) {
	ctx := context.Background()
	data := FlexmetalServerModel{}

	serverRespToPlan(ctx, importedServer(), &data)

	if got := data.Name.ValueString(); got != "wk-0054" {
		t.Errorf("name = %q, want %q", got, "wk-0054")
	}
	if got := data.Location.ValueString(); got != "EU: Rotterdam" {
		t.Errorf("location = %q, want %q", got, "EU: Rotterdam")
	}
	if got := data.InstanceType.ValueString(); got != "bm7.std.8" {
		t.Errorf("instance_type = %q, want %q", got, "bm7.std.8")
	}
	if got := data.Os.Slug.ValueString(); got != "ubuntu-2404-lts" {
		t.Errorf("os.slug = %q, want %q", got, "ubuntu-2404-lts")
	}
	if data.Os.IsNull() || data.Os.IsUnknown() {
		t.Error("os is null/unknown, want a known object")
	}
}

// name, location, instance_type and os.slug are Required, so on the create and
// update paths state has to keep the config value verbatim. The API normalizes
// these (see equalFoldTrimmed in the recovery path), and writing a normalized
// variant back would fail Terraform's "inconsistent result after apply" check.
func TestServerRespToPlanKeepsConfiguredRequiredAttributes(t *testing.T) {
	ctx := context.Background()

	server := importedServer()
	server.Name = "WK-0054"
	server.Location.Name = "eu: rotterdam "
	server.InstanceType.Name = "BM7.STD.8"
	server.Os.Slug = "UBUNTU-2404-LTS"

	data := FlexmetalServerModel{
		FlexmetalServerModel: resource_flexmetal_server.FlexmetalServerModel{
			Name:         types.StringValue("wk-0054"),
			Location:     types.StringValue("EU: Rotterdam"),
			InstanceType: types.StringValue("bm7.std.8"),
			Os: resource_flexmetal_server.NewOsValueMust(
				resource_flexmetal_server.OsValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"slug":            types.StringValue("ubuntu-2404-lts"),
					"ipxe_script_url": basetypes.NewStringNull(),
					"kernel_params":   basetypes.NewListNull(resource_flexmetal_server.KernelParamsValue{}.Type(ctx)),
					"partitions":      basetypes.NewListNull(resource_flexmetal_server.PartitionsValue{}.Type(ctx)),
				},
			),
		},
	}

	serverRespToPlan(ctx, server, &data)

	if got := data.Name.ValueString(); got != "wk-0054" {
		t.Errorf("name = %q, want the configured %q", got, "wk-0054")
	}
	if got := data.Location.ValueString(); got != "EU: Rotterdam" {
		t.Errorf("location = %q, want the configured %q", got, "EU: Rotterdam")
	}
	if got := data.InstanceType.ValueString(); got != "bm7.std.8" {
		t.Errorf("instance_type = %q, want the configured %q", got, "bm7.std.8")
	}
	if got := data.Os.Slug.ValueString(); got != "ubuntu-2404-lts" {
		t.Errorf("os.slug = %q, want the configured %q", got, "ubuntu-2404-lts")
	}
}

// The GET response has no os sub-attributes, so a refresh of a managed server
// must not drop the ones that came from config.
func TestServerRespToPlanKeepsConfiguredOsSubAttributes(t *testing.T) {
	ctx := context.Background()

	kernelParams := basetypes.NewListValueMust(
		resource_flexmetal_server.KernelParamsValue{}.Type(ctx),
		[]attr.Value{
			resource_flexmetal_server.NewKernelParamsValueMust(
				resource_flexmetal_server.KernelParamsValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"key":   types.StringValue("console"),
					"value": types.StringValue("ttyS0"),
				},
			),
		},
	)
	partitions := basetypes.NewListValueMust(
		resource_flexmetal_server.PartitionsValue{}.Type(ctx),
		[]attr.Value{
			resource_flexmetal_server.NewPartitionsValueMust(
				resource_flexmetal_server.PartitionsValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"target":     types.StringValue("/"),
					"filesystem": types.StringValue("ext4"),
					"size":       types.Int64Value(50000),
				},
			),
		},
	)

	data := FlexmetalServerModel{}
	data.Os = resource_flexmetal_server.NewOsValueMust(
		resource_flexmetal_server.OsValue{}.AttributeTypes(ctx),
		map[string]attr.Value{
			"slug":            types.StringValue("ubuntu-2404-lts"),
			"ipxe_script_url": types.StringValue("https://example.test/ipxe"),
			"kernel_params":   kernelParams,
			"partitions":      partitions,
		},
	)

	serverRespToPlan(ctx, importedServer(), &data)

	if got := data.Os.IpxeScriptUrl.ValueString(); got != "https://example.test/ipxe" {
		t.Errorf("os.ipxe_script_url = %q, want it preserved", got)
	}
	if got := len(data.Os.KernelParams.Elements()); got != 1 {
		t.Errorf("os.kernel_params has %d elements, want 1 preserved", got)
	}
	if got := len(data.Os.Partitions.Elements()); got != 1 {
		t.Errorf("os.partitions has %d elements, want 1 preserved", got)
	}
	if got := data.Os.Slug.ValueString(); got != "ubuntu-2404-lts" {
		t.Errorf("os.slug = %q, want %q", got, "ubuntu-2404-lts")
	}
}

// ssh_key and post_install_script are absent from the GET response; a refresh
// must leave whatever state already holds rather than blanking it.
func TestServerRespToPlanLeavesUnreturnedAttributesAlone(t *testing.T) {
	ctx := context.Background()

	data := FlexmetalServerModel{}
	data.PostInstallScript = types.StringValue("#!/bin/sh\necho hi")
	data.SshKey = basetypes.NewListValueMust(types.StringType, []attr.Value{
		types.StringValue("ssh-ed25519 AAAA"),
	})

	serverRespToPlan(ctx, importedServer(), &data)

	if got := data.PostInstallScript.ValueString(); got != "#!/bin/sh\necho hi" {
		t.Errorf("post_install_script = %q, want it preserved", got)
	}
	if got := len(data.SshKey.Elements()); got != 1 {
		t.Errorf("ssh_key has %d elements, want 1 preserved", got)
	}
}

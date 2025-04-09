package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"
	"terraform-provider-i3dnet/internal/provider/datasource_locations"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ datasource.DataSource              = (*locationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*locationsDataSource)(nil)
)

func NewLocationsDataSource() datasource.DataSource {
	return &locationsDataSource{}
}

type locationsDataSource struct {
	client *one_api.Client
}

func (d *locationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *locationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *locationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	generatedSchema := datasource_locations.LocationsDataSourceSchema(ctx)

	locationsAttr, ok := generatedSchema.Attributes["locations"].(schema.SetNestedAttribute)
	if !ok {
		fmt.Println("Failed to get locations attribute")
		return
	}
	id, ok := locationsAttr.NestedObject.Attributes["id"].(schema.Int64Attribute)
	if !ok {
		fmt.Println("Failed to get id attribute")
		return
	}
	name, ok := locationsAttr.NestedObject.Attributes["name"].(schema.StringAttribute)
	if !ok {
		fmt.Println("Failed to get name attribute")
		return
	}
	shortName, ok := locationsAttr.NestedObject.Attributes["short_name"].(schema.StringAttribute)
	if !ok {
		fmt.Println("Failed to get short_name attribute")
		return
	}

	resp.Schema = schema.Schema{
		Description: "Returns a list of all available Bare Metal locations from i3D.net.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				Description: "List of available locations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.Int64Attribute{Description: id.Description, Computed: true},
						"name":       schema.StringAttribute{Description: name.Description, Computed: true},
						"short_name": schema.StringAttribute{Description: shortName.Description, Computed: true},
					},
				},
				Computed: true,
			},
		},
	}
}

type LocationsData struct {
	Locations []LocationsValue `tfsdk:"locations"`
}

// LocationsValue an element in the locations list
// We define our own struct because the API returns more than these 3 fields
// so we cannot use the auto-generated structs
type LocationsValue struct {
	Id        basetypes.Int64Value  `tfsdk:"id"`
	Name      basetypes.StringValue `tfsdk:"name"`
	ShortName basetypes.StringValue `tfsdk:"short_name"`
}

func (d *locationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LocationsData

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read API call logic
	locations, err := d.client.ListLocations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read i3D.net locations",
			err.Error(),
		)
		return
	}

	// Map API response to Terraform state
	var locationValues []LocationsValue
	for _, loc := range locations {
		locationElem := LocationsValue{
			Id:        basetypes.NewInt64Value(int64(loc.ID)),
			Name:      basetypes.NewStringValue(loc.Name),
			ShortName: basetypes.NewStringValue(loc.ShortName),
		}
		locationValues = append(locationValues, locationElem)
	}

	data.Locations = locationValues

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

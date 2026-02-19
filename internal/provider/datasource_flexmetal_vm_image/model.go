package datasource_flexmetal_vm_image

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type VmImageValue struct {
	ID       types.String `tfsdk:"id"`
	Slug     types.String `tfsdk:"slug"`
	Name     types.String `tfsdk:"name"`
	OsFamily types.String `tfsdk:"os_family"`
	Version  types.String `tfsdk:"version"`
}

type VmImagesData struct {
	Slug     types.String   `tfsdk:"slug"`
	OsFamily types.String   `tfsdk:"os_family"`
	Images   []VmImageValue `tfsdk:"images"`
}

func VmImageDataSourceSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Description: "Lists available FlexMetal VM images, optionally filtered by slug or OS family.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Optional:    true,
				Description: "Filter images by slug.",
			},
			"os_family": schema.StringAttribute{
				Optional:    true,
				Description: "Filter images by OS family (e.g. linux, windows).",
			},
			"images": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of VM images.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Image identifier.",
						},
						"slug": schema.StringAttribute{
							Computed:    true,
							Description: "Image slug.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Image name.",
						},
						"os_family": schema.StringAttribute{
							Computed:    true,
							Description: "OS family.",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "Image version.",
						},
					},
				},
			},
		},
	}
}

package modifiers

import (
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// ApplyRequireReplace applies a RequiresReplace plan modifier to resource attributes that cannot be updated in-place
func ApplyRequireReplace(s schema.Schema, attributes []string) {
	for key, attribute := range s.Attributes {
		if !slices.Contains(attributes, key) {
			continue
		}
		switch v := attribute.(type) {
		case schema.StringAttribute:
			v.PlanModifiers = append(v.PlanModifiers, stringplanmodifier.RequiresReplace())
			s.Attributes[key] = v
		case schema.ListAttribute:
			v.PlanModifiers = append(v.PlanModifiers, listplanmodifier.RequiresReplace())
			s.Attributes[key] = v
		case schema.SingleNestedAttribute:
			v.PlanModifiers = append(v.PlanModifiers, objectplanmodifier.RequiresReplace())
			s.Attributes[key] = v
		}
	}
}

// ApplyUseStateForUnknown adds UseStateForUnknown() plan modifier to resource attributes
//
// UseStateForUnknown() Copies the prior state value, if not null.
// This is useful for reducing (known after apply) plan outputs for computed
// attributes which are known to not change over time.
func ApplyUseStateForUnknown(s schema.Schema, attributes []string) {
	for key, attribute := range s.Attributes {
		if !slices.Contains(attributes, key) {
			continue
		}
		switch v := attribute.(type) {
		case schema.StringAttribute:
			v.PlanModifiers = append(v.PlanModifiers, stringplanmodifier.UseStateForUnknown())
			s.Attributes[key] = v
		case schema.Int64Attribute:
			v.PlanModifiers = append(v.PlanModifiers, int64planmodifier.UseStateForUnknown())
			s.Attributes[key] = v
		case schema.ListNestedAttribute:
			v.PlanModifiers = append(v.PlanModifiers, listplanmodifier.UseStateForUnknown())
			s.Attributes[key] = v
		}
	}
}

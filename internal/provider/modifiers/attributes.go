package modifiers

import (
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// UpdateComputed will modify Computed for attributes to value
func UpdateComputed(s schema.Schema, attributes []string, value bool) {
	for key, attribute := range s.Attributes {
		if !slices.Contains(attributes, key) {
			continue
		}
		switch v := attribute.(type) {
		case schema.ListAttribute:
			v.Computed = value
			s.Attributes[key] = v
		case schema.StringAttribute:
			v.Computed = value
			s.Attributes[key] = v
		case schema.BoolAttribute:
			v.Computed = value
			s.Attributes[key] = v
		}
	}
}

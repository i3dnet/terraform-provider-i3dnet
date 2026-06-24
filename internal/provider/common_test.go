package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEmptyStringAsNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan types.String
		want types.String
	}{
		{
			name: "empty string is normalised to null",
			plan: types.StringValue(""),
			want: types.StringNull(),
		},
		{
			name: "null is left as null",
			plan: types.StringNull(),
			want: types.StringNull(),
		},
		{
			name: "non-empty value is left unchanged",
			plan: types.StringValue("my cloud"),
			want: types.StringValue("my cloud"),
		},
		{
			name: "unknown is left unchanged",
			plan: types.StringUnknown(),
			want: types.StringUnknown(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &planmodifier.StringResponse{PlanValue: tt.plan}
			emptyStringAsNull{}.PlanModifyString(
				context.Background(),
				planmodifier.StringRequest{PlanValue: tt.plan},
				resp,
			)

			if !resp.PlanValue.Equal(tt.want) {
				t.Errorf("PlanValue = %v, want %v", resp.PlanValue, tt.want)
			}
		})
	}
}

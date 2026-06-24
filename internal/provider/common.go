package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emptyStringAsNull is a plan modifier that treats an explicitly empty string
// ("") the same as an unset (null) value by normalising "" to null in the plan.
// This keeps an empty configured value from differing against the null that the
// API/state uses for "no value", avoiding a spurious diff (and, when it runs
// before RequiresReplace, a spurious resource replacement).
type emptyStringAsNull struct{}

func (m emptyStringAsNull) Description(_ context.Context) string {
	return "An empty string is treated as null."
}

func (m emptyStringAsNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m emptyStringAsNull) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsUnknown() {
		return
	}
	if !req.PlanValue.IsNull() && req.PlanValue.ValueString() == "" {
		resp.PlanValue = types.StringNull()
	}
}

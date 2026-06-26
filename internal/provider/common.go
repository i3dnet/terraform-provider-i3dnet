package provider

import (
	"context"
	"fmt"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// clientFromProviderData extracts the *one_api.Client that the provider passes
// to resources and data sources via ProviderData. It is shared by the Configure
// methods of every resource and data source.
//
// providerData is nil when Terraform calls Configure before the provider itself
// has been configured; in that case it returns nil without adding a diagnostic,
// and Configure is called again later with a populated value. A non-nil value of
// an unexpected type is reported as an error.
func clientFromProviderData(providerData any, diags *diag.Diagnostics) *one_api.Client {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(*one_api.Client)
	if !ok {
		diags.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *one_api.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}

	return client
}

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

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-i3dnet/internal/one_api"
)

// The API represents an unset description as an empty string, so state has to
// distinguish "omitted from config" from "explicitly empty".
func TestFlexvmVMRespToStateDescription(t *testing.T) {
	tests := map[string]struct {
		state    types.String
		api      string
		wantNull bool
		want     string
	}{
		"omitted stays null when the API returns empty": {
			state: types.StringNull(), api: "", wantNull: true,
		},
		"omitted takes the value the API reports": {
			state: types.StringNull(), api: "set elsewhere", want: "set elsewhere",
		},
		"configured description follows the API value": {
			state: types.StringValue("from config"), api: "from api", want: "from api",
		},
		"configured description cleared by the API becomes empty, not null": {
			state: types.StringValue("from config"), api: "", want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data := FlexvmVMModel{Description: tt.state}
			vm := &one_api.FlexvmVM{Description: tt.api}

			flexvmVMRespToState(vm, &data)

			if tt.wantNull {
				if !data.Description.IsNull() {
					t.Fatalf("Description = %q, want null", data.Description.ValueString())
				}
				return
			}
			if data.Description.IsNull() {
				t.Fatalf("Description is null, want %q", tt.want)
			}
			if got := data.Description.ValueString(); got != tt.want {
				t.Errorf("Description = %q, want %q", got, tt.want)
			}
		})
	}
}

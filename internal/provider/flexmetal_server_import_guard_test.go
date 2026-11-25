package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"terraform-provider-i3dnet/internal/one_api"
)

func TestErrServerManagedByProduct(t *testing.T) {
	managed := testServer("node", "node-1", "delivered", 1)
	managed.ManagedBy = "flexvm"
	own := testServer("own", "wk-0054", "delivered", 1)

	tests := map[string]struct {
		resp    *one_api.ServerResponse
		err     error
		wantErr bool
	}{
		"customer-managed server is importable": {resp: &one_api.ServerResponse{Server: &own}},
		"product-managed server is rejected":    {resp: &one_api.ServerResponse{Server: &managed}, wantErr: true},
		"API error is left to Read": {
			resp: &one_api.ServerResponse{ErrorResponse: &one_api.ErrorResponse{ErrorCode: 404}},
		},
		"failed lookup is left to Read": {err: errors.New("boom")},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			get := func(context.Context, string) (*one_api.ServerResponse, error) { return tt.resp, tt.err }

			err := errServerManagedByProduct(context.Background(), get, "id")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), `"flexvm"`) {
				t.Errorf("error does not name the product: %v", err)
			}
		})
	}
}

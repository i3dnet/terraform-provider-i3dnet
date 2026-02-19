package one_api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListVmPlans_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "vm/plans")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]VmPlan{
			{Slug: "vm-standard-4", Name: "Standard 4", CPU: 4, MemoryGB: 8, GpuCount: 0},
			{Slug: "vm-gpu-1rtx4000", Name: "GPU 1x RTX4000", CPU: 8, MemoryGB: 32, GpuCount: 1, GpuModel: "RTX4000"},
		})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	plans, err := client.ListVmPlans(context.Background())
	require.NoError(t, err)
	require.Len(t, plans, 2)
	require.Equal(t, "vm-standard-4", plans[0].Slug)
	require.Equal(t, 1, plans[1].GpuCount)
}

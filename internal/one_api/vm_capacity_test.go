package one_api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVmCapacity_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "pool-1/capacity")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VmCapacity{PoolID: "pool-1", Available: 5, Total: 10})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	capacity, err := client.GetVmCapacity(context.Background(), "pool-1")
	require.NoError(t, err)
	require.Equal(t, "pool-1", capacity.PoolID)
	require.Equal(t, 5, capacity.Available)
	require.Equal(t, 10, capacity.Total)
}

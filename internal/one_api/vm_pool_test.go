package one_api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListVmPools_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "vm/pools")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]VmPool{{ID: "pool-1", Name: "my-pool", Status: "active"}})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	pools, err := client.ListVmPools(context.Background())
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, "pool-1", pools[0].ID)
	require.Equal(t, "my-pool", pools[0].Name)
}

func TestListVmPools_apiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{ErrorMessage: "unauthorized"})
	}))
	defer srv.Close()

	client, err := NewClient("bad-key", srv.URL)
	require.NoError(t, err)

	_, err = client.ListVmPools(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

func TestCreateVmPool_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(VmPool{ID: "pool-new", Name: "new-pool", Status: "creating"})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.CreateVmPool(context.Background(), CreateVmPoolRequest{
		Name:         "new-pool",
		LocationID:   "EU-NL-01",
		ContractID:   "contract-123",
		Type:         "on_demand",
		InstanceType: "bm9.hmm.gpu.4rtx4000.64",
		VlanID:       100,
		Subnet: []VmPoolSubnet{
			{CIDR: "10.0.0.0/24", Gateway: "10.0.0.1", RangeStart: "10.0.0.10", RangeEnd: "10.0.0.254"},
		},
	})
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
	require.NotNil(t, resp.Pool)
	require.Equal(t, "pool-new", resp.Pool.ID)
}

func TestGetVmPool_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "pool-1")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VmPool{ID: "pool-1", Name: "my-pool", Status: "active"})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.GetVmPool(context.Background(), "pool-1")
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
	require.Equal(t, "pool-1", resp.Pool.ID)
}

func TestUpdateVmPool_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VmPool{ID: "pool-1", Name: "updated-pool", Status: "active"})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.UpdateVmPool(context.Background(), "pool-1", UpdateVmPoolRequest{Name: "updated-pool"})
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
	require.Equal(t, "updated-pool", resp.Pool.Name)
}

func TestDeleteVmPool_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Contains(t, r.URL.Path, "pool-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.DeleteVmPool(context.Background(), "pool-1")
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
}

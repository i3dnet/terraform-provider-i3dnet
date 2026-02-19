package one_api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateVmInstance_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Contains(t, r.URL.Path, "vm/instances")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(VmInstance{
			ID:     "vm-1",
			Name:   "my-vm",
			PoolID: "pool-1",
			Status: "provisioning",
		})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.CreateVmInstance(context.Background(), CreateVmInstanceRequest{
		Name:   "my-vm",
		PoolID: "pool-1",
		Plan:   "vm-standard-4",
		OS:     VmInstanceOS{ImageID: "ubuntu-2404-lts"},
	})
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
	require.NotNil(t, resp.Instance)
	require.Equal(t, "vm-1", resp.Instance.ID)
}

func TestGetVmInstance_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "vm-1")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VmInstance{
			ID:        "vm-1",
			Name:      "my-vm",
			Status:    "running",
			IPAddress: "10.0.0.10",
		})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.GetVmInstance(context.Background(), "vm-1")
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
	require.Equal(t, "vm-1", resp.Instance.ID)
	require.Equal(t, "running", resp.Instance.Status)
}

func TestDeleteVmInstance_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Contains(t, r.URL.Path, "vm-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	resp, err := client.DeleteVmInstance(context.Background(), "vm-1")
	require.NoError(t, err)
	require.Nil(t, resp.ErrorResponse)
}

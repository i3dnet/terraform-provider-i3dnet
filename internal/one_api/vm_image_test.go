package one_api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListVmImages_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "vm/images")
		// verify filter query params are sent
		require.Equal(t, "linux", r.URL.Query().Get("osFamily"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]VmImage{
			{ID: "img-1", Slug: "ubuntu-2404-lts", Name: "Ubuntu 24.04 LTS", OsFamily: "linux"},
			{ID: "img-2", Slug: "debian-12", Name: "Debian 12", OsFamily: "linux"},
		})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	images, err := client.ListVmImages(context.Background(), VmImageFilters{OsFamily: "linux"})
	require.NoError(t, err)
	require.Len(t, images, 2)
	require.Equal(t, "img-1", images[0].ID)
	require.Equal(t, "linux", images[0].OsFamily)
}

func TestGetVmImage_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "img-1")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VmImage{ID: "img-1", Slug: "ubuntu-2404-lts", Name: "Ubuntu 24.04 LTS", OsFamily: "linux"})
	}))
	defer srv.Close()

	client, err := NewClient("test-key", srv.URL)
	require.NoError(t, err)

	img, err := client.GetVmImage(context.Background(), "img-1")
	require.NoError(t, err)
	require.Equal(t, "img-1", img.ID)
	require.Equal(t, "ubuntu-2404-lts", img.Slug)
}

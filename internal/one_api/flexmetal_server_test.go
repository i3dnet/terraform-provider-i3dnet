package one_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListServersPagesThroughRangedData(t *testing.T) {
	// 150 servers over a page size of 100: two requests, the second short.
	const total = 150

	var gotRanges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/flexMetal/servers" {
			t.Errorf("path = %q, want /v3/flexMetal/servers", r.URL.Path)
		}
		gotRanges = append(gotRanges, r.Header.Get("RANGED-DATA"))

		var start int
		if _, err := fmt.Sscanf(r.Header.Get("RANGED-DATA"), "start=%d", &start); err != nil {
			t.Errorf("unparsable RANGED-DATA header %q", r.Header.Get("RANGED-DATA"))
		}

		var page []Server
		for i := start; i < start+flexmetalServersPageSize && i < total; i++ {
			page = append(page, Server{Uuid: fmt.Sprintf("server-%d", i)})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(page); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if resp.ErrorResponse != nil {
		t.Fatalf("unexpected error response: %+v", resp.ErrorResponse)
	}
	if len(resp.Servers) != total {
		t.Fatalf("servers = %d, want %d", len(resp.Servers), total)
	}
	if resp.Servers[total-1].Uuid != fmt.Sprintf("server-%d", total-1) {
		t.Errorf("last server = %q, want server-%d", resp.Servers[total-1].Uuid, total-1)
	}

	wantRanges := []string{
		fmt.Sprintf("start=0,results=%d", flexmetalServersPageSize),
		fmt.Sprintf("start=%d,results=%d", flexmetalServersPageSize, flexmetalServersPageSize),
	}
	if len(gotRanges) != len(wantRanges) {
		t.Fatalf("requests = %d %v, want %d", len(gotRanges), gotRanges, len(wantRanges))
	}
	for i, want := range wantRanges {
		if gotRanges[i] != want {
			t.Errorf("request %d RANGED-DATA = %q, want %q", i, gotRanges[i], want)
		}
	}
}

func TestListServersReturnsErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte(`{"errorCode":403,"errorMessage":"forbidden"}`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if resp.ErrorResponse == nil {
		t.Fatal("ErrorResponse = nil, want the decoded 403")
	}
	if resp.ErrorResponse.ErrorMessage != "forbidden" {
		t.Errorf("ErrorMessage = %q, want %q", resp.ErrorResponse.ErrorMessage, "forbidden")
	}
}

func TestListServersFailsRatherThanTruncate(t *testing.T) {
	// An API that ignores RANGED-DATA hands out a full page every time, so the
	// page cap is reached with a list that is both incomplete and duplicated.
	pages := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++

		page := make([]Server, 0, flexmetalServersPageSize)
		for i := range flexmetalServersPageSize {
			page = append(page, Server{Uuid: fmt.Sprintf("server-%d", i)})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(page); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.ListServers(context.Background())
	if err == nil {
		t.Fatalf("ListServers returned %d servers and no error, want an error", len(resp.Servers))
	}
	if pages != flexmetalServersMaxPages {
		t.Errorf("pages fetched = %d, want %d", pages, flexmetalServersMaxPages)
	}
}

func TestListServersDiscardsPartialPagesOnAnErrorResponse(t *testing.T) {
	// First page succeeds, second fails: the caller must not be handed the
	// incomplete first page alongside the error.
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page > 1 {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte(`{"errorCode":500,"errorMessage":"boom"}`)); err != nil {
				t.Errorf("write: %v", err)
			}
			return
		}

		servers := make([]Server, 0, flexmetalServersPageSize)
		for i := range flexmetalServersPageSize {
			servers = append(servers, Server{Uuid: fmt.Sprintf("server-%d", i)})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(servers); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if resp.ErrorResponse == nil {
		t.Fatal("ErrorResponse = nil, want the decoded 500")
	}
	if len(resp.Servers) != 0 {
		t.Errorf("Servers = %d, want none alongside an error response", len(resp.Servers))
	}
}

package network_api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
)

// errStopRequest aborts a request from a request editor, after the client's
// own editors ran and before anything is sent.
var errStopRequest = errors.New("stop before sending")

// captureRequest builds the fabric listing request without sending it and
// returns its URL and PRIVATE-TOKEN header.
func captureRequest(t *testing.T, api *ClientWithResponses) (rawURL, token string) {
	t.Helper()

	limit := PageSize
	_, err := api.ListFabricsWithResponse(context.Background(), &ListFabricsParams{Limit: &limit},
		func(_ context.Context, req *http.Request) error {
			rawURL = req.URL.String()
			token = req.Header.Get("PRIVATE-TOKEN")
			return errStopRequest
		})
	if !errors.Is(err, errStopRequest) {
		t.Fatalf("ListFabricsWithResponse: %v, want the editor to stop the request", err)
	}

	return rawURL, token
}

func TestNewAPIBuildsTheNetworkEndpointFromTheBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "empty base_url uses the production default",
			baseURL: "",
			want:    "https://api.i3d.net/network/v1/fabrics?limit=100",
		},
		{
			name:    "custom base_url",
			baseURL: "https://api-testing.example",
			want:    "https://api-testing.example/network/v1/fabrics?limit=100",
		},
		{
			name:    "custom base_url with a trailing slash",
			baseURL: "https://api-testing.example/",
			want:    "https://api-testing.example/network/v1/fabrics?limit=100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api, err := NewAPI("secret-key", tt.baseURL)
			if err != nil {
				t.Fatalf("NewAPI: %v", err)
			}

			got, token := captureRequest(t, api)
			if got != tt.want {
				t.Errorf("request URL = %q, want %q", got, tt.want)
			}
			if token != "secret-key" {
				t.Errorf("PRIVATE-TOKEN = %q, want the api key", token)
			}
		})
	}
}

func TestNewAPIRejectsAnUnparseableBaseURL(t *testing.T) {
	t.Parallel()

	if _, err := NewAPI("key", "://no-scheme"); err == nil {
		t.Fatal("NewAPI: nil error, want the parse failure")
	}
}

// The same base_url serves both APIs: network requests under /network/v1,
// One API requests under /v3, and both carry the same PRIVATE-TOKEN.
func TestNetworkAndOneAPIShareTheBaseURLWithTheirOwnPaths(t *testing.T) {
	t.Parallel()

	var (
		mu     sync.Mutex
		paths  []string
		tokens []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		tokens = append(tokens, r.Header.Get("PRIVATE-TOKEN"))
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/network/") {
			_, _ = w.Write([]byte(`{"items":[],"links":{"self":{"href":"/fabrics?limit=100"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"uuid":"uuid-1"}`))
	}))
	t.Cleanup(srv.Close)

	network, err := NewAPI("shared-key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}
	oneAPI, err := one_api.NewClient("shared-key", srv.URL)
	if err != nil {
		t.Fatalf("one_api.NewClient: %v", err)
	}

	if _, err := network.ListAllFabrics(context.Background()); err != nil {
		t.Fatalf("ListAllFabrics: %v", err)
	}
	// Only the request path matters here, not how one_api decodes the body.
	_, _ = oneAPI.GetServer(context.Background(), "uuid-1")

	mu.Lock()
	defer mu.Unlock()

	if len(paths) != 2 {
		t.Fatalf("server saw %d requests (%v), want 2", len(paths), paths)
	}
	if paths[0] != "/network/v1/fabrics" {
		t.Errorf("network request path = %q, want /network/v1/fabrics", paths[0])
	}
	if !strings.HasPrefix(paths[1], "/v3/") {
		t.Errorf("one_api request path = %q, want it under /v3/", paths[1])
	}
	for i, tok := range tokens {
		if tok != "shared-key" {
			t.Errorf("request %d PRIVATE-TOKEN = %q, want shared-key", i, tok)
		}
	}
}

func TestUnauthorizedPointsAtTheAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "problem document",
			contentType: "application/problem+json",
			body:        `{"title":"Unauthorized","status":401,"code":"E010001","traceId":"0af7651916cd43dd8448eb211c80319c"}`,
		},
		{
			name:        "gateway body that is not a problem document",
			contentType: "text/plain",
			body:        "401 Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(srv.Close)

			api, err := NewAPI("wrong-key", srv.URL)
			if err != nil {
				t.Fatalf("NewAPI: %v", err)
			}

			_, err = api.ListAllFabrics(context.Background())
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("ListAllFabrics: %v, want an *APIError", err)
			}
			if apiErr.StatusCode != http.StatusUnauthorized {
				t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
			}
			if !strings.Contains(apiErr.DiagnosticDetail(), "api_key") {
				t.Errorf("DiagnosticDetail() = %q, want it to point at api_key", apiErr.DiagnosticDetail())
			}
		})
	}
}

// The create re-send decision rests on ErrNoResponse: it must mark every
// request that left the caller without a full response, and nothing else.
func TestRequestWithoutAResponseIsErrNoResponse(t *testing.T) {
	t.Parallel()

	t.Run("closed listener", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.NotFoundHandler())
		srv.Close()

		api, err := NewAPI("key", srv.URL)
		if err != nil {
			t.Fatalf("NewAPI: %v", err)
		}

		_, err = api.ListFabricsWithResponse(context.Background(), &ListFabricsParams{})
		if !errors.Is(err, ErrNoResponse) {
			t.Fatalf("err = %v, want it to wrap ErrNoResponse", err)
		}
	})

	t.Run("connection closed without a response", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("Hijack: %v", err)
				return
			}
			_ = conn.Close()
		}))
		t.Cleanup(srv.Close)

		api, err := NewAPI("key", srv.URL)
		if err != nil {
			t.Fatalf("NewAPI: %v", err)
		}

		_, err = api.ListFabricsWithResponse(context.Background(), &ListFabricsParams{})
		if !errors.Is(err, ErrNoResponse) {
			t.Fatalf("err = %v, want it to wrap ErrNoResponse", err)
		}
	})

	t.Run("context deadline", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		t.Cleanup(srv.Close)

		api, err := NewAPI("key", srv.URL)
		if err != nil {
			t.Fatalf("NewAPI: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err = api.ListFabricsWithResponse(ctx, &ListFabricsParams{})
		if !errors.Is(err, ErrNoResponse) {
			t.Fatalf("err = %v, want it to wrap ErrNoResponse", err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want it to wrap context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("call took %s, want it to end with the context deadline", elapsed)
		}
	})

	t.Run("body cut short after the status line", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"id":`))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("Hijack: %v", err)
				return
			}
			_ = conn.Close()
		}))
		t.Cleanup(srv.Close)

		api, err := NewAPI("key", srv.URL)
		if err != nil {
			t.Fatalf("NewAPI: %v", err)
		}

		_, err = api.ListFabricsWithResponse(context.Background(), &ListFabricsParams{})
		if !errors.Is(err, ErrNoResponse) {
			t.Fatalf("err = %v, want it to wrap ErrNoResponse", err)
		}
	})
}

func TestResponseThatArrivedIsNotErrNoResponse(t *testing.T) {
	t.Parallel()

	t.Run("error status", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)

		api, err := NewAPI("key", srv.URL)
		if err != nil {
			t.Fatalf("NewAPI: %v", err)
		}

		resp, err := api.ListFabricsWithResponse(context.Background(), &ListFabricsParams{})
		if err != nil {
			t.Fatalf("ListFabricsWithResponse: %v, want the 500 as a response", err)
		}
		if resp.StatusCode() != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", resp.StatusCode())
		}
	})

	t.Run("redirect loop", func(t *testing.T) {
		t.Parallel()

		// The default CheckRedirect fails after 10 hops, the one case where
		// the HTTP client returns a response together with an error.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/network/v1/fabrics", http.StatusFound)
		}))
		t.Cleanup(srv.Close)

		api, err := NewAPI("key", srv.URL)
		if err != nil {
			t.Fatalf("NewAPI: %v", err)
		}

		_, err = api.ListFabricsWithResponse(context.Background(), &ListFabricsParams{})
		if err == nil {
			t.Fatal("err = nil, want the redirect failure")
		}
		if errors.Is(err, ErrNoResponse) {
			t.Errorf("err = %v, must not be reported as a request without a response", err)
		}
	})
}

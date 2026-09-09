package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
)

// getServerWithRetry only retries requests that never produced a response, so
// a refresh survives a lost or timed-out GET instead of failing the plan.
func TestGetServerWithRetryRetriesIncompleteRequests(t *testing.T) {
	var calls int

	get := func(context.Context, string) (*one_api.ServerResponse, error) {
		calls++
		if calls < 3 {
			return nil, errors.New("failed to do HTTP request: " + one_api.ErrRequestNotCompleted.Error())
		}
		return &one_api.ServerResponse{Server: &one_api.Server{Uuid: "uuid-1"}}, nil
	}

	resp, err := getServerWithRetry(context.Background(), wrapIncomplete(get), "uuid-1", 3, time.Millisecond)
	if err != nil {
		t.Fatalf("getServerWithRetry: %v, want nil after recovering", err)
	}
	if resp.Server.Uuid != "uuid-1" {
		t.Errorf("got server %q, want uuid-1", resp.Server.Uuid)
	}
	if calls != 3 {
		t.Errorf("called %d times, want 3", calls)
	}
}

// wrapIncomplete re-wraps the sentinel so the fake mirrors what the client
// returns for a request that never completed.
func wrapIncomplete(get func(context.Context, string) (*one_api.ServerResponse, error)) func(context.Context, string) (*one_api.ServerResponse, error) {
	return func(ctx context.Context, id string) (*one_api.ServerResponse, error) {
		resp, err := get(ctx, id)
		if err != nil {
			return nil, errors.Join(one_api.ErrRequestNotCompleted, err)
		}
		return resp, nil
	}
}

// An error that came back with a response is a real answer from the API, so
// repeating the request buys nothing.
func TestGetServerWithRetryDoesNotRetryCompletedRequests(t *testing.T) {
	var calls int

	get := func(context.Context, string) (*one_api.ServerResponse, error) {
		calls++
		return nil, errors.New("error decoding response: unexpected EOF")
	}

	if _, err := getServerWithRetry(context.Background(), get, "uuid-1", 3, time.Millisecond); err == nil {
		t.Fatal("getServerWithRetry: nil error, want the decode error")
	}
	if calls != 1 {
		t.Errorf("called %d times, want 1", calls)
	}
}

func TestGetServerWithRetryReturnsTheLastErrorAfterExhaustingAttempts(t *testing.T) {
	var calls int

	get := func(context.Context, string) (*one_api.ServerResponse, error) {
		calls++
		return nil, errors.Join(one_api.ErrRequestNotCompleted, errors.New("Client.Timeout exceeded"))
	}

	_, err := getServerWithRetry(context.Background(), get, "uuid-1", 3, time.Millisecond)
	if !errors.Is(err, one_api.ErrRequestNotCompleted) {
		t.Fatalf("getServerWithRetry: %v, want an error wrapping ErrRequestNotCompleted", err)
	}
	if calls != 3 {
		t.Errorf("called %d times, want 3", calls)
	}
}

func TestGetServerWithRetryStopsWhenTheContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var calls int
	get := func(context.Context, string) (*one_api.ServerResponse, error) {
		calls++
		cancel()
		return nil, errors.Join(one_api.ErrRequestNotCompleted, errors.New("cancelled"))
	}

	_, err := getServerWithRetry(ctx, get, "uuid-1", 3, time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("getServerWithRetry: %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("called %d times, want 1", calls)
	}
}

// The retry has to trigger on what the real client returns for a dropped
// connection, not only on a hand-built sentinel.
func TestGetServerWithRetryRecoversFromADroppedConnection(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if calls.Add(1) == 1 {
			// Close the connection without writing a response, the way a
			// dropped or timed-out request looks to the client.
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("Hijack: %v", err)
				return
			}
			_ = conn.Close()
			return
		}
		_, _ = w.Write(deliveredServerJSON("delivered"))
	}))
	t.Cleanup(srv.Close)

	c, err := one_api.NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := getServerWithRetry(context.Background(), c.GetServer, "uuid-1", 3, time.Millisecond)
	if err != nil {
		t.Fatalf("getServerWithRetry: %v, want nil after the dropped connection", err)
	}
	if resp.Server.Status != "delivered" {
		t.Errorf("got status %q, want delivered", resp.Server.Status)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("made %d requests, want 2", got)
	}
}

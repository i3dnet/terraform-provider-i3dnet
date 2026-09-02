package one_api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCallAPIWrapsTransportFailureAsErrRequestNotCompleted(t *testing.T) {
	// A listener that is closed straight away: the request can never be
	// delivered, so we cannot know whether the API saw it.
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.callAPI(context.Background(), http.MethodPost, "flexMetal", "servers", nil, nil)
	if !errors.Is(err, ErrRequestNotCompleted) {
		t.Fatalf("err = %v, want it to wrap ErrRequestNotCompleted", err)
	}
}

func TestCallAPIRespectsContextDeadlineRatherThanAFixedTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = c.callAPI(ctx, http.MethodPost, "flexMetal", "servers", nil, nil)
	if !errors.Is(err, ErrRequestNotCompleted) {
		t.Fatalf("err = %v, want it to wrap ErrRequestNotCompleted", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("call took %s, want it to end with the context deadline", elapsed)
	}
}

func TestCallAPIDoesNotFlagAnErrorStatusAsIncomplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.callAPI(context.Background(), http.MethodPost, "flexMetal", "servers", nil, nil)
	if err != nil {
		t.Fatalf("callAPI: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestCallAPIDoesNotFlagAReceivedResponseAsIncomplete(t *testing.T) {
	// An endless redirect makes the default CheckRedirect fail, the one case
	// where client.Do returns both a response and an error. A response did
	// arrive, so the request must not be reported as never completed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v3/flexMetal/servers", http.StatusFound)
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.callAPI(context.Background(), http.MethodGet, "flexMetal", "servers", nil, nil)
	if err == nil {
		t.Fatal("err = nil, want the redirect failure")
	}
	if errors.Is(err, ErrRequestNotCompleted) {
		t.Errorf("err = %v, must not be reported as an incomplete request", err)
	}
}

func TestCallAPICapsRequestsWithoutAContextDeadline(t *testing.T) {
	// Read, Update, Delete and the data sources all call in without a
	// deadline; nothing else bounds a body that stalls after the headers.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.requestTimeout = 100 * time.Millisecond

	start := time.Now()
	_, err = c.callAPI(context.Background(), http.MethodGet, "flexMetal", "servers", nil, nil)
	if !errors.Is(err, ErrRequestNotCompleted) {
		t.Fatalf("err = %v, want it to wrap ErrRequestNotCompleted", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("call took %s, want it capped at the default request timeout", elapsed)
	}
}

func TestCallAPIDoesNotCapALongerContextDeadline(t *testing.T) {
	// The original bug: a fixed whole-request cap overrode the deadline
	// derived from the resource timeouts, dooming slow create requests.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(150 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		if _, err := w.Write([]byte(`[]`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	c, err := NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.requestTimeout = 20 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.callAPI(ctx, http.MethodPost, "flexMetal", "servers", nil, nil)
	if err != nil {
		t.Fatalf("callAPI: %v, want the context deadline to govern instead of the default cap", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

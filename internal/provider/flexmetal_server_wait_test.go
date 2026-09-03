package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
)

// waitTestResource serves GET server responses from handler and returns a
// resource wired to it.
func waitTestResource(t *testing.T, handler http.HandlerFunc) *serverResource {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := one_api.NewClient("token", srv.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return &serverResource{client: c}
}

func deliveredServerJSON(status string) []byte {
	s := one_api.Server{Uuid: "uuid-1", Status: status}
	b, _ := json.Marshal([]one_api.Server{s})
	return b
}

// waitForStatus swallows API errors and keeps polling. With a backoff timer
// that means the reschedule has to happen before the error paths continue,
// or the loop parks forever on a drained timer and only the timeout fires.
func TestWaitForStatusKeepsPollingAfterAnErrorResponse(t *testing.T) {
	var calls atomic.Int32

	r := waitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		// Fail the first two polls, then deliver.
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorCode":500,"errorMessage":"boom"}`))
			return
		}
		_, _ = w.Write(deliveredServerJSON("delivered"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.waitForStatus(ctx, "uuid-1", []string{"delivered"}, 8*time.Second, newFixedPoll(time.Millisecond), nil)
	if err != nil {
		t.Fatalf("waitForStatus: %v, want nil after recovering from errors", err)
	}
	if got := calls.Load(); got < 3 {
		t.Errorf("polled %d times, want at least 3 (two failures then success)", got)
	}
}

func TestWaitForStatusReturnsWhenStatusReached(t *testing.T) {
	r := waitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write(deliveredServerJSON("delivered"))
	})

	err := r.waitForStatus(context.Background(), "uuid-1", []string{"delivered", "failed"}, 5*time.Second, newFixedPoll(time.Millisecond), nil)
	if err != nil {
		t.Fatalf("waitForStatus: %v", err)
	}
}

// Backing off must not stop the deadline from firing.
func TestWaitForStatusTimesOutWhileStatusNeverReached(t *testing.T) {
	r := waitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write(deliveredServerJSON("in_progress"))
	})

	start := time.Now()
	err := r.waitForStatus(context.Background(), "uuid-1", []string{"delivered"}, 300*time.Millisecond, newFixedPoll(time.Millisecond), nil)
	if err == nil {
		t.Fatal("waitForStatus returned nil, want a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("timeout took %v, want it to fire near the 300ms deadline", elapsed)
	}
}

func TestWaitForStatusHonoursCancellation(t *testing.T) {
	r := waitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write(deliveredServerJSON("in_progress"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.waitForStatus(ctx, "uuid-1", []string{"delivered"}, time.Minute, newFixedPoll(time.Millisecond), nil)
	if err == nil {
		t.Fatal("waitForStatus returned nil, want a cancellation error")
	}
}

// The backoff must not delay the very first poll beyond the base interval.
func TestWaitForStatusPollsPromptlyOnTheFirstTick(t *testing.T) {
	r := waitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write(deliveredServerJSON("delivered"))
	})

	start := time.Now()
	if err := r.waitForStatus(context.Background(), "uuid-1", []string{"delivered"}, 5*time.Second, newFixedPoll(100*time.Millisecond), nil); err != nil {
		t.Fatalf("waitForStatus: %v", err)
	}

	// 100ms base plus at most a third of jitter, with slack for the request.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("first poll took %v, want roughly the 100ms base interval", elapsed)
	}
}

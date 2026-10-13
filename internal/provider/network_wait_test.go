package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/network_api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// newCreateServer starts a fake Network API that answers the n-th PUT
// (1-based) with reply(n, w, r), and returns the attempt counter and a client
// for it.
func newCreateServer(t *testing.T, reply func(n int, w http.ResponseWriter, r *http.Request)) (*atomic.Int32, *network_api.ClientWithResponses) {
	t.Helper()

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply(int(attempts.Add(1)), w, r)
	}))
	t.Cleanup(srv.Close)

	api, err := network_api.NewAPI("key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}

	return &attempts, api
}

// Fixed id and body for every attempt of one create, as a resource would mint
// and marshal them once before the first PUT.
var (
	testNetworkID   = openapi_types.UUID{0x01, 0x92, 0x5f, 0x2e, 0x7a, 0x10, 0x70, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	testNetworkBody = []byte(`{"fabric":"rotterdam-a","connectivity":{"type":"private"},"name":"web"}`)
)

// sendCreateNetwork is the send func a resource passes to resendCreate.
func sendCreateNetwork(api *network_api.ClientWithResponses) func(ctx context.Context) (network_api.NetworkResponse, error) {
	return func(ctx context.Context) (network_api.NetworkResponse, error) {
		return network_api.Do[network_api.NetworkResponse](api.CreateNetworkWithBody(ctx, testNetworkID, "application/json", bytes.NewReader(testNetworkBody)))
	}
}

func writeAccepted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"fabric":"rotterdam-a"}`))
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"title":%q,"status":%d,"code":%q,"detail":%q,"traceId":"trace-%d"}`,
		http.StatusText(status), status, code, detail, status)
}

// dropConnection closes the connection without a response, the way a lost or
// reset request looks to the client.
func dropConnection(t *testing.T, w http.ResponseWriter) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		t.Errorf("Hijack: %v", err)
		return
	}
	_ = conn.Close()
}

func TestResendCreateResendsTheSamePutAfterALostResponse(t *testing.T) {
	t.Parallel()

	attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			dropConnection(t, w)
			return
		}
		writeAccepted(w)
	})

	resp, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api))
	if err != nil {
		t.Fatalf("resendCreate: %v, want success after the re-sent PUT", err)
	}
	if resp.Fabric != "rotterdam-a" {
		t.Errorf("fabric = %q, want the decoded 202 body", resp.Fabric)
	}

	if got := attempts.Load(); got != 2 {
		t.Fatalf("made %d attempts, want 2", got)
	}
}

func TestResendCreateResendsWhenTheServerAsksForTheSamePut(t *testing.T) {
	t.Parallel()

	attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			writeProblem(w, http.StatusInternalServerError, "E000001",
				"the network was accepted but provisioning could not be started; retry the same PUT")
			return
		}
		writeAccepted(w)
	})

	resp, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api))
	if err != nil {
		t.Fatalf("resendCreate: %v, want success after the re-sent PUT", err)
	}
	if resp.Fabric != "rotterdam-a" {
		t.Errorf("fabric = %q, want the decoded 202 body", resp.Fabric)
	}

	if got := attempts.Load(); got != 2 {
		t.Fatalf("made %d attempts, want 2", got)
	}
}

// Every status whose outcome is unknown is re-sent until a 202.
func TestResendCreateResendsEveryUnknownOutcomeStatus(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Parallel()

			attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
				if n <= 2 {
					writeProblem(w, status, "E000001", "try again")
					return
				}
				writeAccepted(w)
			})

			if _, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api)); err != nil {
				t.Fatalf("resendCreate: %v, want success after re-sending %d", err, status)
			}

			if got := attempts.Load(); got != 3 {
				t.Fatalf("made %d attempts, want 3", got)
			}
		})
	}
}

// A gateway error body that is not a problem document is still classified by
// its status.
func TestResendCreateResendsANonProblemGatewayError(t *testing.T) {
	t.Parallel()

	attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
			return
		}
		writeAccepted(w)
	})

	if _, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api)); err != nil {
		t.Fatalf("resendCreate: %v, want success after the 502", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("made %d attempts, want 2", got)
	}
}

// A 500 is an unknown outcome even when its JSON body does not parse: the
// status decides, not a decode error.
func TestResendCreateResendsA500WithABrokenJSONBody(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"title":`, ""} {
		t.Run(fmt.Sprintf("body %q", body), func(t *testing.T) {
			t.Parallel()

			attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
				if n == 1 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(body))
					return
				}
				writeAccepted(w)
			})

			resp, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api))
			if err != nil {
				t.Fatalf("resendCreate: %v, want success after the re-sent PUT", err)
			}
			if resp.Fabric != "rotterdam-a" {
				t.Errorf("fabric = %q, want the decoded 202 body", resp.Fabric)
			}
			if got := attempts.Load(); got != 2 {
				t.Fatalf("made %d attempts, want 2", got)
			}
		})
	}
}

// A 202 whose body does not decode left the caller without the result: the
// outcome is unknown and the identical PUT is re-sent.
func TestResendCreateResendsA202ThatDoesNotDecode(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"fabric":`, ""} {
		t.Run(fmt.Sprintf("body %q", body), func(t *testing.T) {
			t.Parallel()

			var (
				mu     sync.Mutex
				bodies [][]byte
			)
			attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
				got, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				mu.Lock()
				bodies = append(bodies, got)
				mu.Unlock()

				if n == 1 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(body))
					return
				}
				writeAccepted(w)
			})

			resp, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api))
			if err != nil {
				t.Fatalf("resendCreate: %v, want success after the re-sent PUT", err)
			}
			if resp.Fabric != "rotterdam-a" {
				t.Errorf("fabric = %q, want the decoded second 202", resp.Fabric)
			}
			if got := attempts.Load(); got != 2 {
				t.Fatalf("made %d attempts, want 2", got)
			}

			mu.Lock()
			defer mu.Unlock()
			for i, b := range bodies {
				if !bytes.Equal(b, testNetworkBody) {
					t.Errorf("attempt %d body = %s, want the identical create body", i+1, b)
				}
			}
		})
	}
}

func TestResendCreateReportsAnUnknownOutcomeAtTheDeadline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		reply func(t *testing.T, w http.ResponseWriter)
	}{
		{
			name: "503 on every attempt",
			reply: func(_ *testing.T, w http.ResponseWriter) {
				writeProblem(w, http.StatusServiceUnavailable, "E000002", "unavailable")
			},
		},
		{
			name:  "no response on every attempt",
			reply: dropConnection,
		},
		{
			name: "500 with a broken JSON body on every attempt",
			reply: func(_ *testing.T, w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"title":`))
			},
		},
		{
			name: "202 that does not decode on every attempt",
			reply: func(_ *testing.T, w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"fabric":`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			attempts, api := newCreateServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
				tt.reply(t, w)
			})

			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()

			start := time.Now()
			_, err := resendCreate(ctx, newFixedPoll(5*time.Millisecond), sendCreateNetwork(api))

			if !errors.Is(err, errCreateOutcomeUnknown) {
				t.Fatalf("resendCreate: %v, want errCreateOutcomeUnknown: the API may have accepted an attempt", err)
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("err = %v, want it to wrap context.DeadlineExceeded", err)
			}
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Errorf("gave up after %v, want near the 300ms deadline", elapsed)
			}

			if got := attempts.Load(); got < 2 {
				t.Errorf("made %d attempts, want the PUT re-sent until the deadline", got)
			}
		})
	}
}

// A refusal after an attempt without a known outcome may follow an accepted
// first attempt, so the outcome stays unknown.
func TestResendCreateReportsARefusalAfterAnUnknownOutcomeAsUnknown(t *testing.T) {
	t.Parallel()

	attempts, api := newCreateServer(t, func(n int, w http.ResponseWriter, r *http.Request) {
		if n == 1 {
			dropConnection(t, w)
			return
		}
		writeProblem(w, http.StatusConflict, "E020007", "the network changed since the prefix was requested")
	})

	_, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api))

	if !errors.Is(err, errCreateOutcomeUnknown) {
		t.Fatalf("resendCreate: %v, want errCreateOutcomeUnknown after the lost first attempt", err)
	}

	var apiErr *network_api.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "E020007" {
		t.Errorf("err = %v, want it to wrap the 409 E020007 *APIError", err)
	}

	if got := attempts.Load(); got != 2 {
		t.Fatalf("made %d attempts, want 2 (the 409 is not re-sent)", got)
	}
}

// Refusals and validation failures are a known outcome: never re-sent, and
// nothing was created.
func TestResendCreateNeverResendsARefusal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		code   string
	}{
		{http.StatusConflict, "E020003"},
		{http.StatusConflict, "E020007"},
		{http.StatusConflict, "E020016"},
		{http.StatusConflict, "E020017"},
		{http.StatusConflict, "E020019"},
		{http.StatusUnprocessableEntity, "E020001"},
		{http.StatusBadRequest, "E000400"},
		{http.StatusUnauthorized, "E010001"},
		{http.StatusForbidden, "E010003"},
		{http.StatusNotFound, "E000404"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d %s", tt.status, tt.code), func(t *testing.T) {
			t.Parallel()

			attempts, api := newCreateServer(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
				writeProblem(w, tt.status, tt.code, "refused")
			})

			_, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), sendCreateNetwork(api))

			if errors.Is(err, errCreateOutcomeUnknown) {
				t.Errorf("resendCreate: %v, want no errCreateOutcomeUnknown: a first-attempt refusal created nothing", err)
			}

			var apiErr *network_api.APIError
			if !errors.As(err, &apiErr) || apiErr.Code != tt.code || apiErr.StatusCode != tt.status {
				t.Errorf("err = %v, want it to wrap the %d %s *APIError", err, tt.status, tt.code)
			}

			if got := attempts.Load(); got != 1 {
				t.Errorf("made %d attempts, want exactly 1", got)
			}
		})
	}
}

// After an unknown outcome, the returned error matches both the sentinel and
// the cause, and its message says the resource may have been created.
func TestResendCreateUnknownOutcomeErrorKeepsTheCauseAndSaysSo(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")

	var calls int
	_, err := resendCreate(context.Background(), newFixedPoll(time.Millisecond), func(context.Context) (struct{}, error) {
		calls++
		if calls == 1 {
			return struct{}{}, network_api.ErrNoResponse
		}
		return struct{}{}, cause
	})

	if !errors.Is(err, errCreateOutcomeUnknown) {
		t.Errorf("err = %v, want errCreateOutcomeUnknown", err)
	}
	if !errors.Is(err, cause) {
		t.Errorf("err = %v, want it to wrap the cause", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "may have been created") || !strings.Contains(msg, cause.Error()) {
		t.Errorf("Error() = %q, want it to say the resource may have been created and name the cause", msg)
	}
	if calls != 2 {
		t.Errorf("made %d calls, want 2", calls)
	}
}

func TestWaitUntilStopsOnceTheConditionHolds(t *testing.T) {
	t.Parallel()

	var polls int
	err := waitUntil(context.Background(), newFixedPoll(time.Millisecond), func(context.Context) (bool, error) {
		polls++
		return polls == 3, nil
	})
	if err != nil {
		t.Fatalf("waitUntil: %v", err)
	}
	if polls != 3 {
		t.Errorf("polled %d times, want 3", polls)
	}
}

func TestWaitUntilReturnsThePollError(t *testing.T) {
	t.Parallel()

	stop := errors.New("terminal state")

	var polls int
	err := waitUntil(context.Background(), newFixedPoll(time.Millisecond), func(context.Context) (bool, error) {
		polls++
		if polls == 2 {
			return false, stop
		}
		return false, nil
	})
	if !errors.Is(err, stop) {
		t.Fatalf("waitUntil: %v, want the poll error", err)
	}
	if polls != 2 {
		t.Errorf("polled %d times, want 2", polls)
	}
}

func TestWaitUntilHonoursCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var polls atomic.Int32
	start := time.Now()
	err := waitUntil(ctx, newFixedPoll(time.Millisecond), func(context.Context) (bool, error) {
		polls.Add(1)
		return false, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitUntil: %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("stopped after %v, want near the 100ms deadline", elapsed)
	}
	if polls.Load() == 0 {
		t.Error("never polled before the deadline")
	}
}

// A cancelled wait must not sit out a long poll interval.
func TestWaitUntilDoesNotWaitOutTheIntervalAfterCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := waitUntil(ctx, newFixedPoll(time.Hour), func(context.Context) (bool, error) {
		t.Error("polled before the hour-long interval elapsed")
		return true, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitUntil: %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("stopped after %v, want right after the cancel", elapsed)
	}
}

// The schedule is injected: the first poll waits one schedule delay, not a
// production interval.
func TestWaitUntilUsesTheInjectedSchedule(t *testing.T) {
	t.Parallel()

	start := time.Now()
	var firstPoll time.Duration
	err := waitUntil(context.Background(), newFixedPoll(50*time.Millisecond), func(context.Context) (bool, error) {
		firstPoll = time.Since(start)
		return true, nil
	})
	if err != nil {
		t.Fatalf("waitUntil: %v", err)
	}
	if firstPoll < 50*time.Millisecond {
		t.Errorf("first poll after %v, want it to wait the 50ms schedule delay", firstPoll)
	}
	if firstPoll > 2*time.Second {
		t.Errorf("first poll after %v, want roughly the 50ms schedule delay", firstPoll)
	}
}

func TestSleepCtx(t *testing.T) {
	t.Parallel()

	t.Run("waits the duration", func(t *testing.T) {
		t.Parallel()

		start := time.Now()
		if err := sleepCtx(context.Background(), 20*time.Millisecond); err != nil {
			t.Fatalf("sleepCtx: %v", err)
		}
		if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
			t.Errorf("returned after %v, want at least 20ms", elapsed)
		}
	})

	t.Run("returns early when the context ends", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := sleepCtx(ctx, time.Minute)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("sleepCtx: %v, want context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("returned after %v, want right after the deadline", elapsed)
		}
	})

	t.Run("already cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := sleepCtx(ctx, time.Minute); !errors.Is(err, context.Canceled) {
			t.Fatalf("sleepCtx: %v, want context.Canceled", err)
		}
	})
}

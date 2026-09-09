package provider

import (
	"context"
	"errors"
	"time"

	"terraform-provider-i3dnet/internal/one_api"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// readRetryAttempts and readRetryDelay bound the retry of a refresh. A
	// refresh runs for every resource in the plan, so the budget stays small:
	// it is there to ride out a single lost request, not an outage.
	readRetryAttempts = 3
	readRetryDelay    = 3 * time.Second
)

// getServerFunc is the GET server call, as a function so the retry can be
// tested without a client.
type getServerFunc func(ctx context.Context, id string) (*one_api.ServerResponse, error)

// getServerWithRetry reads a server, retrying requests that never produced a
// response. A GET is idempotent, and a single dropped or timed-out one would
// otherwise fail the whole plan while the server is perfectly fine. Errors that
// did come back from the API are answers, and are returned as they are.
func getServerWithRetry(ctx context.Context, get getServerFunc, id string,
	attempts int, delay time.Duration) (*one_api.ServerResponse, error) {
	for attempt := 1; ; attempt++ {
		resp, err := get(ctx, id)
		if err == nil || !errors.Is(err, one_api.ErrRequestNotCompleted) || attempt >= attempts {
			return resp, err
		}

		tflog.Warn(ctx, "get server did not complete, retrying", map[string]interface{}{
			"id": id, "attempt": attempt, "err": err,
		})

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			// Report why we stopped, not the request that provoked the wait.
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

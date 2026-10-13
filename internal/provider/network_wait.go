package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"terraform-provider-i3dnet/internal/network_api"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// unknownOutcomeStatuses are the statuses after which a create PUT may or may
// not have been accepted. The Network API itself answers 500 when it recorded
// the resource but could not start provisioning, asking for the same PUT.
var unknownOutcomeStatuses = []int{
	http.StatusRequestTimeout,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

// errCreateOutcomeUnknown marks a resendCreate failure after at least one
// attempt ended without a known outcome: the resource may exist, so the caller
// must keep its id in state.
var errCreateOutcomeUnknown = errors.New("the outcome of the create is unknown, the resource may have been created")

// resendCreate sends a create and re-sends it unchanged while its outcome is
// unknown: no response (network_api.ErrNoResponse) or one of
// unknownOutcomeStatuses. It waits schedule.next() between attempts and stops
// when ctx ends. The create is idempotent by its client-minted id, so send
// must issue the identical request on every call.
//
// send returns the response, or the error from network_api.Do: an
// *network_api.APIError for a non-2xx, or an error wrapping
// network_api.ErrNoResponse for a request without a response or a 2xx whose
// body does not decode. Any other non-2xx stops at once. On success it returns
// the 2xx response. Otherwise it returns the last error, wrapped in
// errCreateOutcomeUnknown when any attempt ended without a known outcome.
func resendCreate[R any](ctx context.Context, schedule *pollSchedule, send func(ctx context.Context) (R, error)) (R, error) {
	unknown := false

	for attempt := 1; ; attempt++ {
		resp, err := send(ctx)
		if err == nil {
			return resp, nil
		}

		if !isUnknownOutcome(err) {
			var zero R
			if unknown {
				return zero, fmt.Errorf("%w: %w", errCreateOutcomeUnknown, err)
			}
			return zero, err
		}
		unknown = true

		delay := schedule.next()
		tflog.Warn(ctx, "create outcome unknown, re-sending the same request", map[string]interface{}{
			"attempt": attempt,
			"delay":   delay.String(),
			"error":   err.Error(),
		})

		if waitErr := sleepCtx(ctx, delay); waitErr != nil {
			var zero R
			return zero, fmt.Errorf("%w: %w", errCreateOutcomeUnknown, errors.Join(err, waitErr))
		}
	}
}

func isUnknownOutcome(err error) bool {
	if errors.Is(err, network_api.ErrNoResponse) {
		return true
	}

	var apiErr *network_api.APIError
	return errors.As(err, &apiErr) && slices.Contains(unknownOutcomeStatuses, apiErr.StatusCode)
}

// waitUntil calls poll after each schedule.next() delay until poll reports
// done or returns an error, or ctx ends. poll decides which errors are
// transient: it returns (false, nil) to keep waiting. The caller bounds the
// wait with the ctx deadline.
func waitUntil(ctx context.Context, schedule *pollSchedule, poll func(ctx context.Context) (done bool, err error)) error {
	for {
		if err := sleepCtx(ctx, schedule.next()); err != nil {
			return fmt.Errorf("stopped waiting: %w", err)
		}

		done, err := poll(ctx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

// sleepCtx waits for d or until ctx ends, returning ctx.Err() in that case.
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

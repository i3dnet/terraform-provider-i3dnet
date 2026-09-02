package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
)

// statusCreated is the initial status of a freshly registered FlexMetal server,
// before delivery starts.
const statusCreated = "created"

// createdServerCriteria describes the server a create request tried to register.
type createdServerCriteria struct {
	Name         string
	Location     string
	InstanceType string
	// NotBefore is the oldest createdAt still considered ours, so earlier
	// servers sharing the same name are not adopted. Callers set it to the
	// start of the create request widened by createRecoveryClockSkew, and it
	// therefore reaches slightly before that request began.
	NotBefore int64
}

// selectCreatedServer looks for the server a create request registered among all
// servers on the account. It returns the single server that can safely be
// adopted, plus every candidate that matched the criteria.
//
// Server names are not unique per account, so multiple candidates are only
// disambiguated when exactly one of them still sits in the "created" status.
// Otherwise the match is nil and the caller must not guess.
func selectCreatedServer(servers []one_api.Server, c createdServerCriteria) (*one_api.Server, []one_api.Server) {
	var candidates []one_api.Server
	for _, s := range servers {
		if !matchesCreatedServer(s, c) {
			continue
		}
		candidates = append(candidates, s)
	}

	if len(candidates) == 1 {
		return &candidates[0], candidates
	}

	var created []one_api.Server
	for _, s := range candidates {
		if s.Status == statusCreated {
			created = append(created, s)
		}
	}
	if len(created) == 1 {
		return &created[0], candidates
	}

	return nil, candidates
}

func matchesCreatedServer(s one_api.Server, c createdServerCriteria) bool {
	if !equalFoldTrimmed(s.Name, c.Name) ||
		!equalFoldTrimmed(s.Location.Name, c.Location) ||
		!equalFoldTrimmed(s.InstanceType.Name, c.InstanceType) {
		return false
	}
	if s.CreatedAt < c.NotBefore {
		return false
	}
	// A server on its way out is never the one we just asked for.
	if s.ReleasedAt != 0 || s.Status == "releasing" || s.Status == "released" {
		return false
	}

	return true
}

func equalFoldTrimmed(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

const (
	// createRecoveryTimeout bounds how long we look for a server whose create
	// request never returned a response.
	createRecoveryTimeout = 2 * time.Minute
	// createRecoveryInterval is the poll interval while looking for it.
	createRecoveryInterval = 15 * time.Second
	// createRecoveryClockSkew widens the createdAt window to absorb the
	// difference between our clock and the API's.
	createRecoveryClockSkew = 60
)

var (
	errCreatedServerNotFound     = errors.New("no server matching the create request was registered")
	errAmbiguousCreatedServers   = errors.New("multiple servers match the create request")
	errCreatedServerLookupFailed = errors.New("could not check whether the create request registered a server")
)

// serverLister lists every server of the account.
type serverLister func(ctx context.Context) (*one_api.ServerListResponse, error)

// recoverCreatedServer polls the server list until the server of an unfinished
// create request appears, and returns it so the caller can adopt its UUID.
//
// It gives up with errCreatedServerNotFound once timeout elapses, and with
// errAmbiguousCreatedServers as soon as candidates cannot be told apart -
// waiting longer cannot resolve that.
func recoverCreatedServer(ctx context.Context, list serverLister, c createdServerCriteria,
	timeout, interval time.Duration) (*one_api.Server, []one_api.Server, error) {
	deadline := time.After(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		resp, err := list(ctx)
		switch {
		case err != nil:
			lastErr = err
		case resp.ErrorResponse != nil:
			lastErr = fmt.Errorf("list servers returned %d: %s",
				resp.ErrorResponse.ErrorCode, resp.ErrorResponse.ErrorMessage)
		default:
			lastErr = nil

			match, candidates := selectCreatedServer(resp.Servers, c)
			if match != nil {
				return match, candidates, nil
			}
			if len(candidates) > 1 {
				return nil, candidates, errAmbiguousCreatedServers
			}
		}

		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-deadline:
			// Only claim the server was never registered when we actually
			// managed to look; a failed lookup proves nothing either way.
			if lastErr != nil {
				return nil, nil, fmt.Errorf("%w: %w", errCreatedServerLookupFailed, lastErr)
			}
			return nil, nil, errCreatedServerNotFound
		case <-ticker.C:
		}
	}
}

// createRecoveryWarning describes an adopted server for the practitioner.
func createRecoveryWarning(s *one_api.Server, candidates []one_api.Server) (summary, detail string) {
	detail = fmt.Sprintf("The create request did not return a response, but server %q (%s) was registered "+
		"and has been adopted into state. Delivery is being awaited as usual.", s.Name, s.Uuid)
	if len(candidates) > 1 {
		detail += fmt.Sprintf(" It was picked out of %d servers matching name, location and instance type "+
			"because it is the only one still in status %q.", len(candidates), statusCreated)
	}

	return "Recovered a server from an unfinished create request", detail
}

// createRecoveryError explains why an unfinished create request could not be
// reconciled, and what the practitioner should do about it.
func createRecoveryError(postErr, recoveryErr error, c createdServerCriteria, candidates []one_api.Server) string {
	detail := fmt.Sprintf("The create request failed without a response: %v\n\n"+
		"Server name: %s, location: %s, instance type: %s\n\n", postErr, c.Name, c.Location, c.InstanceType)

	switch {
	case errors.Is(recoveryErr, errAmbiguousCreatedServers):
		detail += fmt.Sprintf("Terraform then found %d servers matching that name, location and instance type "+
			"and cannot tell which one belongs to this resource:\n", len(candidates))
		for _, s := range candidates {
			detail += fmt.Sprintf("  - %s (status %s, created at %d)\n", s.Uuid, s.Status, s.CreatedAt)
		}
		detail += "\nNo server was adopted. Identify the right one and run " +
			"'terraform import <resource address> <uuid>', or release the servers you do not want."
	case errors.Is(recoveryErr, errCreatedServerNotFound):
		detail += fmt.Sprintf("Terraform looked for a matching server for %s and found none, "+
			"so the request most likely never reached the API. Retrying is safe, but check the portal first "+
			"in case the server is registered late.", createRecoveryTimeout)
	default:
		detail += fmt.Sprintf("%v\n\nCheck the portal before retrying: if the server exists, adopt it with "+
			"'terraform import <resource address> <uuid>'.", recoveryErr)
	}

	return detail
}

// createRecoveryPossible reports whether a failed create request may still have
// registered a server worth looking for. Only a request that never produced a
// response qualifies, and only while the create timeout has budget left.
func createRecoveryPossible(postErr, ctxErr error) bool {
	return errors.Is(postErr, one_api.ErrRequestNotCompleted) && ctxErr == nil
}

// createRecoveryUnavailableHint tells the practitioner what to do when a
// request that produced no response cannot be reconciled, because the context
// it would have to look with is already done.
func createRecoveryUnavailableHint(ctxErr error) string {
	reason := "The apply was cancelled"
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		reason = "The create timeout was reached"
	}

	return reason + " before Terraform could check whether the request registered a server, " +
		"so a server may have been registered without Terraform knowing about it. Check the portal: " +
		"if the server is there, adopt it with 'terraform import <resource address> <uuid>' or release it. " +
		"Raising the create timeout gives delivery more room."
}

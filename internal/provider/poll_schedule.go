package provider

import (
	"math/rand/v2"
	"time"
)

const (
	// maxPollDelay caps a single wait between polls, jitter included.
	maxPollDelay = 45 * time.Second

	// minPollInterval is the fastest the delivery schedule ever polls, and the
	// flat interval the provider used before the schedule existed.
	minPollInterval = 15 * time.Second

	// pollNarrowFactor shrinks the interval after each poll. Chosen so the
	// floor is reached well inside five minutes even at full jitter.
	pollNarrowFactor = 0.9

	// pollJitterFraction is how far past the interval a delay may be stretched.
	// Jitter only ever slows a poll down, so an interval is also its floor: a
	// 15s interval yields 15-20s.
	pollJitterFraction = 1.0 / 3.0
)

// maxPollInterval is the widest pre-jitter interval, chosen so that a fully
// jittered delay lands exactly on maxPollDelay.
const maxPollInterval = time.Duration(float64(maxPollDelay) / (1 + pollJitterFraction))

// pollSchedule produces the successive waits between status polls. Each delay
// is jittered upward so that waiters started together by Terraform's
// parallelism drift apart instead of hitting the API in lockstep.
//
// Not safe for concurrent use; each wait loop owns one. The jitter source is
// the math/rand/v2 global, which is goroutine-safe.
type pollSchedule struct {
	interval time.Duration
	floor    time.Duration
	factor   float64
	jitter   float64
}

// newDeliveryPoll narrows from maxPollInterval down to minPollInterval,
// reaching the floor within five minutes. A server is never delivered
// immediately, so the early polls would be near-certain misses; once the
// provision is plausibly finishing, polling is at its fastest.
func newDeliveryPoll() *pollSchedule {
	return &pollSchedule{
		interval: maxPollInterval,
		floor:    minPollInterval,
		factor:   pollNarrowFactor,
		jitter:   pollJitterFraction,
	}
}

// newFixedPoll polls at a flat interval. For a wait that normally finishes in
// seconds, ramping costs more in detection latency than it saves in requests.
func newFixedPoll(interval time.Duration) *pollSchedule {
	return &pollSchedule{interval: interval, floor: interval, factor: 1}
}

// next returns the delay to wait before the next poll and advances the
// interval.
func (p *pollSchedule) next() time.Duration {
	delay := p.interval + time.Duration(rand.Float64()*p.jitter*float64(p.interval))
	p.interval = max(time.Duration(float64(p.interval)*p.factor), p.floor)
	return delay
}

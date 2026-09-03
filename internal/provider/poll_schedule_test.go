package provider

import (
	"testing"
	"time"
)

const samples = 2000

// Delivery is never immediate, so the schedule opens at its widest interval
// rather than polling into a dead zone.
func TestDeliveryPollStartsAtTheWidestInterval(t *testing.T) {
	for i := 0; i < samples; i++ {
		got := newDeliveryPoll().next()
		if got < maxPollInterval {
			t.Fatalf("first delay %v is narrower than the %v starting interval", got, maxPollInterval)
		}
		if got > maxPollDelay {
			t.Fatalf("first delay %v exceeds the %v cap", got, maxPollDelay)
		}
	}
}

func TestDeliveryPollNarrowsTowardsTheFloor(t *testing.T) {
	p := newDeliveryPoll()

	prev := p.interval
	for i := 0; i < 50; i++ {
		p.next()
		if p.interval > prev {
			t.Fatalf("interval widened at step %d: %v then %v", i, prev, p.interval)
		}
		prev = p.interval
	}

	if p.interval != minPollInterval {
		t.Errorf("settled at %v, want the %v floor", p.interval, minPollInterval)
	}
}

// Never poll faster than the flat 15s the provider used before this schedule.
func TestDeliveryPollNeverPollsFasterThanTheFloor(t *testing.T) {
	p := newDeliveryPoll()
	for i := 0; i < samples; i++ {
		if got := p.next(); got < minPollInterval {
			t.Fatalf("delay %d = %v is faster than the %v floor", i, got, minPollInterval)
		}
	}
}

// The requirement: reach the fastest polling rate within five minutes, jitter
// included.
func TestDeliveryPollReachesTheFloorWithinFiveMinutes(t *testing.T) {
	const budget = 5 * time.Minute

	worst := time.Duration(0)
	for trial := 0; trial < 200; trial++ {
		p := newDeliveryPoll()

		var elapsed time.Duration
		for p.interval > minPollInterval {
			elapsed += p.next()
		}
		if elapsed > worst {
			worst = elapsed
		}
		if elapsed > budget {
			t.Fatalf("took %v to reach the %v floor, want within %v", elapsed, minPollInterval, budget)
		}
	}
	t.Logf("worst observed time to reach the floor: %v", worst)
}

// The floor is where a long provision spends most of its time, so the jitter
// has to survive there or every waiter re-synchronizes onto the same tick.
func TestDeliveryPollStillJittersAtTheFloor(t *testing.T) {
	p := newDeliveryPoll()
	for p.interval > minPollInterval {
		p.next()
	}

	seen := map[time.Duration]bool{}
	var maxSeen time.Duration
	for i := 0; i < samples; i++ {
		d := p.next()
		seen[d] = true
		if d > maxSeen {
			maxSeen = d
		}
	}

	if len(seen) < 100 {
		t.Errorf("only %d distinct delays at the floor, want a spread", len(seen))
	}
	if maxSeen > minPollInterval+minPollInterval/3 {
		t.Errorf("max delay at floor %v exceeds the jitter band", maxSeen)
	}
}

func TestDeliveryPollNeverExceedsTheCap(t *testing.T) {
	p := newDeliveryPoll()
	for i := 0; i < samples; i++ {
		if got := p.next(); got > maxPollDelay {
			t.Fatalf("delay %d = %v exceeds the %v cap", i, got, maxPollDelay)
		}
	}
}

func TestDeliveryPollJittersConcurrentWaitersApart(t *testing.T) {
	first := newDeliveryPoll().next()
	for i := 0; i < samples; i++ {
		if newDeliveryPoll().next() != first {
			return
		}
	}
	t.Error("every waiter got the same first delay, so nothing is de-synchronized")
}

// Releasing a server is quick, so that wait polls at a flat interval: ramping
// would cost more in destroy latency than it saves in requests.
func TestFixedPollNeverMovesOrJitters(t *testing.T) {
	p := newFixedPoll(2 * time.Second)

	for i := 0; i < 100; i++ {
		if got := p.next(); got != 2*time.Second {
			t.Fatalf("delay %d = %v, want a flat 2s", i, got)
		}
	}
}

package domain

import "time"

// Tunables of the refresh policy. They are judgement calls, not facts about the
// GitHub API: a wrong value costs latency, never correctness.
const (
	// MinServerWait and MaxServerWait bound every delay that came from the
	// server (Retry-After, reset time), so a hostile or broken value can
	// neither spin the loop nor park it for days.
	MinServerWait = time.Second
	MaxServerWait = 65 * time.Minute

	// MinFailureBase is the first retry delay after a failure when the
	// configured interval is shorter than this.
	MinFailureBase = 30 * time.Second
	// MaxFailureWait caps the exponential backoff (a longer configured
	// interval raises the cap to that interval).
	MaxFailureWait = 5 * time.Minute

	// RateLimitFloor is the least wait after a rate-limited failure that
	// carries no timing information.
	RateLimitFloor = time.Minute

	// ReservePerCost is how many refreshes' worth of budget prv keeps unused:
	// it pauses when Remaining <= ReservePerCost * max(Cost, 1).
	ReservePerCost = 10
)

// Outcome describes the last fetch to NextDelay.
type Outcome struct {
	Failed      bool
	RateLimited bool          // the failure was a rate limit
	RetryAfter  time.Duration // from a Retry-After header; zero when absent
	ResetAt     time.Time     // reset time from the error; zero when absent
}

// Delay is the decision of NextDelay.
type Delay struct {
	Wait        time.Duration // time until the next refresh
	PausedUntil time.Time     // when set, no request may go out before this; zero means no pause
}

// NextDelay decides how long to wait before the next refresh. It is pure: the
// current time is the now parameter. failures counts consecutive failures,
// including the one in last. rl is the last known rate limit.
//
// Order: a Retry-After wins; then a known reset time; then a bounded
// exponential. On success the reserve pause and pacing apply.
func NextDelay(now time.Time, interval time.Duration, failures int, rl RateLimit, last Outcome) Delay {
	interval = max(interval, time.Second)

	if last.Failed {
		if last.RetryAfter > 0 {
			return paused(now, clamp(last.RetryAfter))
		}
		if last.RateLimited {
			reset := last.ResetAt
			if !reset.After(now) {
				reset = rl.ResetAt
			}
			if reset.After(now) {
				return paused(now, clamp(reset.Sub(now)+time.Second))
			}
			return paused(now, max(RateLimitFloor, failureWait(interval, failures)))
		}
		return Delay{Wait: failureWait(interval, failures)}
	}

	d := Delay{Wait: interval}
	if rl.Limit <= 0 || !rl.ResetAt.After(now) {
		return d
	}
	cost := max(rl.Cost, 1)
	untilReset := rl.ResetAt.Sub(now)
	if rl.Remaining <= ReservePerCost*cost {
		return paused(now, clamp(untilReset+time.Second))
	}
	// Pacing: division only, so no interval can overflow.
	left := rl.Remaining / cost
	if int64(untilReset/interval) > int64(left) {
		d.Wait = max(interval, untilReset/time.Duration(left))
	}
	return d
}

func paused(now time.Time, wait time.Duration) Delay {
	return Delay{Wait: wait, PausedUntil: now.Add(wait)}
}

func clamp(d time.Duration) time.Duration {
	return min(max(d, MinServerWait), MaxServerWait)
}

// failureWait is the bounded exponential: the base doubled (failures-1) times,
// stopping at the cap. It loops and stops at the cap, so no failure count can
// overflow it.
func failureWait(interval time.Duration, failures int) time.Duration {
	capWait := max(MaxFailureWait, interval)
	w := min(max(interval, MinFailureBase), capWait)
	for i := 1; i < failures && w < capWait; i++ {
		if w > capWait/2 {
			return capWait
		}
		w *= 2
	}
	return min(w, capWait)
}

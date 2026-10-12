package domain

import (
	"testing"
	"time"
)

func TestNextDelay(t *testing.T) {
	T := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	min := func(n int) time.Duration { return time.Duration(n) * time.Minute }
	sec := func(n int) time.Duration { return time.Duration(n) * time.Second }
	rl := func(cost, remaining int, reset time.Time) RateLimit {
		return RateLimit{Limit: 5000, Cost: cost, Remaining: remaining, ResetAt: reset}
	}
	ok := Outcome{}
	fail := Outcome{Failed: true}

	tests := []struct {
		name     string
		interval time.Duration
		failures int
		rl       RateLimit
		last     Outcome
		wait     time.Duration
		paused   time.Duration // PausedUntil = T + paused; zero means no pause
	}{
		{"success, no rate limit data", sec(60), 0, RateLimit{}, ok, sec(60), 0},
		{"remaining 10 at cost 1 pauses until reset+1s", sec(60), 0, rl(1, 10, T.Add(min(20))), ok, min(20) + sec(1), min(20) + sec(1)},
		{"remaining 11 at cost 1 does not pause", sec(60), 0, rl(1, 11, T.Add(min(10))), ok, sec(60), 0},
		{"remaining 11 with a far reset is paced, not paused", sec(60), 0, rl(1, 11, T.Add(min(20))), ok, min(20) / 11, 0},
		{"remaining 10 with the reset in the past", sec(60), 0, rl(1, 10, T.Add(-min(1))), ok, sec(60), 0},
		{"zero cost counts as cost 1", sec(60), 0, rl(0, 10, T.Add(min(5))), ok, min(5) + sec(1), min(5) + sec(1)},
		{"reserve scales with cost", sec(60), 0, rl(5, 50, T.Add(min(5))), ok, min(5) + sec(1), min(5) + sec(1)},
		{"pacing stretches the delay", sec(60), 0, rl(5, 100, T.Add(min(60))), ok, min(3), 0},
		{"pacing never goes below the interval", sec(60), 0, rl(1, 4000, T.Add(min(60))), ok, sec(60), 0},
		{"pacing with a reset in a year is cut to 65 min", sec(60), 0, rl(1, 11, T.Add(8760*time.Hour)), ok, min(65), 0},
		{"pacing with a reset in 100 years is cut to 65 min", sec(60), 0, rl(1, 5000, T.Add(876000*time.Hour)), ok, min(65), 0},

		{"failure 1", sec(60), 1, RateLimit{}, fail, sec(60), 0},
		{"failure 2", sec(60), 2, RateLimit{}, fail, sec(120), 0},
		{"failure 3", sec(60), 3, RateLimit{}, fail, sec(240), 0},
		{"failure 4 hits the cap", sec(60), 4, RateLimit{}, fail, sec(300), 0},
		{"failure 5", sec(60), 5, RateLimit{}, fail, sec(300), 0},
		{"failure 6", sec(60), 6, RateLimit{}, fail, sec(300), 0},
		{"failure 1000", sec(60), 1000, RateLimit{}, fail, sec(300), 0},
		{"long interval, failure 1", min(10), 1, RateLimit{}, fail, min(10), 0},
		{"short interval uses the 30 s base", sec(15), 1, RateLimit{}, fail, sec(30), 0},
		{"failure count zero counts as one", sec(60), 0, RateLimit{}, fail, sec(60), 0},

		{"retry-after 90 s", sec(60), 1, RateLimit{}, Outcome{Failed: true, RetryAfter: sec(90)}, sec(90), sec(90)},
		{"retry-after 1 ms is raised to 1 s", sec(60), 1, RateLimit{}, Outcome{Failed: true, RetryAfter: time.Millisecond}, sec(1), sec(1)},
		{"retry-after 100 h is cut to 65 min", sec(60), 1, RateLimit{}, Outcome{Failed: true, RetryAfter: 100 * time.Hour}, min(65), min(65)},
		{"retry-after beats a reset time", sec(60), 1, RateLimit{}, Outcome{Failed: true, RateLimited: true, RetryAfter: sec(30), ResetAt: T.Add(min(10))}, sec(30), sec(30)},

		{"rate limited with a reset time", sec(60), 1, RateLimit{}, Outcome{Failed: true, RateLimited: true, ResetAt: T.Add(min(10))}, min(10) + sec(1), min(10) + sec(1)},
		{"rate limited, reset from the last known limit", sec(60), 1, rl(1, 3, T.Add(min(5))), Outcome{Failed: true, RateLimited: true}, min(5) + sec(1), min(5) + sec(1)},
		{"rate limited, stale reset in the error falls back to the last known one", sec(60), 1, rl(1, 3, T.Add(min(5))), Outcome{Failed: true, RateLimited: true, ResetAt: T.Add(-min(1))}, min(5) + sec(1), min(5) + sec(1)},
		{"rate limited with no timing waits at least 60 s", sec(15), 1, RateLimit{}, Outcome{Failed: true, RateLimited: true}, sec(60), sec(60)},
		{"rate limited with no timing, deep in failures", sec(60), 3, RateLimit{}, Outcome{Failed: true, RateLimited: true}, sec(240), sec(240)},
		{"rate limited reset in a year is cut to 65 min", sec(60), 1, RateLimit{}, Outcome{Failed: true, RateLimited: true, ResetAt: T.Add(8760 * time.Hour)}, min(65), min(65)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextDelay(T, tc.interval, tc.failures, tc.rl, tc.last)
			if got.Wait != tc.wait {
				t.Errorf("Wait = %v, want %v", got.Wait, tc.wait)
			}
			var want time.Time
			if tc.paused != 0 {
				want = T.Add(tc.paused)
			}
			if !got.PausedUntil.Equal(want) {
				t.Errorf("PausedUntil = %v, want %v", got.PausedUntil, want)
			}
		})
	}
}

func TestNextDelayNoOverflow(t *testing.T) {
	T := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	for _, failures := range []int{1000, 1 << 30} {
		got := NextDelay(T, time.Minute, failures, RateLimit{}, Outcome{Failed: true})
		if got.Wait != MaxFailureWait {
			t.Errorf("failures %d: Wait = %v, want %v", failures, got.Wait, MaxFailureWait)
		}
	}

	huge := 1000 * time.Hour
	rl := RateLimit{Limit: 5000, Cost: 1, Remaining: 5000, ResetAt: T.Add(time.Hour)}
	if got := NextDelay(T, huge, 0, rl, Outcome{}); got.Wait < huge {
		t.Errorf("huge interval, success: Wait = %v, want >= %v", got.Wait, huge)
	}
	if got := NextDelay(T, huge, 1<<30, RateLimit{}, Outcome{Failed: true}); got.Wait != huge {
		t.Errorf("huge interval, failures: Wait = %v, want %v", got.Wait, huge)
	}
	// An interval so long that doubling it would overflow int64.
	const nearMax = time.Duration(1 << 62)
	if got := NextDelay(T, nearMax, 1<<30, RateLimit{}, Outcome{Failed: true}); got.Wait != nearMax {
		t.Errorf("near-max interval, failures: Wait = %v, want %v", got.Wait, nearMax)
	}
	// An interval below one second is treated as one second.
	if got := NextDelay(T, 0, 0, RateLimit{}, Outcome{}); got.Wait != time.Second {
		t.Errorf("zero interval: Wait = %v, want 1s", got.Wait)
	}
}

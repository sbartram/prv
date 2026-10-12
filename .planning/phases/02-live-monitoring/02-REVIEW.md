---
phase: 02-live-monitoring
reviewed: 2026-10-12T00:00:00Z
depth: standard
files_reviewed: 32
files_reviewed_list:
  - cmd/prv/main.go
  - cmd/prv/main_test.go
  - go.mod
  - go.sum
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/domain/backoff.go
  - internal/domain/backoff_test.go
  - internal/domain/domain_test.go
  - internal/domain/list.go
  - internal/domain/pr.go
  - internal/github/client.go
  - internal/github/client_test.go
  - internal/github/errors.go
  - internal/github/errors_test.go
  - internal/github/ratelimit.go
  - internal/ui/filter_test.go
  - internal/ui/help.go
  - internal/ui/help_test.go
  - internal/ui/keys.go
  - internal/ui/layout.go
  - internal/ui/model.go
  - internal/ui/model_test.go
  - internal/ui/open.go
  - internal/ui/open_test.go
  - internal/ui/ratelimit_test.go
  - internal/ui/refresh.go
  - internal/ui/refresh_test.go
  - internal/ui/selection_test.go
  - internal/ui/styles.go
  - internal/ui/view.go
  - internal/ui/view_test.go
findings:
  critical: 0
  warning: 2
  info: 4
  total: 6
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-10-12
**Depth:** standard
**Files Reviewed:** 32
**Status:** issues_found

## Summary

I reviewed the Phase 02 source (refresh loop, backoff policy, rate-limit capture, config loading, filters, help overlay, open-in-browser). I also read three domain files that are outside the list but are called by it: `domain/clean.go`, `domain/status.go` and the sort helpers. `go vet`, `gofmt -l` and `go test ./...` are all clean.

The Elm-loop design is sound. Ticks are generation-stamped, fetches never overlap, and selection follows the PR ref. `safeURL` correctly rejects the host-spoofing forms I tried (`github.com@evil.com`, `github.com.evil.com`, `github.com\@evil.com`, ports, whitespace). The T-02-08 pacing fix (`min(untilReset/left, MaxServerWait)`) is correct for the branch it patches.

There are no blockers. Two warnings remain:

- The same class of server-supplied-delay bug that T-02-08 closed still exists, one operation away, in the two `+ time.Second` paths.
- The Update methods rely on a Go evaluation order the language does not specify.

## Warnings

### WR-01: `+ time.Second` overflows `Duration` for far-future reset times, so a hostile or broken reset yields a 1 s pause

**File:** `internal/domain/backoff.go:63` and `internal/domain/backoff.go:77`
**Issue:** `time.Time.Sub` saturates at `math.MaxInt64` ns (about 292 years). The code then adds `time.Second` before clamping:

```go
return paused(now, clamp(reset.Sub(now)+time.Second))       // line 63
return paused(now, clamp(untilReset+time.Second))            // line 77
```

For any reset more than about 292 years away, `Sub` returns `MaxInt64`. Adding `time.Second` wraps it to a large negative value, and `clamp` raises that to `MinServerWait` (1 s). I confirmed this with a standalone program: `X-RateLimit-Reset: 99999999999`, or a JSON `resetAt` of `9999-01-01`, gives `+2562047h47m16.85s` from `Sub` and `-2562047h47m15.85s` after `+ time.Second`.

The effect is the opposite of what the policy intends. Instead of parking for `MaxServerWait`, prv pauses for 1 s, then retries. If the limit is really exhausted, that is a once-per-second request loop against GitHub. The code comment at lines 8-10 and T-02-08 in 02-SECURITY.md both say a hostile or broken server value "can neither spin the loop nor park it". Plan 02-06 also claims the saturating parsers (T-02-25) cover this, but `parseReset` has no upper bound and the overflow happens after parsing.

The T-02-08 fix patched the pacing branch (line 83). These two sibling branches have the same hole, so the audit's "all server-derived delays clamped" conclusion is not fully true.

**Fix:** Clamp the raw span first, then add the second. A small helper keeps all three call sites consistent:

```go
// untilReset returns the wait for a reset time, one second past it, bounded.
func untilReset(now, reset time.Time) time.Duration {
    return clamp(min(reset.Sub(now), MaxServerWait) + time.Second)
}
// line 63: return paused(now, untilReset(now, reset))
// line 77: return paused(now, untilReset(now, rl.ResetAt))
```

Also add a `TestNextDelay` row with `ResetAt = time.Unix(99999999999, 0)` for both the rate-limited failure path and the reserve path. The existing far-future row only covers the pacing branch.

### WR-02: `return m, m.mutatingMethod()` depends on unspecified evaluation order

**File:** `internal/ui/model.go:219`, `internal/ui/model.go:221`, `internal/ui/model.go:230`, `internal/ui/model.go:236`, `internal/ui/model.go:281`
**Issue:** `Update` has a value receiver. In these statements it returns `m` and, in the same `return`, calls a pointer-receiver method on `m` that mutates it:

```go
return m, m.setFlash("rate limited — resumes ...", flashFor) // sets m.flash, m.flashUntil
return m, m.beginFetch()                                       // sets m.inFlight = true
return m, m.arm(m.pausedUntil.Sub(now))                        // sets m.nextAt
```

The Go spec fixes the order of function calls but not the order of reading a variable relative to a call. Converting `m` to `tea.Model` can happen before or after `setFlash` or `beginFetch` runs. If it happens first, the returned model has no flash, no `inFlight = true` and no `nextAt`. The mutations are then lost silently, and the next `r` press could start a second overlapping fetch.

The tests pass because the gc compiler currently hoists the calls first. That is an implementation detail, not a guarantee, and it can change with inlining or compiler updates. The same pattern is correct at lines 305, 310 and 322 only because those assign `m.err` and similar fields before the `return`, and the calls there do not need to be seen by the conversion of `m`. Line 322 depends on the same ordering as well: `m.schedule(nil)` sets `m.pausedUntil` and `m.nextAt`.

**Fix:** Sequence the mutation explicitly, then return:

```go
cmd := m.beginFetch()
return m, cmd
```

Do the same for `setFlash`, `arm`, and the `tea.Batch(m.schedule(...), m.startWarning())` returns at lines 305, 310 and 322.

## Info

### IN-01: `wireNode.MergedAt` is requested from the API and decoded but never read

**File:** `internal/github/client.go:75` and `internal/github/client.go:127`
**Issue:** The query asks for `mergedAt` and the struct decodes it, but `mapResponse` ignores it. `ClosedAt` already covers merged PRs. This is dead payload.
**Fix:** Drop `mergedAt` from `discoveryQuery` and `MergedAt` from `wireNode`, or document why it is kept for Phase 3.

### IN-02: `Config.Path` is set in `Load` but never read outside tests

**File:** `internal/config/config.go:36` and `internal/config/config.go:121`
**Issue:** `main.go` tracks the path in its own `shownPath` variable, so `Config.Path` is dead.
**Fix:** Remove the field, or use it in `main.go` in place of `shownPath`.

### IN-03: `maxRetryAfterSeconds = 1 << 31` does not compile on 32-bit targets

**File:** `internal/github/ratelimit.go:54` and `internal/github/ratelimit.go:65`
**Issue:** `min(n, maxRetryAfterSeconds)` converts the constant to `int`. On a 32-bit `GOARCH` that overflows and the build fails. The current release targets are probably 64-bit only, but nothing documents or enforces that.
**Fix:** Declare `const maxRetryAfterSeconds int64 = 1 << 31`, parse with `strconv.ParseInt(..., 10, 64)`, and convert after `min`.

### IN-04: Positional arguments are silently ignored

**File:** `cmd/prv/main.go:66`
**Issue:** After `fs.Parse`, `fs.Args()` is never checked. Running `prv https://github.com/o/r/pull/1` starts the TUI and drops the argument without a message. This is a likely mistake once the add-PR-by-URL feature is planned.
**Fix:** After parsing, add `if fs.NArg() > 0 { fmt.Fprintln(stderr, "prv: unexpected argument"); return 2 }`.

---

_Reviewed: 2026-10-12_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_

---
phase: "02"
slug: "live-monitoring"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-12"
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| api.github.com → prv | Untrusted response bodies and headers (Retry-After, X-RateLimit-*, rateLimit.resetAt) drive refresh timing and classification | Rate-limit timing, PR metadata |
| prv → api.github.com | The refresh loop spends the user's shared token budget | Authenticated GraphQL requests |
| config.toml → prv | Hand-edited, possibly malformed or oversized file | refresh_interval, closed_days, show, unknown keys |
| CLI flags → prv | Hidden `--closed-days` flag | Integer |
| user paste / keys → prv | Pasted text in the number prompt | Arbitrary bytes, escape sequences |
| prv → OS browser launcher | `o` hands a URL to go-gh `pkg/browser` (BROWSER / GH_BROWSER / gh config) | PR URL |
| prv → terminal | Config path, warnings and remote text rendered in footer and help overlay | Cleaned display text |
| Go module proxy → build | Third-party code enters the binary | Source code |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Denial of service | internal/config Load | low | mitigate | `io.LimitReader` 1 MiB cap, oversize → exit 2 (config.go:31,130-137; TestLoadInterval/oversize) | closed |
| T-02-02 | Denial of service | ui refresh tick chain | high | mitigate | Tick armed only from current-generation results; dropped while in flight; stale results dropped; 15 s floor (model.go:276-297,365-378; TestAutoRefreshTickChain, TestTickWhileInFlightDropped, TestStaleTickDropped, TestStaleResultDropped) | closed |
| T-02-03 | Information disclosure | panic inside the fetch Cmd | medium | mitigate | recover with fixed text (refresh.go:36-40; TestFetchPanicShowsErrorPanel) | closed |
| T-02-06 | Tampering | relative XDG_CONFIG_HOME | low | mitigate | Only absolute XDG_CONFIG_HOME honoured (config.go:54-61; TestPath) | closed |
| T-02-SC | Tampering | Go module installs | high | mitigate | Phase 2 added BurntSushi/toml v1.6.0 (direct, prescribed in CLAUDE.md), cli/browser v1.3.0 and google/shlex v0.0.0-20191202100458-e7afc7fbc510 (indirect, exactly the versions pinned by go-gh v2.16.2). Hashes checked against sum.golang.org on 2026-10-12; `go mod verify` passes. Note: plan 02-10 claimed "go.mod/go.sum unchanged" — wrong, disclosed in its SUMMARY deviations; intent holds | closed |
| T-02-23 | Denial of service | `r` presses and overlapping fetches | high | mitigate | No Cmd while in flight; refused during a pause; `gen++` orphans pending tick (model.go:225-232; TestRefreshKey, TestManualRefreshOrphansTick, TestNoOverlappingFetches under -race) | closed |
| T-02-04 | Tampering | header freshness label | medium | mitigate | `lastOK` assigned only on success (model.go:315; TestUpdatedOnlyOnSuccess) | closed |
| T-02-12 | Tampering | selection after a refresh | medium | mitigate | Selection by Ref + nearest-neighbour (model.go:385-422; TestSelectionFollowsRef, TestSelectionInvariant) | closed |
| T-02-13 | Tampering | attention order | low | mitigate | No checks never counts as passing; sort keys polling does not change (status.go:43-44, list.go:59,72-85; TestBucket, TestSort) | closed |
| T-02-24 | Tampering | rows kept after a failed refresh | medium | mitigate | D-07 marker + "data from" replaces "updated", prio 7 (view.go:277-288; TestFailureMarker, TestHeaderPriority) | closed |
| T-02-05 | Information disclosure | Error.Short() in the header | low | mitigate | Fixed strings per Kind (errors.go:70-86; TestShortReason) | closed |
| T-02-27 | Tampering | footer notes / empty screen over partial load | medium | mitigate | Notes before hints, truncated not dropped; honest empty screen (view.go:85-87,137-184; TestFooterNotesFirst, TestEmptyStateShowsNotes). Residual: at ≤ 9-cell width only `api N` fits — unusable width, informational | closed |
| T-02-29 | Tampering | api budget display | low | mitigate | Rendered only when `Limit > 0`; snapshot kept through failures (view.go:140,299, help.go:48-53; TestApiFooter, TestRateWarning) | closed |
| T-02-08 | Denial of service (self) | server-supplied delays inside NextDelay | medium | mitigate | All server-derived delays clamped to [1 s, 65 min]. Audit found the success-path pacing branch unbounded (reset a year away → 796 h wait); fixed in f134a24 with `min(..., MaxServerWait)` and RED test 0d661db (TestNextDelay pacing rows) | closed |
| T-02-09 | Denial of service | refresh loop while limited | high | mitigate | Pause until retry-after/reset; `r` refused; tick during a pause re-arms without fetching (model.go:228-231,279-283, backoff.go; TestSimulatedHour) | closed |
| T-02-07 | Information disclosure | captureTransport | high | mitigate | 3-field record, only `resp.Header.Get` of three names, no request header read (ratelimit.go:14-44; TestCaptureHeadersOnly) | closed |
| T-02-25 | Tampering / DoS | malformed Retry-After / X-RateLimit-Reset | medium | mitigate | Strict parsers, saturation at 2^31 s (ratelimit.go:58-76; TestParseRetryAfter, TestParseReset) | closed |
| T-02-10 | Tampering | 403 classification | medium | mitigate | 403 rate-limited only with Retry-After or remaining 0, else KindForbidden (errors.go:124-127; TestRateLimitDetection) | closed |
| T-02-11 | Information disclosure | rate-limit error text | low | mitigate | Fixed message and hint (errors.go:62-63,147; TestTokenNeverAppears) | closed |
| T-02-14 | Tampering | state toggles and number filter | medium | mitigate | Indicator, "of" count, hidden-count message, unknown never hidden (view.go:44-75,262-267, list.go:110; TestFilteredEmpty, TestHiddenCount, TestUnknownAlwaysVisible) | closed |
| T-02-15 | Tampering | pasted text in the prompt | medium | mitigate | Only ASCII digits, max 7 (model.go:181-189,355; TestNumberPrompt/paste) | closed |
| T-02-17 | Tampering | unknown config key names in warnings | medium | mitigate | `domain.Clean` (config.go:164; TestUnknownKeyCleaned) | closed |
| T-02-18 | Tampering | --closed-days overflow (WR-01) | medium | mitigate | Flag rejected outside 1..365 (exit 2), config clamped (main.go:76-79, config.go:148; TestHiddenClosedDaysFlag, TestClampClosedDays) | closed |
| T-02-22 | Denial of service | malformed config | low | mitigate | Exit 2 with path and line before the TUI (config.go:100-110, main.go:86-91; TestLoadErrors, TestConfigErrorFormat) | closed |
| T-02-28 | Repudiation / integrity | config warning in the footer | low | mitigate | Separate fields, 8 s lifetime, never shortens flash/notes (model.go:34,118-120,330-336, view.go:176-180; TestConfigWarningLifetime, TestConfigWarningKeepsFlash) | closed |
| T-02-16 | Spoofing / EoP | `o` → safeURL → openCmd | high | mitigate | https, host exactly github.com, no port/userinfo/whitespace/control; only rebuilt `u.String()` opened (open.go:24-33, model.go:216-223; TestSafeURL) | closed |
| T-02-26 | Tampering | config path and warnings in the help overlay | medium | mitigate | `domain.Clean` (help.go:46,56; TestHelpWarningsAndCleanPath) | closed |
| T-02-19 | Denial of service | panic in the opener | medium | mitigate | recover into fixed flash (open.go:39-44; TestOpenPanicRecovered, TestOpenFailureFlash) | closed |
| T-02-20 | Tampering | browser launcher output | low | mitigate | `browser.New("", io.Discard, io.Discard)`; launch inside a Cmd (main.go:111, open.go:38-46; TestOpenKey). Residual tty takeover by a terminal browser: see AR-02-02 | closed |
| T-02-21 | Elevation of privilege | BROWSER / GH_BROWSER environment | low | accept | See AR-02-01 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-01 | T-02-21 | BROWSER / GH_BROWSER / gh config `browser` is the user's own environment, honoured the same way `gh` does; prv adds no new trust | plan 02-10 (user-approved plan) | 2026-10-12 |
| AR-02-02 | T-02-20 | A terminal browser named in BROWSER (w3m, lynx) can take over the tty while prv runs; `gh` has the same limit | plan 02-10 (user-approved plan) | 2026-10-12 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-12 | 30 | 28 | 2 (non-blocking: T-02-08, T-02-21) | gsd-security-auditor (ASVS L1, block_on high) |
| 2026-10-12 | 30 | 30 | 0 | orchestrator — T-02-08 fixed (0d661db, f134a24), T-02-21 logged as AR-02-01 |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-12

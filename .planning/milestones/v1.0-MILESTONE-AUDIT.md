---
milestone: v1
audited: 2026-10-13T00:00:00Z
status: tech_debt
scores:
  requirements: 38/38
  phases: 4/4
  integration: 14/14
  flows: 8/8
gaps:
  requirements: []
  integration: []
  flows: []
tech_debt:
  - phase: 01-status-at-a-glance
    items:
      - "Docs: 01-REVIEW-DISPOSITION.md lists WR-01/WR-02/WR-03 as open, but code fixed them in Phase 2 (--closed-days 1..365 in main.go; renderEmpty partial wording; footer notes-first, TestFooterNotesFirst)"
      - "Info findings IN-01..IN-04 open"
      - "UAT: 4 of 5 live checks skipped (live assignee:@me, row-by-row badge fidelity vs github.com, light/dark theme + narrow pane, real 401 flow, ambiguous-width glyphs); only 1 passed"
  - phase: 02-live-monitoring
    items:
      - "UAT: all 7 live checks skipped (real-terminal failure marker, api footer legibility, live rate-limit pause, filter prompt, real browser launch, help overlay, config-to-header wiring)"
      - "Info IN-01..IN-04 open: dead MergedAt payload, dead Config.Path, 32-bit build break on 1 << 31, positional args silently ignored"
      - "README does not document config.toml keys (refresh_interval as duration string, closed_days, show)"
  - phase: 03-blocker-detail
    items:
      - "WR-03 (deferred by user): shared latest-headers slot in captureTransport; concurrent Fetch/FetchDetail/Lookup can classify with another request's Retry-After/reset (integration G1, RFSH-05)"
      - "Detail and add-Lookup rateLimit cost is discarded; header 'api N' and NextDelay reserve reflect only the last list fetch (integration G2, RFSH-05)"
      - "Detail refresh skipped while a detail fetch is in flight; can be one interval stale (integration G5, by design)"
      - "Info IN-01..IN-09 open"
      - "UAT 1, 3, 4 skipped (large CI matrix, team review request, MERGE vs github.com merge box)"
  - phase: 04-curated-watch-list
    items:
      - "WR-01: Store.Add/Remove unlocked read-modify-write; two prv panes can lose an entry (integration G3, WATCH-05/06) — recommended fix: flock on sibling lock file"
      - "WR-02: corrupt backup published before copy completes"
      - "WR-03: Add/Remove over unknown-version file starts fresh with no add-time notice (bytes backed up)"
      - "WR-04: unknown JSON keys dropped; one mistyped entry marks the whole file corrupt"
      - "WR-05: esc after Lookup but before save still persists the PR"
      - "Lookup drops partial-data warnings in the add path; transient until next refresh (integration G4)"
      - "Reconcile clears Auto on a kept row (watch.go:96); only reachable when discovery missed the PR that refresh, so mostly correct (integration G6)"
      - "Info IN-01..IN-03 open"
      - "UAT: SAML/SSO error shape constructed from docs, not recorded live; live smoke skipped"
---

# Milestone v1 Audit: prv

**Audited:** 2026-10-13
**Status:** tech_debt. No blockers. All requirements are satisfied, every cross-phase seam is wired, and every E2E flow is complete. Open review warnings and skipped live UAT checks remain.

## Definition of done

The ROADMAP has 4 vertical phases: Status at a Glance, Live Monitoring, Blocker Detail, and Curated Watch List. All are marked complete, with 32 of 32 plans done. Core value: "At a glance, without opening a browser, the user can see which of their watched PRs are approved, blocked, or failing CI." The code delivers this end to end.

## Phases

| Phase | VERIFICATION | Score | UAT | Security | Open review findings |
|-------|--------------|-------|-----|----------|----------------------|
| 01 Status at a Glance | passed | 5/6 (1 backstop truth routed to human) | 1 pass, 4 skipped | verified, 0 open | WR-01..03 (fixed in code, doc stale), IN-01..04 |
| 02 Live Monitoring | passed | 5/5 | 7 skipped | verified, 0 open | IN-01..04 |
| 03 Blocker Detail | passed | 4/4 + gap-closure truths | 2 pass, 3 skipped, 1 issue fixed (03-11) | verified, 0 open | WR-03 deferred, IN-01..09 |
| 04 Curated Watch List | passed | 4/4 | 2 pass, 1 skipped | verified, 0 open | WR-01..05, IN-01..03 |

No phase is unverified.

## Requirements coverage (3-source cross-reference)

Sources: the VERIFICATION.md requirements tables, the SUMMARY.md `requirements-completed` lists, and the REQUIREMENTS.md traceability table and checkboxes.

| REQ | Phase | VERIFICATION | SUMMARY | REQUIREMENTS | Final |
|-----|-------|--------------|---------|--------------|-------|
| AUTH-01 | 1 | SATISFIED | 01-01 | [x] | satisfied |
| AUTH-02 | 1 | SATISFIED | 01-01 | [x] | satisfied |
| AUTH-03 | 1 | SATISFIED | 01-01, 01-02, 01-04 | [x] | satisfied |
| CONF-01 | 2 | SATISFIED | 02-01, 02-09 | [x] | satisfied |
| CONF-02 | 1 | SATISFIED | 01-01, 01-04 | [x] | satisfied |
| WATCH-01 | 1 | SATISFIED | 01-01, 01-02 | [x] | satisfied |
| WATCH-02 | 1 | SATISFIED (live evidence is a stand-in) | 01-01, 01-02 | [x] | satisfied |
| WATCH-03 | 1 | SATISFIED | 01-02 | [x] | satisfied |
| WATCH-04 | 4 | SATISFIED | 04-01, 04-02, 04-03 | [x] | satisfied |
| WATCH-05 | 4 | SATISFIED | 04-01, 04-04 | [x] | satisfied (multi-pane race: tech debt) |
| WATCH-06 | 4 | SATISFIED | 04-04, 04-06, 04-07 | [x] | satisfied |
| WATCH-07 | 4 | SATISFIED | 04-05, 04-07 | [x] | satisfied |
| LIST-01 | 1 | SATISFIED | 01-01..03 | [x] | satisfied |
| LIST-02 | 1 | SATISFIED | 01-01..03 | [x] | satisfied |
| LIST-03 | 1 | SATISFIED | 01-01, 01-03 | [x] | satisfied |
| LIST-04 | 3 | SATISFIED | 03-04, 03-05 | [x] | satisfied |
| LIST-05 | 1 | SATISFIED | 01-01, 01-03 | [x] | satisfied |
| LIST-06 | 3 | SATISFIED | 03-04, 03-05 | [x] | satisfied |
| LIST-07 | 3 | SATISFIED (accepted >100-context limit) | 03-01, 03-02 | [x] | satisfied |
| LIST-08 | 2 | SATISFIED | 02-03 | [x] | satisfied |
| LIST-09 | 4 | SATISFIED | 04-02, 04-05, 04-07 | [x] | satisfied |
| FILT-01 | 2 | SATISFIED | 02-08 | [x] | satisfied |
| FILT-02 | 2 | SATISFIED | 02-09 | [x] | satisfied |
| FILT-03 | 2 | SATISFIED | 02-08 | [x] | satisfied |
| DETL-01 | 3 | SATISFIED | 03-06..11 | [x] | satisfied |
| DETL-02 | 3 | SATISFIED | 03-06, 03-07, 03-09, 03-11 | [x] | satisfied |
| DETL-03 | 3 | SATISFIED | 03-07, 03-08 | [x] | satisfied |
| RFSH-01 | 2 | SATISFIED | 02-01 | [x] | satisfied |
| RFSH-02 | 2 | SATISFIED | 02-02, 02-06 | [x] | satisfied |
| RFSH-03 | 2 | SATISFIED | 02-02, 02-04 | [x] | satisfied |
| RFSH-04 | 2 | SATISFIED | 02-01, 02-04, 02-07 | [x] | satisfied |
| RFSH-05 | 2 | SATISFIED | 02-05..07, 03-10 | [x] | satisfied (accounting covers list fetch only: G1/G2 tech debt) |
| RFSH-06 | 2 | SATISFIED | 02-02 | [x] | satisfied |
| NAV-01 | 1 | SATISFIED | 01-01, 01-03 | [x] | satisfied |
| NAV-02 | 2 | SATISFIED | 02-10 | [x] | satisfied |
| NAV-03 | 2 | SATISFIED | 02-10 | [x] | satisfied |
| NAV-04 | 1 | SATISFIED | 01-01, 01-02 | [x] | satisfied |
| NAV-05 | 3 | SATISFIED | 03-03, 03-05, 03-08 | [x] | satisfied |

Result: 38/38 satisfied. 0 partial, 0 unsatisfied, 0 orphaned.

RFSH-05 note: the integration checker rated RFSH-05 PARTIAL. The requirement says that the remaining limit is shown and that prv backs off when the limit is low or hit. Both behaviours are present, and they are tested by the simulated-hour test and the end-to-end rate-limit test. The gap is that the shown budget leaves out the cost of detail and add lookups (G2), and that header attribution is shared between requests (G1, Phase 3 WR-03, deferred by the user). This audit records RFSH-05 as satisfied with tech debt, not as a blocker.

## Cross-phase integration

The integration checker found 14 connected seams, 0 orphaned and 0 missing. It ran `go build ./... && go vet ./... && go test ./... -count=1 -race`, and all 6 packages passed.

Key seams it confirmed:
- Phase 4 saved PRs go through the Phase 2 sort, filters, window (InWindow exempts Manual rows), selection by ref, and epoch discard. They also go through the Phase 3 `fetchFailing`.
- Rate-limit errors from Phase 3 detail requests and Phase 4 add lookups go to the same Phase 2 `pausedUntil`. Add and detail requests are refused during a pause.
- One keymap drives the footer, the `?` overlay and the detail help. Each prompt isolates its keys: digits, `q`, `a` and `d` are typed as text inside a prompt.
- Unavailable rows skip the failing-name and detail fetches. `o` still opens the saved ref URL through `safeURL`.

| E2E flow | Status |
|----------|--------|
| (a) Startup with a token: fetch, then list | complete |
| (b) No token: clear exit 1 | complete |
| (c) Auto-refresh with rate-limit pause | complete (G1/G2 caveat) |
| (d) Filter, select, refresh: selection kept | complete |
| (e) Detail, refresh, esc: selection kept | complete |
| (f) Add by URL, save, restart: still listed | complete |
| (g) Saved PR becomes unavailable: reason row, then d+y removes it | complete |
| (h) `o` opens the browser, also on an unavailable row | complete |

### Integration findings (all non-critical)

| ID | Finding | REQ | Disposition |
|----|---------|-----|-------------|
| G1 | Shared rate-limit header slot across concurrent requests | RFSH-05 | Known Phase 3 WR-03, deferred by user |
| G2 | Detail and Lookup rateLimit cost not counted in "api N" or the reserve | RFSH-05 | Tech debt |
| G3 | Watch-list read-modify-write is unlocked across two prv instances | WATCH-05, WATCH-06 | Same as Phase 4 WR-01, the best candidate for a fix |
| G4 | Add path drops partial-data warnings (transient) | LIST-07, LIST-09 | Tech debt |
| G5 | Detail refresh skipped while a detail fetch is in flight | DETL-01 | By design (single-flight) |
| G6 | Reconcile clears `Auto` on a kept row (watch.go:96) | WATCH-03 | Low impact: reached only when this refresh's discovery missed the PR |
| G7 | A key that cancels the remove confirm is consumed | NAV-01 | By design (D-06: any other key cancels) |
| G8 | Phase 1 review disposition says WR-01..03 open, but they are fixed in code | none | Docs only |

## Nyquist

Skipped. No validate-phase step hook is active (verify:post hooks are security and ui-review only).

## Assessment

Nothing blocks completion. The debt falls into three groups:

1. **Live verification debt.** It is the largest group. Phase 2 skipped all 7 live UAT checks, and Phase 1 skipped 4 of 5. The logic is covered by tests that use fixtures and fake transports. Still, no one has compared real rows with github.com, watched a live rate-limit pause, or checked a real browser launch and the terminal themes.
2. **Watch-list robustness** (Phase 4 WR-01..05). WR-01 (two panes can lose an entry) is the only one with a plausible real-world path for a tool that lives in multiplexer panes.
3. **Rate-limit accounting** (G1/G2). This is a known and accepted deferral.

# Requirements: prv

**Defined:** 2026-10-11
**Core Value:** At a glance, without opening a browser, the user can see which of their watched PRs are approved, blocked, or failing CI.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Watch list

- [ ] **WATCH-01**: PRs the user authored appear on the list automatically
- [ ] **WATCH-02**: PRs assigned to the user appear on the list automatically
- [ ] **WATCH-03**: A PR that is both authored and assigned (or also added manually) appears only once
- [ ] **WATCH-04**: User can add a PR by pasting a github.com PR URL or `owner/repo#123`; invalid or inaccessible PRs show an error and are not saved
- [ ] **WATCH-05**: Manually added PRs are saved locally and reappear after restart
- [ ] **WATCH-06**: User can remove a manually added PR
- [ ] **WATCH-07**: Manually added PRs stay listed after they close, until the user removes them (the closed window does not apply to them)

### List display

- [ ] **LIST-01**: Each row shows repo, PR number, title, author, and a draft marker
- [ ] **LIST-02**: Each row shows state: Open, Merged, or Closed
- [ ] **LIST-03**: Each row shows review status: approved, changes requested, review required, or no review status. Repos without required reviews still show approved or changes requested based on actual reviews, and never show "review required" by mistake
- [ ] **LIST-04**: Each row shows reviewers who approved or requested changes, and reviewers still pending
- [ ] **LIST-05**: Each row shows one CI badge: passing, failing, running, or no checks. A PR with no checks never shows as passing
- [ ] **LIST-06**: Each row shows merge-readiness: mergeable, blocked, behind, or conflicts
- [ ] **LIST-07**: For failing PRs, the row shows the name of a failing check
- [ ] **LIST-08**: List is sorted by need for attention (failing CI / changes requested first) in a stable order that does not shuffle between refreshes
- [ ] **LIST-09**: A PR that can't be fetched shows as an "unavailable" row with a reason, and the other PRs still display

### Filtering

- [ ] **FILT-01**: User can toggle Open / Merged / Closed visibility in any combination
- [ ] **FILT-02**: Auto-found PRs that are merged or closed only show if they closed within a configurable window (default 14 days)
- [ ] **FILT-03**: User can type a PR number to narrow the list to matching PRs

### Detail view

- [ ] **DETL-01**: Selecting a PR opens a detail view listing every check/job with its status, failing first
- [ ] **DETL-02**: Detail view lists each reviewer and their review state
- [ ] **DETL-03**: Returning from the detail view keeps the same PR selected

### Refresh

- [ ] **RFSH-01**: Data refreshes automatically on a configurable interval (default 60s, minimum 15s)
- [ ] **RFSH-02**: User can press `r` to refresh immediately
- [ ] **RFSH-03**: The time of the last successful update is shown
- [ ] **RFSH-04**: If a refresh fails, the last data stays on screen with an error indicator
- [ ] **RFSH-05**: Remaining API rate limit is shown, and refreshing backs off automatically when the limit is low or hit
- [ ] **RFSH-06**: The selected PR stays selected after a refresh, even if the list order changes

### Navigation

- [ ] **NAV-01**: User can move through the list with j/k and the arrow keys
- [ ] **NAV-02**: User can press `o` to open the selected PR in the browser
- [ ] **NAV-03**: User can press `?` to see key bindings
- [ ] **NAV-04**: User can quit with `q`
- [ ] **NAV-05**: Layout adapts to terminal resizes and stays usable in narrow panes

### Auth & config

- [x] **AUTH-01**: Uses the same token `gh` would: `GH_TOKEN`, then `GITHUB_TOKEN`, then the `gh` CLI login
- [x] **AUTH-02**: If no token is found, prv exits with a clear message explaining how to log in
- [ ] **AUTH-03**: The token is never logged, displayed, or written to disk
- [ ] **CONF-01**: Settings (refresh interval, closed window) are read from `~/.config/prv/config.toml`, with defaults if the file is missing
- [ ] **CONF-02**: `go install github.com/sbartram/prv/cmd/prv@latest` installs a working binary, and `prv --version` prints its version

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Display

- **DISP-01**: Rows that changed since the last refresh are highlighted until viewed
- **DISP-02**: Unresolved review-thread count per PR
- **DISP-03**: Group PRs by repo

### Filtering

- **FILT2-01**: Text filter on repo and title

### Refresh

- **RFSH2-01**: Adaptive refresh (faster while checks run, slower when idle)
- **RFSH2-02**: Opt-in terminal bell on state change

### Utility

- **UTIL-01**: Copy the selected PR URL to the clipboard
- **UTIL-02**: `--json` output mode
- **UTIL-03**: Start with a PR passed as a command-line argument
- **UTIL-04**: Homebrew release via GoReleaser

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Write actions (merge, approve, comment, re-run workflows) | v1 is watch-only + open in browser; acting happens on github.com |
| PRs where the user's review is requested | User defined "owned" as authored + assigned only |
| Full closed/merged history | Noise; only a recent window is shown |
| GitHub Enterprise hosts | github.com only |
| Desktop/push notifications, notifications inbox | TUI lives in a pane the user glances at |
| Diff view, CI log viewing | Open in browser instead |
| Issues | PRs only |
| Custom search-query sections | Curated watch list is the model, not saved searches (gh-dash covers that) |
| OAuth login flow | Reuse gh's token |

## Technical Defaults (from research conflicts)

Recorded so planning doesn't re-litigate them; each can be revisited if the Phase 1 live spike disproves it.

- **Discovery:** single `ISSUE_ADVANCED` search `is:pr (author:@me OR assignee:@me) (is:open OR closed:>=DATE)`; fall back to separate queries / `viewer.pullRequests` if authored PRs go missing. Dedupe by node ID.
- **Check detail:** list badge from `statusCheckRollup.state` and counts; full check list fetched lazily for the selected PR.
- **Review derivation:** `reviewDecision` if set, else derived from `latestOpinionatedReviews`; verify COMMENTED-after-APPROVED behaviour in spike.

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| WATCH-01 | Phase 1 | Pending |
| WATCH-02 | Phase 1 | Pending |
| WATCH-03 | Phase 1 | Pending |
| WATCH-04 | Phase 4 | Pending |
| WATCH-05 | Phase 4 | Pending |
| WATCH-06 | Phase 4 | Pending |
| WATCH-07 | Phase 4 | Pending |
| LIST-01 | Phase 1 | Pending |
| LIST-02 | Phase 1 | Pending |
| LIST-03 | Phase 1 | Pending |
| LIST-04 | Phase 3 | Pending |
| LIST-05 | Phase 1 | Pending |
| LIST-06 | Phase 3 | Pending |
| LIST-07 | Phase 3 | Pending |
| LIST-08 | Phase 2 | Pending |
| LIST-09 | Phase 4 | Pending |
| FILT-01 | Phase 2 | Pending |
| FILT-02 | Phase 2 | Pending |
| FILT-03 | Phase 2 | Pending |
| DETL-01 | Phase 3 | Pending |
| DETL-02 | Phase 3 | Pending |
| DETL-03 | Phase 3 | Pending |
| RFSH-01 | Phase 2 | Pending |
| RFSH-02 | Phase 2 | Pending |
| RFSH-03 | Phase 2 | Pending |
| RFSH-04 | Phase 2 | Pending |
| RFSH-05 | Phase 2 | Pending |
| RFSH-06 | Phase 2 | Pending |
| NAV-01 | Phase 1 | Pending |
| NAV-02 | Phase 2 | Pending |
| NAV-03 | Phase 2 | Pending |
| NAV-04 | Phase 1 | Pending |
| NAV-05 | Phase 3 | Pending |
| AUTH-01 | Phase 1 | Complete |
| AUTH-02 | Phase 1 | Complete |
| AUTH-03 | Phase 1 | Pending |
| CONF-01 | Phase 2 | Pending |
| CONF-02 | Phase 1 | Pending |

**Coverage:**
- v1 requirements: 38 total
- Mapped to phases: 38
- Unmapped: 0 ✓

---
*Requirements defined: 2026-10-11*
*Last updated: 2026-10-11 after roadmap creation (traceability mapped)*

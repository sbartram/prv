---
phase: 03
review: 03-REVIEW.md
titles: json
findings:
  - id: WR-03
    severity: warning
    disposition: deferred
    title: "Shared \"latest response headers\" is now racy, and detail requests are invisible to the budget"
  - id: IN-01
    severity: info
    disposition: open
    title: "Cached viewport lines go stale for time-dependent summary text"
  - id: IN-02
    severity: info
    disposition: open
    title: "A rate-limited detail result that arrives after esc is dropped without pausing"
  - id: IN-03
    severity: info
    disposition: open
    title: "\"loading…\" can show forever for a PR that left the list before its first fetch"
  - id: IN-04
    severity: info
    disposition: open
    title: "Detail query re-requests reviewers on every page, and review states are parsed twice"
  - id: IN-05
    severity: info
    disposition: open
    title: "Detail view is line-scroll only; help legend omits the commented and dismissed glyphs"
  - id: IN-06
    severity: info
    disposition: open
    title: "`ServerWait` and `NextDelay` agree only by coincidence when the error has no timing, and `ServerWait` ignores the known rate-limit reset"
  - id: IN-07
    severity: info
    disposition: open
    title: "The help legend does not name the new \"unknown\" check class"
  - id: IN-08
    severity: info
    disposition: open
    title: "A kept error on a later page sets `Partial` even though that page's reviewer data is never used"
  - id: IN-09
    severity: info
    disposition: open
    title: "The partial notice is untested at widths below its own length, and its refresh behaviour is untested"
  - id: WR-05
    severity: warning
    disposition: open
    title: "A partial detail read is shown as \"read N of N runs\", or hides \"no checks\""
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Detail rate-limit pause bypasses the server-wait clamp"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "FetchDetail silently undercounts checks and reviewers on partial data"
  - id: WR-04
    severity: warning
    disposition: fixed
    title: "Unknown and neutral checks are folded away under the label \"more passing\" and counted as \"skipped\""
open: 10
total: 14
recorded: 2026-10-13T02:31:18.421Z
---

# Phase 03: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-03 | warning | deferred | UAT 2026-10-12: deferred by user; per-request header capture and detail budget accounting left for a later phase |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | - |
| IN-07 | info | open | - |
| IN-08 | info | open | - |
| IN-09 | info | open | - |
| WR-05 | warning | open | UAT 2026-10-12: user chose fix before ship; tracked as gap G-03-6 (not in the current review) |
| WR-01 | warning | fixed | Fixed by 03-10 Task 1 (domain.ServerWait clamp in pauseForRateLimit); confirmed resolved by re-review 2026-10-12 (not in the current review) |
| WR-02 | warning | fixed | Fixed by 03-09 (FetchDetail marks Truncated on partial reads); confirmed resolved in the data layer by re-review 2026-10-12 — display follow-up is WR-05 (not in the current review) |
| WR-04 | warning | fixed | Fixed by 03-10 Task 2 (CheckCounts.Other, unknown checks listed, fold line reworded); confirmed resolved by re-review 2026-10-12 (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.

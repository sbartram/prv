---
phase: 03
review: 03-REVIEW.md
titles: json
findings:
  - id: WR-05
    severity: warning
    disposition: open
    title: "A partial detail read is shown as \"read N of N runs\", or hides \"no checks\""
  - id: IN-06
    severity: info
    disposition: open
    title: "`ServerWait` and `NextDelay` agree only by coincidence when the error has no timing, and `ServerWait` ignores the known rate-limit reset"
  - id: IN-07
    severity: info
    disposition: open
    title: "The help legend does not name the new \"unknown\" check class"
  - id: WR-01
    severity: warning
    disposition: open
    title: "Detail rate-limit pause bypasses the server-wait clamp"
  - id: WR-02
    severity: warning
    disposition: open
    title: "FetchDetail silently undercounts checks and reviewers on partial data"
  - id: WR-03
    severity: warning
    disposition: deferred
    title: "Shared \"latest response headers\" is now racy, and detail requests are invisible to the budget"
  - id: WR-04
    severity: warning
    disposition: open
    title: "Unknown and neutral checks are folded away under the label \"more passing\" and counted as \"skipped\""
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
open: 11
total: 12
recorded: 2026-10-13T01:46:07.404Z
---

# Phase 03: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-05 | warning | open | - |
| IN-06 | info | open | - |
| IN-07 | info | open | - |
| WR-01 | warning | open | UAT 2026-10-12: fix before ship (gap G-03-6-WR01) (not in the current review) |
| WR-02 | warning | open | UAT 2026-10-12: fix before ship (gap G-03-6-WR02) (not in the current review) |
| WR-03 | warning | deferred | UAT 2026-10-12: deferred by user; per-request header capture and detail budget accounting left for a later phase (not in the current review) |
| WR-04 | warning | open | UAT 2026-10-12: fix before ship (gap G-03-6-WR04) (not in the current review) |
| IN-01 | info | open | - (not in the current review) |
| IN-02 | info | open | - (not in the current review) |
| IN-03 | info | open | - (not in the current review) |
| IN-04 | info | open | - (not in the current review) |
| IN-05 | info | open | - (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.

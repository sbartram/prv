---
phase: 03
review: 03-REVIEW.md
titles: json
findings:
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
open: 8
total: 9
recorded: 2026-10-12T23:59:33.648Z
---

# Phase 03: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | UAT 2026-10-12: fix before ship (gap G-03-6-WR01) |
| WR-02 | warning | open | UAT 2026-10-12: fix before ship (gap G-03-6-WR02) |
| WR-03 | warning | deferred | UAT 2026-10-12: deferred by user; per-request header capture and detail budget accounting left for a later phase |
| WR-04 | warning | open | UAT 2026-10-12: fix before ship (gap G-03-6-WR04) |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.

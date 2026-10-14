---
phase: 01
review: 01-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "`--closed-days` has no upper bound; large values overflow and silently drop closed PRs"
  - id: WR-02
    severity: warning
    disposition: open
    title: "Empty-state screen claims \"No open PRs\" even when the fetch was partial or truncated"
  - id: WR-03
    severity: warning
    disposition: open
    title: "Footer puts the \"incomplete list\" indicators last, so `cut` drops them first on narrow panes"
  - id: IN-01
    severity: info
    disposition: open
    title: "Dead data: `wireNode.MergedAt` and the text of `Snapshot.Warnings`"
  - id: IN-02
    severity: info
    disposition: open
    title: "Extra positional arguments are silently ignored"
  - id: IN-03
    severity: info
    disposition: open
    title: "`Clean` leaves implicit bidi marks and Unicode line/paragraph separators"
  - id: IN-04
    severity: info
    disposition: open
    title: "`classify` reports every `net.Error` as \"could not reach api.github.com\""
open: 7
total: 7
recorded: 2026-10-12T01:24:00.141Z
---

# Phase 01: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.

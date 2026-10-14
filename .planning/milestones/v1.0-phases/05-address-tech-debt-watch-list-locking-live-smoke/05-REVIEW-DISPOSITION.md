---
phase: 05
review: 05-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "Live test skips on expired CI data before it reports a Review or State regression on the same pin"
  - id: WR-02
    severity: warning
    disposition: open
    title: "Removing from an unparseable file reports success, writes nothing, and the \"started over\" notice cannot fire"
  - id: IN-01
    severity: info
    disposition: open
    title: "A backup is published without syncing the directory, and an existing backup of the same 8-digit name counts as complete"
  - id: IN-02
    severity: info
    disposition: open
    title: "Test hook `afterRead` lives in the production `Store` and is wired only into `AddRef`"
  - id: IN-03
    severity: info
    disposition: open
    title: "`sync` local variable shadows the imported package in the helper test"
  - id: IN-04
    severity: info
    disposition: open
    title: "Windows branches in tests cannot run"
open: 6
total: 6
recorded: 2026-10-14T03:55:34.338Z
---

# Phase 05: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.

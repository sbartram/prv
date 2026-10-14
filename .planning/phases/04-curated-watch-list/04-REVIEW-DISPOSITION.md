---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Add/Remove is an unlocked read-modify-write; concurrent prv processes lose updates"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "Backup is not crash-safe, and a partial backup is treated as complete"
  - id: WR-03
    severity: warning
    disposition: fixed
    title: "Add/Remove silently replaces a corrupt or newer-version file, and the user is not told in the add flow"
  - id: WR-04
    severity: warning
    disposition: deferred
    title: "Round-tripping through typed structs drops unknown fields, and one wrongly typed entry makes the whole file \"corrupt\""
  - id: WR-05
    severity: warning
    disposition: fixed
    title: "Esc while a save is in flight does not cancel the save and gives no feedback"
  - id: IN-01
    severity: info
    disposition: deferred
    title: "Lookup and Fetch share one \"latest response headers\" slot, so classification can use the other request's headers"
  - id: IN-02
    severity: info
    disposition: fixed
    title: "Add failure reasons are collapsed into one generic message"
  - id: IN-03
    severity: info
    disposition: deferred
    title: "Parse errors are hidden during a rate-limit pause"
open: 0
total: 8
recorded: 2026-10-14T00:06:26.077Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | fixed | 05-01: flock on watchlist.json.lock; TestMultiProcessAdd |
| WR-02 | warning | fixed | 05-01: temp + Sync + os.Link backup |
| WR-03 | warning | fixed | 05-01 + 05-02: version refusal; 'previous list saved to' flash |
| WR-04 | warning | deferred | Phase 5 D-01: unknown JSON keys and per-entry tolerant decode deferred |
| WR-05 | warning | fixed | 05-02: saving state; esc ignored once the save starts |
| IN-01 | info | deferred | Phase 5 CONTEXT Deferred G1: per-request rate-limit header capture, deferred by the user |
| IN-02 | info | fixed | 05-02: saveReason; watchlist.ErrFull |
| IN-03 | info | deferred | Phase 5 CONTEXT Deferred: parse errors hidden during a rate-limit pause |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.

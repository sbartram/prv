# Phase 3: Blocker Detail - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md. This log keeps the alternatives that were considered.

**Date:** 2026-10-12
**Phase:** 03-blocker-detail
**Areas discussed:** Check-fetch strategy (C3 gate), Row layout for blockers, Detail view shape, Narrow-pane priorities

---

## Check-fetch strategy (C3 gate)

| Option | Description | Selected |
|--------|-------------|----------|
| Two-step | Lean list; a batched nodes(ids:) contexts query for FAILURE PRs only; the detail fetch on open | ✓ |
| Eager in list query | contexts(first:100) on every PR; one request; about 30x the payload | |
| Eager for list, lazy detail | A small contexts(first:~25) in the list, full paginated list on open | |

| Option (several failing) | Selected |
|--------|----------|
| First name + count (`✗ lint +2`) | ✓ |
| Name only | |
| Count only when >1 | |

| Option (re-runs) | Selected |
|--------|----------|
| Latest run per workflow+name | ✓ |
| Show every context | |

| Option (only-cancelled failing PR) | Selected |
|--------|----------|
| Name it, marked `(cancelled)` | ✓ |
| Treat like failure | |

**User's choice:** All recommended options.

---

## Row layout for blockers

| Option | Description | Selected |
|--------|-------------|----------|
| New columns | REVIEWERS + MERGE columns, CI cell carries the failing name, one line per PR | ✓ |
| Second line per PR | A faint blocker line under each row | |
| One BLOCKERS column | Free-text summary of what is in the way | |

| Option (reviewers) | Selected |
|--------|----------|
| Glyph + login, then +N | ✓ |
| Counts only | |
| Names only for blockers | |

| Option (UNSTABLE/UNKNOWN) | Selected |
|--------|----------|
| UNSTABLE→mergeable, UNKNOWN→– | ✓ |
| UNSTABLE→its own 'unstable' | |
| You decide | |

**Notes:** The selected row preview left out AUTHOR. It was recorded as kept, and it drops first in narrow panes.

---

## Detail view shape

| Option | Description | Selected |
|--------|-------------|----------|
| Full screen | Replaces the list, like the ? overlay | ✓ |
| Split pane | List on top, detail below | |

| Option (keys) | Selected |
|--------|----------|
| enter opens, esc back, q quits | ✓ |
| enter opens, esc or q back | |

| Option (order) | Selected |
|--------|----------|
| Checks, then reviewers | ✓ |
| Reviewers, then checks | |

| Option (PR leaves list) | Selected |
|--------|----------|
| Stay, mark it, esc to neighbour | ✓ |
| Close detail automatically | |

| Option (passing checks, follow-up to resolve a preview/answer conflict) | Selected |
|--------|----------|
| Collapse passing, toggle to expand | ✓ |
| List everything, scroll | |

| Option (live detail) | Selected |
|--------|----------|
| Yes, on each refresh tick | ✓ |
| Only on open and on r | |

---

## Narrow-pane priorities

| Option (drop order) | Selected |
|--------|----------|
| AUTHOR → STATE → REVIEWERS → MERGE | ✓ |
| AUTHOR → REVIEWERS → MERGE → STATE | |
| You decide | |

| Option (minimum width) | Selected |
|--------|----------|
| Compact mode under ~40 cols | ✓ |
| Just cut at the right edge | |

---

## Claude's Discretion

- MERGE and reviewer glyphs; how team and bot reviewers render
- Detail loading and error states; the row fallback when the failing-contexts query fails
- The key that expands collapsed checks; column widths and caps; the exact compact threshold
- Help legend additions

## Deferred Ideas

None.

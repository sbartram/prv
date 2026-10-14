# Phase 2: Live Monitoring - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-11
**Phase:** 02-live-monitoring
**Areas discussed:** Attention sort, Status bar & refresh, Filter keys & PR# search, Config file & help

---

## Attention sort

| Option | Description | Selected |
|--------|-------------|----------|
| Act-on-it first | failing · changes · approved+passing · running · review/none · drafts · merged · closed | ✓ |
| Problems first, ready last | failing · changes · running · waiting · approved+passing · drafts · merged/closed | |
| Open by attention, then D-06 | Two groups for open, D-06 for the rest | |

| Option | Description | Selected |
|--------|-------------|----------|
| createdAt newest, then ref | Keys that polling does not change | ✓ |
| Ref only | Alphabetical within bucket | |
| updatedAt | Recent activity floats up, but rows shuffle | |

| Option | Description | Selected |
|--------|-------------|----------|
| Jump immediately | Cursor follows ref | ✓ |
| Only the selected row stays put | Pin the selection until cursor moves | |
| Freeze while cursor moving | Re-sort after idle | |

| Option | Description | Selected |
|--------|-------------|----------|
| Draft bucket always | Drafts stay in bucket 6 | ✓ |
| Status wins over draft | Failing draft sorts with failing | |

**User's choice:** All recommended options.

---

## Status bar & refresh

| Option | Description | Selected |
|--------|-------------|----------|
| Clock time + quiet limit | `updated HH:MM:SS`, limit only when low | ✓ |
| Relative age + always-on limit | `45s ago`, redraws every second | |
| Clock time + always-on limit | Literal reading of SC2 | |

| Option | Description | Selected |
|--------|-------------|----------|
| Dim in footer, loud when low | Faint `api N` in footer; header + yellow under 20% | ✓ |
| Only when low, amend criterion | Hidden while healthy | |
| Always in header after all | Simplest to verify | |

| Option | Description | Selected |
|--------|-------------|----------|
| Header marker + short reason | Rows not dimmed | ✓ |
| Header marker + dim all rows | Collides with draft/merged faint | |
| Footer line only | Easy to miss | |

| Option | Description | Selected |
|--------|-------------|----------|
| Refuse with message | No request during backoff | ✓ |
| Allow one manual refresh | Force a request | |
| Allow if limit not exhausted | Refuse only at 0 / retry-after | |

| Option | Description | Selected |
|--------|-------------|----------|
| Error panel + auto-retry | Recovers by itself | ✓ |
| Error panel, manual retry only | As in Phase 1 | |

**User's choice:** All recommended options.
**Notes:** Claude pointed out that the user's first choice ("quiet limit") conflicts with the SC2 wording "The status bar shows the remaining API rate limit". The follow-up question settled this: a faint footer value is always shown.

---

## Filter keys & PR# search

| Option | Description | Selected |
|--------|-------------|----------|
| 1 / 2 / 3 | Toggle by position | ✓ |
| Shift O / M / C | Mnemonic; next to `o` | |
| One `s` key that cycles | Not every combination | |

| Option | Description | Selected |
|--------|-------------|----------|
| `/` prompt, prefix match, live | Digits only; enter keep, esc clear | ✓ |
| `/` prompt, exact match | | |
| `/` prompt, substring | | |

| Option | Description | Selected |
|--------|-------------|----------|
| All on; config sets default | Session-only toggles | ✓ |
| All on, not configurable | | |
| Remember last session | Writes state in Phase 2 | |

| Option | Description | Selected |
|--------|-------------|----------|
| Filtered-empty message | Different from D-10 empty state | ✓ |
| Empty table + header count | | |

**User's choice:** All recommended options.

---

## Config file & help

| Option | Description | Selected |
|--------|-------------|----------|
| Flat, duration strings | `refresh_interval = "60s"`, `closed_days`, `show` | ✓ |
| Flat, integer seconds | | |
| Sectioned tables | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Exit on syntax/type error, warn on unknown keys | Clamp out-of-range values | ✓ |
| Never exit; warn and use defaults | | |
| Strict: exit on anything | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Keep as override, bounded | Flag > config > default, 1..365 (fixes WR-01) | ✓ |
| Public flags for both | | |
| Remove; config only | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Full-screen overlay | Keys, legend, settings, api | ✓ |
| Expanded footer | | |

**User's choice:** All recommended options.

---

## Claude's Discretion

- Whether approved + no checks counts as ready (bucket 3) or other (bucket 5)
- Header/footer wording; footer order and cut priority (WR-03); partial-fetch warnings on empty screens (WR-02)
- `o` failure feedback; how long transient messages stay; debug log path/gating

## Deferred Ideas

None.

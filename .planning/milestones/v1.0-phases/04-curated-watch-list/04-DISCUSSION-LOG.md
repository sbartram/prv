# Phase 4: Curated Watch List - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-12
**Phase:** 04-curated-watch-list
**Areas discussed:** Add flow, Manual marker & removal, Unavailable rows, Manual PRs in the list

---

## Add flow

| Option | Description | Selected |
|--------|-------------|----------|
| a, bottom line | One-line prompt at the bottom like `/` | ✓ |
| a, centred modal | Bordered box over the list | |
| + key, bottom line | Same prompt bound to `+` | |

| Option | Description | Selected |
|--------|-------------|----------|
| Prompt stays, inline result | `checking ⠋`; error inline with text kept; success closes and jumps to row | ✓ |
| Prompt closes, footer result | Background check, footer reports | |
| Placeholder 'checking' row | Row appears at once and fills in | |

| Option | Description | Selected |
|--------|-------------|----------|
| Close prompt, jump to row | No new row; footer `already watching`; notes hidden by filters | ✓ |
| Inline error, prompt stays | Treat duplicate as an error | |
| Auto-found → also save as manual | Pin auto-found PRs on paste | |

| Option | Description | Selected |
|--------|-------------|----------|
| Lenient on github.com PR forms | Trailing /files etc, fragments, query, no scheme, www, whitespace; reject other hosts and issues | ✓ |
| Strict: exactly two forms | | |
| Lenient + several per paste | | |

## Manual marker & removal

| Option | Description | Selected |
|--------|-------------|----------|
| Glyph before the ref | `+ cli/cli#123`, survives compact mode | ✓ |
| `added` tag after title | Drops out when TITLE shrinks | |
| No marker | | |

| Option | Description | Selected |
|--------|-------------|----------|
| d, then y/n confirm | Footer confirmation | ✓ |
| d, instant + u undo | | |
| x, then y/n confirm | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Shows +, d unpins only | Row stays if auto-found; window applies again | ✓ |
| No +, d refuses | | |
| Auto-sweep the saved entry | Conflicts with never-auto-remove rule | |

| Option | Description | Selected |
|--------|-------------|----------|
| List only; footer note on auto rows | | ✓ |
| d also works in detail | | |
| Hide the d hint on auto rows | | |

## Unavailable rows

| Option | Description | Selected |
|--------|-------------|----------|
| Ref + warning glyph + reason in place of title | Warning colour, not faint | ✓ |
| Keep last known data + marker | Needs caching | |
| Faint row + reason | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Three reasons + SSO hint | not found / SSO / could not load | ✓ |
| Pass GitHub's message through | | |
| Single 'unavailable' + reason in detail | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Top bucket; o works, enter shows reason | | ✓ |
| Bottom; o works, enter does nothing | | |
| Top bucket; enter does nothing | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Split by cause | Whole-request failure keeps data; NOT_FOUND/SSO flips at once; other errors after 2 in a row | ✓ |
| Always flip immediately | | |
| Always keep last data | | |

## Manual PRs in the list

| Option | Description | Selected |
|--------|-------------|----------|
| Toggles apply to all rows | Unavailable rows always shown by state toggles | ✓ |
| Manual rows always shown | | |
| Toggles apply, unavailable hidden by # | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Header count only; empty state mentions a | `· N unavailable` when any | ✓ |
| Split count in header | | |
| No change | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Re-read before write + on refresh | No clobbering; other pane's changes appear within one interval | ✓ |
| Re-read before write only | | |
| Single instance assumed | | |

## Claude's Discretion

- Marker and unavailable glyphs, prompt implementation (extend vs textinput), manual-lookup query shape and chunk size, failure-counter storage, JSON schema details and corrupt-backup naming, flash timings, exact reason wording.

## Deferred Ideas

- Multi-ref paste; pin auto-found PRs on paste; undo for removal.

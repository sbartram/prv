# Phase 1: Status at a Glance - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-11
**Phase:** 01-status-at-a-glance
**Areas discussed:** Row look & badges, Phase 1 ordering, Launch & failure states, Authored vs assigned

---

## Row look & badges

| Option | Description | Selected |
|--------|-------------|----------|
| Glyph + short word | Colour on top; reads without colour | ✓ |
| Glyph only, colour-coded | Most compact; needs a legend | |
| Words only, colour-coded | Safest for fonts | |

| Option | Description | Selected |
|--------|-------------|----------|
| Status first | CI, then review, then state, then PR, then title | |
| Identity first | PR, title, author, state, review, CI | ✓ |
| You decide | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Login, dim when it's you | | |
| 'me' for yourself | Needs the viewer login | ✓ |
| Always plain login | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Dim whole row + 'draft' tag | | ✓ |
| Prefix in title only | | |
| Replace STATE with 'draft' | | |

| Option | Description | Selected |
|--------|-------------|----------|
| ANSI 16 only | Theme colours, no detection | ✓ |
| GitHub-like hex | Needs LightDark detection | |

**User's choice:** Glyph + word, identity-first columns, `me` for self, dimmed draft row + tag, ANSI 16.

---

## Phase 1 ordering

| Option | Description | Selected |
|--------|-------------|----------|
| State group, then recently updated | Open, then merged, then closed; updatedAt desc; repo#n tiebreak | ✓ |
| Recently updated only | Search order | |
| By repo, then number | Alphabetical | |

| Option | Description | Selected |
|--------|-------------|----------|
| Dim merged/closed, no divider | | ✓ |
| Section headers | | |
| No special treatment | | |

---

## Launch & failure states

| Option | Description | Selected |
|--------|-------------|----------|
| Alt screen + spinner | TUI starts at once | ✓ |
| Fetch before TUI starts | | |

| Option | Description | Selected |
|--------|-------------|----------|
| In-TUI error + 'r' to retry | Minimal `r` pulled into Phase 1 | ✓ |
| Error panel, quit only | | |
| Exit non-zero with message | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Friendly empty state naming @login | | ✓ |
| Empty table headers only | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Both remedies (gh auth login, or env vars) | | ✓ |
| gh-only remedy | | |

---

## Authored vs assigned

| Option | Description | Selected |
|--------|-------------|----------|
| Implicit via AUTHOR | No extra column | ✓ |
| Small marker column | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Draft shows review as normal | | ✓ |
| Blank review for drafts | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Follow rollup state | Matches github.com | ✓ |
| Failing wins, always | | |

---

## Claude's Discretion

- Exact glyphs and wording, spinner style, selection highlight, column widths, `--version` format, and how partial results appear in Phase 1.

## Deferred Ideas

None.

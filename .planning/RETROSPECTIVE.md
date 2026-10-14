# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — MVP

**Shipped:** 2026-10-14
**Phases:** 5 | **Plans:** 35 | **Sessions:** not tracked

### What Was Built
- A Go + Bubble Tea v2 TUI that lists the user's authored and assigned PRs with state, review, CI and merge-readiness badges from one batched GraphQL fetch (gh-compatible auth via go-gh)
- Long-running monitoring: single-flight auto-refresh, attention sort, rate-limit backoff from real response headers, filters, config.toml, help overlay, open-in-browser
- Blocker detail: failing check names in rows, MERGE and REVIEWERS columns, a full-screen detail of every check and reviewer
- A curated watch list (add by URL or `owner/repo#n`, remove, persist) with unavailable rows that never vanish silently
- Tech-debt close-out: flock-safe concurrent writes from several panes, crash-safe backups, version refusal, an honest save prompt, and an opt-in live smoke test against pinned public PRs

### What Worked
- Thin vertical slices: every phase ended with a working binary, so each phase's verification checked real behaviour, not scaffolding
- A live-verification spike at the start of Phase 1 pinned the GraphQL query before it was frozen
- Pure functions at the seams (`domain.NextDelay`, `DeriveMerge`, `layoutColumns`, `saveReason`) made the hard logic testable without a terminal or the network; golden files then pinned each screen
- Code review findings fed directly into a dedicated phase: Phase 4 WR-01 (unlocked read-modify-write) became Phase 5's flock work, proven with 8 real processes and a mutation run
- The milestone audit turned scattered review and UAT debt into one list that Phase 5 could plan against

### What Was Inefficient
- Live UAT was deferred phase after phase: phases 1–4 recorded 5 passes, 1 issue and 15 skips, and VERIFICATION.md was set to passed on those deferrals. Phase 5 had to carry the backlog, and some checks (live config edits) are still unrun after two deferrals
- Phases 2 and 3 grew to 10 and 11 plans, several of them gap-closure plans written after verification; tighter must-haves up front would have caught some of those earlier
- Verification reports for phases 1–4 went stale once later phases touched shared files, so v1.0 had to close as an override instead of a verified closeout
- Small process friction at the end: a skip with a reason was assumed to unblock completion, but the predicate accepts only a `Deferred follow-up:` reason

### Patterns Established
- Remote strings are cleaned once at the mapper seam (`domain.Clean`); errors reach the screen as fixed phrases, never `err.Error()` with paths
- Partial data is never shown as complete: separate flags (Truncated, Partial) each drive their own notice
- Watch-list store keeps no state: every write re-reads under an exclusive lock and writes via temp file + rename; reads stay lock-free
- Opt-in, env-gated tests (`PRV_LIVE=1`, `PRV_SHOW=1`) for anything that touches the network or needs a human eye; `go test ./...` stays offline

### Key Lessons
1. Run the live UAT at the end of each phase, or record it as an explicit `Deferred follow-up:` with a backlog entry; a plain deferral turns into milestone-end debt
2. Re-run the milestone audit after any closure phase, so the close can be a verified closeout instead of an override
3. When code review flags a concurrency or data-loss issue, give it its own phase with a test that fails without the fix (mutation run), not a quick patch
4. Keep verification reports fresh: re-verify earlier phases when a later phase edits the files they cover

### Cost Observations
- Model mix: `adaptive` profile (opus for planning and auditing agents, haiku for plan checking); exact split not tracked
- Sessions: not tracked
- Notable: 301 commits and ~21.2k lines of Go (~14.7k test, 350 test functions) in 4 days

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | not tracked | 5 | Added a tech-debt closure phase after the milestone audit; live UAT deferred until then |

### Cumulative Quality

| Milestone | Tests | Coverage | Zero-Dep Additions |
|-----------|-------|----------|-------------------|
| v1.0 | 350 test functions + golden files + opt-in live smoke | not measured | n/a |

### Top Lessons (Verified Across Milestones)

1. (Needs a second milestone to cross-validate.)

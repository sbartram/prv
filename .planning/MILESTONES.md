# Milestones

## v1.0 MVP (Shipped: 2026-10-14)

**Phases completed:** 5 phases, 35 plans, 71 tasks

**Delivered:** A single-binary Go + Bubble Tea v2 TUI that watches the user's authored and assigned PRs plus any PR added by URL, with state, review, CI and merge-readiness badges, a per-PR detail view, rate-limit-aware auto-refresh, and a watch list that two panes can write at once.

**Stats:** 301 commits, 295 files, ~21.2k lines of Go (~14.7k test) · 2026-10-11 → 2026-10-14 · tag v1.0

**Highlights:**
- Live status at a glance: one batched GraphQL fetch through go-gh, gh-compatible auth, ANSI-16 badges readable without colour (Phase 1)
- Long-running monitoring: single-flight refresh, attention sort, rate-limit backoff from real headers, filters, config.toml, help, open-in-browser (Phase 2)
- Blocker detail: failing check names in rows, MERGE and REVIEWERS columns, a full-screen detail of every check and reviewer (Phase 3)
- Curated watch list: add by URL or owner/repo#n, remove, persist, unavailable rows that never vanish silently (Phase 4)
- Tech-debt close-out: flock-safe concurrent writes, crash-safe backups, version refusal, honest save prompt, opt-in live smoke test (Phase 5)

**Closeout type:** override_closeout (chosen by the user on 2026-10-14)

**Known verification overrides:** 0 newly acknowledged, 0 carried forward from a prior close (open-artifact scan was clear). Closed as an override because:
- Phases 1–4 verification reports read `stale` at close; only Phase 5 was `passed`.
- The milestone audit (v1.0-MILESTONE-AUDIT.md, status `tech_debt`, 38/38 requirements) predates Phase 5 and was not re-run; Phase 5 closed Phase 4 WR-01/02/03/05 and 6 deferred live UAT checks.
- Live UAT still deferred: D-17 items (SSO, live rate-limit pause, assignee:@me, CJK glyphs, team reviews, large CI matrix, bad-token 401, theme/narrow pane) and ROADMAP backlog 999.1 (api footer legibility) and 999.2 (live config.toml edits, never run live).

**All accomplishments (one per plan):**

**Key accomplishments:**
- Go + Bubble Tea v2 `prv` that resolves a gh-style token, runs one live `ISSUE_ADVANCED` GraphQL search through go-gh, and lists each PR with state, a review badge derived from `reviewDecision` plus opinionated reviews, and a CI badge taken from GitHub's rollup state
- The PR list now holds every authored and assigned PR in the exact closed window once each, from up to five pages in a stable Open/Merged/Closed order, and launch, failure, empty and partial states tell the user what is happening and how to fix it without ever showing the token
- The PR list now reads at a glance: ANSI-16 green/red/yellow/faint badges that also read without colour, dimmed draft/merged/closed rows, a `>` plus reverse-video selection, cell-width column layout where only TITLE is cut with `…`, cursor-following scrolling, and 13 golden files locking every screen at 80x24, 120x40 and 0x0
- README with install, token precedence, read-only scopes, keys and badge legend, an automated release gate that passed in one run, and a user-confirmed `go install github.com/sbartram/prv/cmd/prv@latest` that installs a working `prv` from the pushed main
- prv re-fetches itself on a `refresh_interval` from config.toml (60 s default, 15 s floor) through one generation-stamped tick chain, and a failed refresh never hides the rows on screen.
- `r` refreshes from every screen without ever overlapping a running fetch, the header shows the clock time of the last successful update with a spinner while refreshing, and the cursor follows the selected PR by owner/repo#n instead of by row index.
- The list is sorted by need for attention (failing CI, changes requested, ready to merge, CI running, other open, open drafts, merged, closed, unknown) with createdAt, repo and integer number as tie-breaks, so the order stays put between refreshes.
- A failed refresh now keeps the rows and marks the header "✗ refresh failed HH:MM (reason) · data from HH:MM:SS" in ANSI yellow through a priority-based header composer, the error panel says when it retries, and the footer puts integrity notes before key hints taken from the keymap.
- Every fetch now reports the GraphQL rate limit, the footer shows a faint right-aligned "api N" from the first successful response on, and the header warns in ANSI yellow with the reset time when less than 20% is left.
- One pure `domain.NextDelay` decides every refresh delay (Retry-After, then reset time, then a bounded exponential, plus a reserve pause and pacing), and the model uses it after every result, shows "paused until HH:MM (rate limit)", and refuses `r` with a "rate limited — resumes HH:MM" flash while a pause is active.
- A header-capturing transport under go-gh and `KindRateLimited` with `Backoff()` recognise GitHub's HTTP-200 primary limit and its 403/429 secondary limit with their retry-after and reset times, so the 02-06 policy pauses prv on real responses.
- Keys 1/2/3 toggle Open/Merged/Closed with a "[open merged ·]" header indicator and "N of M PRs" count, and "/" opens a digits-only prompt that narrows the list by PR-number prefix, with a filtered-empty screen that says how many PRs the filters hide.
- config.toml now sets refresh_interval, closed_days (clamped 1..365) and show; a bad file stops prv with "prv: <path> line N: <problem>", the --closed-days flag beats the file inside the same bound, and unknown keys show as a bounded 8 s footer warning that never hides a flash.
- A keymap-driven help overlay (`?`) and a safe `o` that hands only exact https://github.com links to go-gh's browser from inside a tea.Cmd.
- Two-step GitHub fetch (lean discovery, then one read-only `FailingChecks nodes(ids:)` query per 25 failing PRs) with pure re-run de-duplication, so a failing row reads `✗ lint +2` or `✗ deploy (cancelled)`
- Failing-check names proven correct on superseded re-runs, truncated pages, partial GraphQL errors, 120-PR chunking, rate limits and cancel, plus a suffix-preserving CI cell cut that is safe for CJK and ZWJ emoji names.
- Pure layoutColumns drops AUTHOR, STATE and TITLE in the D-13 order, then compact `> web#7 ✗✗` rows below 40 cells, with a property test over 1500 sizes proving no overflow
- DeriveMerge (state and draft before status, UNKNOWN+CONFLICTING is conflicts) and MergeReviewers (pending wins, hidden teams, bots, detail-only states) fill PR.Merge and PR.Reviewers from an extended 2-point discovery query, proven through the real mapper
- MERGE and REVIEWERS columns in the PR list (`✗bob …carol ✓alice`, `⊘ blocked`), folding REVIEWERS to a REVS count before dropping it in the D-13 order, with 22-cell cap, hidden-team token and `+N` for anyone not shown
- `FetchDetail` returns a PR's de-duplicated, failing-first checks, every reviewer with their state (commented and dismissed included) and counts that never include superseded re-runs, paging up to 500 check runs and reporting how many were read of how many exist
- Enter opens a full-screen, scrollable detail of the selected PR from one FetchDetail call (checks failing-first with passing ones folded behind `e`, every reviewer with their state, `read N of M runs` on a partial read), and esc returns to the same PR
- The open detail view refetches after every successful list refresh (one Detail query, none when closed, in flight, paused or gone), says "no longer in list" and keeps its last data when its PR leaves the rows, shows a failure line on a failed refresh, extends the rate-limit pause on a rate-limited detail error, drops to glyph-and-name lines below 40 cells, and has goldens for all four new states
- FetchDetail now sets Truncated when a null check node leaves Read below Total or when a page's GraphQL error was kept, so the detail screen says "read N of M runs" instead of showing exact-looking counts on partial data.
- `domain.ServerWait` gives the detail view's rate-limit pause the list path's 1 s to 65 min clamp, and unknown checks are now listed, counted as "unknown", and kept out of the "more passing or skipped" fold line.
- `Detail.Partial` splits a kept GraphQL error from a short check-run read, so the detail view shows a faint "some details could not be loaded" line for the first and "read N of M runs" only for the second.
- Paste a github.com PR link after pressing `a`: prv checks it with one variables-only aliased GraphQL Lookup, saves the canonical ref to `watchlist.json` with an atomic temp-file rename, lists it with full status and a `+` marker, and every later refresh (and restart) merges the saved list with discovery by key and node ID.
- `lookupRefs` now groups GraphQL errors by alias (`errors[].path[0]`) and `unavailFor` turns each alias's own errors into one of three D-10 reasons: SSO (with the org SSO page and `gh auth refresh`), not found, or `could not load (SHORT)`. The add prompt and an unavailable row show the same text because both go through this path.
- Pasting a PR that is already listed moves the cursor to it (or says the filters hide it) and saves nothing; every add mistake is explained in the prompt line with the typed text kept; and an epoch on every fetch discards any refresh that began before an add, so a new row never disappears.
- watchlist.json now survives a corrupt, hand-edited, oversized or unreadable file and two prv panes: a bad file is copied once to `watchlist.json.corrupt-HASH8` (O_EXCL, SHA-256 name) and never rewritten by a read, entries cap at 500, writes go through a rename seam that cleans its temp file, and startup problems show once as `watch list: ...` warnings.
- A saved PR that GitHub cannot load now shows as a yellow "⚠ unavailable: REASON" row at the top of the list, a saved PR that closed long ago stays listed, and a one-off GitHub error keeps the last data until it repeats.
- Pressing d on a + row asks "remove REF from watch list? y/n"; only y removes it (through Watchlist.Remove in a Cmd), q and every other key cancel, and a PR that discovery also lists is unpinned instead of dropped.
- enter on an unavailable saved PR now shows why (reason, SSO hint, how to remove it) without asking GitHub again, a add and d remove appear in the footer, the empty screen and the ? overlay, and every new screen is pinned by golden files.
- Exclusive flock on watchlist.json.lock around every add and remove (proved by 8 real processes and a mutation run), a crash-safe hash-named backup, and a version-mismatch refusal that never overwrites a v0, v2 or version-less file.
- The add prompt now flashes the real backup name when an add replaced a broken watchlist.json, ignores esc and typing once a save has started (with a "saving REF" line and a live spinner), and gives every failed add or removal a fixed, path-free reason.
- An opt-in PRV_LIVE=1 test runs prv's real Fetch on 11 pinned public PRs and checks state, review, CI and MERGE (GitHub-side drift becomes a re-pin skip, a prv review-mapping bug still fails), plus a PRV_SHOW=1 render of the real low-budget warning and an 8-check 05-UAT.md.

---

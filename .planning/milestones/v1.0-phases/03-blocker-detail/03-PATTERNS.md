# Phase 3: Blocker Detail - Pattern Map

**Mapped:** 2026-10-12
**Files analyzed:** 21 (9 new, 12 modified, plus fixtures and goldens)
**Analogs found:** 21 / 21 (all analogs are git-tracked; verified through `git ls-files`)

Revised decisions applied: D-02 (+N from de-duplicated contexts, none when truncated), D-07 (UNKNOWN + `mergeable: CONFLICTING` gives conflicts), D-11 (separate sequenced detail Cmd with seq and ref guard), D-14 (0x0 means unsized, renders everything uncut like `list_0x0.golden`).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match |
|---|---|---|---|---|
| `internal/domain/pr.go` (M: PR gains Merge, Reviewers, FailingCheck) | model | transform | itself (`PR`, `Review`, `CI` enums, lines 50-99) | exact |
| `internal/domain/status.go` (M: Merge enum, DeriveMerge, Glyph methods) | utility | transform | `DeriveReview`, `ClassifyRollup`, `Badge()` (same file) | exact |
| `internal/domain/reviewers.go` (N: Reviewer, MergeReviewers) | utility | transform | `internal/domain/status.go` `DeriveReview` | role-match |
| `internal/domain/checks.go` (N: Check, ClassifyCheck, LatestChecks, SummarizeFailing, Detail) | utility | transform | `internal/domain/status.go` `ClassifyRollup`; `list.go` for sort style | role-match |
| `internal/domain/domain_test.go` (M, or new `merge_test.go`, `checks_test.go`) | test | table tests | `TestDeriveReview`, `TestBadge`, `TestClassifyRollup` | exact |
| `internal/github/client.go` (M: discoveryQuery, wireNode, mapResponse, Fetch step 2) | service | request-response | itself (lines 64-278) | exact |
| `internal/github/checks.go` (N: failingQuery, detailQuery, wire structs, FetchDetail) | service | request-response | `internal/github/client.go` `Fetch` loop and wire structs | exact |
| `internal/github/client_test.go` (M: route fixtureTransport by operation) | test | request-response | itself, `fixtureTransport` lines 22-60 | exact |
| `internal/github/testdata/*.json` (N: failing_contexts, detail_*, reviewers fixtures; M: status.json if needed) | fixture | n/a | `testdata/status.json`, `search_page1.json` | exact |
| `internal/ui/layout.go` (M: column table, layoutColumns, compact threshold) | utility | transform | itself (`columnWidths`, `truncateCell`) | exact |
| `internal/ui/cells.go` (N: reviewersCell, mergeCell, ciCell, compactRow) | utility | transform | `internal/ui/layout.go` `truncateCell`; `styles.go` `reviewStyle`/`ciStyle` | role-match |
| `internal/ui/styles.go` (M: mergeStyle, checkStyle, reviewerStyle) | utility | transform | `reviewStyle`, `ciStyle` | exact |
| `internal/ui/detail.go` (N: detailState, detailLines, detail update helpers, render) | component | event-driven | `internal/ui/help.go` `renderHelp` (full-screen) plus `model.go` selection fields | role-match |
| `internal/ui/keys.go` (M: Enter, Back, Expand, DetailShortHelp) | config | n/a | itself | exact |
| `internal/ui/model.go` (M: detail viewState/fields, Update branch, Fetcher/DetailFetcher) | controller | event-driven | itself (`showHelp` branch, `Open` case, `rebuild`) | exact |
| `internal/ui/refresh.go` (M: detailMsg, fetchDetail Cmd) | service | request-response | `fetch()` / `beginFetch()` in same file | exact |
| `internal/ui/view.go` (M: render() switch, renderList columns, footer for detail) | component | transform | itself (`render`, `renderList`, `fit`, `footer`) | exact |
| `internal/ui/help.go` (M: legend for merge/reviewer/check glyphs) | component | transform | itself (`Badges` block lines 27-44) | exact |
| `internal/ui/*_test.go` (N: detail_test.go, layout_test.go; M: view_test.go, selection_test.go) | test | table/golden | `view_test.go`, `refresh_test.go`, `selection_test.go` | exact |
| `internal/ui/testdata/TestGolden/*.golden` (regenerate; add 60x15 and detail screens) | fixture | golden | `list_80x24.golden` etc. | exact |
| `cmd/prv/main.go` (M only if the Fetcher wiring needs it; `*github.Client` already satisfies it) | config | n/a | itself | exact |

## Pattern Assignments

### `internal/domain/status.go` + `pr.go` (utility/model, transform)

**Analog:** `internal/domain/status.go` lines 11-84, and enum style in `pr.go` lines 50-71.

Enum style (pr.go 50-71): zero value is the least favourable, no `iota` skipping, doc comment says so.
```go
// CI is the check status shown for a pull request. The zero value is the
// non-favourable CINone ("no checks").
type CI int

const (
	CINone CI = iota
	CIPassing
	CIFailing
	CIRunning
	CIUnknown
)
```
Total-mapping function with a nil/empty guard (status.go 42-55):
```go
func ClassifyRollup(state *string) CI {
	if state == nil { return CINone }
	switch *state {
	case "SUCCESS": return CIPassing
	case "FAILURE", "ERROR": return CIFailing
	case "PENDING", "EXPECTED": return CIRunning
	}
	return CIUnknown
}
```
Badge style (status.go 59-84): glyph plus word, default branch last, string literal per case. Add `Merge.Badge()` (`✓ mergeable`, `⊘ blocked`, `↓ behind`, `✗ conflicts`, `–`, and `""` for MergeNone) and `Glyph()` methods; add a test that `Badge()` starts with `Glyph()` (RESEARCH Pitfall 11).

`DeriveMerge(st State, draft bool, status, mergeable string) Merge` (RESEARCH lines 223-234): check `st != StateOpen` first, then `draft`, then the status switch, then `UNKNOWN + CONFLICTING` gives conflicts (D-07 revised), else `MergeUnknown`.

New PR fields (next to `Review`, `CI` in `PR` struct, pr.go 85-99): `Merge Merge`, `Reviewers []Reviewer`, `Failing FailingCheck`.

**Fixture/test analog:** `domain_test.go` `TestBadge` (lines 99-124) uses a `map[Type]string` and `t.Errorf("X(%d).Badge() = %q, want %q", ...)`. Use the same shape for Merge and for a Glyph-prefix invariant (see also `TestDeriveReviewInvariant`, line 44).

---

### `internal/domain/reviewers.go`, `checks.go` (utility, transform)

**Analog:** `DeriveReview` (status.go 11-33): loop over remote strings, `switch`, early return for the dominant case. Same file style: pure, no I/O, no imports beyond stdlib. Strings arriving from the github seam are already `domain.Clean`ed (see mapper below); domain functions do not clean again.

Sorting analog: `internal/domain/list.go` `Sort` (uses `slices.SortStableFunc`; see `TestSort`, domain_test.go:179, for the table style). Use `slices.SortStableFunc` with `strings.ToLower` keys for reviewer order within a group (RESEARCH: no reshuffling between refreshes).

Use RESEARCH lines 485-502 (`ClassifyCheck`) and the table at lines 249-259 verbatim as the contract; `LatestChecks` key is (Workflow, Name), winner by largest (SuiteCreatedAt, ID) (F3).

---

### `internal/github/client.go` (service, request-response; modify)

**Analog:** itself.

**Query + wire struct pattern** (lines 64-84, 117-139): query is a `const` raw string; variables always passed, never concatenated. Extend the PullRequest selection with `mergeStateStatus mergeable reviewRequests(first: 20) {...}` and change `LatestOpinionatedReviews.Nodes` from `[]*struct{ State string }` to carry `Author *struct{ Login string }`. Keep the comment about opinionated-only (lines 132-134).

**Mapper pattern** (lines 230-278): nil-guard every pointer, `domain.Clean` every remote string at the seam, build `domain.PR` in one literal.
```go
author := "ghost"
if n.Author != nil {
	author = domain.Clean(n.Author.Login)
}
...
snap.PRs = append(snap.PRs, domain.PR{
	ID:      n.ID,
	Ref:     domain.Ref{Repo: domain.Clean(n.Repository.NameWithOwner), Number: n.Number},
	Title:   domain.Clean(n.Title),
	...
	Review:  domain.DeriveReview(decision, reviewStates),
	CI:      domain.ClassifyRollup(rollup),
```
Add `Merge: domain.DeriveMerge(domain.ParseState(n.State), n.IsDraft, n.MergeStateStatus, mergeable)` and `Reviewers: domain.MergeReviewers(...)`. `requestedReviewer` is `*struct{Typename string \`json:"__typename"\`; Login string; CombinedSlug string}`, nil means hidden team (F6).

**Step 2 insertion point:** after the discovery loop and `slices.DeleteFunc(... InWindow ...)` (lines 215-218) and before `domain.Sort`. Collect IDs where `pr.CI == domain.CIFailing`, chunk by 25, cap 100 total.

**Error policy to copy** (lines 157-174): rate-limit and context cancellation abort the whole fetch; other errors keep data and add a warning.
```go
err = classify(err, c.capture.latest())
var ce *Error
isErr := errors.As(err, &ce)
if isErr && ce.Kind == KindRateLimited {
	return domain.Snapshot{}, err
}
...
snap.Warnings = addWarning(snap.Warnings, "Only part of the list could be loaded")
```
**Rate-limit accounting** (lines 200-203): `snap.RateLimit.Cost +=`, Limit/Remaining/ResetAt from the last response.

---

### `internal/github/checks.go` (service, request-response; new)

**Analog:** `internal/github/client.go` (same package: `c.gql.DoWithContext(ctx, query, vars, &resp)`, `classify(err, c.capture.latest())`, wire structs with pointer nullables). The detail method is `func (c *Client) FetchDetail(ctx context.Context, id string) (domain.Detail, error)`, paging `contexts(first:100, after:)` with the same `maxPages = 5` cap (client.go 141-144). Wire `Nodes []*wireFailNode` (GraphQL can null an element; `nodes(ids:)` returns null for vanished PRs, skip nils). Queries are the verbatim texts in RESEARCH lines 424-446 and 451-480; document that the step-2 text has not been run as written.

---

### `internal/github/client_test.go` + `testdata` (test/fixture)

**Analog:** `fixtureTransport` (lines 22-60):
```go
func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(req.Body)
	f.bodies = append(f.bodies, raw)
	if f.calls >= len(f.files) {
		f.t.Fatalf("unexpected request %d; only %d fixtures", f.calls+1, len(f.files))
	}
```
It serves files strictly in order and fails on extras. `status.json` contains failing PRs, so every `Fetch` test now makes a step-2 request (Pitfall 9). Change the transport to route by operation name in the body (`query Watch`, `query FailingChecks`, `query Detail`), with per-operation file lists, and keep `newFixtureClient(t, files...)` working for the existing callers (default: an empty `nodes: []` step-2 response). `fetchStatus` (lines 62-75) is the helper to extend.

---

### `internal/ui/layout.go` + `cells.go` (utility, transform)

**Analog:** `layout.go` lines 8-65.

Replace the `iota` column block with the new set (`colMarker, colPR, colTitle, colAuthor, colState, colReview, colReviewers, colMerge, colCI, colCount`) and make `columnWidths`/`renderList` consume a visibility mask from a new pure `layoutColumns(cells, total)`. Keep the constants and measuring idiom:
```go
for _, row := range cells {
	for c := 0; c < colCount && c < len(row); c++ {
		ws[c] = max(ws[c], lipgloss.Width(row[c]))
	}
}
...
if total <= 0 {
	return ws // unsized: everything uncut (D-14 revised; matches list_0x0.golden)
}
```
Cell truncation uses the existing helper (lines 59-65):
```go
func truncateCell(s string, w int) string {
	if lipgloss.Width(s) > w {
		return ansi.Truncate(s, w, "…")
	}
	return s
}
```
Drop order D-13: shrink TITLE to `minTitleW`, then drop AUTHOR, STATE, REVIEWERS (via `+N` first), MERGE. Compact mode below about 40 cells. `chromeLines = 3` (line 27) must stay accurate if the compact header hint stays on the existing header line.

Note `table` StyleFunc uses `col < colCount-1` for gap padding (view.go:383); with dropped columns, base this on the last visible column instead.

---

### `internal/ui/styles.go` (utility, transform)

**Analog:** `reviewStyle` / `ciStyle` (lines 24-50): `switch` on enum, `lipgloss.Green/Red/Yellow` ANSI-16 only, default `faintStyle`. Add `mergeStyle(domain.Merge)`, per-check style where CANCELLED, NEUTRAL, SKIPPED are never red, and reviewer-state style. `TestStylesUseAnsi16` (view_test.go:171) enforces the palette.

---

### `internal/ui/detail.go` (component, event-driven; new)

**Analog (render):** `help.go` `renderHelp` (lines 15-65): full-screen string, fit to `m.h` by cutting and pinning the last hint line, wrap with `m.wrap`.
```go
lines := strings.Split(m.wrap(b.String()), "\n")
if m.h > 0 && len(lines) > m.h {
	lines = append(lines[:m.h-1], returnHint)
}
```
**Analog (header/footer):** `view.go` `fit(segs, sep, width)` (lines 227-257) and `header()` (261-304) for the detail header segments with priorities; `footer()` (135-191) for the hint line, which takes `m.keys.ShortHelp()` — add a `DetailShortHelp()` and select it by state. The `h := m.help // a copy: SetWidth must not change the model` idiom applies.

**State analog:** `model.go` lines 83-93 (`showHelp bool`, selected/rows/cursor fields). Detail state is a struct field `detail detailState` with its own `ref`, last-`pr` copy, `gone`, `data`, `err`, `expanded`, `vp viewport.Model`. It must never read `m.rows[m.cursor]`.

---

### `internal/ui/model.go` (controller, event-driven; modify)

**Analog:** itself.

**Key precedence** (lines 190-212): help branch first, then prompt, then the key switch. Add the detail branch after the prompt block and before the `switch`, so `esc`/`j`/`k`/`o`/`r` mean detail actions there.
```go
if m.showHelp {
	if msg.String() == "ctrl+c" { return m, tea.Quit }
	m.showHelp = false
	return m, nil
}
if m.prompting { ... }
```
**Quit/Open/Refresh cases to reuse** (lines 214-248): `bkey.Matches(msg, m.keys.Quit)`, the `safeURL` + `m.opener == nil` + `openCmd` sequence (use `m.detail.pr.URL`, not `m.rows[m.cursor].URL`), and the refresh guard (`m.inFlight`, `m.pausedUntil`, `m.gen++`, `m.beginFetch()`).

**`fetchedMsg` success branch:** after existing handling, when the detail is open: look up `ref` in `m.snap.PRs` to refresh `detail.pr` or set `gone`, and start a detail Cmd if none in flight. `rebuild()`/`nearest()` (lines 395-442) run unchanged; they already move `m.selected` to the nearest neighbour.

**Interfaces:** `Fetcher` is declared at lines 19-22. Add the optional `DetailFetcher` and type-assert it (`m.fetcher.(DetailFetcher)`), so `fakeFetcher` in tests keeps compiling. Rate-limit errors reuse the `backoffer` interface (lines 123-129) and `schedule` (375-389).

**viewState** (lines 47-54): do not add the detail to `viewState` if it must coexist with refresh state; a separate `detail.open` flag mirrors `showHelp` and keeps `render()` simple. Decision for the planner.

---

### `internal/ui/refresh.go` (service, request-response; modify)

**Analog:** `fetch()` and `beginFetch()` (lines 31-56). Copy the panic-wrapped Cmd closure that captures only ctx, fetcher and ids, never the Model:
```go
func fetch(ctx context.Context, f Fetcher, gen int) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = fetchedMsg{gen: gen, err: errors.New("unexpected internal error while loading PRs")}
			}
		}()
		snap, err := f.Fetch(ctx)
		return fetchedMsg{gen: gen, snap: snap, err: err}
	}
}
```
New `detailMsg{seq int; ref domain.Ref; data domain.Detail; err error}` and `fetchDetail(ctx, df, id, ref, seq)`. Update drops a `detailMsg` whose `seq != m.detailSeq` or `ref != m.detail.ref` (Pitfall 8), mirroring the generation check on `fetchedMsg`/`tickMsg` (lines 13-29).

---

### `internal/ui/view.go` (component, transform; modify)

**Analog:** itself.

`render()` (lines 26-42): add `if m.detail.open { return m.renderDetail() }` after the `showHelp` check.

`renderList()` (340-413): header cell row at line 343 and the row literal at 357-361 gain `REVIEWERS` and `MERGE`; CI cell uses the failing-name form. Reuse the viewer-as-`me` rule from lines 353-356 for reviewer logins:
```go
author := pr.Author
if strings.EqualFold(pr.Author, m.snap.Viewer) {
	author = "me"
}
```
The `StyleFunc` switch (394-399) gains `colMerge` and `colReviewers` cases; `dimmed(pr)` (styles.go 55-57) already faints drafts and merged/closed rows. Final `m.cut(l)` per line (409-411) stays.

---

### `internal/ui/keys.go` (config; modify)

**Analog:** itself, lines 10-67. Add `Enter`, `Back`, `Expand` bindings in `keyMap` and `defaultKeys()`; follow the `bkey.NewBinding(bkey.WithKeys(...), bkey.WithHelp(...))` form. `promptKeys` already binds `enter`/`esc` for the prompt (lines 31-37), so the prompt branch must keep precedence. Note `Clear` already binds `esc` (line 55), used only when `numFilter != ""` in list mode; the detail branch runs earlier. Add `DetailShortHelp()` beside `ShortHelp`/`FullHelp` (59-67) and add the new keys to `FullHelp` so `?` stays the single source (Phase 2 D-17).

---

### `internal/ui/help.go` (component; modify)

**Analog:** itself, lines 27-44: per-enum loop rendering `style(x).Render(x.Badge())`. Add `merge` and `reviewer` legend lines in the same loop shape.

---

### Tests: `internal/ui/*_test.go`

**Analog:** `view_test.go`.

Helpers to reuse: `newTestModel(t, snap, w, h)` (lines 25-32), `stateModel(t, state, w, h)` (226-251, add `"detail"`, `"list-failing"` states), `fixtureThirty`/`fixtureSix` (extend PRs with the new fields), `lines(s)` (268) and `press(m, keys...)`.
Golden loop (253-264): add sizes `{60, 15}` to `sizes := [][2]int{{80, 24}, {120, 40}, {0, 0}}`; name `state_WxH`; assertion `golden.RequireEqual(t, ansi.Strip(m.View().Content))`. Regenerate with `go test ./internal/ui -update` as its own reviewed task (Pitfall 10). `TestTitleTruncatesFirst` (line 270) asserts the exact header list and must be rewritten for the new columns.
Refresh/out-of-order tests: model on `TestStaleResultDropped` (refresh_test.go:264), `blockingFetcher` (350), and `result(m, snap, err)` (162) for `detailMsg` stale-sequence tests. Selection tests: model on `selection_test.go` for "refresh drops the PR while detail is open, then esc".

## Shared Patterns

### Sanitise remote strings at the github seam
**Source:** `internal/github/client.go` lines 232, 246, 264-265 and `internal/domain/clean.go` `Clean`.
**Apply to:** every check name, workflow name, login, team slug, context name in `checks.go` and the mapper. Domain and ui never re-clean; ui still truncates with `truncateCell`.

### Display-width, never byte or rune counts
**Source:** `internal/ui/layout.go` 37-65, `view.go` 243-256.
**Apply to:** all new cells (`lipgloss.Width`, `ansi.Truncate`), reviewer `+N` fill, detail lines.

### Glyph plus word, ANSI-16 only
**Source:** `status.go` `Badge()`, `styles.go` lines 11-50.
**Apply to:** Merge, per-check, and reviewer-state badges. CANCELLED, NEUTRAL, SKIPPED never red.

### Single-flight, generation guard, panic-wrapped Cmds, no I/O in Update/View
**Source:** `refresh.go` 13-56, `model.go` 98-105.
**Apply to:** the detail Cmd and any step-2 work (step 2 lives inside `Client.Fetch`, so it is already inside the single-flight Cmd).

### Selection by `domain.Ref`
**Source:** `model.go` `rebuild`/`nearest`/`indexOf` (395-442).
**Apply to:** detail open/close; the detail view keeps its own ref and PR copy and does not touch `m.cursor`.

### Rate-limit cost accounting and backoff
**Source:** `client.go` 200-203, `model.go` `backoffer` and `schedule` (123-129, 375-389).
**Apply to:** step-2 and detail queries (sum Cost, last Remaining; rate limit aborts).

### Partial-failure policy
**Source:** `client.go` 157-188.
**Apply to:** step 2: keep discovery data, rows stay unnamed, one `addWarning`. Never guess a name from a truncated contexts page.

## No Analog Found

| File/Concern | Role | Data Flow | Reason |
|---|---|---|---|
| `charm.land/bubbles/v2/viewport` use in `detail.go` | component | scroll | No viewport usage exists in the repo; follow RESEARCH Pattern 5 (`SetWidth/SetHeight/SetContentLines`, programmatic `ScrollDown/ScrollUp`, not its own KeyMap). |
| Compact-mode row (`> web#7 ✗⊘✗`) | utility | transform | No compact rendering exists; build from `Glyph()` methods plus `truncateCell`. |
| Chunked `nodes(ids:)` helper (`chunk[T any]`) | utility | batch | No batching helper exists; stdlib `slices` is enough (RESEARCH "Don't Hand-Roll"). |
| Check-context union decoding (`CheckRun`/`StatusContext`) | wire | transform | No union decoding yet; use one tagged struct with `Typename` and both field sets, as `wireNode.Typename` does for `search` nodes (client.go:117-118, 241). |

## Metadata

**Analog search scope:** `internal/domain`, `internal/github`, `internal/ui`, `cmd/prv` (all tracked; no gitignored mirrors involved)
**Files read:** status.go, pr.go, clean.go, client.go, layout.go, keys.go, refresh.go, help.go, view.go, model.go, styles.go, open.go, selected parts of client_test.go, view_test.go, domain_test.go
**Pattern extraction date:** 2026-10-12

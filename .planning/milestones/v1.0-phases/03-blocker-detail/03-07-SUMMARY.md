---
phase: 03-blocker-detail
plan: 07
subsystem: ui-detail-view
tags: [go, bubbletea-v2, detail-view, viewport, keymap, goldens, tdd]

requires:
  - phase: 03-blocker-detail
    provides: "plan 03-06 FetchDetail and domain.Detail (de-duplicated, sorted checks; every reviewer; exact counts; Truncated/Read/Total); plan 03-05 merge and reviewer badges in the list"
provides:
  - "Full-screen detail view: enter opens the selected PR, esc returns to the same PR, j/k and the arrows scroll a viewport, o/r/q/? as in the list"
  - "ui.DetailFetcher (optional interface; *github.Client implements it, asserted at compile time in cmd/prv)"
  - "CHECKS section: de-duplicated counts, one line per check (failing first), passing/skipped/unknown folded into '… +N more passing', e expands; partial reads say 'read N of M runs'"
  - "REVIEWERS section: glyph, name (me, @org/slug, hidden teams on one line) and review state"
  - "keyMap.DetailShortHelp(), Enter/Back/Scroll/Expand bindings, checkStyle and reviewerStyle (ANSI-16)"
  - "detail and detail-expanded goldens at 60x15, 80x24, 120x40 and 0x0; help overlay checks legend"
affects: [03-08]

plan_head_before: 7f92f4513572dccf774cb303b456aede0fffef41
plan_head_after: 4aa3e7c837dba0aaa0c76ae47c3ebf0b1bd71572

actuals:
  tokens: 18069
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Detail data has its own sequenced Cmd: detailSeq is bumped on every start and every close, and a result with a stale seq, a stale ref, or arriving after close is dropped without touching detailInFlight"
    - "Detail code never reads the list row at the cursor: Update passes the PR to openDetail, and the view keeps its own copy (grep gate in the plan verify)"
    - "The body scrolls in a bubbles viewport driven only by prv's keymap; its lines are right-trimmed; unsized (0x0) the view draws detailLines(0) uncut"
    - "openURL and manualRefresh helpers shared by the list and detail key handlers"

key-files:
  created:
    - internal/ui/detail.go
    - internal/ui/detail_test.go
    - internal/ui/testdata/TestGolden/detail_0x0.golden
    - internal/ui/testdata/TestGolden/detail_60x15.golden
    - internal/ui/testdata/TestGolden/detail_80x24.golden
    - internal/ui/testdata/TestGolden/detail_120x40.golden
    - internal/ui/testdata/TestGolden/detail-expanded_0x0.golden
    - internal/ui/testdata/TestGolden/detail-expanded_60x15.golden
    - internal/ui/testdata/TestGolden/detail-expanded_80x24.golden
    - internal/ui/testdata/TestGolden/detail-expanded_120x40.golden
  modified:
    - internal/ui/model.go
    - internal/ui/refresh.go
    - internal/ui/keys.go
    - internal/ui/view.go
    - internal/ui/styles.go
    - internal/ui/help.go
    - internal/ui/help_test.go
    - internal/ui/view_test.go
    - cmd/prv/main.go
    - "internal/ui/testdata/**/*.golden (footer hint line of list/empty/filtered/compact; help overlay)"

key-decisions:
  - "openDetail takes the PR as an argument and Update passes the list row: the plan's grep gate forbids the cursor-row expression anywhere in detail.go"
  - "r in the detail also restarts the detail fetch when there is an error and no data, so the CHECKS line's '· r retry' is true; a list refresh alone would not refetch until plan 03-08"
  - "The footer drops whole hints from the end when bubbles help would let an item overflow the pane, instead of relying on the hard cut"
  - "A long CHECKS summary or reviewer line is cut by display cells (as the plan says), not wrapped"

patterns-established:
  - "TDD stubs: the RED commit carries type-only stubs (DetailFetcher, detailMsg, detailState, Model fields) so tests compile and fail on assertions"
  - "Golden-state helper stateModel gains 'detail' and 'detail-expanded' built from a detailFake and bigDetail()"

requirements-completed: [DETL-01, DETL-02, DETL-03]

coverage:
  - id: D1
    description: "enter on a list row opens a full-screen detail (header 'prv · OWNER/REPO#N · updated HH:MM:SS', title with 'STATE · REVIEW · MERGE', CHECKS, REVIEWERS) from one FetchDetail call; esc returns to the list with the same PR selected and on screen"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailOpenClose"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailRender"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every check is listed failing first with its word and source; the CHECKS summary shows the de-duplicated counts; passing, skipped and unknown checks fold into '… +N more passing' and e expands or collapses them; a partial read ends with 'read N of M runs'"
    requirement: DETL-01
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailFold"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailTruncatedSummary"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every reviewer is listed with a glyph, name (me, @org/slug, hidden teams on one line) and review state; empty states read 'CHECKS  no checks', 'REVIEWERS  none', 'loading…', 'details not available'"
    requirement: DETL-02
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailRender"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailEmpty"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailWithoutDetailFetcher"
        status: pass
    human_judgment: false
  - id: D4
    description: "Keys in the detail: j/k/arrows scroll without moving the list cursor; q and ctrl+c quit; ? opens the help and any key returns; o opens only the detail PR's URL through safeURL; r refreshes; 1, 2, 3, / and enter do nothing; footer hints come from DetailShortHelp"
    requirement: DETL-03
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailKeys"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailFooterHints"
        status: pass
    human_judgment: false
  - id: D5
    description: "Stale, closed and panicking detail fetches are handled: a result with an old seq or another ref or after close is dropped; a panic becomes the fixed message and never reaches the screen"
    verification:
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailStaleDropped"
        status: pass
      - kind: unit
        ref: "internal/ui/detail_test.go#TestDetailPanicRecovered"
        status: pass
    human_judgment: false
  - id: D6
    description: "The detail and detail-expanded views fit 60x15, 80x24 and 120x40, do not panic unsized, and have golden files at all four sizes; the help overlay lists enter, esc, j/k, e and the check-glyph legend"
    verification:
      - kind: unit
        ref: "internal/ui/view_test.go#TestGolden"
        status: pass
      - kind: unit
        ref: "internal/ui/view_test.go#TestViewFitsScreen"
        status: pass
      - kind: unit
        ref: "internal/ui/help_test.go#TestHelpContent"
        status: pass
    human_judgment: false
  - id: D7
    description: "In a real terminal, on a PR with a large CI matrix: enter opens the detail, failing checks are on top, passing ones fold into '… +N more passing', e expands and collapses, j/k scroll, esc returns to the same PR; the Detail query as written (03-06 D5) returns the shapes the fixtures model"
    requirement: DETL-01
    verification: []
    human_judgment: true
    rationale: "The plan's own human-check: layout, readability and a live query against a real PR cannot be asserted by the golden tests; the verifier harvests it at end of phase"

duration: 30min
completed: 2026-10-12
status: complete
---

# Phase 3 Plan 07: Selecting a PR opens a detail view of every check and every reviewer Summary

**Enter opens a full-screen, scrollable detail of the selected PR from one FetchDetail call (checks failing-first with passing ones folded behind `e`, every reviewer with their state, `read N of M runs` on a partial read), and esc returns to the same PR**

## Performance

- **Duration:** about 30 min (approximate; the start time was not captured)
- **Started:** 2026-10-12T23:12Z (approximate)
- **Completed:** 2026-10-12T23:42Z
- **Tasks:** 2 (each as a RED and a GREEN commit)
- **Files modified:** 35 (9 Go files plus 2 new, 22 golden files of which 8 are new, plus planning files)

## Accomplishments

- `internal/ui/detail.go`: `detailState`, `openDetail`/`startDetailFetch`/`closeDetail`, `detailKey`, `applyDetail` (drops results with a stale seq or ref, after close, or canceled), `syncDetail` (viewport sized to the pane minus header and footer), `detailLines` (title and status side by side or on two lines, CHECKS, REVIEWERS), `renderDetail` (handles heights 1 and 2 and the unsized 0x0 case uncut).
- The CHECKS section keeps the de-duplicated counts at every size; only failing, cancelled and running checks are always listed; the rest fold into one faint `… +N more passing` line until `e`; past the page cap the summary ends with `read N of M runs` and never switches to GitHub's per-state counts.
- The REVIEWERS section shows the viewer as `me`, a team as `@org/slug`, and all hidden teams as one `… N teams (names hidden)  pending` line; commented and dismissed reviewers are faint.
- `ui.DetailFetcher` is optional (type-asserted on the Fetcher); `*github.Client` is asserted to implement it in `cmd/prv/main.go`. A plain Fetcher shows `CHECKS  details not available`.
- Keys: `Enter`, `Back`, `Scroll`, `Expand`; `DetailShortHelp()` drives the detail footer; the `?` overlay gained the enter/esc/j-k/e column and a checks legend line. `openURL` and `manualRefresh` are shared by the list and the detail.
- Goldens: detail and detail-expanded at 60x15, 80x24, 120x40 and 0x0; the list/empty/filtered/compact footers and the help overlay gained the new hints.

## Task Commits

Each task was committed atomically as a RED and a GREEN commit:

1. **Task 1: enter opens a full-screen detail of the selected PR with every check, failing first, and every reviewer; esc returns to the same PR**
   - RED `a32e8b1` (test)
   - GREEN `2f17af1` (feat)
2. **Task 2: Big check matrices stay readable — passing checks fold, e expands, and a partial read says how many runs it covers**
   - RED `20a545d` (test)
   - GREEN `4aa3e7c` (feat)

**Plan metadata:** recorded in the docs commit that follows this file (docs: complete plan)

## TDD Gate Compliance

The plan type is `execute` with `tdd="true"` tasks, so there is no plan-level gate; each task followed RED then GREEN, with no REFACTOR commit needed.

- **RED (Task 1, `a32e8b1`):** seven new `TestDetail*` tests. The commit carries type-only stubs (`DetailFetcher`, `detailMsg`, `detailState`, the Model fields) so the package compiles. Six of the seven fail on the planned assertions ("enter did not open the detail view", "after j offset = 0, want 1", "the detail should be back", "1 changed the view"); `TestDetailEnterOnlyOnRows` is a negative test and passes by design. Semantic assessment: every failure names the planned behaviour, none is a compile, load or fixture fault.
- **RED (Task 2, `20a545d`):** `TestDetailFold`, `TestDetailTruncatedSummary`, `TestDetailFooterHints` and the new checks-legend subtest of `TestHelpContent` fail on the planned assertions ("collapsed view lists pass-01", "footer missing 'e expand'", "missing the read segment", "checks legend missing ✗ failing"). The golden states had no files yet.
- `gsd_run check tdd-red-evidence` supports TAP, JUnit, swift-testing and unittest reports; plain `go test` output is none of these, so the classifier was not run. The semantic assessment above was done by reading the failure text of each run.
- **GREEN:** `2f17af1` and `4aa3e7c`; `go test ./... -count=1 -race` passes.

## Files Created/Modified

- `internal/ui/detail.go` - detail state, key handling, line building, rendering
- `internal/ui/detail_test.go` - detailFake, fixtureDetail, bigDetail and ten detail tests
- `internal/ui/model.go` - DetailFetcher, Model fields, Update routing (detail keys after the help and prompt branches), Enter case, `openURL` and `manualRefresh` helpers
- `internal/ui/refresh.go` - `detailMsg`, panic-safe `fetchDetail`
- `internal/ui/keys.go` - Enter/Back/Scroll/Expand, `DetailShortHelp`, FullHelp column
- `internal/ui/view.go` - detail branch in `render`, `statusSegments` split out of `header`, footer hints per view, footer hint overflow fix
- `internal/ui/styles.go` - `checkStyle`, `reviewerStyle` (ANSI-16 only)
- `internal/ui/help.go` - checks legend line
- `internal/ui/help_test.go`, `internal/ui/view_test.go` - legend/expand expectations; `detail` and `detail-expanded` states in TestGolden and TestViewFitsScreen
- `cmd/prv/main.go` - `var _ ui.DetailFetcher = (*github.Client)(nil)`
- `internal/ui/testdata/**` - regenerated and new goldens

## Decisions Made

- `openDetail(pr)` receives the PR from `Update`, so `detail.go` never contains the cursor-row expression that the plan's grep gate forbids; the view keeps its own copy of the PR and ref.
- `r` in the detail restarts the detail fetch when there is an error and no data (see deviation 2).
- Footer hints are dropped whole from the end when they do not fit (see deviation 1).
- Over-long CHECKS summary and reviewer lines are cut by display cells, as the plan specifies; at 60 cells the summary can end mid-segment (see golden `detail_60x15`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Footer hints could overflow the pane**
- **Found during:** Task 1 (adding the `enter details` hint)
- **Issue:** `help.ShortHelpView` adds an item even when it overflows if its ellipsis does not fit either. The footer was exactly 120 cells in `TestFooterNotesFirst` before; one more hint made it 136 cells, and `TestViewFitsScreen/empty_80x24` and `TestApiFooter/80` failed (the api segment was pushed off the line).
- **Fix:** after rendering the hints, drop whole hints from the end until the line fits the room, then cut with an ellipsis as a last resort.
- **Files modified:** internal/ui/view.go
- **Verification:** `TestFooterNotesFirst`, `TestApiFooter`, `TestViewFitsScreen`, all footer goldens
- **Committed in:** `2f17af1`

**2. [Rule 2 - Missing critical] "r retry" needed a retry**
- **Found during:** Task 1 (CHECKS error line)
- **Issue:** the plan's error line promises `· r retry`, but `r` only refreshed the list; until plan 03-08 nothing would refetch the detail.
- **Fix:** in the detail, `r` also calls `startDetailFetch` when `detail.data == nil && detail.err != nil`; `detailInFlight` still guards single-flight. Covered by a `TestDetailPanicRecovered` subtest.
- **Files modified:** internal/ui/detail.go, internal/ui/detail_test.go
- **Committed in:** `2f17af1`

**3. [Plan deviation, minor] `openDetail` signature**
- The plan describes `openDetail()` copying the PR at the cursor. Its grep gate (`! grep -nF 'm.rows[m.cursor]' internal/ui/detail.go`) forbids that expression in detail.go, so `openDetail(pr domain.PR)` takes the PR and `Update` passes it.

**4. [Plan deviation, minor] Test expressions**
- `TestDetailRender` presses `e` first (Task 2's fold would hide the passing checks the Task 1 test lists).
- `TestHelpContent` matches the `e expand` binding with a regexp, because the key table pads each key to its column (`e     expand`); the footer shows the literal `e expand`.
- The help legend line is aligned with the other legend lines (`  checks    `) instead of the plan's two spaces.

---

**Total deviations:** 4 (1 bug fix, 1 missing-critical addition, 2 minor plan variations)
**Impact on plan:** the two fixes were needed for correctness (a footer wider than its pane; a retry hint that did nothing). No scope creep.

## Issues Encountered

- At 80 cells the list footer now ends at `… 1/2/3 states   api 4812`: with `enter details` added, `? help` and `q quit` no longer fit next to the api budget and are dropped from the end (the plan's "where it fits"). Both stay visible at 100 cells and wider, and `q`/`?` still work. If this matters, a later change could keep Help and Quit and drop Filter or States first.
- At 60 cells the detail's CHECKS summary is cut mid-segment (`… 10 passing · 2`), because lines are cut, not wrapped.
- A first attempt in this run to create a branch was refused by the permission system and was not retried; all work stayed on `gsd/phase-01-status-at-a-glance`.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. (`CHECKS  details not available` is the designed state for a Fetcher without `FetchDetail`; `*github.Client` implements it.)

## Threat Flags

None. The only outbound action is `o`, which opens `detail.pr.URL` through `safeURL` (T-03-13); no module was added (`git diff --exit-code go.mod go.sum` is clean).

## Next Phase Readiness

- Ready for 03-08: the detail has `detailSeq`/`detailInFlight`, `startDetailFetch` and `applyDetail`, so a refetch after each successful list refresh (D-11) and a "gone" flag for a PR that leaves the list (D-12) fit into `detailState` without new plumbing. Today a list refresh while the detail is open does not refetch the detail.
- The plan's human-check (large CI matrix in a real terminal) and a live run of the Detail query (03-06 D5) are still open for end-of-phase verification.

## Self-Check: PASSED

- Files: `internal/ui/detail.go`, `internal/ui/detail_test.go`, the eight new `detail*_*.golden` files exist.
- Commits `a32e8b1`, `2f17af1`, `20a545d`, `4aa3e7c` are ancestors of HEAD.
- `go vet ./...` and `go test ./... -count=1 -race` pass; `git diff --exit-code go.mod go.sum` is clean; golden hygiene check prints nothing; `! grep -nF 'm.rows[m.cursor]' internal/ui/detail.go` and the ANSI-16 grep pass.

---
*Phase: 03-blocker-detail*
*Completed: 2026-10-12*

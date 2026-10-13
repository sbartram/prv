# Phase 4: Curated Watch List - Pattern Map

**Mapped:** 2026-10-12
**Files analyzed:** 16 (new/modified)
**Analogs found:** 16 / 16 (all analogs are git-tracked; `internal/ui/view_test.go` confirmed tracked)

All paths are relative to repo root `/Users/scottb/dev/bartram/prv`. Excerpts below were read from the current tree.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match |
|---|---|---|---|---|
| `internal/watchlist/watchlist.go` (new: Path, Load, Add, Remove, backup) | store/utility | file-I/O | `internal/config/config.go` (Path, Load) | role-match |
| `internal/watchlist/watchlist_test.go` (new) | test | file-I/O | `internal/config/config_test.go` (t.TempDir, t.Setenv XDG) | exact |
| `internal/domain/ref.go` (new: ParseRef, Normalize) | utility | transform | `internal/domain/pr.go` `Ref`; regexp style in `internal/config/config.go` `wrongType` | partial |
| `internal/domain/pr.go` (mod: `Manual`, unavailable reason) | model | transform | itself (`PR`, `Snapshot`) | exact |
| `internal/domain/list.go` (mod: InWindow exemption, Bucket 0, Visible) | utility | transform | itself | exact |
| `internal/github/lookup.go` (new: aliased `repository{pullRequest}` query, chunks 25) | service | batch request-response | `internal/github/checks.go` `fetchFailing` | exact |
| `internal/github/errors.go` (mod: per-alias reasons) | utility | transform | itself (`classify`, `Short`, `Hint`) | exact |
| `internal/github/client.go` (mod: `Fetch` merges manual refs, dedupe by ID) | service | request-response | itself (`Fetch`, `mapResponse`) | exact |
| `internal/github/testdata/lookup_*.json` (new) | fixture | n/a | `testdata/failing_partial.json` | exact |
| `internal/github/lookup_test.go` (new) | test | request-response | `internal/github/checks_test.go` (`manyTransport`, `opName`, `rtFunc`) | exact |
| `internal/ui/keys.go` (mod: `Add`, `Remove`, `addPromptKeys`) | config | n/a | itself (`keyMap`, `promptKeys`) | exact |
| `internal/ui/model.go` (mod: add prompt, confirm, flash, merge) | component | event-driven | itself (`promptKey`, `prompting`, `rebuild`) | exact |
| `internal/ui/refresh.go` (mod: add-validation Cmd, watchlist re-read Cmd) | component | event-driven | itself (`fetch`, `fetchDetail`) | exact |
| `internal/ui/view.go` (mod: add prompt line, marker, header/footer) | component | transform | itself (`bottom`, `compactRow`, `statusSegments`) | exact |
| `internal/ui/detail.go` (mod: reason-only detail) | component | transform | itself | exact |
| `cmd/prv/main.go` (mod: open store, pass in `ui.Options`) | config | request-response | itself (`config.Load` + `ui.New`) | exact |
| `internal/ui/*_test.go` + `testdata/TestGolden` (mod/new) | test | transform | `internal/ui/view_test.go`, `help_test.go` (golden) | exact |

## Pattern Assignments

### `internal/watchlist/watchlist.go` (store, file-I/O)

**Analog:** `internal/config/config.go`

**Path** (lines 51-62): mirror with `XDG_DATA_HOME` and `.local/share`:
```go
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" && filepath.IsAbs(dir) {
		return filepath.Join(dir, "prv", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "prv", "config.toml"), nil
}
```
**Tolerant read** (Load, lines ~113-135): size-capped, missing file is not an error:
```go
f, err := os.Open(path)
if errors.Is(err, fs.ErrNotExist) { return cfg, nil }
...
data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
if len(data) > maxFileSize { return Config{}, fmt.Errorf("%s: file is larger than 1 MiB", path) }
```
Load warnings: config appends to `cfg.Warnings` and passes them through `ui.Options.Warnings` (one-time TUI warning, D-15). Do the same for "watchlist.json was corrupt, backed up to X" (return a warning string, not an error). Wrap decode errors through `domain.Clean`.
**No in-repo analog for atomic write** (temp file in same dir + `os.Rename`, 0o600/0o700 perms), and none for corrupt-file backup naming. Use CLAUDE.md guidance. Add/Remove must each re-load, apply one change, write (D-16). Store does no TUI work; UI calls it only inside `tea.Cmd`s.
Header convention: package doc comment says what it never does (config: "never writes the file and prints nothing").

### `internal/watchlist/watchlist_test.go`
**Analog:** `internal/config/config_test.go`: `t.TempDir()`, `t.Setenv("XDG_DATA_HOME", ...)` (lines ~16-42), `Load(filepath.Join(t.TempDir(), "nope", ...))` for the missing-file case.

### `internal/domain/ref.go` (ParseRef)
**Analog:** `internal/domain/pr.go` lines 78-86 define `Ref{Repo string; Number int}` and `String()`. Parser returns `Ref` plus a specific-reason error (D-04 strings). One anchored regexp per form (CLAUDE.md); precompile with `regexp.MustCompile` at package level as `wrongType` does in `internal/config/config.go`. Normalise with `strings.ToLower` on `Repo` for matching (D-03) while keeping API-returned casing for display. Pure, no I/O.

### `internal/domain/pr.go` (mod)
Add `Manual bool` to `PR` (struct at lines ~90-110) and an unavailable notion. Recommended: `Unavailable string` (reason) on `PR` so `Ref`-keyed cursor, `rebuild()`, `Sort`, `Visible` and rendering stay uniform (D-14/D-09). Keep the zero value safe (a zero `PR` stays available/auto). Unavailable rows have no `ID`, so dedupe by `ID` must skip empty IDs.

### `internal/domain/list.go` (mod)
- `InWindow(pr, now, closedDays)` (lines 12-21): add `pr.Manual` early return true (WATCH-07), before the `ClosedAt.IsZero()` drop.
- `Bucket` (lines 44-65): add first case `if pr.Unavailable != "" { return 0 }` and update the doc table (comment lists buckets 1-9).
- `Sort` (lines 70-81): inside bucket 0 order by ref, not `CreatedAt` (unavailable rows have zero CreatedAt, so the existing keys already fall through to Repo, Number; verify with a test).
- `Visible` (lines 100-118): add unavailable bypass of the state filter, mirroring the existing `StateUnknown` bypass:
```go
if pr.State != StateUnknown && !show.Has(pr.State) { continue }
if number != "" && !strings.HasPrefix(strconv.Itoa(pr.Ref.Number), number) { continue }
```
Change the first test to `pr.Unavailable == "" && pr.State != StateUnknown && ...`. Number filter still applies (D-13).

### `internal/github/lookup.go` (batched aliased lookup)
**Analog:** `internal/github/checks.go` `fetchFailing` (lines 112-195) and `failingQuery`/`failingChunk` (lines 16-45). Copy:
- Query as a `const` document, ids/values only as variables: `c.gql.DoWithContext(ctx, failingQuery, map[string]any{"ids": ids}, &resp)`. For aliases build the document text from alias NAMES only (`r0`, `r1`, ...) and pass owner, name, number as variables `$o0:String!, $n0:String!, $p0:Int!`. Never interpolate user text.
- Chunk loop: `for start := 0; start < len(x); start += chunk { chunk := x[start:min(start+chunk, len(x))] ... }` with `const failingChunk = 25` style constant.
- Error policy block (lines 138-160): classify, cancel and `KindRateLimited` return whole-fetch error; otherwise keep partial data:
```go
err = classify(err, c.capture.latest())
if errors.Is(err, context.Canceled) { return err }
var ce *Error
isErr := errors.As(err, &ce)
if isErr && ce.Kind == KindRateLimited { return err }
var ge *api.GraphQLError
if !isErr || ce.Kind != KindGraphQL || !errors.As(err, &ge) { ... }
```
  Here the partial `*api.GraphQLError` is the normal path: read `ge.Errors[i].Path[0]` (alias) and `.Type` (NOT_FOUND, FORBIDDEN) and map to the alias (spike Q3; SAML shape unobserved, use a constructed fixture). Network/5xx/401 (non-GraphQL) fail the whole request -> D-12 `refresh failed` marker, no row turns unavailable.
- Rate limit accounting (lines 162-168): `snap.RateLimit.Cost += resp.RateLimit.Cost`, and set Limit/Remaining/ResetAt when `Limit > 0`. Include `rateLimit { limit cost remaining resetAt }` in the query.
- Wire structs: pointer for every nullable, `json:"__typename"` tags, as `wireFailResp`/`wireFailNode`.
- Reuse the discovery PR field selection and `mapResponse`-style mapping (client.go lines 299-350) so manual PRs map identically; clean all strings with `domain.Clean` at the seam. Reuse `FetchDetail`-style single-ref path for add-validation (a `Lookup(ctx, ref)` that calls the same chunk function with one ref).

### `internal/github/errors.go` (mod)
Extend, do not replace. Existing: `ErrorKind` iota (lines 14-24), `Short()` fixed strings per kind (lines 62-80), `classify` GraphQL branch (lines 128-148) takes `ge.Errors[0].Message` via `domain.Clean`. Add a small pure function mapping `api.GraphQLErrorItem.Type`/message to the D-10 reason (`not found (deleted, renamed, or no access)`, `SSO: authorize your token for <org>`, `could not load (<short>)`) and keep reason text built only from fixed strings plus cleaned API text (the `Error` doc comment forbids headers/tokens). Note `rateLimitedGraphQL` (lines 152-162) must still win before per-alias mapping.

### `internal/github/client.go` (mod `Fetch`)
Lines 214-290. Insert manual lookup after discovery and before `InWindow` filtering; the window filter at lines 276-278 must skip `pr.Manual` (now in `domain.InWindow`, so it just works). Dedupe uses the existing `seen := map[string]bool{}` by `pr.ID` (lines 217-218, 262-267); a manual PR already seen is not re-added (D-03: auto wins, but a saved ref still counts as manual for the marker and `d`: carry `Manual` on the existing PR instead of dropping it). Saved refs must reach `Client`: new `Options` field or a `Fetch(ctx, refs)` signature change (planner's choice; `ui.Fetcher` interface at `internal/ui/model.go` line ~20 currently is `Fetch(ctx) (domain.Snapshot, error)`; the "re-read file each tick" requirement favors the UI passing refs per fetch, since the watchlist re-read must happen on each tick).
Failing-check step `fetchFailing(ctx, &snap)` operates on `snap.PRs` where `CI == CIFailing`: unavailable rows must not break it (empty ID).

### `internal/github/lookup_test.go` + fixtures
**Analog:** `internal/github/checks_test.go`: `rtFunc`, `jsonResp(req, 200, body)`, `opName(raw)`, `readBody`, `manyTransport` switching on operation name (lines 177-200), `newFixtureClient(t, "status.json")`. Fixtures in `internal/github/testdata/*.json` (`failing_partial.json` is the partial-data-with-errors shape). Add: not_found, saml_forbidden (constructed), other error, partial data, and a chunk-size test mirroring `TestFailingChunks`. Assert variables are used (no owner/name text in the query document).

### `internal/ui/keys.go` (mod)
**Analog:** itself. Add `Add, Remove bkey.Binding` to `keyMap` (lines 11-31) and to `defaultKeys()` in the same style:
```go
Filter: bkey.NewBinding(bkey.WithKeys("/"), bkey.WithHelp("/", "PR #")),
```
Add to `ShortHelp()` (line 71) and `FullHelp()` (line 81); NOT to `DetailShortHelp` (a/d do not work in detail, D-08). The add prompt keys follow `promptKeys`/`defaultPromptKeys()` (lines 33-45): Accept enter, Cancel esc, Delete backspace; reuse them rather than defining a second set, unless textinput is chosen. If `bubbles/textinput` is chosen, still keep enter/esc routed through this keymap.
Help legend for the marker/unavailable glyphs lives in `internal/ui/help.go` (not read; planner to open it).

### `internal/ui/model.go` (mod)
**Analog:** itself.
- Prompt state: fields `numFilter`, `prompting`, `pkeys` (lines 83-89). Add parallel `adding bool`, `addText string`, `addErr string`, `addChecking bool`, `addSeq int`, plus `confirmRemove domain.Ref`. Late results are dropped by sequence number exactly like `detailSeq` ("bumped for every start and every close, so a result that arrives late is dropped", lines 120-125).
- Update dispatch order (lines 194-240): `tea.PasteMsg` case currently digit-filters only when `m.prompting` (lines 196-204). Add the add-prompt branch accepting free text (trim, strip newlines, cap length). KeyPress order: help overlay, then prompt (`if m.prompting { m.promptKey(msg); if msg.String()=="ctrl+c" {return m, tea.Quit}; return m, nil }`), then detail. Insert add-prompt and confirm-remove handling at the same level so `q` does not quit while typing; `ctrl+c` still quits. The `d` confirm: any key other than `y` cancels.
- List-only guard pattern (line 240): `if m.state == stateList { m.prompting = true }`; use for `a`. `a`/`d` are placed before the `m.detail.open` check's list switch so they never fire in detail (`m.detail.open` returns `m.detailKey(msg)` first at line 220).
- Footer messages: `cmd := m.setFlash("could not open the browser", flashFor); return m, cmd` (line 298 pattern).
- `rebuild()` (lines 442-465) and `nearest`/`indexOf` (468-489) already handle cursor-by-ref and removal; to jump to a new/existing row set `m.selected = ref` then `m.rebuild()`; if hidden by filters, flash the `· hidden by filters` suffix (D-03).
- 2-in-a-row counter (D-12): store `map[domain.Ref]int` on Model, or compute in the github layer via last-good data passed in; Model owns `m.snap` so merging "keep last data on first other-error" belongs where both old and new snapshots are visible (the `fetchedMsg` handler).
- `Options` (lines 36-47): add watchlist store/path, or an interface (`ui` declares its own interfaces and never imports `internal/github`; follow that, e.g. `type Watchlist interface{ Load()...; Add(ref)...; Remove(ref)... }`).
- `schedule(err)` (lines 418-436): per-tick watchlist re-read is a separate Cmd batched with `fetch`, not inside `schedule`.

### `internal/ui/refresh.go` (mod)
**Analog:** itself. New Cmds copy `fetchDetail` (lines 53-66) exactly: closure captures only args, panic-wrapped with a fixed message, result msg stamped with seq and ref:
```go
func fetchDetail(ctx context.Context, f DetailFetcher, id string, ref domain.Ref, seq int) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = detailMsg{seq: seq, ref: ref, err: errors.New("unexpected internal error while loading details")}
			}
		}()
		...
```
Create `addCheckMsg{seq, ref, pr, err}` + `checkAdd(...)`, `watchlistMsg{...}` + `loadWatchlist(...)`, `saveMsg` for add/remove writes. Single-flight: `beginFetch` (lines 74-77) sets `inFlight` and batches `m.spin.Tick`; spinner for `checking owner/repo#n ⠋` reuses `m.spin`. `setFlash` (lines 87-91) for D-03/D-07/D-08 messages.

### `internal/ui/view.go` (mod)
- Add prompt line: copy `bottom()` (lines 120-130): prompt replaces the footer, ends with `return m.cut(line)`, so it costs no list row. Same function branches on `m.adding` first. Render caret `█`, error as `✗ <reason>`; keep text on error.
- Confirm remove: a footer variant in `footer()` (line 138+), same hint segments machinery (`segment`, `fit`, lines 244-270); integrity notes (`· N unavailable`) must survive narrowing before hints (WR-03 rule documented at lines 134-137; `statusSegments` line 293 builds the header segments).
- Marker in full rows: `renderList` (line 361+, not read in detail; planner open it) PR column; unavailable row spans TITLE..CI with warning style (`⚠ unavailable: <reason>`), not faint (`dimmed(pr)` is the faint rule, used at `compactRow`: unavailable PRs must not hit it).
- Compact: `compactRow` (lines 549-578) builds `marker + shortRef + " "` then three glyph segments, each styled separately; add the manual marker into the `text` prefix and include its width in `budget`; for unavailable rows replace the three glyph segments with a single `⚠`. Use `lipgloss.Width` only (display-width rule).
- Empty state `renderEmpty` (line 85): append `· press a to add any PR`.

### `internal/ui/detail.go` (mod)
Add early branch in `openDetail`/render for `pr.Unavailable != ""`: no `fetchDetail`, show ref and full reason (D-11). `enter` call site is model.go line ~252 (`m.openDetail(m.rows[m.cursor])`).

### `cmd/prv/main.go` (mod)
Lines 107-126: after `config.Load` and `github.NewClient`, resolve `watchlist.Path()` the way config does (`var configPath = config.Path`, line 31, is the test seam; add a `watchlistPath` var likewise), and extend the `ui.Options{...}` literal (lines 118-125). Append store load warnings to `Warnings: cfg.Warnings`. Add the `var _ ui.X = (*github.Client)(nil)` assertion style (line 43) for any new interface the client implements.

### Golden tests
**Analog:** `internal/ui/help_test.go` lines 339-343 and `internal/ui/view_test.go` (`fixedNow`, `golden.RequireEqual(t, ansi.Strip(m.View().Content))`); sizes 0x0, 60x15, 80x24, 120x40 under `internal/ui/testdata/TestGolden/<name>_<w>x<h>.golden`. Add: add-prompt (typing, checking, error), remove-confirm, unavailable rows, marker (full and compact), empty state with `a` hint.

## Shared Patterns

### Seam sanitising
**Source:** `internal/github/checks.go` `mapContext`, `internal/github/client.go` `mapResponse`. Every remote string passes through `domain.Clean(...)` at the github seam; domain and ui never see raw strings.

### No I/O in Update/View
**Source:** `internal/ui/refresh.go`. All file and network work in panic-wrapped `tea.Cmd`s returning stamped messages; stale results dropped via seq/gen.

### Error classification and whole-vs-partial
**Source:** `internal/github/errors.go` `classify` + `checks.go` lines 138-160. Rate limit and cancel abort; GraphQL error with data keeps data.

### Single keymap feeds footer, help and Update
**Source:** `internal/ui/keys.go`. Match with `bkey.Matches(msg, m.keys.X)`; import alias `bkey`.

### Warnings
**Source:** `internal/config/config.go` `Config.Warnings` -> `ui.Options.Warnings`; shown for `warnFor` (8s) from the first fetch result (model.go lines 55-58, 383-389).

## No Analog Found

| File/Concern | Reason |
|---|---|
| Atomic JSON write (temp + rename) and corrupt-file backup | The repo writes no files today (config is read-only). Use CLAUDE.md spec; test with `t.TempDir()`. |
| Free-text add prompt incl. `tea.PasteMsg` text | The only prompt is digits-only (`promptKey`, model.go 395-415). Either generalise it or use `bubbles/textinput` (RESEARCH covers the choice). |
| SAML/SSO FORBIDDEN `errors[]` shape | Never observed (01-SPIKE.md Q3); use a constructed fixture. |
| Aliased `repository(owner:,name:){pullRequest(number:)}` document building | `fetchFailing` uses `nodes(ids:)`, not aliases; only chunking, error and accounting patterns carry over. |

## Metadata

**Analog search scope:** `internal/{config,domain,github,ui}`, `cmd/prv`
**Files read:** config.go, keys.go, checks.go, errors.go, list.go, pr.go, refresh.go, model.go (parts), client.go (parts), view.go (parts), main.go (parts), checks_test.go (parts)
**Not read (planner should open):** `internal/ui/help.go`, `internal/ui/detail.go`, `internal/ui/cells.go`, `renderList` in view.go
**Pattern extraction date:** 2026-10-12

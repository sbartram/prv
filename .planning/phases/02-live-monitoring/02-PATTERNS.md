# Phase 2: Live Monitoring - Pattern Map

**Mapped:** 2026-10-11
**Files analyzed:** 19 (7 new, 12 modified)
**Analogs found:** 17 / 19

All analog paths below were checked against `git ls-files` (tracked source; no gitignored mirrors). Prescriptive design (Patterns 1-11, bucket rules, TOML traps, URL allow-list) lives in `02-RESEARCH.md`. This file only says which existing code to copy from.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/config/config.go` (new) | utility/config | file-I/O, transform | `internal/github/client.go` (`NewClient` options + defaults) ; `cmd/prv/main.go` | role-partial |
| `internal/config/config_test.go` (new) | test | transform | `internal/domain/domain_test.go` (table tests) | role-match |
| `internal/domain/pr.go` (mod: `CreatedAt`, `RateLimit.Limit`) | model | transform | itself (lines 90-115) | exact |
| `internal/domain/list.go` (mod: `Bucket`, `Sort`, `Visible`) | utility | transform | itself (`stateRank`, `Sort`, `InWindow`) | exact |
| `internal/domain/backoff.go` (new: `NextDelay`) | utility | transform | `internal/domain/list.go` `InWindow` (pure, time passed in) | role-match |
| `internal/domain/domain_test.go` (mod: rewrite `TestSort`, add Bucket/Visible/NextDelay) | test | transform | itself `TestSort` (178-227) | exact |
| `internal/github/client.go` (mod: query `createdAt`, `limit`; map) | service | request-response | itself (query const ~line 56-75, `mapResponse`) | exact |
| `internal/github/errors.go` (mod: `Short`, `Backoff`, `KindRateLimited`) | utility | request-response | itself (`Hint`, `classify`) | exact |
| `internal/github/ratelimit.go` (new: header-capture RoundTripper) | middleware | request-response | `Options.Transport` seam in `client.go:24,43` | partial |
| `internal/github/testdata/*.json`, `client_test.go` | test fixture | request-response | existing fixtures | exact |
| `internal/ui/model.go` (mod: state machine) | model | event-driven | itself | exact |
| `internal/ui/refresh.go` (new: tick/fetch Cmds, `safeCmd`) | service | event-driven | `Model.fetchCmd` (model.go:71-84) | exact |
| `internal/ui/keys.go` (new: `keyMap`) | config | event-driven | `Update` key switch (model.go:94-114) | partial |
| `internal/ui/open.go` (new: `safeURL`, `openCmd`, `Opener`) | service | request-response | `fetchCmd` + `Fetcher` interface | role-match |
| `internal/ui/view.go` (mod: header/footer composer, filtered-empty, WR-02) | component | transform | itself (`renderList`, `renderError`, `cut`, `wrap`) | exact |
| `internal/ui/help.go` (new: overlay) | component | transform | `view.go` `renderError` + `wrap` | role-match |
| `internal/ui/layout.go` (mod: `len(m.rows)`) | utility | transform | itself (lines 68-90) | exact |
| `internal/ui/styles.go` (mod: help/warn styles) | config | transform | itself | exact |
| `internal/ui/model_test.go`, `view_test.go` + goldens | test | event-driven | itself (`runCmd`, `newModel`, golden) | exact |
| `cmd/prv/main.go` (mod: config, flag precedence, Opener, transport) | controller | request-response | itself `run` | exact |
| `cmd/prv/main_test.go` (mod: isolate config path) | test | request-response | itself | exact |

## Pattern Assignments

### `internal/ui/refresh.go` + `internal/ui/model.go` (model/service, event-driven)

**Analog:** `internal/ui/model.go` lines 71-89 (panic-safe Cmd) and 91-144 (Update).

Panic-safe Cmd to generalize into `safeCmd`/gen-stamped fetch (copy, but capture `ctx`, `fetcher`, `gen` instead of `m`, per RESEARCH Pattern 1):
```go
func (m Model) fetchCmd() tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if recover() != nil {
				msg = fetchedMsg{err: errors.New("unexpected internal error while loading PRs")}
			}
		}()
		snap, err := m.fetcher.Fetch(m.ctx)
		return fetchedMsg{snap: snap, err: err}
	}
}
```
Keep this fixed panic message (panic value never shown). Reuse for `openCmd` returning `openedMsg{err}`.

Spots to change in `Update` (all in model.go):
- line 87-89 `Init`: keep `tea.Batch(m.spin.Tick, fetch)`; add NO tick. Set `gen: 1, inFlight: true` in `New()` (lines 52-64).
- lines 95-114 key switch: `msg.String()` switch is what `key.Matches` replaces; keep `"q"`, `"ctrl+c"`, `"j"`, `"k"`, `"down"`, `"up"` working (tests send them).
- line 118-123 `spinner.TickMsg`: gate currently `m.state == stateLoading`; change to `m.inFlight || m.state == stateLoading`.
- line 125-127: KEEP `if errors.Is(msg.err, context.Canceled) { return m, nil }` first, then gen check.
- line 129 `m.cursor, m.offset = 0, 0`: REMOVE (D-03); replace with `rebuild()` + reselect by ref.
- lines 130-134: on error with data present, do not touch `m.snap`/`m.state`; only stateError if no snapshot yet (D-07/D-09).

Seam to reuse for the virtual-hour test: `Fetcher` interface (model.go:16-19) and `Options` struct (21-24; add `Interval, Show, ConfigPath, Warnings, Opener`). Add injectable `now func() time.Time` and `tick func(time.Duration, int) tea.Cmd` fields defaulting to `time.Now` / `tea.Tick`.

### `internal/ui/view.go` (component, transform)

**Analog:** itself.

Footer/header to replace with the priority composer (this is the WR-03 cause):
```go
// view.go:141-155
footer := "j/k move · q quit"
if m.snap.Truncated { footer += " · list truncated" }
if len(m.snap.Warnings) > 0 { footer += " · some data could not be loaded" }
lines := []string{"prv · @" + m.snap.Viewer + " · " + strconv.Itoa(len(m.snap.PRs)) + " PRs"}
lines = append(lines, strings.Split(t.String(), "\n")...)
lines = append(lines, footer)
for i, l := range lines { lines[i] = m.cut(l) }
```
Keep: `chromeLines = 3` (header + table header + footer; prompt REPLACES footer line), `m.cut` (ansi.Truncate + `lipgloss.Width`), table build with `StyleFunc`, widths computed once over all cells. Switch loops from `m.snap.PRs` to `m.rows` (lines 77, 79, 107, 125) and cursor marker `i == m.cursor` stays index-in-rows.

Error-panel hint pattern to copy for `shorter`/`backoffer` local interfaces (view.go:48-64):
```go
type hinter interface{ Hint() string }
...
var h hinter
if errors.As(m.err, &h) { if hint := h.Hint(); hint != "" { ... } }
```
Add `type shorter interface{ Short() string }` etc. beside it; `ui` must not import `internal/github`. Add "retrying at HH:MM" line to `renderError` (D-09).

Empty state (view.go:28-31) drops `Truncated`/`Warnings`: fix per WR-02. Wrapping: always route message screens through `m.wrap(...)` (view.go:37-46).

### `internal/ui/layout.go` (utility)

**Analog:** itself. Replace `len(m.snap.PRs)` with `len(m.rows)` at line 71 (`visibleRows` pre-size branch) and line 85 (`clampOffset`):
```go
m.offset = max(0, min(m.offset, len(m.snap.PRs)-v))
```
`chromeLines = 3` must not change (goldens).

### `internal/ui/styles.go` (config)

**Analog:** itself. ANSI-16 only via `lipgloss.Green/Red/Yellow`, `faintStyle`, `headerStyle`. Add `warnStyle` (Yellow/Red) the same way; override `help.Styles` with `faintStyle`-derived styles (help defaults use hex colours and break `TestStylesUseAnsi16`).

### `internal/ui/keys.go`, `help.go`, `open.go` (new)

No close analog for `keyMap`/`bubbles/key` (no prior use); take shape from RESEARCH Pattern 10. For `open.go` copy the `Fetcher` interface style (small local interface in `ui`, implementation injected via `Options`) and the `fetchCmd` recover wrapper. For `help.go` copy `renderError` string-builder + `m.wrap`.

### `internal/domain/list.go` (utility, transform)

**Analog:** itself. Replace `stateRank` + `Sort` (lines 24-52) with `Bucket(pr)` and keep the same shape:
```go
func Sort(prs []PR) {
	slices.SortStableFunc(prs, func(a, b PR) int {
		if c := cmp.Compare(Bucket(a), Bucket(b)); c != 0 { return c }
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 { return c }
		if c := cmp.Compare(a.Ref.Repo, b.Ref.Repo); c != 0 { return c }
		return cmp.Compare(a.Ref.Number, b.Ref.Number)
	})
}
```
`Visible(prs, show, numFilter)` follows the `InWindow` switch-on-`pr.State` style (list.go:13-22); keep `StateUnknown` always visible. `NextDelay` in `backoff.go`: same pure style as `InWindow(pr, now, closedDays)`: `now` as a parameter, never `time.Now()`.

### `internal/domain/pr.go`

Add `CreatedAt time.Time` next to `UpdatedAt` (pr.go ~line 95) and `Limit int` to `RateLimit` (pr.go ~101-105: `Cost, Remaining, ResetAt`). Do not renumber the `State/Review/CI` iota enums.

### `internal/domain/domain_test.go`

**Analog:** `TestSort` (lines 178+): local `mk` constructor, `want := []string{...}` of IDs, `slices.Equal`. Rewrite keyed on `CreatedAt` and buckets; add `TestBucket` as a table over (State, IsDraft, Review, CI).

### `internal/github/client.go`

Query const `discoveryQuery` (~line 56-75): `rateLimit { cost remaining resetAt }` becomes `rateLimit { limit cost remaining resetAt }`; field list `id number title url isDraft state closedAt mergedAt updatedAt` gains `createdAt`. Map both in `mapResponse`. `domain.Sort(snap.PRs)` stays in `Fetch` (~line 200). `Options.Transport` (line 24) is where the header-capture wrapper is injected; `NewClient` must keep `LogIgnoreEnv: true`, no `EnableCache`.

### `internal/github/errors.go`

**Analog:** itself. `Hint()` switch (43-57) is the model for `Short()`; extend `classify` (64-103): the `*api.HTTPError` branch (71-87) already has `he.StatusCode` and `he.Headers` for 403/429 `Retry-After`; the `*api.GraphQLError` branch (88-97) must consult the captured headers. Append `KindRateLimited` AFTER `KindUnknown` (const block lines 17-24; `KindAuth` is the zero value). Keep `var _ interface{ Hint() string } = (*Error)(nil)` compile-time assertion style for the new interfaces. Never include headers or the token in `Message`.

### `internal/github/ratelimit.go` (new, no close analog)

Use RESEARCH Pattern 3. Only the `Transport http.RoundTripper` seam exists; wrap it, read exactly `Retry-After`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, mutex-guard.

### `internal/config/config.go` (new)

Weak analog: `cmd/prv/main.go` for error-to-stderr style, `client.go` `NewClient` for "zero value means default" handling:
```go
c := &Client{gql: gql, closedDays: opts.ClosedDays, now: opts.Now}
if c.closedDays <= 0 { c.closedDays = domain.DefaultClosedDays }
if c.now == nil { c.now = time.Now }
```
Follow RESEARCH Pattern 6 for TOML specifics (`duration` `UnmarshalText`, regexp for non-ParseError, `Undecoded()`).

### `cmd/prv/main.go` (controller)

**Analog:** itself `run` (45-87). Order to preserve: flagset, parse, `--version` return BEFORE config load, closed-days validation, token lookup, client, program.
```go
closedDays := fs.Int("closed-days", domain.DefaultClosedDays, "") // hidden: not in usage
...
if *closedDays < 1 { fmt.Fprintln(stderr, "prv: --closed-days must be at least 1"); return 2 }
```
Add upper bound 365 here (WR-01); use `fs.Visit` to know if the flag was set (flag > config > default). Config path via a package var like `var tokenForHost = auth.TokenForHost` (line 25) so tests can replace it. Pass `Interval/Show/Warnings/Opener` into `ui.Options` at line 81; construct `browser.New("", io.Discard, io.Discard)` once. Errors go through `fmt.Fprintln(stderr, "prv: ...")`, exit code 2 for usage/config, 1 for runtime.

### Tests (ui)

**Analog:** `internal/ui/model_test.go` helpers: `runCmd` (lines ~40-56; recurses into `tea.BatchMsg`, never runs Cmds returned by Update, hence Init must not return a tick), `key(s)`, `plain(m)`, `newModel(f, closedDays)` (120x40), `onePR()`, `hintErr` (implements `Hint`). Add a gen-stamping helper for `fetchedMsg` literals. Golden tests: `view_test.go` with `internal/ui/testdata/TestGolden/*.golden` at 0x0, 80x24, 120x40; regenerate with `go test ./internal/ui -update`.

## Shared Patterns

### Update/View do no I/O; all I/O in a `tea.Cmd` with recover
**Source:** `internal/ui/model.go:71-84`. Apply to fetch, open-browser, any config/log writes.

### `ui` never imports `internal/github`
**Source:** `internal/ui/view.go:48-50` (`hinter`). Apply to `shorter`, `backoffer`, `Opener`.

### Canceled context is ignored on quit
**Source:** `model.go:125-127`. Apply to every new result-message handler (`fetchedMsg`, `openedMsg`).

### ANSI-16 palette, status never colour-alone
**Source:** `internal/ui/styles.go:1-40`. Apply to header warning, yellow rate-limit text, help styles; guarded by `TestStylesUseAnsi16`.

### Width safety
**Source:** `view.go:66-73` `cut`, `layout.go` `truncateCell`, `view.go:37-46` `wrap` (all `ansi.Truncate`/`lipgloss.Width`). Apply to header, footer, prompt, help overlay.

### Token hygiene / error text
**Source:** `internal/github/errors.go:26-37` (messages built from status, fixed strings, `domain.Clean` text only). Apply to `Short()`, rate-limit capture, config warnings (pass keys through `domain.Clean`).

### Time as input
**Source:** `domain.InWindow(pr, now, closedDays)` and `Options.Now` in `client.go:23`. Apply to `NextDelay`, model `now`/`tick`.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/ui/keys.go` | config | event-driven | No `bubbles/key`/`help` usage yet; use RESEARCH Pattern 10 |
| `internal/github/ratelimit.go` | middleware | request-response | No RoundTripper wrappers yet; only the `Transport` seam; RESEARCH Pattern 3 |
| `internal/config/*` | config | file-I/O | No file reads or TOML yet; RESEARCH Pattern 6 and the Code Examples sketch |

## Metadata

**Analog search scope:** all 31 tracked files under `cmd/` and `internal/` (git ls-files).
**Files read:** `internal/ui/{model,view,layout,styles}.go`, `internal/ui/model_test.go` (helpers), `internal/domain/{list,pr}.go`, `internal/domain/domain_test.go` (TestSort), `internal/github/{client,errors}.go`, `cmd/prv/main.go`.
**Pattern extraction date:** 2026-10-11

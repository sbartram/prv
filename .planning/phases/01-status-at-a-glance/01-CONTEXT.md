# Phase 1: Status at a Glance - Context

**Gathered:** 2026-10-11
**Status:** Ready for planning

<domain>
## Phase Boundary

The user runs `prv`. From live GitHub data, it shows every PR the user authored or is assigned to (open, plus merged/closed in the last 14 days). Each row shows repo#n, title, author, draft marker, state, review status and a CI badge. The user can move with j/k and the arrow keys and quit with `q`. Data is fetched once at launch. Auto-refresh, the attention sort, filters, config, open-in-browser and help come in Phase 2.

The phase opens with the live-verification spike (research/SUMMARY.md, 10 questions). It runs before the discovery query, review derivation and token resolution are frozen.

Requirements: AUTH-01, AUTH-02, AUTH-03, CONF-02, WATCH-01, WATCH-02, WATCH-03, LIST-01, LIST-02, LIST-03, LIST-05, NAV-01, NAV-04.

</domain>

<decisions>
## Implementation Decisions

### Row look & badges
- **D-01:** Each badge is a glyph plus a short word, with ANSI colour added on top. Examples: `✓ approved`, `✗ changes`, `– none`, `✓ passing`, `✗ failing`, `● running`. A badge must still read correctly when colour is stripped.
- **D-02:** Columns are identity first: `PR` (owner/repo#n), `TITLE`, `AUTHOR`, `STATE`, `REVIEW`, `CI`. TITLE is the flexible column that gets truncated first.
- **D-03:** AUTHOR shows `me` when the author is the viewer, and the login otherwise. The query must therefore get `viewer { login }`. It is one cheap field, so add it to the discovery request.
- **D-04:** A draft PR has its whole row dimmed and carries a `draft` tag, so it reads without colour too. Its STATE stays the true state (open).
- **D-05:** The palette uses the 16 ANSI colours only (terminal-theme red/green/yellow, plus faint/dim). No hex colours and no background detection. It must read on light and dark themes and inside multiplexers.

### Ordering (interim, until the Phase 2 attention sort)
- **D-06:** Sort by state group (Open, then Merged, then Closed), then `updatedAt` newest first. The final tiebreak is owner/repo#n, so the order is deterministic. Phase 2 may reuse this as the secondary key of the attention sort.
- **D-07:** Merged and closed rows are dimmed like drafts. There is no divider and there are no section headers.

### Launch & failure states
- **D-08:** The TUI enters the alt screen at once and shows a spinner with "Fetching your PRs…" during the first fetch. `q` and `ctrl+c` work while the fetch runs.
- **D-09:** When the initial fetch fails (network, 401, 5xx, GraphQL error), the TUI shows an error panel with an actionable hint (for example, 401 → "run `gh auth login`"). `r` retries and `q` quits. **Scope note:** this pulls a minimal `r` into Phase 1, only for the error screen. Phase 2 (RFSH-02) generalises it into refresh-anytime and single-flight.
- **D-10:** With zero PRs, show a friendly empty state that names the account: "No open PRs authored by or assigned to @login, and none closed in the last 14 days." This makes a wrong-account token obvious.
- **D-11:** With no token (AUTH-02), print to stderr before the TUI starts: "prv needs a GitHub token. Run `gh auth login`, or set GH_TOKEN / GITHUB_TOKEN." Then exit with code 1.

### Authored vs assigned
- **D-12:** There is no provenance column. The AUTHOR column implies it: `me` means authored, and another login means assigned. A PR that is both shows `me`.
- **D-13:** A draft PR shows its true REVIEW and CI values. The dimmed row plus the `draft` tag is the only treatment.
- **D-14:** The CI badge follows `statusCheckRollup.state` as GitHub reports it (FAILURE/ERROR → failing, PENDING/EXPECTED → running, SUCCESS → passing). A null rollup → `no checks`. CI must match github.com (success criterion 4), so there is no client-side override of the precedence.

### Carried forward (already decided upstream; do not re-litigate)
- Token resolution: go-gh `auth.TokenForHost("github.com")`, which checks GH_TOKEN, then GITHUB_TOKEN, then the gh login. The token is never logged, shown or persisted (`LogIgnoreEnv: true`).
- Discovery: a single `ISSUE_ADVANCED` search `is:pr (author:@me OR assignee:@me) (is:open OR closed:>=DATE)` with a fixed 14-day window, deduplicated by node ID. Fallback is separate queries or `viewer.pullRequests`, if spike Q4/Q8 shows missing PRs.
- Review: use `reviewDecision` when it is set. Otherwise derive from `latestOpinionatedReviews`. A null decision never maps to "review required". Spike Q1 checks the comment-after-approve case.
- One total check-state mapping covers CheckRun and StatusContext, with an unknown case. CANCELLED, NEUTRAL and SKIPPED are never red.
- Strip control characters and ANSI from remote strings. No I/O in Update/View. Read the raw GraphQL envelope (partial data plus `errors[]`).

### Claude's Discretion
- The exact glyph set and wording for each badge value, within D-01 and D-05
- Spinner style, header/status-line content and the cursor/selection highlight style
- Column widths and truncation rules at a normal width (full narrow-pane work is Phase 3, NAV-05)
- The `prv --version` output format (ldflags `main.version` with a `debug.ReadBuildInfo()` fallback)
- How partial results (data plus `errors[]`) appear in Phase 1, if the spike shows they happen during discovery

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Scope and requirements
- `.planning/ROADMAP.md` § Phase 1: goal, success criteria, early gate and pitfalls list
- `.planning/REQUIREMENTS.md`: requirement IDs for this phase, plus "Technical Defaults (from research conflicts)"
- `.planning/PROJECT.md`: core value, constraints and the Key Decisions table

### Research
- `.planning/research/SUMMARY.md` § "Live-Verification Spike": the 10 spike questions. Q1, Q4, Q5 and Q8 decide this phase. Record the findings in this phase directory.
- `.planning/research/STACK.md`: library versions and Bubble Tea v2 API facts (also summarised in `.claude/CLAUDE.md`)
- `.planning/research/ARCHITECTURE.md`: GraphQL query shapes and model structure
- `.planning/research/PITFALLS.md`: null `reviewDecision`/rollup handling and check-state mapping
- `.planning/research/FEATURES.md`: prior-art feature comparison (gh-dash)

### Project instructions
- `.claude/CLAUDE.md`: the prescriptive tech stack (charm.land v2 imports, go-gh v2 GraphQL, what NOT to use)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None. This is a greenfield repo with no Go code yet. Phase 1 creates the module `github.com/sbartram/prv` with the `cmd/prv` layout.

### Established Patterns
- None yet. Phase 1 sets them up. Use `.claude/CLAUDE.md` as the source for conventions: pure Update/View with golden tests, `tea.Cmd` for all I/O, and `lipgloss/table` with our own cursor index.

### Integration Points
- `cmd/prv/main.go`: token resolution happens before `tea.NewProgram`. The no-token path (D-11) exits here.
- The GraphQL client sits behind a small interface, so tests can use recorded fixtures. Phase 2 adds refresh and Phase 4 adds aliased lookups on the same client.

</code_context>

<specifics>
## Specific Ideas

- The row layout is modelled on the preview the user picked:
  ```
  PR              TITLE              AUTHOR  STATE  REVIEW      CI
  acme/api#481    Fix token refresh  me      open   ✗ changes   ✗ failing
  sbartram/prv#12 Add list view      me      open   ✓ approved  ✓ passing
  ```
- prv lives in a multiplexer pane, so ANSI-16 colours and a badge that still reads without colour both matter.

</specifics>

<deferred>
## Deferred Ideas

None. The discussion stayed within phase scope. (A minimal `r` retry on the error screen was pulled forward deliberately; see D-09.)

</deferred>

---

*Phase: 01-status-at-a-glance*
*Context gathered: 2026-10-11*

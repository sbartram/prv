---
phase: 01-status-at-a-glance
plan: 04
subsystem: release
tags: [go, release-readiness, go-install, readme, checkpoint]

requires:
  - phase: 01-status-at-a-glance
    provides: "01-03 readable rows: coloured badges, column layout, scrolling, golden tests"
provides:
  - "README with the go install command, token precedence, least-privilege scopes, keys and badge legend"
  - ".gitignore entries for a local prv binary and the GSD runtime files"
  - "A recorded automated release gate (format, vet, race tests, Go floor, static local install, --version, no-token exit, hygiene)"
  - "User-confirmed go install github.com/sbartram/prv/cmd/prv@latest through the module proxy path"
affects: [phase-02, phase-03, release]

actuals:
  tokens: 790
  tasks: 2
  commits: 2

plan_head_before: 7256735f17ece9d31936e6dec91a41024cbef027
plan_head_after: 6f4846503b65dc4188faf865dc5fd501b730e9ed
commits: 2

tech-stack:
  added: []
  patterns:
    - "Release gate as one re-runnable block of greps and go commands; every check exits non-zero on failure"

key-files:
  created:
    - README.md
  modified:
    - .gitignore

key-decisions:
  - "Hidden window flag is not documented in the README (user decision 1)"
  - "GSD runtime files .planning/state.json and .planning/milestone.lock are gitignored; .planning itself stays tracked so commit_docs keeps working"

patterns-established:
  - "Remote go install is proven with GOPROXY=direct and an empty GOBIN right after a push, to bypass a stale proxy cache"

requirements-completed: [CONF-02, AUTH-03]

coverage:
  - id: D1
    description: "README lets a new user install (go install ...@latest, Go 1.26+), authenticate (GH_TOKEN, then GITHUB_TOKEN, then gh login; env token overrides gh), use read-only scopes, and read the keys and badge legend; the hidden window flag is not documented"
    requirement: "AUTH-03"
    verification:
      - kind: other
        ref: "release gate block 4: grep for install command, GH_TOKEN, GITHUB_TOKEN, gh auth login, no closed-days, /prv in .gitignore"
        status: pass
    human_judgment: false
  - id: D2
    description: "Automated release gate: gofmt clean, go vet, go test -race, go 1.26.0 with no toolchain line, CGO_ENABLED=0 go install into an empty GOBIN, --version prints one line, build settings record CGO_ENABLED=0, no-token run exits 1"
    requirement: "CONF-02"
    verification:
      - kind: other
        ref: "release gate blocks 1 and 2 (gofmt -l, go vet ./..., go test ./... -count=1 -race, local CGO_ENABLED=0 go install, go version -m, env -i run)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Token and output hygiene: LogIgnoreEnv true, no stdout/stderr/disk writes in non-test internal code, no v1 Charm imports, no store/config/browser package, no token-shaped string in any tracked or untracked non-ignored file"
    requirement: "AUTH-03"
    verification:
      - kind: other
        ref: "release gate block 3 (greps over internal/, go files and git ls-files)"
        status: pass
    human_judgment: false
  - id: D4
    description: "go install github.com/sbartram/prv/cmd/prv@latest installs a working binary from the pushed main and --version prints its version; the user ran the installed binary in a real terminal"
    requirement: "CONF-02"
    verification:
      - kind: manual_procedural
        ref: "Task 2 checkpoint: GOBIN=<empty> GOPROXY=direct CGO_ENABLED=0 go install ...@latest, prv --version, user reply 'approved'"
        status: pass
    human_judgment: true
    rationale: "A remote install and a real terminal run need the user's push and eyes; confirmed by the user at the blocking checkpoint."

duration: ~40min across a checkpoint
completed: 2026-10-12
status: complete
---

# Phase 1 Plan 4: Ship-ready Summary

**README with install, token precedence, read-only scopes, keys and badge legend, an automated release gate that passed in one run, and a user-confirmed `go install github.com/sbartram/prv/cmd/prv@latest` that installs a working `prv` from the pushed main**

## Performance

- **Duration:** about 40 min of wall clock including the checkpoint wait (Task 1 committed 2026-10-11 20:45 -0400; continuation committed 21:18 -0400)
- **Completed:** 2026-10-12T01:18Z
- **Tasks:** 2 (1 auto, 1 blocking-human checkpoint)
- **Files modified:** 2 (`README.md` created, `.gitignore` modified)

## Accomplishments

- README (97 lines) in plain short sentences: core value, install with Go 1.26+, `prv --version`, token order GH_TOKEN then GITHUB_TOKEN then `gh auth login` (an env token overrides the gh login), read-only fine-grained scopes (Pull requests, Checks, Commit statuses, Metadata), keys, and the STATE, REVIEW and CI legend. The hidden window flag is not documented.
- Automated release gate passed in one run, all four blocks exit 0 (recorded below).
- The user pushed (the orchestrator merged and pushed on their request) and confirmed the remote `go install ...@latest` works.

## Release gate results (Task 1)

All run by the prior executor, all exit 0:

| Check | Result |
|-------|--------|
| `gofmt -l .` | empty |
| `go vet ./...` | ok |
| `go test ./... -count=1 -race` | ok: cmd/prv, internal/domain, internal/github, internal/ui |
| `go.mod` floor | `go 1.26.0`, no `toolchain` line |
| `CGO_ENABLED=0 go install ./cmd/prv` into an empty GOBIN | ok |
| `prv --version` | `prv v0.0.0-20261012004434-7256735f17ec+dirty` (`+dirty` only from pre-existing uncommitted files) |
| `go version -m` | records `CGO_ENABLED=0` |
| `env -i` run with no gh on PATH | exit 1 |
| Hygiene greps | `LogIgnoreEnv: true` present; no `fmt.Print`, `log.*`, `os.Stdout`/`os.Stderr`/disk writes in non-test internal code; no v1 Charm import paths; no `internal/store`, `internal/config`, `internal/browser`; no token-shaped string in any tracked or untracked non-ignored file |
| README content | install command, `GH_TOKEN`, `GITHUB_TOKEN`, `gh auth login`, "read-only", "Pull requests" present; no `closed-days` |

## Task 2 checkpoint outcome (blocking-human)

User reply: **approved**.

- The user asked the orchestrator to merge and push. The branch was merged into main with `--no-ff` as merge commit `d2057f9` and pushed: `aa11d7e..d2057f9 main -> main`. Git HTTPS credentials were absent, so the push used gh as a one-off credential helper. No tag was created (optional step, skipped).
- Remote install: `GOBIN=<empty dir> GOPROXY=direct CGO_ENABLED=0 go install github.com/sbartram/prv/cmd/prv@latest` downloaded `github.com/sbartram/prv v0.0.0-20261012010335-d2057f9d080b`. `prv --version` printed `prv v0.0.0-20261012010335-d2057f9d080b` and exited 0.
- The user ran the installed binary in a real terminal and approved: the list and empty state showed, and `q` restored the terminal.

## Task Commits

1. **Task 1: README and automated release gate** - `1cb204d` (docs)
2. **Task 2: user push and go install check** - checkpoint, no code commit; evidence above
3. **User-requested addition at the checkpoint: gitignore GSD runtime files** - `6f48465` (chore)

**Plan metadata:** committed separately as `docs(01-04): complete ship-ready plan`.

## Files Created/Modified

- `README.md` - install, auth, scopes, keys and legend
- `.gitignore` - `/prv` (Task 1), plus `.planning/state.json` and `.planning/milestone.lock`

## Decisions Made

- The hidden window flag stays undocumented (user decision 1).
- `.planning/state.json` and `.planning/milestone.lock` are ignored. `milestone.lock` is a per-session advisory claim (session id, pid, TTL). `state.json` is a snapshot regenerated from STATE.md and ROADMAP.md at step boundaries. Neither belongs in history.

## Deviations from Plan

### User-requested change

**1. [User request at checkpoint] Gitignore GSD runtime files**
- **Found during:** Task 2 checkpoint, after approval
- **Request:** append `.planning/state.json` and `.planning/milestone.lock` to `.gitignore` and commit separately. `.gitignore` is already in this plan's `files_modified`.
- **Files modified:** `.gitignore`
- **Verification:** `git check-ignore .planning/state.json .planning/milestone.lock` prints both paths; `git check-ignore .planning` prints nothing (exit 1), so `commit_docs` stays on.
- **Committed in:** `6f48465`

---

**Total deviations:** 1 (user request, not a defect fix)
**Impact on plan:** None on scope; two ignore lines.

## Issues Encountered

- Git HTTPS credentials were absent when pushing; the push used gh as a one-off credential helper. No credential was stored.
- The `+dirty` suffix in the Task 1 version string came from pre-existing uncommitted files, not from the plan. The remote install shows a clean pseudo-version.
- Note on history: merge commit `d2057f9` on main contains the plan up to `1cb204d`. `6f48465` and the plan metadata commit follow on the phase branch and are not yet in main. Nothing was pushed by this continuation.

## Authentication Gates

None.

## Known Stubs

None.

## Threat Flags

None. T-01-16 (repository contents before push) was mitigated by the token-pattern grep over all tracked and untracked non-ignored files. T-01-17 (published install path) was mitigated by the user-gated push and the remote install check. T-01-18 (README token guidance) was mitigated: the README recommends read-only scopes, says prv never stores the token, and never asks for a token in a file.

## User Setup Required

None.

## Next Phase Readiness

- Phase 1 plans are all complete. Phase-level verification (and the carried human checks from 01-03, such as colours on light and dark themes and a narrow pane) remains for `/gsd-verify-work`.
- Open items carried forward: RESEARCH A7 (ambiguous-width badge glyphs in CJK-locale terminals); no `v0.1.0` tag yet, so `--version` shows a pseudo-version.

## Self-Check: PASSED

- `README.md` and `.gitignore` exist.
- Commits `1cb204d` and `6f48465` are ancestors of HEAD.
- `git check-ignore` prints both runtime files and nothing for `.planning`.
- Nothing pushed by this continuation.

---
*Phase: 01-status-at-a-glance*
*Completed: 2026-10-12*

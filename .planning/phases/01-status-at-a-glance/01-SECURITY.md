---
phase: "01"
slug: "status-at-a-glance"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-12"
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| process env / gh keyring → prv | The token enters memory; it leaves only as the Authorization header to api.github.com | GitHub token (secret) |
| api.github.com → prv | Untrusted remote strings, enum values, HTTP status, error messages, partial data | PR metadata (public/private repo data) |
| prv → terminal | Everything rendered; an escape sequence in remote text could control the terminal | Cleaned display text |
| Go module proxy → build / test build | Third-party code enters the binary or test binary | Source code |
| local repository → github.com (push) | Everything committed becomes public | Repository contents |
| module proxy → user machine (go install) | The published module is fetched and built on the user's machine | Source code |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-01 | Information disclosure | internal/github/client.go NewClient (go-gh HTTP logging) | high | mitigate | `LogIgnoreEnv: true`; no `Log` or `EnableCache` set | closed |
| T-01-02 | Information disclosure | cmd/prv/main.go stdout/stderr | high | mitigate | Token only in `github.Options.Token`; constant D-11 message; `TestNoToken`; no fmt/log output in non-test internal code | closed |
| T-01-03 | Tampering | discovery query construction | medium | mitigate | Constant `discoveryQuery`; search string only as GraphQL variable `q` | closed |
| T-01-04 | Tampering / Spoofing | remote strings rendered in the terminal | high | mitigate | `domain.Clean` at the mapper seam (title, author, repo, viewer); `TestCleanAtSeam`, `TestMapperCleansRemoteStrings` | closed |
| T-01-05 | Spoofing | wrong identity from a stale env token | medium | mitigate | Header shows `@<viewer.login>` on every list screen | closed |
| T-01-06 | Information disclosure | test fixtures | medium | mitigate | Synthetic fixtures; no token-shaped strings in testdata | closed |
| T-01-07 | Elevation of privilege | `gh` subprocess lookup | low | transfer | go-gh `safeexec.LookPath` / `GH_PATH`; prv imports no `os/exec` | closed |
| T-01-08 | Information disclosure | internal/github/errors.go Error(), Hint(), Snapshot.Warnings | high | mitigate | Messages from status codes, fixed strings and `domain.Clean(API message)`; `TestTokenNeverAppears` | closed |
| T-01-09 | Information disclosure | panic inside fetchCmd | medium | mitigate | Deferred `recover()` returns a fixed generic message; `TestFetchPanicShowsErrorPanel` | closed |
| T-01-10 | Spoofing | 401 handling | medium | mitigate | No fallback token source; hint names `gh auth login`, GH_TOKEN/GITHUB_TOKEN | closed |
| T-01-11 | Denial of service | pagination | low | mitigate | `maxPages = 5`; Truncated flag in footer | closed |
| T-01-12 | Tampering (integrity of what is shown) | partial data and later-page failures | medium | mitigate | Footer notes; `TestPartialData`, `TestLaterPageFailure`, `TestFooterNotes` (see review WR-02/WR-03 below) | closed |
| T-01-13 | Tampering | API error messages rendered in the terminal | medium | mitigate | `domain.Clean` applied in `classify` and partial-data warnings | closed |
| T-01-14 | Tampering | layout of wide, combining or CJK text | low | mitigate | `lipgloss.Width` / `ansi.Truncate`, no byte-length maths; `TestCellWidth`, `TestViewFitsScreen` | closed |
| T-01-15 | Spoofing (misread status) | status shown by colour only | low | mitigate | Glyph + word badges, `>` marker with reverse video; `TestBadgesReadWithoutColour`, `TestSelectionNotColourOnly` | closed |
| T-01-16 | Information disclosure | repository contents before the push | high | mitigate | Release-gate token grep over tracked and untracked files; re-checked at audit with `git grep` (no matches outside .planning) | closed |
| T-01-17 | Tampering | published install path | medium | mitigate | User-gated push (checkpoint approved); origin/main = d2057f9; remote `go install …@latest` verified | closed |
| T-01-18 | Information disclosure | README token guidance | low | mitigate | README recommends read-only scopes; never asks for a token in a file | closed |
| T-01-SC | Tampering | Go module installs (go get / go install) | high | mitigate | Pinned versions (bubbletea v2.1.0, go-gh v2.16.2, …); `go mod verify` passes; golden v0.1.0 test-only | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

Related advisory code-review findings (01-REVIEW.md, not threats in this register): WR-01 unbounded hidden `--closed-days` (overflow of the search window), WR-02 empty state hides partial-fetch warnings, WR-03 footer integrity notes truncated first on narrow panes. All three are integrity-of-display issues that touch T-01-12; they are tracked as open review findings, not as open threats.

---

## Accepted Risks Log

No accepted risks.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-12 | 19 | 19 | 0 | gsd-secure-phase (orchestrator, ASVS L1 grep verification; auditor skipped per short-circuit rule) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-12

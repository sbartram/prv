# Phase 5: Address tech debt: watch-list locking + live smoke - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-13
**Phase:** 05-address-tech-debt-watch-list-locking-live-smoke
**Areas discussed:** Watch-list fix scope, Lock behaviour & platforms, Live smoke form, Which live checks

---

## Watch-list fix scope

| Option | Description | Selected |
|--------|-------------|----------|
| WR-02 backup crash-safety | Temp file, then os.Link publish; copy the hashed bytes | ✓ |
| WR-03 no silent clobber | Refuse to write over a version-N file; flash the backup path on start-over | ✓ |
| WR-04 preserve unknown keys | RawMessage decode, per-entry skip | |
| WR-05 esc during save | Ignore esc once the save is dispatched | ✓ |

| Option | Description | Selected |
|--------|-------------|----------|
| IN-02 specific save errors | Short cleaned reason, full-list sentinel | ✓ |
| G4 add drops partial warnings | | |
| Neither | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Multi-process test | Re-exec the test binary as N children | ✓ |
| Goroutines, separate Stores | | |
| You decide | | |

**User's choice:** WR-02, WR-03, WR-05 + IN-02; multi-process proof.

---

## Lock behaviour & platforms

| Question | Options | Selected |
|----------|---------|----------|
| Platforms | macOS + Linux only / Keep Windows compiling | macOS + Linux only |
| Contention | Block with timeout / Block indefinitely / Fail fast | Block with timeout |
| Reads | Lock-free / Shared lock | Lock-free |
| Lock file | Sibling .lock, never deleted / You decide | Sibling .lock, never deleted |

---

## Live smoke form

| Question | Options | Selected |
|----------|---------|----------|
| Form | Both / Manual UAT only / Automated only | Both |
| Target | Stable public PRs / Own sbartram/prv PRs / Both | Stable public PRs |
| CI | Local only / CI on a schedule | Local only |

---

## Which live checks

| Question | Options | Selected |
|----------|---------|----------|
| Automated | State/review/CI/MERGE mapping; Team review requests; Large CI matrix detail; Bad-token 401 flow | State/review/CI/MERGE mapping |
| Manual UAT | Phase 2 UI set; Badges + MERGE by eye; Detail view interaction; Light/dark + narrow pane | Phase 2 UI set; Badges + MERGE by eye |
| Still deferred | SAML/SSO; Live rate-limit pause; assignee:@me + CJK width | All three |

**Notes:** Unselected checks (team reviews, large-matrix detail, 401 flow, detail interaction, light/dark) stay deferred. The user confirmed this at the wrap-up.

## Claude's Discretion

- The lock timeout and poll interval, and the error wording
- The shape of the start-over notice from Add/Remove
- x/sys/unix vs syscall
- The live-test env var name and placement, and the helper-process mechanism
- The wording of 05-UAT.md

## Deferred Ideas

WR-04, G4, G1/G2/IN-01, IN-03, G8 docs fix, the Phase 2 IN-01..04 items and the README config keys, a CI workflow, Windows lock support, and the live checks listed in D-17.

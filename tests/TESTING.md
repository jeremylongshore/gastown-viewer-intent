# Testing Policy — Gastown Viewer Intent

**Status:** v1.0 · Authored 2026-05-24 as the closeout artifact of bead
`gastown-3uf` per the 2026-05-23 AT-DECR Q3 (CSO SOP-consistency + CISO
bifurcated-coverage binding constraints).

This document is the canonical statement of what test coverage this repo
commits to, where the gaps are by deliberate decision, and what
follow-ups are scheduled to close them. The `@intentsolutions/audit-harness`
hash manifest (`.harness-hash`) is pinned against this file; AI-proposed
edits to either must come with re-init evidence.

## 1. Repo type and applicable layers

This repo is a **multi-component Go-and-TypeScript service** (Go HTTP
daemon + React/Vite web client + Go TUI). Per the 7-layer testing
taxonomy (see `~/.claude/skills/audit-tests/references/taxonomy.md`),
these layers apply:

| Layer | Tool / approach | Today's bar |
|---|---|---|
| L1 (git hooks) | `audit-harness` pre-commit gates | installed via `gastown-4ri`; hash-pin policy this file describes |
| L2 (static)    | `go vet`, `golangci-lint`, `eslint` (web), `markdownlint-cli2` + Vale + lychee + custom frontmatter validator (docs) | all wired in CI |
| L3 (unit)      | Go `testing` against MockExecutor; React `vitest` (when adopted) | Go: full coverage of pure-function code paths. Web: **deliberate gap — see § 3** |
| L4 (integration) | Go `testing` against real `bd`/`gt` CLI in CI | covered for issue/board/graph/sync/human handlers |
| L5 (system)    | `gvid` daemon end-to-end smoke (run daemon, hit endpoints, validate JSON) | partial — handler tests act as system-level smoke today |
| L6 (E2E)       | Browser-driven flows (Playwright / similar) | **deliberate gap — see § 3** |
| L7 (acceptance) | None (no formal acceptance criteria layer) | n/a for this repo |

## 2. Calibrated investment principle

Test investment is bounded by the council's repo-type-applicability rule:
this is an internal-use Mission Control dashboard with one human user
per daemon process. The cost of a defect is "Jeremy notices the panel is
wrong and goes back to the terminal" — a graceful degradation to the
existing CLI workflow, not a production-impact incident.

Coverage targets follow from that calibration:

| Surface area | Today's target | Why this level |
|---|---|---|
| `internal/beads/` parser + adapter | 100% of public functions exercised, integration tests where MockExecutor is too thin | Wrong data here propagates through every handler; cheapest place to catch parse drift against bd version bumps |
| `internal/api/` security middleware (Origin / SessionToken / loopback) | Full threat-model coverage — every defense has at least one regression test per attack class | Per `THREAT_MODEL.md`, these are the load-bearing defenses; under-tested middleware is the worst quality-debt class |
| `internal/api/` handlers | Smoke-level — 200/503/404 path per route, response-shape assertion | Handlers are thin orchestration over the adapter; deep handler tests duplicate adapter tests without finding new bugs |
| `internal/gastown/` adapter | Pure-function tests + graceful-degradation tests against missing `gt` | `gt` is a personal CLI; CI can't depend on it being installed. Posture is "every code path tested with gt absent" |
| `web/src/` React components | **L3 deliberate gap** (see § 3) | Zero web tests today by design; backfill scheduled |
| `web/src/` E2E browser flows | **L6 deliberate gap** (see § 3) | Same |

## 3. Deliberate gaps

These gaps are documented per the council's CSO binding constraint: the
**signal** of the Intent Solutions Testing SOP is the harness install +
explicit policy here, not the coverage percentage. Gaps are real and
acknowledged; they have follow-up beads with hard deadlines.

### 3a. Web unit test coverage (L3)

**Current state:** zero unit tests under `web/src/`.

**Why it's a gap:** the React app polls every 5s and renders adapter
responses; the failure mode is "panel doesn't render" which is visible
on the first refresh. No state-changing affordances exist (read-only by
council Q2 invariant); the smallest defect class is an obviously-broken
display, not a silent data corruption.

**Follow-up:** **bead `backfill-web-tests-by-2026-07-15`** (filed alongside
this doc per Q3 binding constraint). Auto-escalates if the date slips —
the council reconvenes before any further viewer work.

**Coverage target when backfilled:**

- one smoke test per panel (Vitest + React Testing Library): renders without crash, displays mocked API data correctly
- redaction-logic unit tests on the memories panel (when `gastown-fp0`
  lands the panel) — Class A + Class B coverage per
  `000-docs/005-PP-POLICY-memories-classification-2026-05-24.md`

- 60% line coverage floor on new code only — pre-existing code is not
  retroactively gated

### 3b. End-to-end browser flows (L6)

**Current state:** no Playwright or equivalent harness.

**Why it's a gap:** the dashboard's single user is Jeremy. Adding a
Playwright suite means maintaining a CI environment that has Chrome
installed plus the daemon running, for one user's benefit. The
opportunity cost is real.

**Follow-up:** no scheduled bead. L6 will be considered if/when:

- the viewer opens to OSS contributors (currently private — VP DevRel
  council seat recommended staying private)

- a defect ships that would have been caught by L6 specifically (not by
  L4 integration). This trigger has not fired in the history of the
  repo; if it fires twice, file the bead unconditionally.

## 4. State-changing handler gate (future)

Per `gastown-hu4`'s `RequireTokenMiddleware`, every POST/PUT/PATCH/DELETE
route — when they ship — must come with an **auth-pattern test suite**
as a merge gate (CISO bifurcated-coverage binding constraint):

1. Missing token → 401 TOKEN_REQUIRED
2. Wrong-origin → 403 ORIGIN_REJECTED (delegated to OriginAllowlistMiddleware)
3. Wrong token → 401 TOKEN_INVALID
4. Successful call → handler invoked, audit log written, actor field populated

No state-changing routes exist today; this section is the schema-slot
reservation. The first POST route that lands must add this test class.

## 5. Hash pinning

The audit-harness `.harness-hash` manifest is pinned against:

- this file (`tests/TESTING.md`)
- `.markdownlint-cli2.jsonc`
- `.vale.ini`
- `lychee.toml`
- `scripts/validate-frontmatter.py`
- `THREAT_MODEL.md`
- `000-docs/005-PP-POLICY-memories-classification-2026-05-24.md`

After any AI-proposed edit to a pinned file, re-init with
`pnpm exec audit-harness init` (web/) and commit the new hash alongside
the policy change in a single commit. Pre-commit refuses AI-proposed
edits whose hash doesn't match the pin.

## 6. Coverage gates currently enforced in CI

| Gate | Workflow | Enforcement |
|---|---|---|
| `go test ./...` | `.github/workflows/ci.yaml` | hard fail |
| `go vet ./...` | implicit via `go test` | hard fail |
| `golangci-lint` | `.github/workflows/ci.yaml` lint job | hard fail |
| ESLint (web) | `.github/workflows/ci.yaml` web job | hard fail |
| TypeScript build (web) | same | hard fail |
| markdownlint-cli2 | `.github/workflows/doc-quality.yml` | hard fail (currently failing on pre-existing README/SECURITY debt — `gastown-rj5` cleans up) |
| Frontmatter validator | same | hard fail |
| Vale | same | soft (PR check only) — promotes to hard when prose-debt baseline is clean |
| Lychee | same | hard fail |

## 7. When to update this document

This file is rebuilt when any of these change:

- A new layer is added to the 7-layer taxonomy.
- The repo gains a new component (a `cmd/...` binary, a new package, a
  new web target).

- A deliberate gap from § 3 is closed.
- The council ratifies a new test-investment level.

Edits must be paired with a re-init of `.harness-hash` (see § 5).

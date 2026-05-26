# CLAUDE.md

<!-- bd-doctor-divergence: ok -->
<!-- AGENTS.md and CLAUDE.md intentionally serve different audiences:
     AGENTS.md is generic agent/bd onboarding (any harness); CLAUDE.md
     is Claude-specific project architecture, build commands, and
     doc-quality gates. The divergence is by design — bd doctor's
     "Agent Doc Divergence" warning is opted out here. -->

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Gastown Viewer Intent** is a local-first Mission Control dashboard for **Beads** (a local issue tracker with dependency support) and **Gas Town** (a multi-agent orchestrator). It provides board views, dependency graphs, agent status dashboards, molecule tracking, and convoy progress via an HTTP daemon, TUI, and React Web UI.

## Build & Development Commands

```bash

make dev              # Daemon (localhost:7070) + web (localhost:5173) in parallel
make daemon           # Daemon only
make web              # Web dev server only (Vite hot reload)
make tui              # TUI client (requires running daemon)
make build            # Build web, copy to internal/api/web_dist, then build Go binaries
make test             # Go tests + web lint
make clean            # Remove bin/, dist/, web/dist/, internal/api/web_dist/

# Go tests

go test -v ./...                         # All tests
go test -v ./internal/beads/...          # Single package
go test -v -run TestParseIssueList ./internal/beads/...  # Single test

# Web

cd web && npm run dev       # Dev server
cd web && npm run build     # TypeScript check + Vite build
cd web && npm run lint      # ESLint

# Verify daemon

curl http://localhost:7070/api/v1/health
```

## Architecture

Two adapters feed data into a single HTTP server with a security
middleware chain in front:

- **Beads Adapter** (`internal/beads/`): Shells out to `bd` CLI for issue data. Never parses `.beads/` files directly. Uses the `Executor` interface (`DefaultExecutor` for production, `MockExecutor` for tests). Surfaces `bd memories`, `bd dolt status`, `bd human list` in addition to the standard issue/board/graph routes.
- **Gastown Adapter** (`internal/gastown/`): Reads Gas Town filesystem at `~/gt` and shells to `gt` CLI. Molecules now read from `gt wisps list --json` (gt 0.9 surface); legacy `.beads/molecule.json` file reads were removed in `gastown-7fq`.
- **Security middleware** (`internal/api/security.go`): Origin allowlist (DNS-rebind + CSRF defense), session token at `~/.config/gvid/token` (mode 0600), loopback bind enforcement at `Start()`. See `THREAT_MODEL.md` for the full model.
- **Memory redaction** (`internal/api/memoryredact.go`): Applies the partner-name + secret-pattern denylists from `000-docs/005-PP-POLICY-memories-classification-2026-05-24.md` before any memory crosses the HTTP boundary.

Both adapters are interface-based for testability. The `Server` (`internal/api/server.go`) composes both and registers routes on `net/http.ServeMux` using Go 1.22+ method routing (`"GET /api/v1/issues/{id}"`).

**Data flow**: Web UI/TUI → Origin allowlist → CORS → HTTP API (gvid :7070) → Adapters → `bd`/`gt` CLI + filesystem.

**SSE**: The `SSEBroker` (`internal/api/sse.go`) manages client connections with heartbeat at `/api/v1/events`.

## Key Design Decisions

- **Fail-fast**: If `bd` not found, return 503 `BD_NOT_FOUND`. If `.beads/` not initialized, return 503 `BEADS_NOT_INIT`. Every beads handler calls `checkBeadsInitialized()` first.
- **CLI shelling, not file parsing**: Both adapters shell to their respective CLIs rather than parsing internal state files. This keeps the viewer decoupled from internal formats.
- **No external router**: Uses stdlib `net/http.ServeMux` with Go 1.22+ pattern matching. No Gin/Chi/Echo.
- **CORS + Origin allowlist**: CORS headers configured for `http://localhost:5173`; a hard Origin allowlist middleware runs outermost and rejects mismatched cross-origin requests with 403 `ORIGIN_REJECTED`. Native clients (no `Origin` header) bypass the gate by design.
- **Memories panel is read-only-forever**: Council Q2 architectural invariant. Zero state-mutating endpoints under `/api/v1/memories/*`. The bd CLI is the canonical writer. A test (`TestMemoriesHandler_NoPOSTRouteRegistered`) tripwires this rule.
- **Sync pill never errors**: `/api/v1/sync` always returns 200 with a `DoltSyncState` body; failure cases are encoded as `health: "unknown"` with a tooltip string. The header pill must never break the dashboard.

## Testing

Prefer integration tests that hit the real `bd` CLI over mocks. Parser tests (`parser_test.go`) test pure functions and need no CLI. Adapter tests should use `DefaultExecutor` against real beads state when possible. `MockExecutor` exists but is a last resort, not the default approach.

## API Routes

Two route groups defined in `server.go:registerRoutes()`:

- **Beads** (`/api/v1/`): health, issues, board, graph, events, sync, human, memories (+ search + by-key)
- **Gas Town** (`/api/v1/town/`): status, rigs, agents, convoys, molecules, mail

Graph endpoint supports `?format=json` (default) and `?format=dot` (Graphviz DOT). Memory endpoints accept `?reveal=true` to opt into un-redacted content per `005-PP-POLICY`.

## Web UI

React 19 + Vite 7 + TypeScript. Single-page app with five tab views plus a header sync pill:

- **Board**: Kanban columns from `/api/v1/board`
- **Graph**: D3.js force-directed visualization from `/api/v1/graph`
- **Gas Town**: Agent dashboard with molecules and convoys
- **Memory**: Read-only `bd memories` viewer with default-redacted content, per-card reveal toggle (does NOT persist), and `Copy bd recall <key>` CLI passthrough
- **Triage**: Read-only queue of beads with the `human` label
- **Sync pill** (in header): green/yellow/red/gray pill bound to `/api/v1/sync`

All API types and fetch functions in `web/src/api.ts`. Polls every 5 seconds; the Memory panel additionally debounces its search on input change.

## Security model

Full details in `THREAT_MODEL.md`. Short version:

- Loopback bind enforced at `Start()`; `--host=0.0.0.0` is refused with an actionable error.
- Origin allowlist middleware (`OriginAllowlistMiddleware`) rejects cross-origin requests at 403; defends against DNS rebinding and CSRF from any tab on the dev box.
- Session token (`SessionToken`) at `~/.config/gvid/token` (mode 0600). Required by `RequireTokenMiddleware` on any state-changing route. None ship today — installed for future POST routes.
- Memory content classification per `000-docs/005-PP-POLICY-memories-classification-2026-05-24.md`; redaction applied in `internal/api/memoryredact.go`.

## Beads Work Tracking

```bash

bd ready              # Show unblocked issues
bd blocked            # Show dependency graph
bd show <id>          # View issue details
```

## Doc-Quality Gates (Phase 2 pre-flight, 2026-05-23)

CI workflow `.github/workflows/doc-quality.yml` runs four gates on every PR
that touches `**/*.md` or the gate configs:

| Gate | Tool | Config | Local invocation |
|---|---|---|---|
| Markdown lint | `markdownlint-cli2` (web devDep) | `.markdownlint-cli2.jsonc` | `make markdownlint` |
| Frontmatter + filename | `scripts/validate-frontmatter.py` (Python stdlib) | inline regex | `make frontmatter` |
| Prose style | Vale 3.7.1 | `.vale.ini` (Microsoft + write-good packages) | `vale 000-docs` (CI uses `errata-ai/vale-action@reviewdog`) |
| Link check | lychee | `lychee.toml` | `lychee --config lychee.toml '**/*.md'` (CI uses `lycheeverse/lychee-action@v2`) |

The frontmatter validator enforces the Document Filing Standard v4.3
(`NNN-CC-ABCD-description.ext` for project docs; `000-CC-ABCD-...` for canonical
cross-repo standards). Legacy `NNN-XXX-` (pre-v4) and `6767-` (v4.2) prefixes
emit warnings — they are scheduled for renaming in the janitorial sweep but
don't block the gate today.

**Test-enforcement harness** — `@intentsolutions/audit-harness` is installed as
a web/ devDep. Hash manifest at `.harness-hash` (initially empty; will
be populated when `tests/TESTING.md` ships in the closeout bead).
`make audit-harness-verify` runs the manifest check locally.

**Architectural invariant (Phase 2 council decision Q2):** the `bd memories`
panel is **read-only-forever**. No `POST/PUT/PATCH/DELETE` endpoints under
`/api/v1/memories/*`. The bd CLI is the canonical writer; the viewer Edit
affordance shells out to `bd remember <id>` or copies the command to clipboard.
This is documented in `THREAT_MODEL.md` (lands in `gastown-hu4` bead).

See `000-docs/004-AT-DECR-gastown-viewer-option-b-council-2026-05-23.md` for
the full council decision record.

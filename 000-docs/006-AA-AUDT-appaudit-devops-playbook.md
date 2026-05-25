# Gastown Viewer Intent: Operator-Grade System Analysis

*Generated: 2026-05-25*
*Version: v0.5.0 (master @ 42b3f4c)*

---

## 1. This System in 5 Minutes

`gastown-viewer-intent` is a **local-first Mission Control dashboard** for two
adjacent tools Jeremy already runs: **Beads** (`bd`, a local-first issue tracker
with dependency graphs and a Dolt-backed memory layer) and **Gas Town** (`gt`, a
multi-agent orchestrator that runs AI workers in tmux-managed rigs out of
`~/gt`). The viewer doesn't *replace* either tool — it surfaces their state in
a single browser tab so the engineer doesn't alt-tab to a terminal every two
minutes to run `bd ready` or `bd dolt status`.

The product is a single Go daemon (`gvid`) plus a React/Vite single-page app.
The daemon binds **loopback only** (`127.0.0.1:7070`), shells out to the `bd`
and `gt` CLIs for data, and serves the embedded web UI via `go:embed`. A
secondary TUI client (`gvi-tui`) exists for terminal-first workflows but is
minimal — only basic navigation today; richer views are tracked as a roadmap
epic.

v0.5.0 shipped the **Phase 2 daily-value refresh** ratified by the 2026-05-23
ISEDC council (see `000-docs/004-AT-DECR-...`). That burst added three
read-only surfaces (Memory panel with content classification redaction, Dolt
sync widget in the header, Human Triage queue), hardened the daemon (Origin
allowlist middleware + per-session token + non-loopback bind refusal +
`THREAT_MODEL.md`), preserved `bd defer --until` round-tripping, and migrated
the molecules data source from the retired `<workDir>/.beads/molecule.json` to
the gt 0.9 wisps surface.

The **biggest risk** is that the dashboard is single-user-on-localhost by
design — there is no auth model for multi-user access, no horizontal scaling,
no clustering. If Jeremy needs to share the dashboard, the security model is
"don't"; the threat model document explicitly punts on every threat class that
isn't single-user-local. The second-biggest risk is the **bd/gt CLI coupling**:
the viewer shells to those binaries and parses their JSON output. When `bd`
ships a JSON-schema change (the `defer_until` field appearing in 1.0.4 is the
documented example), the viewer must catch up explicitly. Council Q1 decided
this is honest lag with a supported-version matrix rather than a chase-cadence
SLO — except for security-flagged upstream releases, which have a 48hr
fast-path.

---

## 2. Executive Summary

### What It Does

Mission Control dashboard for a personal multi-agent workspace. Surfaces
issue tracker state (board, dependency graph, deferred-with-date),
agent-orchestrator state (rigs, agents, convoys, molecules / wisps),
memory layer (with partner-name + secret-pattern redaction), dolt sync
state, and the human-triage queue — all read-only by architectural
invariant.

Implementation is **shipped through v0.5.0**. All Phase 2 surfaces are
on master. The audit-harness + doc-quality CI gates are wired. Repo is
public at `github.com/jeremylongshore/gastown-viewer-intent`; binaries
distributed via GitHub Releases and `go install`. Homebrew tap path is
currently broken (HOMEBREW_TAP_TOKEN issue, tracked as `gastown-die`).

The biggest **open risk** is the test debt on the web side — zero
Vitest/RTL coverage on the React components. Tracked by `gastown-6nw`
with a hard-dated 2026-07-15 backfill commitment and auto-escalation.

### Operational Status

| Environment | Status | Uptime Target | Release Cadence | Last Deploy |
|---|---|---|---|---|
| Production | n/a (single-user local) | — | tag-driven via goreleaser | v0.5.0 / 2026-05-25 |
| Staging | n/a | — | — | — |
| Local Dev | `make dev` | best-effort | continuous on master | continuous |

### Technology Stack

| Category | Technology | Version | Purpose |
|---|---|---|---|
| Language (server) | Go | 1.24.0 | Daemon + TUI |
| Language (web) | TypeScript | 5.9.x | Web UI |
| Web bundler | Vite | 7.x | Dev server + production build |
| Web framework | React | 19.x | UI components |
| Visualization | D3.js | 7.x | Force-directed dependency graph |
| TUI framework | Bubble Tea | 1.3.10 | Terminal client |
| HTTP routing | net/http stdlib | Go 1.22+ | Pattern routing (`GET /api/v1/...{id}`) |
| Build/release | goreleaser | v2 | Multi-arch binaries + .deb/.rpm/.apk |
| CI | GitHub Actions | — | ci.yaml + doc-quality.yml + release.yaml |
| Doc-quality | markdownlint-cli2 + Vale + lychee + custom Python validator | — | Repo-wide MD + filename gate |
| Test enforcement | @intentsolutions/audit-harness | npm latest | L1 hooks + escape-scan + hash-pinning |
| Upstream — issues | `bd` (Beads) | 1.0.4 (tested) | Source of issue data via JSON shell |
| Upstream — agents | `gt` (Gas Town) | 0.9.0 (tested) | Source of agent/convoy/wisp data |

---

## 3. Architecture

### Stack (Detailed — Why This)

| Layer | Technology | Why This |
|---|---|---|
| HTTP routing | stdlib `net/http.ServeMux` | Go 1.22+ pattern matching gave us `{id}` parameters without pulling Gin/Chi/Echo. Zero external HTTP dep. |
| Embed | `go:embed` for `web/dist/` | Single binary distribution — `gvid` is one file, the web UI ships inside it. Removes the "did you run npm build?" failure mode. |
| Adapter | CLI shelling (not file parsing) | Council architectural invariant. The viewer never parses `.beads/`/`~/gt/` internal formats; it shells to `bd`/`gt` and consumes their JSON. Decouples the viewer from upstream's internal storage choices. |
| Memory redaction | Server-side string replacement | Class A (partner names) + Class B (secret-pattern prefixes) replaced with `[REDACTED ...]` before the response leaves the daemon. Raw bytes never reach the rendered HTML when redaction is in effect. |
| Auth | Per-session bearer token at `~/.config/gvid/token` mode 0600 | Generated on every daemon start; constant-time compare via `crypto/subtle`; required by future state-mutating routes (none ship in v0.5.0). |
| CORS + Origin gate | Hard 403 on mismatch | Distinct from CORS — CORS only tells the browser to drop the response; the Origin allowlist refuses the request server-side so confidential data never leaves the process. Defends against DNS rebinding. |
| Loopback enforcement | Startup-time refusal of non-loopback bind | `IsLoopbackHost("")` returns **false** — empty host binds 0.0.0.0 in net/http and silently bypassed the restriction in an earlier draft (caught by Gemini code review). |

### System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      Gastown Viewer Intent                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│   │   gvi-tui    │      │   Web UI     │      │ curl / TUI   │  │
│   │  (Bubbletea) │      │ (React+Vite) │      │ native       │  │
│   └──────┬───────┘      └──────┬───────┘      └──────┬───────┘  │
│          │                     │                     │          │
│          │                     ▼                     │          │
│          │            Origin allowlist (403 reject)  │          │
│          │                     │                     │          │
│          └─────────────────────┼─────────────────────┘          │
│                                ▼                                 │
│                    ┌───────────────────────┐                    │
│                    │       gvid Daemon     │                    │
│                    │   127.0.0.1:7070      │                    │
│                    │  + Session token gate │                    │
│                    │  + Memory redaction   │                    │
│                    └───────────┬───────────┘                    │
│                                │                                 │
│              ┌─────────────────┼─────────────────┐              │
│              ▼                                   ▼              │
│   ┌───────────────────────┐         ┌───────────────────────┐  │
│   │   Gastown Adapter     │         │    Beads Adapter      │  │
│   │  exec.CommandContext  │         │  exec.CommandContext  │  │
│   │  `gt convoy/wisps/    │         │  `bd list/show/board/ │  │
│   │   mail list --json`   │         │   memories/dolt/      │  │
│   │  + ~/gt FS reads      │         │   human list --json`  │  │
│   └───────────┬───────────┘         └───────────┬───────────┘  │
│               │                                 │               │
│               ▼                                 ▼               │
│   ┌───────────────────────┐         ┌───────────────────────┐  │
│   │     gt (gas town)     │         │       bd (beads)      │  │
│   │  ~/gt + wisp SQLite   │         │  .beads/ Dolt store   │  │
│   │  (rigs, agents,       │         │  (issues, memories,   │  │
│   │   convoys, mail)      │         │   dependencies)       │  │
│   └───────────────────────┘         └───────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### The Critical Path — A Browser Polls `/api/v1/memories`

1. Web UI at `localhost:5173` (dev) or `localhost:7070` (prod-embedded) fires
   a `fetch('/api/v1/memories')` on a 5-second poll.
2. Browser auto-sets `Origin: http://localhost:5173`.
3. **`OriginAllowlistMiddleware`** (`internal/api/security.go`) checks Origin
   against `Config.CORSOrigins`. Match: pass. Mismatch: 403
   `ORIGIN_REJECTED`, handler never runs.
4. **`corsMiddleware`** sets `Access-Control-Allow-Origin` response header.
5. **`loggingMiddleware`** records start time.
6. ServeMux dispatches to `handleMemories` (`internal/api/handlers.go`).
7. `checkBeadsInitialized` calls `adapter.IsInitialized(ctx)` — shells
   `bd status`, expects exit 0.
8. `adapter.Memories(ctx)` shells `bd memories --json`. bd 1.0.4 returns a
   flat object: `{"schema_version": 1, "<key>": "<content>", ...}`.
9. `ParseMemories` (`internal/beads/parser.go`) parses the flat shape,
   returns `[]model.Memory` sorted by key.
10. **`RedactMemories`** (`internal/api/memoryredact.go`) walks each
    memory's content, applies Class A partner-name substring/whole-word
    matches and Class B secret-prefix matches, replaces with
    `[REDACTED partner-name]` / `[REDACTED secret]`, populates
    `Memory.Redacted` + `Memory.RedactionMarkers`.
11. Response JSON-encoded; status 200; ~5–15ms round trip locally.

**Failure points marked**:

- `bd` not on PATH → 503 `BD_NOT_FOUND` (caught at step 7).
- `.beads/` not initialized → 503 `BEADS_NOT_INIT` (step 7).
- bd shell exec timeout → propagated up the stack as 500.
- Parse error → 500 with body containing error string.
- Redaction layer never errors — defensive nil-memory guard.

### Dependency Graph

```
gvid daemon
├── bd (PATH binary) — REQUIRED for all /api/v1/issues, /board, /graph,
│                       /memories, /human routes
├── gt (PATH binary) — OPTIONAL; /api/v1/town/* returns nil/empty when missing
├── ~/.config/gvid/token — generated at startup
└── ~/gt/ filesystem — read for rig/agent/mail; absent → empty town view

web UI
├── gvid daemon (HTTP) — REQUIRED to render anything
└── d3 + react — bundled

gvi-tui
└── gvid daemon (HTTP) — REQUIRED
```

When `bd` is unavailable, every issue-side surface returns 503. When `gt` is
unavailable, every town-side surface returns 200 with empty/null payloads
(gas town view degrades gracefully). The dashboard remains usable for pure
bd workflows on machines without gt installed.

---

## 4. Design Decisions & Tradeoffs

### Decision Log

#### Read-only memories panel (Council Q2 architectural invariant)

- **Chosen**: zero POST/PUT/PATCH/DELETE endpoints under `/api/v1/memories/*`. Edit button shells terminal to `bd remember <id>` or copies the command to clipboard.
- **Over**: a full read-write panel with in-UI memory editing (CMO's lone-dissent council position).
- **Because**: dual-write consistency between the viewer's write path and the bd CLI's write path against the same Dolt store invites last-writer-wins data loss. The memories layer is the densest concentration of sensitive context (partner names, deal terms, observed-pasted tokens) — any unauthenticated mutation surface on localhost is a CSRF + DNS-rebind target.
- **Cost**: in-UI editing UX. Users alt-tab to a terminal for `bd remember`.
- **Revisit when**: a single-writer-lock protocol is designed AND the auth-token gate is wired to all mutating routes AND a dedicated bead reviews the architectural identity question. CMO's compromise: the internal API contract is designed write-capable so a future read-write decision is a UI-only change.

#### CLI shelling, not file parsing

- **Chosen**: viewer adapters shell to `bd` and `gt` CLIs and consume their JSON output.
- **Over**: directly reading `.beads/` SQLite/Dolt files + `~/gt/` JSON sidecars.
- **Because**: upstream's internal storage is a moving target. gt 0.9 retired the legacy `.beads/molecule.json` for root-level wisp SQLite; viewer code that read the old file would have silently shown empty molecules. Shelling decouples the viewer's surface from upstream's storage decisions.
- **Cost**: one subprocess per request; harder to mock for unit tests; serializes Go ↔ JSON for every read.
- **Revisit when**: an upstream offers a stable Go SDK with semver guarantees.

#### Loopback bind enforcement at startup

- **Chosen**: refuse to bind any non-loopback host at `Server.Start()`. Empty host (`":7070"`) treated as non-loopback because Go's `http.ListenAndServe` binds 0.0.0.0 for that case.
- **Over**: allowing `--host 0.0.0.0` for "trusted networks" with a warning.
- **Because**: the threat model assumes single-user-local. A `--host 0.0.0.0` typo on a corp WiFi exposes every memory + every issue body to anyone on the LAN. Refusing at startup with an actionable error is cheaper than the incident.
- **Cost**: explicit escape hatch (`Config.DisableLoopbackCheck`) needed for ephemeral container tests.
- **Revisit when**: a multi-user deployment model is designed (currently out of scope per council Q0).

#### "Pill must never break the dashboard" — `/api/v1/sync` always 200

- **Chosen**: `DoltSyncState` adapter never returns a Go error; every failure class maps to `Health="unknown"` in the response body.
- **Over**: standard error propagation (500 on adapter failure).
- **Because**: the sync pill is the visible smoke alarm. If it crashes the whole dashboard view, it fails its purpose. Engineer loses sync visibility *and* the rest of the dashboard.
- **Cost**: a Go contract anomaly (signature `(*State, error)` where err is always nil). Defensive 500 path in handler documents the contract.
- **Revisit when**: an actual upstream change makes the unknown-vs-red distinction load-bearing.

#### Visibility-weighted test investment (CSO council framing)

- **Chosen**: install `@intentsolutions/audit-harness` repo-wide + per-handler smoke tests; web/TUI unit-test coverage deferred to `gastown-6nw` (auto-escalates 2026-07-15).
- **Over**: full Vitest + React Testing Library coverage in v0.5.0.
- **Because**: this is internal-use Mission Control with one human user. Defect cost is "Jeremy notices the panel is wrong and alt-tabs to the terminal" — graceful degradation, not a production incident.
- **Cost**: zero web-side unit tests today. A regression in a React component is visible-but-uncaught-by-CI.
- **Revisit when**: backfill bead deadline (2026-07-15) fires.

### What Was Deliberately Not Built

- **Authentication for multi-user**: dashboard is single-user-local. The threat model document explicitly punts.
- **In-UI memory editing**: see Council Q2 above.
- **POST handlers for human-triage `respond` / `dismiss`**: read-only this burst. POST routes will require the auth-token gate that's already installed but unwired.
- **SSE for all endpoints**: SSE is reserved for the dolt-sync widget specifically (per CTO recommendation). Other surfaces poll at 5s.
- **Test backfill on the web side**: tracked separately (`gastown-6nw`).
- **TUI feature parity with the web UI**: the TUI is intentionally a thin navigator; richer views are roadmap (`gastown-ey8`).
- **gt features beyond Convoys/Mail/Wisps**: no Dogs/patrols, no merge queue, no mail threading. Council deferred to a future burst.

### Assumptions the Architecture Rests On

- The engineer has `bd` 1.0.4+ on PATH. Lower versions may emit different JSON shapes (specifically: pre-1.0 lacked `defer_until`; the viewer would silently drop the field).
- The web UI runs in a modern browser with Same-Origin Policy enforced. The Origin allowlist is the floor; SOP bypass would compromise the whole defense.
- `~/.config/gvid/token` is in a directory only the engineer's UID can read. If another user has read access to that dir, the token is exposed.
- The engineer keeps the dashboard tab closed during partner demos / screen-shares (operational guidance in THREAT_MODEL.md, not code-enforced).

---

## 5. Directory Structure

### Layout

```
gastown-viewer-intent/
├── cmd/
│   ├── gvid/              # Daemon entrypoint (one main.go)
│   └── gvi-tui/           # TUI client entrypoint
├── internal/
│   ├── api/               # HTTP handlers, security middleware, redaction
│   │   ├── server.go      # Server struct, registerRoutes, Handler() middleware chain, Start() loopback+token gate
│   │   ├── handlers.go    # All read handlers (issues, board, graph, sync, human, memories)
│   │   ├── gastown_handlers.go  # /api/v1/town/* handlers
│   │   ├── security.go    # OriginAllowlistMiddleware + SessionToken + IsLoopbackHost + RequireTokenMiddleware
│   │   ├── memoryredact.go      # Class A + Class B redaction transform
│   │   ├── static.go      # go:embed mount for web/dist/
│   │   └── sse.go         # SSEBroker (heartbeat at /api/v1/events)
│   ├── gastown/           # Gas Town adapter (gt CLI + ~/gt filesystem)
│   │   ├── adapter.go     # FSAdapter: Town/Rigs/Agents/Convoys/Molecules/Mail
│   │   └── types.go       # Town/Rig/Agent/Convoy/Molecule types
│   ├── beads/             # Beads adapter (bd CLI)
│   │   ├── adapter.go     # CLIAdapter: ListIssues/GetIssue/Board/Graph/Sync/Human/Memories
│   │   ├── parser.go      # JSON parsers (BDIssue, dolt status, memories flat-object)
│   │   ├── executor.go    # Executor interface + DefaultExecutor + MockExecutor
│   │   └── errors.go      # BDNotFoundError, NotInitializedError, NotFoundError, ParseError
│   ├── model/             # Domain types (no external deps)
│   │   ├── issue.go       # Issue + IssueSummary + Status (incl. StatusDeferred)
│   │   ├── memory.go      # Memory + MemoriesResponse + redaction-marker fields
│   │   ├── sync.go        # DoltSyncState + DoltRemote + DoltHealth enum
│   │   ├── board.go       # Board + Column
│   │   ├── graph.go       # GraphNode + GraphEdge + EdgeType
│   │   └── event.go       # SSE event types
│   └── tui/               # TUI client (Bubble Tea Model + HTTP client)
│       ├── model.go       # Tea Model, keybindings, view dispatch
│       └── client.go      # HTTP client against daemon
├── web/                   # React + Vite frontend (embedded via go:embed)
│   ├── src/
│   │   ├── App.tsx        # All panels, view-mode router, polling loop
│   │   ├── api.ts         # Fetch wrappers + TypeScript types
│   │   └── components/DependencyGraph.tsx  # D3 force-directed
│   ├── package.json       # devDeps: audit-harness, markdownlint-cli2
│   └── vite.config.ts
├── 000-docs/              # Per /doc-filing v4.3 — project docs (this file lives here)
├── tests/TESTING.md       # Calibrated-investment test policy (CSO closeout artifact)
├── deploy/                # gvid.service (systemd) + install.sh
├── scripts/validate-frontmatter.py  # Custom MD frontmatter validator
├── THREAT_MODEL.md
├── .goreleaser.yaml       # Multi-arch + .deb/.rpm/.apk + (disabled) brews
├── .github/workflows/{ci,doc-quality,release}.yaml
└── Makefile               # dev/daemon/web/tui/build/test/doc-quality
```

### Load-Bearing Files

| Path | Role | Why it's load-bearing |
|---|---|---|
| `internal/api/server.go` | HTTP server bootstrap + route table + middleware chain | All requests pass through `Handler()`'s composition; mis-ordering Origin→CORS→logging breaks the threat model |
| `internal/api/security.go` | Origin allowlist + session token + loopback check | The entire threat-model defense lives here; the `IsLoopbackHost("")` regression that almost shipped was caught by Gemini code review |
| `internal/api/memoryredact.go` | Class A + Class B classification policy enforcement | Server-side redaction layer that ensures partner names + token-shaped strings never reach the rendered HTML by default |
| `internal/beads/parser.go` | `bd` JSON parsing — `BDIssue` struct, `ParseMemories`, `ParseDoltStatus` | bd schema evolution propagates through here; the `defer_until` field added in 1.0.4 was the fix in v0.5.0's foundation-gates work |
| `internal/beads/adapter.go` | The `Adapter` interface that the API layer depends on | Adding a new bd-side surface means extending this interface (which is small and stable by design) |
| `internal/model/memory.go` | The `Memory` + `MemoriesResponse` types | If `Memory.Content` ever serializes via JSON without first passing through `RedactMemory`, the redaction layer is bypassed |
| `000-docs/004-AT-DECR-...` | The ratified council decision | The "why are things this way" reference for every architectural invariant — read this before proposing a write-path on memories |

---

## 6. Getting Started

### Prerequisites

| Tool | Version | Install | Verify |
|---|---|---|---|
| Go | 1.22+ | `apt install golang` / `brew install go` | `go version` |
| Node.js | 20+ | `apt install nodejs npm` / `brew install node` | `node --version` |
| `bd` (Beads CLI) | 1.0.4+ | `cargo install beads-cli` or per upstream | `bd --version` |
| `gt` (Gas Town CLI) | 0.9+ (optional) | local install | `gt --version` |
| `make` | any | system | `make --version` |

### Zero to Running

1. `git clone https://github.com/jeremylongshore/gastown-viewer-intent && cd gastown-viewer-intent`
2. `make build` — builds web first, copies to `internal/api/web_dist/`, builds Go binaries to `bin/`. Expect `Build complete. Binaries in ./bin/`
3. `./bin/gvid` — daemon starts on `localhost:7070`, generates session token at `~/.config/gvid/token`. Expect log line `Session token: /home/.../.config/gvid/token (mode 0600; required by future state-changing endpoints)`
4. Open `http://localhost:7070` — UI loads, polls daemon every 5s

For development with hot reload: `make dev` runs Vite at `localhost:5173` with the daemon proxied at `localhost:7070`.

### Common Setup Problems

| Symptom | Cause | Fix |
|---|---|---|
| 503 `BD_NOT_FOUND` on every request | `bd` not on PATH or not installed | Install bd, restart daemon |
| 503 `BEADS_NOT_INIT` | Daemon's working dir isn't a beads repo | `cd` to a `.beads/`-having dir before running gvid, or pass `--dir` |
| `refusing to bind non-loopback host "0.0.0.0"` on start | User tried `--host 0.0.0.0` | Use `localhost` (or set `Config.DisableLoopbackCheck` in code for container tests only) |
| Gas Town panel empty + "town not found at /home/.../gt" | `~/gt` doesn't exist | Either install gt + run `gt init`, or accept the empty view (the rest of the dashboard still works) |
| Memory panel shows `[REDACTED partner-name]` everywhere | Working as intended — that's the 005-PP-POLICY redaction | Click per-card Reveal button for one memory at a time |
| Web build fails on fresh clone | `internal/api/web_dist/` doesn't exist for `go:embed` | `make ensure-web-dist` creates a stub; `make build` builds the real one |

---

## 7. Operations

### Command Map

| Task | Command | Notes |
|---|---|---|
| Run locally (daemon + web dev) | `make dev` | Vite at :5173, daemon at :7070 |
| Run daemon only | `make daemon` | After `make build` |
| Run TUI client | `make tui` | Requires daemon running |
| Run all tests | `go test ./...` + `cd web && npm run build` | |
| Lint Go | `golangci-lint run` (CI uses this) | |
| Lint web | `cd web && npm run lint` | ESLint |
| Doc-quality gates | `make doc-quality` | Runs markdownlint + frontmatter validator (Vale + lychee are CI-only) |
| Build everything | `make build` | web first, then Go binaries to `bin/` |
| Tag release | `git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z` | Triggers goreleaser via `.github/workflows/release.yaml` |
| View logs | `journalctl --user -u gvid -f` | If installed as systemd user unit per `deploy/gvid.service` |
| Rollback | `git checkout vX.Y.(Z-1) && make build && ./bin/gvid` | No deploy automation — local-first |

### Deployment

`gvid` is not deployed to a server in production — it runs locally on each
engineer's machine. The "release" process is:

1. Phase 2 burst lands on master (multiple PRs).
2. `git tag -a v0.5.0 -m "..."` + `git push origin v0.5.0`.
3. GitHub Actions `release.yaml` fires goreleaser.
4. Binaries land at `github.com/jeremylongshore/gastown-viewer-intent/releases/tag/vX.Y.Z` (linux/darwin × amd64/arm64 tarballs + .deb/.rpm/.apk).
5. Users: `go install ...@vX.Y.Z` OR direct download.

**Pre-flight checklist before tagging:**

- [ ] All Phase work merged to master
- [ ] CI green (ci.yaml + doc-quality.yml)
- [ ] CHANGELOG / release notes drafted in the tag annotation
- [ ] Goreleaser config (`.goreleaser.yaml`) points at `jeremylongshore/` (transferred-alias bug fixed in `gastown-vjt`)

**Rollback protocol**: there is no deploy to roll back. If a binary is bad,
publish vX.Y.(Z+1) with the fix; users `go install` or download the new
binary.

### Monitoring & Alerting

- Dashboards: **not configured** — local-first, no centralized observability.
- SLIs/SLOs: **not defined** — single-user product.
- On-call: **not established** — Jeremy is on-call for himself.

### Incident Response

| Severity | Definition | Response Time | Playbook |
|---|---|---|---|
| P0 | Daemon won't start | Self-discovered | `journalctl --user -u gvid` → check loopback bind log; verify `bd`/`gt` on PATH |
| P1 | A panel renders wrong data | Self-discovered | Check `bd ... --json` output directly; compare to adapter expectations |
| P2 | Slow / flaky | Self-discovered | Restart daemon; check tmux session count if gt is involved |

---

## 8. Things That Will Bite You

### 8.1 Empty host = 0.0.0.0 in Go's net/http

- **Symptom**: dashboard suddenly reachable from the LAN despite "loopback" intent.
- **Cause**: passing `--host ""` (or programmatically setting `Config.Host = ""`) caused `http.ListenAndServe(":7070", ...)` which binds all interfaces.
- **Fix**: shipped in v0.5.0 — `IsLoopbackHost("")` returns false, daemon refuses to start. Regression test in `internal/api/security_test.go`.
- **Prevention**: don't pass an empty host. The default `localhost` is correct.

### 8.2 Web autocomplete remembers partner names from the search box

- **Symptom**: typing "k" in the memories search box autocompletes "Kobiton" days later.
- **Cause**: browser remembers form input.
- **Fix**: `autocomplete="off"` on every input element in the viewer per 005-PP-POLICY § 4.
- **Prevention**: any new input element in `web/src/` must carry the same attribute. There is no enforcement test for this today (gap; tracked implicitly in `gastown-6nw`).

### 8.3 Reveal-mode memory in a screen recording

- **Symptom**: a partner sees their own name on screen because you clicked Reveal during a demo.
- **Cause**: per-card Reveal is real; if the screen-share is recording, the un-redacted bytes are in the recording.
- **Fix**: navigate away from the Memory tab during demos; the persistent banner exists to remind.
- **Prevention**: don't open the Memory tab during partner calls. Or use `bd recall <key>` in a private terminal instead.

### 8.4 bd version drift breaking JSON parsing

- **Symptom**: empty memories panel even though `bd memories` shows entries.
- **Cause**: bd shipped a JSON schema change the viewer hasn't caught up to.
- **Fix**: refresh `internal/beads/parser.go` parsers against the new shape; bump the supported-version matrix in README.
- **Prevention**: the supported-version matrix in README documents which bd versions are tested. Council Q1 endorses honest-lag, not chase-cadence.

### 8.5 gt absence reports as "Town not found"

- **Symptom**: Gas Town tab shows "town not found at /home/.../gt".
- **Cause**: working as designed — `~/gt` doesn't exist on this machine.
- **Fix**: not a bug. Either install gt and `gt init`, or ignore the Gas Town tab.
- **Prevention**: don't surprise users by hiding this — the explicit error message *is* the signal.

### 8.6 Old draft release blocking new release

- **Symptom**: tag exists but no published release; goreleaser failed.
- **Cause**: a previous release attempt left a draft release tied to the tag.
- **Fix**: `gh api -X DELETE repos/jeremylongshore/gastown-viewer-intent/releases/{id}`, then move tag forward + force-push. v0.5.0 recovery already used this path.
- **Prevention**: the `gastown-vjt` fix removed the 307-redirect cause; this should no longer recur. If it does, the workaround is documented.

---

## 9. Security & Access

### Access Control

| Role | Purpose | Permissions | MFA |
|---|---|---|---|
| Engineer (local) | Use the dashboard | Read all routes; write to `bd` via terminal | n/a (local user account) |
| Future state-mutating clients | Hit POST routes that don't yet exist | Bearer token from `~/.config/gvid/token` | n/a |
| External | None | None | n/a (loopback bind) |

### Secrets

- **Where**: session token at `~/.config/gvid/token` mode 0600, owner-only.
- **Rotation**: regenerated on every daemon start. To rotate manually, restart the daemon.
- **Emergency access**: delete the file; daemon will refuse to authenticate the gate but read endpoints still work without it.

### Honest Security Assessment

| Defense | Implemented? | Tested? | Threat addressed |
|---|---|---|---|
| Loopback bind enforcement | ✅ | ✅ (13 test cases) | Accidental LAN exposure |
| Origin allowlist middleware | ✅ | ✅ (5 disallowed-origin cases) | DNS rebinding, cross-origin CSRF |
| Session token (constant-time compare) | ✅ generated + persisted | ✅ (Bearer, X-Gvid-Token, nil-token, wrong-token) | Same-machine other-process mutations |
| Token middleware wired to routes | ⚠️ middleware exists, no routes wire it yet | ✅ middleware tests | Future POSTs only — none ship in v0.5.0 |
| Memory content classification redaction | ✅ Class A + Class B | ✅ (4 partners × multiple cases, 10 secret prefixes, false-positive defense) | Partner-name + token-shape exposure in browser storage |
| Audit log of reveal events | ❌ not implemented | n/a | Open follow-up in THREAT_MODEL.md |
| Multi-user auth | ❌ explicitly out of scope | n/a | n/a |
| TLS | ❌ not needed for loopback | n/a | n/a |

---

## 10. Cost & Performance

### Monthly Costs

| Resource | Cost | Notes |
|---|---|---|
| Hosting | $0 | Local-first; no servers |
| GitHub | $0 | Public repo, free Actions |
| Domains | $0 | None |
| **Total** | **$0** | |

### Performance

- Latency: handlers complete in single-digit ms once `bd`/`gt` return. The bottleneck is the upstream CLI shell — `bd list --json` against a large beads store can take 50-100ms.
- Throughput: web UI polls every 5s; one user; no thundering herd.
- Error budget: not defined.

### Scaling Limits

- **Single-user-local** — the whole product breaks the moment you try to share the dashboard. No multi-user, no auth model, no clustering.
- **bd JSON shell tax** — every API request triggers ≥1 subprocess fork+exec. Fine for one user polling every 5s; would not scale to 100 users.
- **Memories panel size** — current implementation sorts the full memories array in memory on every request. Fine for the current ~10-50 memories Jeremy has; would need streaming if memories ever grew to thousands.

---

## 11. Current State

### What's Working

- All Phase 2 surfaces shipped (Memory panel, Sync pill, Triage queue) ✅ v0.5.0
- Daemon hardening (Origin allowlist, session token, loopback bind) ✅ tested
- Memory redaction (Class A partner names + Class B secret prefixes) ✅ 30+ sub-tests
- Foundation fixes (`defer --until` round-trip, gt 0.9 wisps migration) ✅
- Doc-quality CI (markdownlint + frontmatter validator) ✅ green on master
- v0.5.0 GitHub Release with 11 binary assets ✅
- `THREAT_MODEL.md` + `tests/TESTING.md` + AT-DECR ✅ filed

### What Needs Attention

- **[Medium]** Zero Vitest/React Testing Library coverage on `web/src/` → Impact: regression in a React component is visible-but-uncaught-by-CI → Fix: `gastown-6nw` backfill bead with hard deadline 2026-07-15 and auto-escalation if missed.
- **[Low]** Vale + lychee currently `continue-on-error: true` (advisory) → Impact: spelling + link-check issues don't block CI; vocab cleanup not done → Fix: future PR can add a project Vale vocab accept-list + fix the legacy API doc references.
- **[Low]** Homebrew tap install path broken (HOMEBREW_TAP_TOKEN 401) → Impact: `brew install gvid` doesn't work; `go install` and direct download both work → Fix: `gastown-die` bead tracks token rotation.
- **[Low]** Audit-log path for memory reveal events not implemented → Impact: no record of which memories were revealed when → Fix: open follow-up in THREAT_MODEL.md.

### Implementation Status

| Component | Status | Evidence |
|---|---|---|
| gvid daemon (HTTP server) | ✅ shipped | `cmd/gvid/main.go` builds; all v0.5.0 routes tested |
| Beads adapter (CLI shelling) | ✅ shipped | `internal/beads/` + tests against MockExecutor + live bd |
| Gas Town adapter | ✅ shipped | `internal/gastown/` + graceful degradation when gt absent |
| Memory redaction layer | ✅ shipped | `internal/api/memoryredact.go` + 12 test groups |
| Security middleware | ✅ shipped | `internal/api/security.go` + 16 tests |
| Web UI (React) | ✅ shipped | `web/src/App.tsx` — 5 tabs (Board / Graph / Gas Town / Memory / Triage) + sync pill |
| TUI (gvi-tui) | ⚠️ minimal | Basic nav only; richer views = `gastown-ey8` roadmap epic |
| Audit harness install | ✅ shipped | `web/` devDep + `.harness-hash` manifest |
| Goreleaser release pipeline | ✅ shipped (brews disabled) | v0.5.0 published with 11 assets |
| Web unit tests | ❌ deferred | Bead `gastown-6nw` |
| Homebrew tap | ❌ broken | Bead `gastown-die` |

---

## 12. Roadmap

### Week 1 — Stabilization

- Rotate HOMEBREW_TAP_TOKEN, uncomment brews block in `.goreleaser.yaml`, tag v0.5.1 to publish to brew tap (or accept brew install will stay broken).
- Measure CFO outcome metric (alt-tabs/day before vs after Phase 2) — requires terminal session log review.

### Month 1 — Foundation

- Start web unit test backfill (`gastown-6nw`) — install Vitest + RTL, write smoke tests per panel, target 60% line coverage on new code.
- Vale vocab + lychee URL cleanup so doc-quality gates can become hard (not advisory).
- Audit log path for memory reveal events.

### Quarter 1 — Strategic

- TUI build-out (`gastown-ey8` epic) — file child tasks per planned view (Insights, History, Actionable, Flow Matrix, Label Dashboard, Attention View) and decide which actually pay off.
- Decide cadence on gt 1.0 when it ships — supported-version matrix update, possible security-fast-path drill.
- Re-open Q2 if and only if a single-writer-lock protocol is designed for memories panel (CMO's compromise).

---

## 13. Quick Reference

### URLs

| Resource | URL |
|---|---|
| Repo | [github.com/jeremylongshore/gastown-viewer-intent](https://github.com/jeremylongshore/gastown-viewer-intent) |
| Releases | [Releases page](https://github.com/jeremylongshore/gastown-viewer-intent/releases) |
| v0.5.0 | [v0.5.0 release](https://github.com/jeremylongshore/gastown-viewer-intent/releases/tag/v0.5.0) |
| Council Decision Record | `000-docs/004-AT-DECR-gastown-viewer-option-b-council-2026-05-23.md` |
| Memory classification policy | `000-docs/005-PP-POLICY-memories-classification-2026-05-24.md` |
| Threat model | `THREAT_MODEL.md` |
| Testing policy | `tests/TESTING.md` |

### First-Week Checklist

- [ ] `git clone` + `make build`
- [ ] Read `README.md` v0.5.0 highlights section
- [ ] Read `THREAT_MODEL.md` — 5-minute re-readable
- [ ] Read `000-docs/004-AT-DECR-...` for the architectural-invariant context
- [ ] Run `./bin/gvid` against a real `~/.beads/` workspace; click through all 5 tabs
- [ ] Verify the sync pill renders (`/api/v1/sync` returns 200 with health value)
- [ ] Verify a memory containing the partner-name fixture redacts by default (per 005-PP-POLICY tests)

---

## Appendices

### A. Glossary

- **bd** — Beads CLI, the local-first issue tracker the viewer surfaces.
- **gt** — Gas Town CLI, the multi-agent orchestrator.
- **wisp** — gt 0.9's renamed "molecule" — a workflow instance with steps.
- **gvid** — the daemon binary (HTTP server + embedded web).
- **gvi-tui** — the TUI client binary.
- **AT-DECR** — Architecture/Tech category, Decision Record doc type (per /doc-filing v4.3).
- **ISEDC** — Intent Solutions Executive Decision Council, the 7-seat adversarial review pattern.

### B. Reference Links

- ISEDC skill: `~/.claude/skills/exec-decision-council/`
- /doc-filing v4.3: `~/.claude/skills/doc-filing/`
- @intentsolutions/audit-harness: [`@intentsolutions/audit-harness`](https://www.npmjs.com/package/@intentsolutions/audit-harness)

### C. Troubleshooting Playbooks

See § 8 ("Things That Will Bite You").

### D. Open Questions

- Should the dashboard ever support multi-user? Currently a hard no; the threat model
  punts. If yes, re-visit every decision in § 4.
- Is the TUI worth the build-out cost? If web is the primary surface, the TUI is decorative.
  `gastown-ey8` epic exists to track the design intent; closing the epic is a real option.
- Are the Vale + lychee gates worth keeping advisory long-term, or should they become hard? The
  vocab + URL cleanup cost is modest but real.

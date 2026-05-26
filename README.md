# gastown-viewer-intent v0.6.0

> Local-first Mission Control dashboard for Beads + Gas Town, with a read-only-forever memory panel.

[![CI](https://github.com/jeremylongshore/gastown-viewer-intent/actions/workflows/ci.yaml/badge.svg)](https://github.com/jeremylongshore/gastown-viewer-intent/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/v/release/jeremylongshore/gastown-viewer-intent)](https://github.com/jeremylongshore/gastown-viewer-intent/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Links:** [Gist One-Pager](https://gist.github.com/jeremylongshore/cd5d24298d05140eca8a3ef2cb2773f3) · [GitHub](https://github.com/jeremylongshore/gastown-viewer-intent) · [v0.6.0 Release](https://github.com/jeremylongshore/gastown-viewer-intent/releases/tag/v0.6.0) · [CHANGELOG](CHANGELOG.md)

## What's New in v0.6.0 (TUI Tier-1)

Foundation refactor of the terminal client (`gvi-tui`) plus two new
read-only tabs that bring the TUI to parity with the web UI's Phase 2
surfaces. No daemon changes — the TUI remains a thin HTTP client; the
daemon at `localhost:7070` continues to own all `bd`/`gt` shelling.

### Registry-driven keybindings

- A single `KeyRegistry` feeds both the dispatcher and the help overlay,
  so help cannot drift from the bindings the TUI actually responds to.
  Adopted from the [Dicklesworthstone/beads_viewer](https://github.com/Dicklesworthstone/beads_viewer)
  reference repo.
- `Focus` enum replaces the binary `View` toggle: **Board · Memories ·
  Triage · Detail**. `Update()` routes through `Dispatch(focus, key)`
  so every binding declares the focus it applies to.
- Combo-key state machine: `gg` jumps to top of column, `G` jumps to
  bottom. 250 ms pending-key window via `tea.Tick`.
- Help overlay (`?`) groups bindings by category (Navigation / Tabs /
  Actions / Global), filtered by current focus.

### Memories tab (read-only, daemon-redacted)

- New Memories tab reachable via `2` or `m`. Lists `bd memories` with
  the daemon's 005-PP-POLICY redaction already applied; redacted entries
  show a `[redacted]` marker and the classification class.
- `/` triggers server-side substring search via
  `/api/v1/memories/search?q=…`.
- `enter` expands the selected memory; a `Copy bd recall <key>` hint
  surfaces the canonical reveal path. **Reveal is intentionally NOT
  exposed in the TUI** — Council Q2 read-only-forever invariant says the
  bd CLI is the canonical writer / revealer.

### Triage tab (read-only)

- New Triage tab reachable via `3` or `t`. Lists every bead carrying the
  `human` label from `/api/v1/human`.
- `enter` reuses the existing Board issue-detail fetch — reviewers land
  on the same detail screen they already know.

### Foundation fixes

- HTTP client status check: `Memories`, `SearchMemories`, and
  `HumanFlags` now reject non-200 responses with a readable error before
  attempting JSON decode (shared `decodeJSON` helper).
- UTF-8 rune-safe truncation across 4 sites — board card titles, issue
  descriptions, memory previews, triage row titles. Previously byte
  slicing could split a multi-byte codepoint and emit malformed bytes.
- TUI dispatcher releases its `RWMutex` read lock before invoking
  handlers, mirroring `DispatchCombo`'s order; closes a latent deadlock
  if a future handler re-enters the registry.
- `gvi-tui` version is now injected by goreleaser ldflags
  (`-X main.version=…`) — previously every release binary reported
  `0.1.0` regardless of tag.

### Previous highlights (v0.5.0)

Phase 2 (Option B-minus): daemon hardening, read-only memories panel,
triage queue, sync pill. See [v0.5.0 in CHANGELOG](CHANGELOG.md#v050---2026-05-23) for the full list.

### Previous highlights (v0.4.0)

- **Embedded Web UI**: single binary serves the React dashboard via
  `go:embed`.
- **Convoy Dashboard**: batch work progress with Done/Active/Blocked
  counts.
- **Interactive Dependency Graph**: D3.js force-directed visualization.
- **Smart Agent Status**: Active/Idle/Stuck detection.

## Supported-version matrix

The viewer follows a "honest lag" cadence rather than chasing every
upstream release. Refreshes are opportunistic, on user-pain trigger,
EXCEPT for security-flagged upstream releases which follow a 48-hour
fast-path SLA.

| Upstream | Tested range | Notes |
|---|---|---|
| `bd` (Beads CLI) | 1.0.4 | `defer --until` preserved; `human list`, `dolt status`, `memories` surfaced |
| `gt` (Gas Town CLI) | 0.9.0 | Wisps surface used; legacy `.beads/molecule.json` no longer read |
| Go | 1.22+ | Building from source |
| Node.js | 20+ | Web dev |

## What it does

Real-time visibility into your Beads issue tracker and Gas Town agent
swarms.

| Surface | What's there |
|---|---|
| **Board** | Kanban view of Beads issues |
| **Graph** | Interactive D3.js dependency visualization |
| **Gas Town** | Agent dashboard with molecules, convoys, rigs |
| **Memory** | Read-only `bd memories` viewer with classification redaction |
| **Triage** | Read-only human-needed bead queue |
| **Header sync pill** | Live dolt sync state |

## Quickstart

### Install

#### Homebrew (macOS/Linux)

```bash
brew tap intent-solutions-io/tap
brew install gvid
```

#### Direct download

Download binaries from the
[releases page](https://github.com/intent-solutions-io/gastown-viewer-intent/releases).

#### From source

```bash
go install github.com/intent-solutions-io/gastown-viewer-intent/cmd/gvid@latest
```

### Prerequisites

- [Beads](https://github.com/steveyegge/beads) (`bd` CLI in `$PATH`).
- [Gas Town](https://github.com/steveyegge/gastown) installed at `~/gt`
  (optional — the dashboard works without it; the Gas Town tab simply
  reports the town as absent).

For development:

- Go 1.22+
- Node.js 20+

### Run

```bash
# If installed via brew or binary:
gvid                          # Start daemon + web UI on :7070

# For development with hot reload:
make dev                      # Vite on :5173, API proxied to :7070
```

Open [http://localhost:7070](http://localhost:7070) (or
[http://localhost:5173](http://localhost:5173) during development) and
switch tabs.

### Verify

```bash
# Health check
curl http://localhost:7070/api/v1/health

# Gas Town status
curl http://localhost:7070/api/v1/town/status

# Dolt sync state (header pill source)
curl http://localhost:7070/api/v1/sync

# Memories (default-redacted)
curl http://localhost:7070/api/v1/memories

# Human triage queue
curl http://localhost:7070/api/v1/human

# Dependency graph as Graphviz DOT
curl "http://localhost:7070/api/v1/graph?format=dot" | dot -Tsvg > deps.svg
```

## Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                      Gastown Viewer Intent                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│   │   gvi-tui    │      │   Web UI     │      │  External    │  │
│   │  (Bubbletea) │      │ (React+Vite) │      │   Clients    │  │
│   └──────┬───────┘      └──────┬───────┘      └──────┬───────┘  │
│          │                     │                     │          │
│          └─────────────────────┼─────────────────────┘          │
│                                │                                 │
│                                ▼                                 │
│                    ┌───────────────────────┐                    │
│                    │       gvid Daemon     │                    │
│                    │   localhost:7070      │                    │
│                    │  + Origin allowlist   │                    │
│                    │  + Session token gate │                    │
│                    └───────────┬───────────┘                    │
│                                │                                 │
│              ┌─────────────────┼─────────────────┐              │
│              ▼                                   ▼              │
│   ┌───────────────────────┐         ┌───────────────────────┐  │
│   │   Gastown Adapter     │         │    Beads Adapter      │  │
│   │  (`gt` CLI + ~/gt)    │         │   (shells to `bd`)    │  │
│   └───────────┬───────────┘         └───────────┬───────────┘  │
│               │                                 │               │
│               ▼                                 ▼               │
│   ┌───────────────────────┐         ┌───────────────────────┐  │
│   │      Gas Town         │         │     bd / Dolt store   │  │
│   │  ~/gt (rigs, agents)  │         │   (issues, memories)  │  │
│   └───────────────────────┘         └───────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Gas Town concepts

| Concept | Description |
|---|---|
| **Town** | Workspace root (`~/gt`) containing all rigs and town-level agents |
| **Mayor** | Town coordinator — routes work across rigs |
| **Deacon** | Town patrol — monitors health and escalates issues |
| **Rig** | Project container with its own agent pool |
| **Witness** | Rig-level overseer — manages polecat lifecycle |
| **Refinery** | Merge queue processor for the rig |
| **Polecats** | Transient workers spawned for specific tasks |
| **Crew** | Persistent user-managed workers in a rig |
| **Convoy** | Batch work tracking across multiple rigs |
| **Molecule** / **Wisp** | Workflow instance — `wisp` is the gt 0.9 name |
| **Formula** | Template defining molecule structure and steps |

## API endpoints

### Beads

| Endpoint | Description |
|---|---|
| `GET /api/v1/health` | Health check |
| `GET /api/v1/board` | Kanban board view |
| `GET /api/v1/issues` | List issues |
| `GET /api/v1/issues/{id}` | Issue details |
| `GET /api/v1/graph?format={json,dot}` | Dependency graph |
| `GET /api/v1/events` | SSE event stream |
| `GET /api/v1/sync` | Dolt sync state (header pill) |
| `GET /api/v1/human` | Human triage queue (read-only) |
| `GET /api/v1/memories` | Memory layer (default-redacted; `?reveal=true` to opt in) |
| `GET /api/v1/memories/{key}` | Single memory recall |
| `GET /api/v1/memories/search?q=...` | Substring search |

### Gas Town

| Endpoint | Description |
|---|---|
| `GET /api/v1/town/status` | Town health, agent + rig counts |
| `GET /api/v1/town` | Full town structure |
| `GET /api/v1/town/rigs` | List all rigs |
| `GET /api/v1/town/rigs/{name}` | Single rig details |
| `GET /api/v1/town/agents` | All agents with status |
| `GET /api/v1/town/convoys` | Active convoys |
| `GET /api/v1/town/convoys/{id}` | Single convoy details |
| `GET /api/v1/town/molecules` | Active molecules (sourced from `gt wisps`) |
| `GET /api/v1/town/molecules/{id}` | Single molecule details |
| `GET /api/v1/town/mail/{address}` | Agent mail inbox |

## Security model

See `THREAT_MODEL.md` for the full threat model. Key points:

- Loopback bind enforced at startup; `--host=0.0.0.0` is refused.
- Origin allowlist middleware rejects cross-origin requests
  (DNS-rebind / CSRF defense).
- Session token at `~/.config/gvid/token` (mode 0600). State-mutating
  endpoints behind the token gate (none shipped yet — all current
  endpoints are read-only).
- Memories panel is **read-only-forever** by architectural invariant
  (Council Q2). The bd CLI is the canonical writer.

## Configuration

```bash
# Custom Gas Town location
gvid --town /path/to/gt

# Custom port
gvid --port 8080

# All options
gvid --help
```

## Project structure

```text
gastown-viewer-intent/
├── cmd/
│   ├── gvid/              # Daemon entrypoint
│   └── gvi-tui/           # TUI client
├── internal/
│   ├── api/               # HTTP handlers, security, redaction
│   ├── gastown/           # Gas Town adapter (reads ~/gt + gt CLI)
│   ├── beads/             # Beads adapter (bd CLI)
│   └── model/             # Domain types
├── web/                   # React + Vite frontend (embedded via go:embed)
├── 000-docs/              # Project docs (per /doc-filing v4.3)
├── tests/                 # Testing policy
└── Makefile
```

## License

MIT — see `LICENSE`.

## Related projects

- [Gastown](https://github.com/steveyegge/gastown) — multi-agent
  workspace orchestrator.
- [Beads](https://github.com/steveyegge/beads) — local-first issue
  tracking with dependencies.

ADR-001 — Gastown Viewer Intent MVP Architecture

Status: Accepted
Date: 2026-01-01
Repo: intent-solutions-io/gastown-viewer-intent
Docs location: 000-docs/ (flat)

⸻

Context

We need a local-first "mission control" viewer for Beads-managed work. Users want a board view, issue details (children + dependencies), a dependency graph export, and optional live updates. The solution must be quick to run in any repo, minimize coupling to Beads internals, and support both terminal and web workflows.

⸻

Decision

Adopt a control-plane architecture:
    1.    Local daemon (Go) provides a stable HTTP API on localhost:7070.
    2.    UI clients (TUI + Web) consume the daemon API only.
    3.    Beads integration for MVP shells out to the bd CLI (no direct .beads/ parsing).
    4.    Docs are stored flat in 000-docs/ only.

⸻

Drivers
    •    Fast MVP iteration while keeping a stable internal contract (API).
    •    Reduce breakage risk from Beads storage/schema changes.
    •    Support SSH/terminal users and visual users without duplicating logic.
    •    Make it easy to add future integrations (SSE, Slack, mobile, multi-repo) without rewriting clients.

⸻

Key Decisions

D1: Language and UI stack
    •    Go for daemon + TUI
    •    Bubbletea for TUI
    •    Vite + React + TypeScript for web UI

D2: Data source contract
    •    MVP uses bd CLI as the integration contract.
    •    Implement internal/beads.Adapter that executes bd commands and parses output with graceful degradation.

D3: API-first internal boundary

The daemon exposes /api/v1/* endpoints (health, issues, issue detail, board, graph, events).
Clients must not call bd directly.

D4: Live updates
    •    Preferred: SSE at GET /api/v1/events.
    •    Acceptable MVP fallback: polling (web and/or TUI).
If SSE is deferred, create a follow-on Beads issue.

D5: Documentation standard

All documentation files live flat in 000-docs/ (no docs/ directory, no subfolders).

⸻

Consequences

Positive
    •    Clear separation of concerns: integration vs presentation.
    •    One parsing surface (daemon) instead of duplicating parsing in each UI.
    •    Easier testing (mock Adapter, test API contract).
    •    Easier future packaging (desktop wrapper, mobile remote client).

Negative
    •    Requires running a daemon (extra process).
    •    CLI parsing can be brittle; must invest in robust error handling.
    •    SSE adds complexity; polling may be used initially.

⸻

Alternatives considered

A1: Web-only app that reads .beads/ directly

Rejected: high coupling to storage format; harder to keep stable; duplicates logic between clients.

A2: Fork beads_viewer and extend it

Rejected for MVP: fork maintenance burden; product shape differs (control plane + multi-clients).

A3: Single monolithic TUI that calls bd directly

Rejected: blocks web UI reuse; no stable contract; harder to add mobile/remote later.

⸻

Implementation notes
    •    Daemon should fail fast with actionable errors when:
    •    bd is missing
    •    .beads/ not initialized
    •    Adapter should degrade gracefully:
    •    return partial data when fields are missing
    •    never panic on unexpected output

⸻

Follow-ups (post-MVP)
    •    Add optional insights provider (e.g., bv integration) behind an interface.
    •    Add persistence cache layer (sqlite) for faster startup on large graphs.
    •    Add remote mode (mobile client talks to daemon over Tailscale/LAN).
    •    Expand event model and change detection.

⸻

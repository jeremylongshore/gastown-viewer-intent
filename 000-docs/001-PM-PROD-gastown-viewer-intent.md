# Product Requirements Document (PRD)

**Project**: Gastown Viewer Intent
**Version**: 1.0.0
**Date**: 2026-01-01
**Status**: Active
**Repo**: intent-solutions-io/gastown-viewer-intent

---

## 1. Executive Summary

### 1.1 One-Liner

**Product Vision**: Local-first Mission Control for Beads + Gastown-style agent swarms.

### 1.2 Problem Statement

**Who hurts today**: Developers using Beads (a local issue tracker with first-class dependency support) lack visual tools for understanding and navigating work graphs.

**Current pain points**:

- No visual board view for issue status at a glance
- Difficult to trace dependency chains in complex projects
- No real-time visibility into status changes
- CLI-only access limits collaboration and quick scanning
- No web-based sharing for async team review

**Why now**: As Beads adoption grows for agent orchestration (Gastown-style swarms), the need for visual observability becomes critical. Agent workflows generate many interdependent tasks that are hard to track via CLI alone.

**Cost of inaction**: Developers waste time mentally reconstructing dependency graphs, miss blocked issues, and lack confidence in work prioritization.

---

## 2. Target Users

### 2.1 Primary Persona: Solo Developer

- **Profile**: Individual developer using Beads for personal task management
- **Goals**: Quick visual scan of project status, navigate to blocked issues, understand dependency paths
- **Pain Points**: CLI output is hard to scan; mental model of dependencies gets stale
- **Tech Proficiency**: High (comfortable with terminal, but values visual dashboards)

### 2.2 Secondary Persona: Agent Operator

- **Profile**: Developer running Gastown-style agent swarms where Claude agents create/update Beads issues
- **Goals**: Monitor agent-generated work, see real-time status updates, verify dependency integrity
- **Relationship to Primary**: Uses more advanced features (events stream, graph export)

---

## 3. MVP Scope

### 3.1 In Scope (Must Have)

| Feature | Description | Success Criteria |
|---------|-------------|------------------|
| **Daemon API** | HTTP server on localhost:7070 with JSON API | `curl /api/v1/health` returns JSON |
| **Board View** | Issues grouped by status (pending, in_progress, done, blocked) | All issues visible in correct columns |
| **Issue Details** | View single issue with children and dependencies | Children and deps displayed correctly |
| **Dependency Graph** | Export graph as JSON (nodes + edges) | Valid graph structure returned |
| **Events Stream** | SSE endpoint for real-time updates | Events stream when issues change |
| **TUI Client** | Terminal UI using Bubbletea | Board view + issue navigation works |
| **Web UI** | React app with board and detail panel | Clickable board, side panel for details |

### 3.2 Out of Scope (MVP)

| Feature | Why Excluded | Future Phase |
|---------|--------------|--------------|
| Direct .beads/ parsing | Stability risk; bd CLI is stable contract | Post-MVP |
| Write operations | Read-only viewer for MVP | V2 |
| Authentication | Local-first, single-user for MVP | V2 |
| Remote deployment | Localhost only for MVP | V2 |
| Beads Viewer (bv) integration | Additional complexity | Post-MVP |

### 3.3 Key Assumptions

**Technical**:

- `bd` CLI is installed and in PATH
- `bd` output format is stable (or we degrade gracefully)
- Go 1.22+ and Node 20+ available

**User**:

- Single user per instance
- Local network access only
- Comfortable with terminal for initial setup

---

## 4. Functional Requirements

### 4.1 Daemon (gvid)

**FR-1**: Start HTTP server on configurable port (default: 7070)

- Acceptance: `go run ./cmd/gvid` binds to :7070

**FR-2**: Health endpoint returns beads initialization status

- Acceptance: `/api/v1/health` returns `{status, beads_initialized, version}`

**FR-3**: List issues with optional filters

- Acceptance: `/api/v1/issues?status=pending` returns filtered list

**FR-4**: Get single issue with full details

- Acceptance: `/api/v1/issues/:id` returns issue with children, blocks, blocked_by

**FR-5**: Board view returns issues grouped by status

- Acceptance: `/api/v1/board` returns columns with issue summaries

**FR-6**: Graph export returns dependency structure

- Acceptance: `/api/v1/graph` returns nodes and edges

**FR-7**: SSE endpoint streams status changes

- Acceptance: `/api/v1/events` sends events on issue updates

**FR-8**: Fail fast if beads not initialized

- Acceptance: Returns 503 with actionable error message

### 4.2 TUI Client (gvi-tui)

**FR-9**: Display board view on startup

- Acceptance: Shows columns with issue counts

**FR-10**: Navigate issues with keyboard

- Acceptance: Arrow keys move selection, Enter opens details

**FR-11**: Show issue details panel

- Acceptance: Children and dependencies visible

**FR-12**: Error state when daemon unavailable

- Acceptance: Clear message if connection fails

### 4.3 Web UI

**FR-13**: Board view with drag-free columns

- Acceptance: Issues grouped by status, clickable

**FR-14**: Issue detail side panel

- Acceptance: Click issue opens panel with full details

**FR-15**: Polling for updates (MVP)

- Acceptance: Board refreshes periodically

**FR-16**: Error state when daemon unavailable

- Acceptance: Clear message if API fails

---

## 5. Non-Functional Requirements

### 5.1 Performance

- API response time: < 200ms for all endpoints
- TUI startup: < 1 second
- Web initial load: < 2 seconds

### 5.2 Reliability

- Graceful degradation when bd output is unexpected
- No panics on malformed data
- Clear error messages for all failure modes

### 5.3 Usability

- Keyboard-driven TUI with vim-style navigation
- Responsive web UI (mobile-friendly layout)
- Self-documenting API (OpenAPI spec)

### 5.4 Security

- Bind to localhost only (no external access by default)
- No sensitive data stored
- CORS restricted to dev origins

---

## 6. Technical Architecture

See: `000-docs/002-ADR-gastown-viewer-intent.md`

**Summary**:

- Go daemon + TUI (single binary potential)
- Vite + React + TypeScript web
- Beads adapter shells to `bd` CLI (no .beads/ parsing)
- API-first internal boundary

---

## 7. Success Metrics

### 7.1 MVP Launch Criteria

| Criteria | Measurement | Target |
|----------|-------------|--------|
| Daemon starts | `make dev` succeeds | 100% |
| API returns valid JSON | All endpoints tested | 100% |
| TUI shows board | Manual verification | Pass |
| Web shows board | Manual verification | Pass |
| Error handling | Beads not init test | Graceful 503 |

### 7.2 Quality Gates

- [ ] All Beads MVP issues closed
- [ ] `bd ready` returns empty (no blocked work)
- [ ] README has working quickstart
- [ ] Demo walkthrough recorded

---

## 8. Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| bd output format changes | Medium | High | Graceful degradation; parser returns partial data |
| bd CLI not installed | Low | Critical | Fail fast with install instructions |
| Performance with large graphs | Low | Medium | Pagination, lazy loading in V2 |
| SSE complexity | Medium | Medium | Polling fallback acceptable for MVP |

---

## 9. Beads Work Graph

This PRD is tracked by Beads epic: **Gastown Viewer Intent MVP**

### Child Issues

- A: Domain model + event schema
- B: Beads adapter via bd CLI
- C: Daemon HTTP API + SSE events
- D: TUI client consuming API
- E: Web UI consuming API
- F: Dev tooling + docs
- G: MVP demo + sanity checks

### Dependency Order

```

A → B → C → D → G
         ↘ E → G
     F ────────→ G
```

Run `bd board` to see current implementation status.

---

## 10. Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-01-01 | Claude | Initial PRD |

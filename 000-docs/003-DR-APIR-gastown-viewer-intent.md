# API Reference: Gastown Viewer Intent

**Version**: 1.0.0
**Base URL**: `http://localhost:7070/api/v1`
**Date**: 2026-01-01

---

## Overview

The Gastown Viewer Intent daemon (`gvid`) exposes a RESTful JSON API for querying Beads issue data. All endpoints are read-only for MVP.

### Authentication

None required (local-first, single-user).

### Content Types

- Request: N/A (GET only for MVP)
- Response: `application/json` (except SSE endpoint)

### CORS

Default headers for local development:

```

Access-Control-Allow-Origin: http://localhost:5173
Access-Control-Allow-Methods: GET, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

---

## Endpoints

### GET /health

Health check endpoint. Returns daemon status and Beads initialization state.

**Request**

```http

GET /api/v1/health HTTP/1.1
Host: localhost:7070
```

**Response (200 OK)**

```json

{
  "status": "ok",
  "beads_initialized": true,
  "version": "0.1.0",
  "bd_version": "0.29.0"
}
```

**Response (503 Service Unavailable)** — Beads not initialized

```json

{
  "status": "error",
  "beads_initialized": false,
  "error": "Beads not initialized. Run 'bd init' in your project directory.",
  "version": "0.1.0"
}
```

---

### GET /issues

List all issues. Supports optional query filters.

**Request**

```http

GET /api/v1/issues?status=pending&parent=gvi-0 HTTP/1.1
Host: localhost:7070
```

**Query Parameters**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | No | Filter by status: `pending`, `in_progress`, `done`, `blocked` |
| `parent` | string | No | Filter by parent issue ID |
| `search` | string | No | Search in title and description |
| `limit` | integer | No | Max results (default: 100) |
| `offset` | integer | No | Pagination offset (default: 0) |

**Response (200 OK)**

```json

{
  "issues": [
    {
      "id": "gvi-1",
      "title": "Domain model + event schema",
      "status": "done",
      "priority": "high",
      "parent": "gvi-0",
      "children": ["gvi-1a", "gvi-1b"],
      "blocks": ["gvi-2"],
      "blocked_by": [],
      "created_at": "2026-01-01T10:00:00Z",
      "updated_at": "2026-01-01T12:00:00Z"
    },
    {
      "id": "gvi-2",
      "title": "Beads adapter via bd CLI",
      "status": "in_progress",
      "priority": "high",
      "parent": "gvi-0",
      "children": [],
      "blocks": ["gvi-3"],
      "blocked_by": ["gvi-1"],
      "created_at": "2026-01-01T10:00:00Z",
      "updated_at": "2026-01-01T14:00:00Z"
    }
  ],
  "total": 2,
  "limit": 100,
  "offset": 0
}
```

---

### GET /issues/{id}

Get single issue with full details including children and dependency information.

**Request**

```http

GET /api/v1/issues/gvi-2 HTTP/1.1
Host: localhost:7070
```

**Response (200 OK)**

```json

{
  "id": "gvi-2",
  "title": "Beads adapter via bd CLI",
  "description": "Implement internal/beads.Adapter interface that shells out to bd CLI.\n\nDone when:\n- internal/beads/adapter.go defines Adapter interface\n- Unit tests with mock bd output pass",
  "status": "in_progress",
  "priority": "high",
  "parent": {
    "id": "gvi-0",
    "title": "Gastown Viewer Intent MVP"
  },
  "children": [],
  "blocks": [
    {
      "id": "gvi-3",
      "title": "Daemon HTTP API + SSE events",
      "status": "pending"
    }
  ],
  "blocked_by": [
    {
      "id": "gvi-1",
      "title": "Domain model + event schema",
      "status": "done"
    }
  ],
  "done_when": [
    "internal/beads/adapter.go defines Adapter interface",
    "internal/beads/cli.go implements CLI executor",
    "internal/beads/parser.go handles output parsing",
    "Unit tests with mock bd output pass",
    "Returns clear error if bd not found"
  ],
  "created_at": "2026-01-01T10:00:00Z",
  "updated_at": "2026-01-01T14:00:00Z"
}
```

**Response (404 Not Found)**

```json

{
  "error": "issue not found",
  "code": "ISSUE_NOT_FOUND",
  "id": "gvi-999"
}
```

---

### GET /board

Board view with issues grouped by status columns.

**Request**

```http

GET /api/v1/board HTTP/1.1
Host: localhost:7070
```

**Response (200 OK)**

```json

{
  "columns": [
    {
      "status": "pending",
      "label": "Pending",
      "count": 3,
      "issues": [
        {
          "id": "gvi-4",
          "title": "TUI client consuming API",
          "priority": "high"
        },
        {
          "id": "gvi-5",
          "title": "Web UI consuming API",
          "priority": "high"
        },
        {
          "id": "gvi-6",
          "title": "Dev tooling + docs",
          "priority": "medium"
        }
      ]
    },
    {
      "status": "in_progress",
      "label": "In Progress",
      "count": 1,
      "issues": [
        {
          "id": "gvi-3",
          "title": "Daemon HTTP API + SSE events",
          "priority": "high"
        }
      ]
    },
    {
      "status": "done",
      "label": "Done",
      "count": 2,
      "issues": [
        {
          "id": "gvi-1",
          "title": "Domain model + event schema",
          "priority": "high"
        },
        {
          "id": "gvi-2",
          "title": "Beads adapter via bd CLI",
          "priority": "high"
        }
      ]
    },
    {
      "status": "blocked",
      "label": "Blocked",
      "count": 0,
      "issues": []
    }
  ],
  "total": 6
}
```

---

### GET /graph

Dependency graph as nodes and edges. Supports multiple output formats.

**Request**

```http

GET /api/v1/graph?format=json HTTP/1.1
Host: localhost:7070
```

**Query Parameters**

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `format` | string | No | Output format: `json` (default), `dot` |
| `root` | string | No | Filter to subgraph rooted at issue ID |

**Response (200 OK) — JSON format**

```json

{
  "nodes": [
    {
      "id": "gvi-1",
      "title": "Domain model + event schema",
      "status": "done",
      "priority": "high"
    },
    {
      "id": "gvi-2",
      "title": "Beads adapter via bd CLI",
      "status": "in_progress",
      "priority": "high"
    },
    {
      "id": "gvi-3",
      "title": "Daemon HTTP API + SSE events",
      "status": "pending",
      "priority": "high"
    }
  ],
  "edges": [
    {
      "from": "gvi-1",
      "to": "gvi-2",
      "type": "blocks"
    },
    {
      "from": "gvi-2",
      "to": "gvi-3",
      "type": "blocks"
    }
  ],
  "stats": {
    "node_count": 3,
    "edge_count": 2,
    "max_depth": 3
  }
}
```

**Response (200 OK) — DOT format** (`?format=dot`)

```

Content-Type: text/plain

digraph dependencies {
  rankdir=LR;
  node [shape=box fontname="Helvetica"];

  "gvi-1" [label="Domain model" style=filled fillcolor="#90EE90"];
  "gvi-2" [label="Beads adapter" style=filled fillcolor="#FFD700"];
  "gvi-3" [label="Daemon API" style=filled fillcolor="#FFFFFF"];

  "gvi-1" -> "gvi-2";
  "gvi-2" -> "gvi-3";
}
```

---

### GET /events (SSE)

Server-Sent Events stream for real-time updates. Connection stays open.

**Request**

```http

GET /api/v1/events HTTP/1.1
Host: localhost:7070
Accept: text/event-stream
Cache-Control: no-cache
```

**Response Headers**

```http

HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Event Types**

| Event | Description |
|-------|-------------|
| `issue_created` | New issue added |
| `issue_updated` | Issue status/fields changed |
| `issue_deleted` | Issue removed |
| `heartbeat` | Keep-alive (every 30s) |

**Event Format**

```

event: issue_updated
data: {"id":"gvi-2","status":"done","previous_status":"in_progress","updated_at":"2026-01-01T14:30:00Z"}

event: heartbeat
data: {"timestamp":"2026-01-01T14:30:30Z"}

event: issue_created
data: {"id":"gvi-8","title":"New feature","status":"pending","created_at":"2026-01-01T14:31:00Z"}
```

**Reconnection**
Clients should reconnect on connection drop. No `Last-Event-ID` support in MVP.

---

## Data Types

### Issue (Summary)

```typescript

interface IssueSummary {
  id: string;
  title: string;
  status: "pending" | "in_progress" | "done" | "blocked";
  priority: "high" | "medium" | "low";
}
```

### Issue (Full)

```typescript

interface Issue {
  id: string;
  title: string;
  description: string;
  status: "pending" | "in_progress" | "done" | "blocked";
  priority: "high" | "medium" | "low";
  parent: IssueSummary | null;
  children: IssueSummary[];
  blocks: IssueSummary[];
  blocked_by: IssueSummary[];
  done_when: string[];
  created_at: string; // ISO 8601
  updated_at: string; // ISO 8601
}
```

### Board

```typescript

interface Board {
  columns: Column[];
  total: number;
}

interface Column {
  status: string;
  label: string;
  count: number;
  issues: IssueSummary[];
}
```

### Graph

```typescript

interface Graph {
  nodes: GraphNode[];
  edges: GraphEdge[];
  stats: GraphStats;
}

interface GraphNode {
  id: string;
  title: string;
  status: string;
  priority: string;
}

interface GraphEdge {
  from: string;
  to: string;
  type: "blocks" | "parent";
}

interface GraphStats {
  node_count: number;
  edge_count: number;
  max_depth: number;
}
```

---

## Error Responses

All endpoints return consistent error format on failure.

### Error Schema

```json

{
  "error": "human-readable message",
  "code": "MACHINE_READABLE_CODE",
  "details": {}
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `BEADS_NOT_INIT` | 503 | Beads not initialized in current directory |
| `BD_NOT_FOUND` | 503 | bd CLI not found in PATH |
| `ISSUE_NOT_FOUND` | 404 | Requested issue ID does not exist |
| `PARSE_ERROR` | 500 | Failed to parse bd output (with partial data if possible) |
| `BD_ERROR` | 500 | bd command returned non-zero exit |
| `INVALID_PARAM` | 400 | Invalid query parameter |

### Example Error Response

```json

{
  "error": "Beads not initialized. Run 'bd init' in your project directory.",
  "code": "BEADS_NOT_INIT",
  "details": {
    "working_dir": "/home/user/project",
    "suggestion": "bd init"
  }
}
```

---

## Rate Limiting

No rate limiting in MVP (local daemon).

---

## Versioning

API version is in URL path: `/api/v1/*`

Breaking changes will increment version: `/api/v2/*`

---

## Example Workflow

### 1. Check Health

```bash

curl http://localhost:7070/api/v1/health
```

### 2. Get Board View

```bash

curl http://localhost:7070/api/v1/board | jq '.columns[] | {status, count}'
```

### 3. View Issue Details

```bash

curl http://localhost:7070/api/v1/issues/gvi-3 | jq
```

### 4. Export Dependency Graph

```bash

curl http://localhost:7070/api/v1/graph?format=dot > deps.dot
dot -Tpng deps.dot -o deps.png
```

### 5. Stream Events

```bash

curl -N http://localhost:7070/api/v1/events
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-01-01 | Claude | Initial API spec |

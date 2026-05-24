# Threat Model — Gastown Viewer Intent

**Status:** v1.0 · Authored 2026-05-24 as part of bead `gastown-hu4` per the
`000-docs/004-AT-DECR-gastown-viewer-option-b-council-2026-05-23.md` council
decision Q0 (CISO binding constraint).

This document is the operating threat model for the `gvid` daemon and its
React web UI. It is meant to be short, concrete, and re-readable in five
minutes. Updates land alongside the security code they describe.

## What this product is

A local-first Mission Control dashboard. The `gvid` Go daemon binds to a
loopback address (default `localhost:7070`) and shells out to the `bd`
(beads) and `gt` (Gas Town) CLIs for data. The Vite/React web UI runs at
`localhost:5173` in dev and is embedded as static assets in the built
daemon. A TUI client is a thin alternative front-end against the same
HTTP API.

There is exactly one expected human user per daemon process: the engineer
whose dev box is running it.

## Assets we are protecting

In rough order of how bad it would be to lose them:

1. **`bd memories`** — the persistent cross-session knowledge layer. Contains
   partner names (Kobiton, Nixtla, Mudit/Polygon, Lit, Elm), engagement
   context, recovery-context notes, and occasionally pasted API tokens. See
   [`000-docs/005-PP-POLICY-memories-classification-2026-05-24.md`](000-docs/005-PP-POLICY-memories-classification-2026-05-24.md)
   for the content classification policy.
2. **Issue state mutations** — closing, deferring, or commenting on beads
   issues is a writable surface. The viewer does NOT expose any
   state-mutation endpoint as of `gastown-hu4`; future endpoints (notably
   `bd human respond` / `dismiss` in `gastown-3uf`) will sit behind the
   token gate this bead installs.
3. **Issue and town metadata** — issue titles/descriptions and gt town
   layout. Less sensitive than memories but still confidential.

## Trust boundary

The daemon's process boundary is the only trust boundary. Inside that
boundary the daemon trusts the `bd` and `gt` binaries on `$PATH`, the
filesystem at `~/.beads/` and `~/gt/`, and the engineer's
`~/.config/gvid/token` file. Outside that boundary every byte is hostile
until proven otherwise.

## In-scope threats

These are the threats this product actively defends against. Each one has
a corresponding code path or operational control.

### 1. DNS rebinding from any tab open on the dev box

**Scenario:** Engineer has any web page open in any browser on the dev box.
That page resolves `victim.example` via DNS to a public IP initially, then
the second resolution returns `127.0.0.1`. JavaScript on that page issues
`fetch('http://victim.example:7070/api/v1/memories')`. Without an Origin
allowlist on the daemon, the daemon would happily return the memories JSON.

**Defense:** `OriginAllowlistMiddleware` in `internal/api/security.go`.
Every request that has an `Origin` header is checked against
`Config.CORSOrigins`; mismatches return HTTP 403 `ORIGIN_REJECTED` and the
handler is never reached. Browser fetch() always sends `Origin`, so this
gate fires on every rebinding attempt. Native clients (curl, the TUI,
self-tests) deliberately do NOT send `Origin` and pass through — they are
inside the trust boundary.

### 2. CSRF from a malicious local web page

Same shape as (1) but without DNS rebinding — a malicious page on
`localhost:9999`, `file://`, or a sandboxed iframe (Origin: `null`) tries
to issue requests against the daemon. Same defense: the Origin allowlist
rejects everything but the configured front-end origin.

### 3. Same-machine other process attempting to mutate state

**Scenario:** A malicious npm `postinstall`, a compromised MCP server, or a
drive-by markdown preview spawns a request to `localhost:7070`. These
local processes can read `Origin` headers freely (they're not browsers),
so the Origin gate is insufficient.

**Defense (future state):** `RequireTokenMiddleware` wraps every
state-mutating endpoint. The token is generated at daemon startup, written
to `~/.config/gvid/token` with mode `0600`, and never logged. Same-machine
processes that do not run under the engineer's UID cannot read the file;
processes that do run under that UID are already inside the trust
boundary. As of this bead the middleware is **installed but not yet wired
to any route** because the AT-DECR Q2 decision pins the memories panel
read-only-forever and the human-triage POSTs are deferred to bead
`gastown-3uf`. The cost of installing the gate now is one Go file; the
benefit is that the schema slot exists when the POST routes ship, instead
of being a retrofit after a security review notices it's missing.

### 4. Accidental binding to a non-loopback address

**Scenario:** Engineer types `--host=0.0.0.0` thinking they're enabling
something local, exposing the dashboard to whatever network the dev box
shares (corporate WiFi, hotel Ethernet, cloud VPC).

**Defense:** `IsLoopbackHost` is called from `Server.Start()` before any
listener is opened. A non-loopback host returns an error and the daemon
fails to start. There's an escape hatch (`Config.DisableLoopbackCheck`)
for ephemeral container test environments, gated by an explicit warning
log line at startup.

### 5. Confidential data rendered into uncontrolled browser storage

**Scenario:** Engineer opens the dashboard, the memories panel renders
partner names from `bd memories`. Browser autocomplete starts suggesting
"Kobiton" in any future input on any site. Screen-recording during a
partner call captures the open panel.

**Defense:** policy document
[`000-docs/005-PP-POLICY-memories-classification-2026-05-24.md`](000-docs/005-PP-POLICY-memories-classification-2026-05-24.md)
specifies the partner-name and secret-pattern denylists. When the memories
panel ships in `gastown-3uf` it must apply these denylists as a redaction
overlay before any partner name reaches the DOM, and all input elements in
the viewer must carry `autocomplete="off"`. The architectural invariant
(memories panel is read-only-forever) eliminates the form-input vector
entirely — there is no input field where partner names could be typed.

## Out-of-scope threats

Explicitly NOT in this product's threat model. If the scenario below
applies, the engineer is already in a worse situation than the dashboard
can mitigate.

- **Attacker with arbitrary code execution as the engineer's UID.** The
  attacker can already read the token file, the `.beads/` SQLite store,
  every memory, and `~/.ssh/`. This dashboard cannot help.
- **Attacker with kernel-level access on the dev box.** Same — the
  defenses below are user-space and provide no protection.
- **Network-level attackers outside the dev box.** The loopback bind check
  prevents the dashboard from being reachable from the network at all.
  If an attacker is on-path between the engineer and `127.0.0.1` they are
  in the kernel, which is the out-of-scope case above.
- **Browser zero-day that bypasses Same-Origin Policy.** SOP is the floor
  the Origin allowlist is built on. If SOP itself is bypassed, every
  cookie/token on the engineer's machine is in play.
- **Physical access to the unlocked dev box.** Lock your screen.

## Operational controls

These are the things the engineer is expected to do, not the code:

- Lock screen when stepping away. The dashboard does not authenticate the
  engineer — anyone at the keyboard can read every memory.
- During screen-sharing with partners (Anthropic cohort calls, partner
  demos), close any browser tab showing the dashboard, or use the
  redaction overlay's "show" toggle ONLY for non-sensitive memories.
- Do not paste API tokens into `bd remember`. The viewer will
  best-effort-redact common token patterns
  (`sk-`, `ghp_`, `AKIA`, `gho_`, `glpat-`), but the defense in depth is
  not putting tokens there in the first place. The bd CLI is the right
  surface for tokens that have to live somewhere; SOPS-encrypted env
  files are the right surface for production-impact secrets.

## Defense map — code locations

| Threat                          | Defense                                | File                            |
|---------------------------------|----------------------------------------|---------------------------------|
| DNS rebind / cross-origin CSRF  | `OriginAllowlistMiddleware`            | `internal/api/security.go`      |
| Same-machine state mutation     | `RequireTokenMiddleware` (installed; routes wired in `gastown-3uf`) | `internal/api/security.go`      |
| Token persistence + comparison  | `SessionToken.Persist` (0600) + `SessionToken.Equal` (constant-time) | `internal/api/security.go`      |
| Non-loopback bind               | `IsLoopbackHost` startup check         | `internal/api/security.go` + `internal/api/server.go` |
| Memories rendering surface      | Architectural invariant (read-only-forever) | Council decision Q2 + `internal/api/server.go` route registration |
| Partner-name + secret redaction | Classification policy + denylists      | `000-docs/005-PP-POLICY-memories-classification-2026-05-24.md` (applied in `gastown-3uf`) |

## Open follow-ups

Tracked as beads under the `gastown-cr5` epic. None of these block this
bead from shipping; they're the residual threat-model work the council
explicitly deferred.

- `gastown-3uf` — wire the memories panel (read-only) and the human-triage
  read-view. Apply the classification policy denylists. No POST routes.
- Future bead (post-Phase 2) — wire `RequireTokenMiddleware` onto the
  human-triage POST routes (`/api/v1/human/{id}/respond`,
  `/api/v1/human/{id}/dismiss`) when they ship. Will require the audit-log
  emit path that CISO specified as the binding constraint.
- Future bead — add a startup-time check that
  `Config.SessionTokenPath` resolves under the engineer's `$HOME`. Today
  the persist call will happily write the token anywhere the daemon can
  write; the policy intent is "user-config-dir only."

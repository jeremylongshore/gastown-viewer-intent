# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.6.0] - 2026-05-26

### Added

- TUI Tier-1 build-out (#24): focus-based key registry, combo-key state
  machine (`gg` jump-to-top), and a help overlay rendered from the same
  registry the dispatcher consumes. Adopts the registry-driven pattern
  from [Dicklesworthstone/beads_viewer](https://github.com/Dicklesworthstone/beads_viewer).
- TUI Memories tab (#24): read-only `bd memories` viewer with
  daemon-applied redaction markers; `/` triggers server-side substring
  search via `/api/v1/memories/search`; `Copy bd recall <key>` hint
  surfaces the canonical reveal path. Reveal is intentionally NOT
  exposed in the TUI per Council Q2 read-only-forever invariant.
- TUI Triage tab (#24): read-only `human`-flag queue from
  `/api/v1/human`; `enter` jumps to the existing Board issue-detail
  viewport.
- TUI status code enforcement: new `Memories`, `SearchMemories`, and
  `HumanFlags` HTTP client methods reject non-200 responses with a
  readable error before attempting JSON decode (shared `decodeJSON`
  helper).
- Operator-grade DevOps playbook (`000-docs/006-AA-AUDT-…`) covering
  build, deploy, monitoring, secrets, and rollback for the viewer
  daemon (#23).
- Tier-2/3 follow-up beads filed under epic `gastown-ey8`:
  `gastown-6je` (kanban swimlanes), `gastown-yla` (TUI status bar with
  sync pill), `gastown-ay7` (ASCII DAG + insights/history/sprint).

### Changed

- TUI architecture: replaced the binary `View` enum (Board / Issue)
  with a richer `Focus` enum (Board / Memories / Triage / Detail);
  `Update()` routes through a centralized `Dispatch(focus, key)` so
  bindings, help, and tab bar stay in sync (#24).
- `cmd/gvi-tui/main.go` now takes its version from goreleaser ldflags
  (`-X main.version={{.Version}}`) instead of a hardcoded constant.
  Previously every binary reported `0.1.0` regardless of release.
- `.goreleaser.yaml`: wired ldflags injection for the `gvi-tui` build
  target so release binaries report the correct version.
- README header synced with the canonical gist one-pager and new
  canonical URLs (#22).
- `go.sum` cleanup: removed orphan transitive entries
  (`go-udiff v0.2.0`, `x/exp/golden`) that no module currently imports.

### Fixed

- TUI dispatcher held an `RWMutex` read lock across handler invocation
  — a latent deadlock if a handler ever re-entered the registry. Mirror
  `DispatchCombo`'s release-before-invoke order.
- UTF-8 byte-slice truncation across 4 sites (board card titles, issue
  descriptions, memory previews, triage row titles) could split a
  multi-byte codepoint and emit malformed bytes. Replaced with a new
  rune-safe `truncateRunes()` helper; locked in by
  `TestTruncateRunes_UTF8Safe` covering ASCII, Cyrillic, and emoji.
- `.goreleaser.yaml`: temporarily disabled the Homebrew tap block until
  `HOMEBREW_TAP_TOKEN` rotation completes — release builds were failing
  on the brews step (#19).
- `.goreleaser.yaml`: pointed at the canonical
  `jeremylongshore/gastown-viewer-intent` repo URL (#18).

## [v0.5.0] - 2026-05-23

Phase 2 (Option B-minus): daemon hardening, read-only memories panel,
triage queue, sync pill.

### Added

- Memories panel: read-only `bd memories` viewer with policy-driven
  redaction (`000-docs/005-PP-POLICY-memories-classification`). Default
  view shows redaction markers; full content requires explicit
  `?reveal=true` in the daemon URL.
- Triage queue: `/api/v1/human` exposes beads carrying the `human`
  label (read-view; respond / dismiss deferred to a future bead behind
  the session-token gate).
- Dolt sync pill: `/api/v1/sync` returns a composed `DoltSyncState`
  body; never errors (failures encode as `health: "unknown"` with a
  tooltip string). Header pill in the web UI is bound to this surface.
- `gt wisps list --json` integration: molecules now read from gt 0.9's
  wisps surface rather than the legacy `.beads/molecule.json` file.
- 7-layer test enforcement: `@intentsolutions/audit-harness` installed
  as a web devDep with a hash manifest at `.harness-hash`.
- Doc-quality CI gates: `markdownlint-cli2`, frontmatter validator,
  Vale prose style, lychee link check on every PR touching
  `**/*.md`.

### Changed

- Daemon binds loopback-only by default; `--host=0.0.0.0`, `::`,
  private LAN, and link-local addresses are rejected at startup with
  an actionable error message.
- Origin-allowlist middleware sits outermost on every HTTP request;
  cross-origin requests are rejected with `403 ORIGIN_REJECTED`
  (defense against DNS rebinding + CSRF from any tab on the dev box).
- Session-token plumbing: `~/.config/gvid/token` (mode 0600) is
  generated at first start; `RequireTokenMiddleware` is wired but
  registered against zero state-changing routes today — installed
  ahead of any future POST endpoints.

### Security

- THREAT_MODEL.md committed alongside the daemon hardening work;
  documents the loopback bind, origin allowlist, session token, and
  memory-classification invariants.
- Memory redaction (`internal/api/memoryredact.go`) applies the
  partner-name + secret-pattern denylists from
  `005-PP-POLICY-memories-classification` before any memory crosses
  the HTTP boundary.

[Unreleased]: https://github.com/jeremylongshore/gastown-viewer-intent/compare/v0.6.0...HEAD
[v0.6.0]: https://github.com/jeremylongshore/gastown-viewer-intent/compare/v0.5.0...v0.6.0
[v0.5.0]: https://github.com/jeremylongshore/gastown-viewer-intent/releases/tag/v0.5.0

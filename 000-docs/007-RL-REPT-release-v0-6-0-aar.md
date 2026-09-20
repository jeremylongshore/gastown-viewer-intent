---
title: "Release v0.6.0 — TUI Tier-1 build-out (AAR)"
date: 2026-05-26
audience: engineering
owner: jeremylongshore
status: published
---

# Release Report: gastown-viewer-intent v0.6.0

## Executive Summary

| Field | Value |
|---|---|
| Version | v0.6.0 |
| Released | 2026-05-26T02:19:04Z |
| Released by | jeremy |
| Bump type | MINOR (one user-facing feature, no breaking changes) |
| Tag SHA | 7f84197 |
| Approval base SHA | b31b339 |
| Release duration | ~3 minutes (ceremony to tag push); ~90 s (goreleaser workflow) |

## Pre-Release State

| Surface | Status |
|---|---|
| Open PRs | 0 |
| Branches ahead of master | 7 stale local feature branches (already merged upstream — cleanup non-blocking) |
| Working tree | Clean (only `.beads/export-state.json` untracked — ephemeral) |
| Unpushed commits | 0 |
| Stashes | 3 old (from prior feature branches; not release-relevant) |
| Security advisories | 0 |
| Dependabot alerts | 0 |
| Beads in-progress | 0 |
| Beads open (snapshot) | 11 (epic + Tier-2/3 follow-ups + unrelated work) |

## Changes Included

### Features (1)

- **TUI Tier-1 build-out** (#24, `gastown-bkv` closed). Registry-driven
  keybindings, Focus enum (Board / Memories / Triage / Detail),
  combo-key state machine, help overlay, Memories tab (read-only,
  daemon-redacted, no reveal — Council Q2 invariant), Triage tab
  (read-only human-flag queue). Adopted from
  [Dicklesworthstone/beads_viewer](https://github.com/Dicklesworthstone/beads_viewer).
  ~1,300 LOC across `internal/tui/`, 9 new dispatcher tests.

### Fixes (round-1 Gemini review on #24)

- HTTP client status-code enforcement (shared `decodeJSON` helper).
- KeyRegistry.Dispatch releases RLock before invoking handler.
- UTF-8 rune-safe truncation across 4 sites (`truncateRunes()`).
- Help overlay comment corrected (Detail does NOT inherit Board bindings).

### Release infrastructure fixes

- `cmd/gvi-tui/main.go`: hardcoded `const version = "0.1.0"` → `var version = "dev"` with goreleaser ldflags injection. Every prior release binary reported `0.1.0` regardless of tag — fixed.
- `.goreleaser.yaml`: added `-X main.version={{.Version}}` to the gvi-tui build.
- `web/package.json` + lockfile: 0.0.0 → 0.6.0 (never bumped from Vite default).
- `go.sum`: `go mod tidy` cleanup (dropped `go-udiff v0.2.0`, `x/exp/golden`).

### Breaking changes

None.

## Documentation Updates

| Artifact | Change |
|---|---|
| `CHANGELOG.md` | Created (Keep a Changelog 1.1.0 format) with v0.6.0 + retroactive v0.5.0 sections + comparison links |
| `README.md` | Header bumped to v0.6.0; new "What's New in v0.6.0 (TUI Tier-1)" section; v0.5.0 What's New moved to Previous highlights |
| Gist `cd5d24298d05140eca8a3ef2cb2773f3` | Bulk-replaced v0.5.0 → v0.6.0; stale "TUI minimal nav only" claim corrected; "What's New in v0.6.0" subsection prepended; full CHANGELOG appended per `/release` v2.3 standard |

## Phase 2.6 CHANGELOG + SemVer Conformance Gate

All deterministic checks passed:

- ✅ Target `v0.6.0` is valid SemVer 2.0.0
- ✅ Monotonic forward bump: v0.5.0 → v0.6.0
- ✅ Dated header `## [v0.6.0] - 2026-05-26` present
- ✅ 3 Keep-a-Changelog sections (Added / Changed / Fixed)
- ✅ All sections contain bullet items
- ○ `web/package.json` has no `files[]` (non-blocking — not an npm-published package)

## Security & Compliance

| Gate | Status |
|---|---|
| Secrets scan in diff since v0.5.0 | ✅ clean |
| `.env` files tracked | ✅ none |
| npm audit (web/, prod deps) | ✅ 0 vulnerabilities |
| License | ✅ MIT |
| Branch protection on master | ⚠ NOT configured (open follow-up — non-blocking for this release) |
| GitHub security advisories | ✅ 0 open |
| Dependabot alerts | ✅ 0 open |

## Metrics

| Metric | Value |
|---|---|
| Commits since v0.5.0 | 8 |
| Files changed | 12 |
| Lines added | +2,074 |
| Lines removed | -195 |
| Net delta | +1,879 |
| Days since v0.5.0 | 3 |
| Contributors | 1 |

## Release Pipeline Outcome

| Step | Result |
|---|---|
| Master push | ✅ b31b339..7f84197 |
| Tag push (v0.6.0) | ✅ new tag |
| `release.yaml` GH Actions run | ✅ success (~90 s) |
| GitHub Release | ✅ published with 11 assets (linux/darwin × amd64/arm64 tarballs + .deb + .rpm + .apk + checksums) |
| Homebrew tap | ⏸ disabled — pending `HOMEBREW_TAP_TOKEN` rotation (bead `gastown-die`) |
| Gist (one-pager + audit + changelog) | ✅ updated 02:19:15Z |

## Quality Gates

| Gate | Status |
|---|---|
| `go build ./...` | ✅ green |
| `go test ./...` | ✅ green (10 packages, 0 fail) |
| `go vet ./...` | ✅ clean |
| `gvi-tui --version` via ldflags | ✅ reports `v0.6.0` (was stuck at 0.1.0) |
| PR #24 CI (test / lint / web) | ✅ green |
| PR #24 Gemini review | ✅ round-1 findings (8) all addressed in commit 0466daf |
| `bd doctor` | not run this session |

## Rollback Procedure

If issues are discovered:

```bash
# Remove from remote
git push origin --delete v0.6.0
gh release delete v0.6.0 --yes

# Revert local
git tag -d v0.6.0
git revert 7f84197
git push origin master
```

The gist update is non-destructive (gh gist edit preserves history) — no rollback required there. The bead state is append-only — no rollback required.

## Lessons Learned

- **`gvi-tui --version` had been wrong since the first release.** The hardcoded `const version = "0.1.0"` was masked because nobody read the output until the operator audit doc surfaced it. Locking in ldflags injection plus a Phase 1.1 check that ALL `version` strings in source match the tag would have caught this 4 releases ago. Adding this to `/release` Phase 1.1 next.
- **`web/package.json` version drift** is the same class of bug — the file's been at `0.0.0` since the Vite scaffolder created it, no one looked. Adding to the Phase 1.1 audit.
- **Phase 2.6 awk gate has a copy-paste typo** — the heredoc-style shell substitution mangled the awk `` reference. Worked around in this run; the skill SKILL.md should be patched with proper awk-quoting (filed as follow-up to `/release` skill maintenance).
- **TaskCreate gentle-reminder system reminder** fired ~10 times across this session. Project rules (`~/.claude/CLAUDE.md` + `~/000-projects/gastown-viewer-intent/CLAUDE.md`) explicitly route task tracking through `bd`. The reminders are noise here — but ignoring them silently is the right call; do not switch tools.

## Post-Release Checklist

- [x] Verify tag exists locally + remote
- [x] Verify GitHub Release published with assets
- [x] Verify gist updated
- [x] Verify release workflow green
- [x] Close PR-related bead with evidence
- [ ] Monitor error rates for 24 h (no error surface to monitor — local-first single-user)
- [ ] Update project board / roadmap (handled by closing `gastown-bkv`)
- [ ] Announce in relevant channels (per user preference)
- [ ] Clean stale local branches (non-blocking — defer to next `/repo-sweep`)

## Open Follow-Ups (filed)

| Bead | Title | Type |
|---|---|---|
| `gastown-ey8` | TUI build-out epic (parent) | Epic, open |
| `gastown-6je` | Tier-2: kanban swimlanes + detail viewport | Task, P3, open |
| `gastown-yla` | Tier-3: TUI status bar with sync pill | Task, P3, open |
| `gastown-ay7` | Tier-3: ASCII DAG + insights/history/sprint views | Task, P4, open |
| `gastown-die` | Homebrew tap re-enable after `HOMEBREW_TAP_TOKEN` rotation | (pre-existing) |
| `gastown-6nw` | Web-side unit tests (auto-escalates 2026-07-15) | (pre-existing) |

— Jeremy Longshore  
intentsolutions.io

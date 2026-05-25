# 005 · PP · POLICY · Memories Classification — 2026-05-24

| Field | Value |
|---|---|
| **Date** | 2026-05-24 |
| **Status** | In force from the day the memories panel ships (`gastown-3uf`) |
| **Authoring bead** | `gastown-hu4` |
| **Source decision** | `000-docs/004-AT-DECR-gastown-viewer-option-b-council-2026-05-23.md` Q2 + GC binding constraint |
| **Reviewers** | Jeremy Longshore (data owner); Acting Head of Board on Council 2026-05-23 |

## 1. Purpose

Specify how the `bd memories` content layer is classified, redacted, and
rendered in the Gastown Viewer Intent dashboard. The council's binding
constraint on memories (read-only-forever architectural invariant) takes
the **write** surface off the table; this policy governs the **read**
surface — what the dashboard is allowed to display, what it must redact,
and what affordances must be present on every memories-touching element.

This policy is normative for any code path that surfaces memory content in
the browser. It does NOT govern the `bd remember` CLI surface — that is
the canonical writer and its operational guidance lives in the global
`~/.claude/CLAUDE.md` § "Auto memory" section.

## 2. Scope

Applies to:

- Every HTTP handler under `/api/v1/memories/*` (read-only).
- Every React component in `web/src/` that renders memory content.
- Any future SSE event payload that streams memory deltas.

Does NOT apply to:

- The `bd remember` / `bd memories` / `bd recall` CLI commands. Those are
  governed by bd's own posture and the global Claude memory file.
- Memory content that never reaches the viewer (e.g., memories stored by
  other tools in the same `.beads/` Dolt schema).

## 3. Classification

Two classes of memory content trigger redaction in the viewer:

### Class A — Partner names

The viewer treats the following case-insensitive substring matches as
partner-name signal:

- `kobiton`
- `nixtla`
- `mudit` (covers both "Mudit Gupta" and "Mudit Goyal" variants)
- `polygon`
- `lit` (matched as a whole-word token, NOT a substring — otherwise it
  would false-positive on "literature", "literally", "splitting", etc.)
- `elm` (whole-word token; false-positive risk on "elmer", "helm",
  "realm" otherwise)

The denylist is conservatively narrow on purpose. Adding a new partner is
a one-line change to `internal/api/memoryredact.go` (lands with
`gastown-3uf`) and a corresponding update to this section of this policy.
Removing a partner from the denylist is a separate decision because it
means data that was previously redacted becomes visible — pre-existing
screen-recording exposure does not unrender retroactively.

### Class B — Secret-pattern strings

The viewer redacts any token-like substring matching the following
patterns (case-sensitive — these prefixes are case-sensitive in their
canonical issuers' formats):

| Pattern        | Issuer / kind                                  |
|----------------|------------------------------------------------|
| `sk-...`       | OpenAI API keys (also Anthropic legacy format) |
| `sk-ant-...`   | Anthropic API keys                             |
| `ghp_...`      | GitHub personal access tokens (PAT)            |
| `gho_...`      | GitHub OAuth tokens                            |
| `ghs_...`      | GitHub Apps server-to-server tokens            |
| `ghr_...`      | GitHub refresh tokens                          |
| `AKIA...`      | AWS access key IDs (long-term)                 |
| `ASIA...`      | AWS access key IDs (temporary / STS)           |
| `glpat-...`    | GitLab personal access tokens                  |
| `tlk_...`      | Twilio API keys                                |

The matching rule is: any continuous run of non-whitespace characters of
length ≥ 16 that begins with one of these prefixes. Length floor exists
to avoid false-positives on prose ("we use sk-as-an-acronym").

Patterns may be added without a separate review. Removing a pattern from
the denylist requires the same paper-trail discipline as Class A removal.

## 4. Rendering behavior

The viewer applies the following behavior to all memory content rendered
in the browser:

1. **Default state — redacted.** When the memories panel first renders,
   every Class-A and Class-B match in every visible memory is replaced
   with a `[REDACTED partner-name]` or `[REDACTED secret]` overlay
   element. The original bytes are NOT included in the rendered HTML —
   the redaction happens server-side in the handler before the JSON ever
   leaves the daemon process.

2. **Show toggle per memory.** Each memory has a "show" button that
   reveals the un-redacted version IF the engineer clicks. The reveal
   re-fetches the memory from the daemon with `?reveal=true`; the daemon
   logs the reveal event to the audit log (path TBD with the panel work
   in `gastown-3uf`). The reveal lasts until the next page reload or
   navigation away; it does not persist.

3. **No autocomplete surface.** Every input element in the viewer — even
   read-only search boxes on the memories panel — sets
   `autocomplete="off"`. The architectural invariant (read-only-forever)
   means there are no write inputs, but the search box is itself an
   input the browser could remember.

4. **No clipboard auto-copy.** The memory body is NOT auto-selected when
   focused. Clicking a memory copies its ID to the clipboard, never the
   content. Engineer wants the content → CLI: `bd recall <id>`.

5. **Screen-share warning banner.** When the memories panel mounts, a
   non-dismissible banner at the top reads "MEMORIES — sensitive content;
   close before screen-sharing." Banner stays for the lifetime of the
   panel; engineer must navigate away to remove it.

## 5. False-positive handling

Engineer can mark a specific memory as "false-positive — render in full"
via a CLI: `bd memories --no-redact <id>`. This sets a flag on the memory
that the viewer's redaction layer honors. Flag is per-memory, not
per-pattern — turning off redaction for one memory does not silence the
denylist globally.

## 6. Logging

The daemon logs:

- Reveal events (memory ID, timestamp, but NOT the revealed content).
- False-positive marks (memory ID, timestamp, who marked it — always the
  daemon-owning user, but logged for parity).

The daemon does NOT log:

- Memory content. Ever.
- The session token (see THREAT_MODEL.md).
- Any partner-name string from a redacted memory, even when reveal is on.

## 7. Review cadence

This policy is reviewed:

- When a new partner is added to the engagement portfolio (additive).
- When a memory leak is suspected (root-cause analysis updates the
  classification rules).
- Annually as part of `gastown-cr5`'s closeout review, regardless of
  whether incidents occurred.

## 8. Open items

Tracked under the `gastown-cr5` epic. None block `gastown-hu4` from
shipping; these complete the policy-to-code wiring in subsequent bursts.

- `gastown-3uf` — implement `internal/api/memoryredact.go` applying
  this policy. Add unit tests with one fixture per Class-A name and one
  per Class-B pattern.
- Future bead — surface the reveal audit log in a viewer panel so the
  engineer can see "I revealed X memories today" without grepping logs.
- Future bead — extend the secret-pattern denylist with project-specific
  prefixes from a config file (today the list is compiled in; the future
  list reads `internal/api/secret-patterns.yaml`).

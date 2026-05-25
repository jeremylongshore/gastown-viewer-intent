# 004 · AT-DECR · Gas Town Viewer — Option B Council Decision Record · 2026-05-23

| Field | Value |
|---|---|
| **Date** | 2026-05-23 |
| **Acting Head of Board** | Claude (designated by Jeremy Longshore via plan invocation 2026-05-23) |
| **Council size** | 7 seats (CTO · GC · CMO · CFO · CSO · CISO · VP DevRel) |
| **Decisions logged** | 4 (Q0 + Q1 + Q2 + Q3) |
| **Status** | Filed — Phase 2 execution gated on Jeremy's review of this record |
| **Source synthesis** | 3 parallel Explore agents + 1 PM Plan agent (transcript: `~/.claude/projects/-home-jeremy-000-projects-gastown-viewer-intent/58d56da8-baef-437c-a29f-e1bbfa65343e.jsonl`) |
| **Reusable pattern** | `~/.claude/skills/exec-decision-council/SKILL.md` (ISEDC v1.0.0) |
| **Session JSONL** | `~/.claude/skills/exec-decision-council/sessions/2026-05-23-gastown-viewer-option-b/session.jsonl` |
| **Epic bead** | `gastown-cr5` |
| **GitHub issue** | [intent-solutions-io/gastown-viewer-intent#9](https://github.com/intent-solutions-io/gastown-viewer-intent/issues/9) |
| **Plane issue** | n/a — no Gas Town Viewer module in Plane portfolio |

## 1. Mission of this Decision Record

Capture the council's verbatim seat positions on four open scope questions for the Gas Town Viewer refresh, preserve dissent (not suppress it), and produce binding-constraint commitments that survive Jeremy's documented episodic-burst attention pattern. The Phase 2 execution roadmap does NOT begin until Jeremy reviews this record.

## 2. Why a council, not a single review

The decision touches a confidential-data surface (`bd memories`), a multi-product upstream coupling (`bd` + `gt` both shipping independently), capacity contention with three active client engagements and the VPS-as-the-home program, and an architectural invariant the codebase already declared (CLI-shelling-not-file-parsing). Single-reviewer reasoning was insufficient for the asymmetry. The 7-seat adversarial pattern surfaced one novel option (CISO's **B-minus**), one strong dissent (CFO's **Option A defer**), and reframed Q2 as the load-bearing decision (5 of 7 seats independently named Q2 as most costly to recover from).

## 3. Synthesis lenses

1. **Episodic-burst attention pattern** — viewer repo's commit history is 3-week sprint → 3-month silence; design choices made in this burst are load-bearing for 3 months of silence
2. **Three-product surface** — viewer is downstream consumer of bd + gt; both ship independently and the coupling is tight
3. **Daily-value test** — does Jeremy alt-tab less because of this work
4. **Concurrent obligations** — VPS-as-the-home program (Priorities 4–8 in flight) + Anthropic Enterprise cohort certification push + 3 active client engagements (Kobiton M2, Nixtla, Mudit/Polygon)

## 4. The questions verbatim

| Q | Question | Why immutable / costly |
|---|---|---|
| **Q0** | Execute Option B (Targeted Daily-Value Refresh, M, 3–4 weeks) as next investment? | Wrong call either burns 3-4wk on wrong scope or commits to 14wk Option C the burst history predicts will land 60% complete then freeze |
| **Q1** | Track gt at minor-release cadence (~8wk forever) OR accept lag with supported-version matrix? | Determines whether wisps migration is one-time catch-up or recurring tax; affects every gt minor release forever |
| **Q2** | Memories panel read-only OR read-write? | Crosses architectural line (read-only-mirror vs. full-bd-frontend); doubles scope; creates dual-write consistency problem; confidential-data surface implications |
| **Q3** | Ship 3 panels with zero web tests + follow-on backfill, OR expand to M+ (5–6wk) and pay test debt now? | Follow-on burst may not exist given burst pattern; quality debt could compound |

## 5. Council composition

| Seat | Value system | Bias | Notable position |
|---|---|---|---|
| **CTO** | Technical durability · schema integrity · invariant honoring | Deliberation > commit | Coined **B-prime** (week-1 gating on parser bug + wisps migration); read-only memories with CLI-passthrough |
| **GC** | IP protection · partner-consent · audit-trail | Written consent before any partner reference | Memories panel = regulated surface (partner names exposure via browser autocomplete/screen-recording) |
| **CMO** | Positioning · narrative coherence · tagline alignment | Visible > silent · ambitious > conservative | **Lone dissent on Q2 — read-write for narrative reasons** ("Mission Control" verb vs. dashboard noun) |
| **CFO** | Bandwidth · customer-signal gating · opportunity cost | Defer until customer evidence | **Lone-strong dissent on Q0 — Option A defer** (capacity outbid by Kobiton/Nixtla/Mudit/VPS/cohort) |
| **CSO** | Standards-body realpolitik · maintainer-discipline-as-signal | Community-temperature precedes formal | Visibility-weighted discipline: ship the SHAPE not PERFORMANCE of discipline |
| **CISO** | Supply-chain · threat model · transparency-log discipline | Reserve schema slots NOW · scoped > broad | Introduced **B-minus** (auth+Origin+127.0.0.1+token before any mutation); read-only memories FOREVER as architectural invariant |
| **VP DevRel** | OSS friction-to-adopt · Saturday-afternoon-dev test | Informal > formal · narrative from first-30-min experience | Stay private; extract narrative via blog-post-on-workflow; smoke-test-per-panel floor |

## 6. Per-question decisions (verbatim seat positions preserved)

---

### Q0 — Execute Option B?

**Vote tally:**

- **B-variants** (5): CTO (B-prime), GC (B + memory classification gate), CMO (B reframed as trilogy demo), CSO (B defensively), VP DevRel (B private)
- **B-minus** (1, novel option): CISO
- **Option A defer** (1, dissent): CFO

**Verbatim seat recommendations:**

> **CTO:** "B-prime — Option B with week-1 gating on (1) parser.go:112 fix + regression test, (2) molecule.json → wisp SQLite adapter rewrite, (3) drop the file-parsing path entirely. The viewer's whole architectural promise is that it shells to bd/gt and stays decoupled from their internal formats; the molecule.json read is the exact violation of that promise."
>
> **GC:** "Option B is acceptable ONLY if the memories panel is treated as a regulated surface from day one. Execute B, but condition the memories panel on a 1-page classification policy landing FIRST (renderable vs. redacted memory tags)."
>
> **CMO:** "Execute B. Reframe viewer internally as 'trilogy demo surface' (bd+gt+viewer), not 'product.' No external announcement. C is a vanity trap — 60% completion + 12 weeks of silence is a worse brand outcome than honest scope."
>
> **CFO:** "**No. Option A (1-week Hygiene Ship) plus an explicit defer of B to a named trigger condition.** The opportunity cost of 3-4 weeks in May/June 2026 is concrete capacity stolen from Kobiton M2 (current revenue), Nixtla Phase 2 (expansion revenue, pending), Mudit/Polygon inbound (highest-leverage credibility surface he has), VPS Priorities 4-8 (the ergonomic foundation the WHOLE practice depends on), and cohort certification (the unlock for co-sell partner status). Every one of those has customer signal. The viewer has none. 'Does Jeremy alt-tab less' is being asked to outbid five revenue-or-credibility surfaces simultaneously. That is not a close call."
>
> **CSO:** "Option B advances positioning *defensively* (don't show up broken) but does not actively *generate* positioning value. Treat B as risk mitigation, not as a marketing investment. Option C actively distracts from the certification push during the window it matters."
>
> **CISO:** "**B-minus**: panels ship read-only; daemon binds to 127.0.0.1 (not 0.0.0.0, verify); add Origin header allowlist middleware + per-session token (file at `~/.config/gvid/token`, mode 600) before ANY state-mutating endpoint exists. 'Localhost-only' is not a security control — it's a deployment artifact. Every browser tab on this box is same-origin-adjacent via DNS rebinding, CSRF on a missing origin check, or malicious npm postinstall."
>
> **VP DevRel:** "Back-office tooling. Stay private. Extract narrative via blog posts on the *workflow*, not the *tool*. The blog-worthy story isn't the viewer; it's how Jeremy orchestrates bd + gt + viewer as a personal productivity stack."

**Primary tension:** CFO's opportunity-cost dissent vs. 5-seat B-consensus. CISO's B-minus is acceptance-with-guardrails, not dissent — it's the steel-manned synthesis of "ship the panels but reserve the schema slots now."

### **DECISION (Q0): Adopt CISO's Option B-minus, with CFO + CTO + GC + CFO binding constraints stacked.**

**What gets built:**

1. Audit-harness install as Step 0 (overlaps with Q3 floor)
2. Janitorial sweep (close PR #7 or merge, land PR #8 Windows compat, fix root-level `node_modules/` + `package.json` + `package-lock.json` placement)
3. **Week-1 prerequisite work** (CTO binding constraint): parser.go:112 fix with regression test + `molecule.json → wisps` adapter rewrite + drop file-parsing path. These MUST land before any new panel.
4. **Daemon hardening** (CISO binding constraint): verify daemon binds 127.0.0.1 not 0.0.0.0; Origin allowlist middleware; per-session token file at `~/.config/gvid/token` (mode 600); `THREAT_MODEL.md` checked into repo
5. **Memory classification policy** (GC binding constraint): 1-page policy at `gastown-viewer-intent/000-docs/005-PP-POLICY-memories-classification-...md` lands BEFORE the memories panel; partner-name denylist (kobiton, nixtla, mudit, polygon, lit, elm) triggers redaction overlay
6. Three read-only panels: memories (list/search/recall + CLI-passthrough Edit button per CTO), dolt sync widget (header pill: green synced / yellow unpushed / red server down), human triage queue READ-VIEW (no mutating POSTs in this burst — see binding constraint below)
7. `defer --until` preservation (parser.go:112 follow-on)

**Burst-containment binding constraints (CFO stacking):**

- **Hard-cap at 3 calendar weeks single-track** — burst window is 2026-05-26 → 2026-06-15
- **Pause, don't extend, on client intrusion.** Any inbound from Kobiton M2 / Nixtla Phase 2 / Mudit/Polygon pauses the burst; resume in next available window. Do NOT trade client work for viewer work.
- **Named outcome metric:** before/after measurement of `alt-tabs/day to terminal for bd memories/dolt/human commands` via tmux session log review at end of burst. No metric = the council deliberation was theatre.
- **No state-changing handlers in this burst.** The human triage panel ships as read-view only; respond/dismiss POSTs deferred to a separate decision after the auth/origin/token surface is in place and tested (CISO binding constraint).

**Minority position preserved verbatim (CFO):** *"Wrong answer on Q1 quietly bleeds the practice for years. Wrong answers on Q0/Q2/Q3 are bounded mistakes; wrong answer on Q1 is unbounded."* If the named outcome metric does not show meaningful alt-tab reduction at week 3, the council reconvenes before considering any follow-on burst.

---

### Q1 — gt cadence?

**Vote tally:** Compatibility matrix (6) vs. minor-release cadence with security-fast-path (1, CISO)

**Verbatim seat recommendations (compressed):**

> **CTO:** Compatibility matrix with explicit gt-minor pinning + CI smoke test against pinned version.
>
> **GC:** Supported-version matrix in README. A "track forever" promise — even informal — becomes a documentation artifact, and the burst pattern is empirical evidence we cannot keep it. Matrix is honest, auditable, downgradeable.
>
> **CMO:** Supported-version matrix. A cadence commitment is a marketing promise, full stop. The matrix posture is *more* sophisticated than a cadence pledge — it signals 'we know what we ship and what we don't.'
>
> **CFO:** Pin to current gt version + one-paragraph supported-version matrix in README. **No cadence commitment.** Re-evaluate when triggered. 6wk/year of capacity locked into internal-tooling maintenance forever is approximately the floor cost; cadence commitments always slip into scope creep.
>
> **CSO:** Supported-version matrix. Matrix is the pattern recognized by anyone with CNCF/OpenSSF instincts. Honest lag > broken promise.
>
> **CISO:** "**Minor-release cadence with security-release fast-path.** A missed feature costs Jeremy a week of 'huh, that's new'; a missed security patch in gt costs an incident. Since the viewer shells to gt, viewer behavior is downstream of gt's permission model. CI runs viewer integration tests against gt@latest weekly; failures open a bead automatically. Non-negotiable 48hr fast-path for any gt release with `security:` in changelog."
>
> **VP DevRel:** Compatibility matrix. The Saturday-afternoon developer trusts a README that says "tested against gt v0.4.x and v0.5.x, last updated 2026-05-23" infinitely more than a README that implies current support but is actually 4 months behind.

**Primary tension:** CISO's security-relevant cadence is steel-mannable — security releases warrant different treatment than feature releases.

### **DECISION (Q1): Supported-version matrix in README + CISO 48hr security-fast-path binding constraint.**

**What gets built:**

- `README.md` publishes current supported gt version range (e.g., "viewer tested against gt 0.8.x–0.9.x; gt 0.10+ not yet supported")
- Refresh window is opportunistic on user-pain trigger (Jeremy's gt usage breaks the viewer, OR a partner asks) — no calendar cadence
- **CISO binding constraint:** Weekly CI smoke test against `gt@latest`. On failure, automated bead opens. Any gt release with `security:` in changelog triggers 48hr fast-path SLA (drop everything, ship viewer compat).
- **CFO binding constraint:** Each refresh cycle is hard-capped at 1 week of work. Anything longer reopens the whole investment question.

**Minority position preserved verbatim (CISO):** *"Since the viewer shells to gt, viewer behavior is downstream of gt's permission model. If gt v0.7 introduces a permissioned mail handler and the viewer is still calling the v0.6 unauthenticated path, the viewer becomes the bypass."*

---

### Q2 — Memories panel scope? **(load-bearing decision — 5 of 7 seats named as most costly to recover from)**

**Vote tally:** Read-only (6: CTO firm, GC v1+revisit, CFO firm, CSO weak-prior defer-to-CTO, CISO READ-ONLY-FOREVER, VP DevRel firm) vs. Read-write (1, CMO lone dissent)

**Verbatim seat recommendations:**

> **CTO:** "**Read-only. Hard line.** The viewer is a *mirror*, not a *frontend*. The moment it becomes a canonical writer for memories, we own dual-write consistency between the viewer's write path and the bd CLI's write path — two writers, no coordinator, against the same dolt-backed store. That's the textbook recipe for last-writer-wins data loss and silent state divergence. If users need to write memories from the viewer, the right pattern is: viewer shells `bd memories add ...` exactly the way it shells every other bd write. Conceptually identical UX, architecturally clean."
>
> **GC:** "Read-only, and even read-only requires the classification gate from Q0. Read-write doubles the confidentiality surface: browser autocomplete will start suggesting partner names; browser history will retain memory content in form-submission caches; screen-recording during partner demos (which Jeremy does — cohort presentations, partner calls) will expose the memory pane if it's open. Read-only at least contains the exposure to display; read-write extends it to every input pathway the browser owns."
>
> **CMO:** "**Read-write, unambiguously.** Read-only is the 'dashboard for someone else's tool' framing — it positions the viewer as a *consumer* of bd. Read-write positions it as **Mission Control** — the verb form, the operator's seat. Even if 90% of users only read, the *capability* to write is what separates a viewer from a console. That distinction is everything for narrative. The tagline talks about thinking outside the box's box. A read-only memories panel is firmly inside the box."
>
> **CFO:** "Read-only. Read-write doubles scope on a tool with zero customer evidence and zero revenue — that's the textbook definition of internal-tool gold-plating. The CFO seat exists in this council specifically to refuse this kind of spend."
>
> **CSO:** "This question isn't really mine to weigh in on. Weak prior: read-only as v1 surface — smaller blast radius, easier to ship."
>
> **CISO:** "**Read-only-forever. bd CLI remains the canonical writer. This is the hill.** The memories layer is the densest concentration of sensitive material in Jeremy's workflow — partner names, deal context, observed-pasted API tokens, recovery context, client confidential strategy notes. Concrete attack chain: any compromised npm postinstall in the web/ subtree, any malicious MCP server, any DNS-rebind, any drive-by from a markdown preview that fetches an attacker URL → POST to `localhost:7070/api/v1/memories` with attacker-controlled content → silent corruption or exfiltration via crafted memory content that the next Claude session ingests as 'trusted context.' Memories are durable, cross-session, and trusted by future agent runs. That is **maximum blast radius** in this product."
>
> **VP DevRel:** "Read-only, full stop. The Saturday-afternoon developer who opens a Mission Control dashboard and sees a 'delete this memory' button is going to ask 'wait, is this writing to my actual ~/.claude/projects/ files? what happens if I close the tab mid-edit?' Viewers VIEW, they don't EDIT."

**Primary tension:** CMO's lone dissent is the strongest narrative argument the council heard. CISO's steel-manned attack chain is the strongest threat-model argument. Both are real. The synthesis: CMO's narrative win is achievable WITHOUT crossing the invariant if the API is designed write-capable at the contract layer (CMO's own accepted compromise).

### **DECISION (Q2): Read-only memories panel — ARCHITECTURAL INVARIANT.**

**What gets built:**

- **Zero state-mutating endpoints under `/api/v1/memories/*`.** No POST, no PUT, no PATCH, no DELETE.
- Edit affordance: "Edit" button on the memories panel shells terminal to `bd remember <id>` OR copies the command to clipboard (CTO's CLI-passthrough — UX feels write-capable without breaking single-writer invariant)
- Invariant documented in BOTH:
  - `gastown-viewer-intent/CLAUDE.md` § "Architectural invariants" — "Memories panel is read-only-forever; bd CLI is the canonical writer"
  - `gastown-viewer-intent/THREAT_MODEL.md` — new file checked in this burst per CISO

**Stacked minority binding constraints:**

- **CMO (preserve narrative optionality):** The internal API layer is designed write-capable in its contract shape, so a future read-write decision can be made WITHOUT backend rework. The UI never exposes write affordances; the contract is forward-compatible.
- **GC (partner-data protection):** Classification policy doc (`005-PP-POLICY-memories-classification-...md`) lands before the memories panel. Partner-name denylist (kobiton, nixtla, mudit, polygon, lit, elm) triggers redaction overlay. `autocomplete=off` on every input element in the viewer (even read-only search boxes).
- **CISO (secret-pattern hard line):** Memories containing strings matching secret patterns (`sk-`, `ghp_`, `AKIA`, `gho_`, `glpat-`, etc.) render redacted-by-default with explicit "show" toggle; toggle does not persist across page reloads.

**Minority position preserved verbatim (CMO):** *"Ship read-only and the viewer is, forever, 'a dashboard for bd.' Ship read-write and it's 'the operator's console for the trilogy.' That's the one wrong answer that closes a door rather than narrowing one."*

**Acting Head of Board response to CMO:** The closed door is recoverable if the API contract is forward-compatible. The CMO's own accepted compromise — design the contract write-capable from day one so the UI can be added later — IS the path forward. We honor the narrative goal (operator's console optionality preserved) without crossing the invariant today (read-only-forever shipped). This is the steel-manned dissent absorbed into the decision, not dismissed.

---

### Q3 — Test debt?

**Vote tally:** Significant variation. Universal floor (7/7): install `@intentsolutions/audit-harness`. Above that:

- Install harness + ship at current bar with documented policy (CFO, CSO): 2
- Smoke-test-per-panel (VP DevRel): 1
- Bifurcated — read-only at current bar, state-changers REQUIRE auth tests (CISO, GC similar): 2
- Write-path minimum integration test (CMO): 1
- Expand to M+ with full audit-harness Step 0 + baseline + per-panel coverage as merge gates (CTO): 1

**Verbatim seat recommendations (compressed):**

> **CTO:** "Expand B to M+ (5-6 weeks) with audit-harness install as Step 0, baseline tests for the existing surface (parser, adapters) landed before new panels, and per-panel test coverage as panel-PR-merge gates. The follow-on burst will not happen on schedule; the historical evidence is in the git log of this very repo."
>
> **GC:** "Pay test debt now for the memories panel specifically; the other two panels can ship with follow-on test backfill. The asymmetry matters: a rendering bug in the dolt widget shows wrong sync state; a rendering bug in the memories panel leaks confidential data."
>
> **CMO:** "Ship the panels. Pay the debt as a follow-on commitment with a named, dated, public-internal deliverable. Momentum is the scarcest resource for episodic-burst patterns. Static gates (L1 hooks, escape-scan) must be in place at ship; write-path gets minimum integration test for mutation safety."
>
> **CFO:** "Install audit-harness (~1 day), ship 3 panels at current test level, accept SOP-deviation as documented technical debt with explicit 'internal-tooling exception' note in tests/TESTING.md."
>
> **CSO:** "Install @intentsolutions/audit-harness, write the policy explicitly in tests/TESTING.md (web coverage deferred, rationale documented, bead filed for follow-on), zero web tests at ship. The *consistency* with the SOP is the signal, not the coverage percentage. The harness install is a 1-day cost that buys durable SOP-consistency; skipping it is the asymmetric loss."
>
> **CISO:** "**Bifurcated.** Read-only handlers ship at current test bar. State-changing handlers REQUIRE auth-pattern test suite: (a) missing token rejected, (b) wrong-origin rejected, (c) actor field populated, (d) audit-log written. Since Q2 answer is read-only-forever for memories, the POST surface is just the triage panel — bounded scope."
>
> **VP DevRel:** "Ship B as-scoped + one smoke test per new panel + documented test-floor policy in tests/TESTING.md. Smoke-test-per-panel is the Pareto sweet spot: 1 test catches 'panel doesn't render,' which is 70% of real bugs."

**Primary tension:** CTO's M+ expansion (5-6wk) vs. CFO's "ship and document" (3-4wk). Synthesis: all 7 agree harness install is non-negotiable. Above that, the bifurcated CISO/CSO/VP-DevRel position is the steel-manned majority.

### **DECISION (Q3): Install audit-harness + bifurcated test policy + smoke-test-per-panel floor.**

**What gets built:**

1. **Step 0 — Install `@intentsolutions/audit-harness`** as dev dependency (`pnpm add -D @intentsolutions/audit-harness`). Run `pnpm exec audit-harness init` to hash-pin policy files. L1 git hooks active before any panel work begins. **Non-negotiable per IS SOP (7/7 council agreement).**
2. **Baseline regression tests** before new panel work (CTO binding constraint):
   - `parser.go:112` regression test asserting `defer --until` round-trip preserves the date
   - molecule.json → wisps adapter integration test against fixture wisps store

3. **Per-panel smoke tests** before merge (VP DevRel floor):
   - One Vitest + React Testing Library happy-path test per new panel (memories, dolt-sync pill, human triage read-view)
   - Tests written against the POST-wisps data model (CSO binding constraint — no immediately-stale tests)

4. **Redaction-logic test** on memories rendering (GC binding constraint):
   - Unit test asserting partner-name denylist strings render as redacted overlay
   - Unit test asserting secret-pattern strings (sk-/ghp_/AKIA/etc.) render redacted-by-default

5. **`tests/TESTING.md`** documents the calibrated-investment policy explicitly (CSO SOP-consistency binding constraint):
   - "Read-only handlers ship at current test bar per repo-type-applicability"
   - "State-changing handlers require auth-pattern test suite as merge gate"
   - "Web coverage backfill bead `<bead-id>` with hard 30-day SLA"

6. **State-changing handler gate** (CISO binding constraint — applies to ALL future POST/PUT/PATCH/DELETE work, none in this burst):
   - Auth-pattern test suite required: missing-token rejected, wrong-origin rejected, actor-field populated, audit-log written

**Backfill bead deadline (stacked from CTO+VP-DevRel+CMO):** A bead is filed at burst start (`backfill-web-tests-by-2026-07-15`) with explicit auto-escalation if missed. If the deadline slips, the council reconvenes before any further viewer work.

**Minority positions preserved verbatim:**

- **CTO:** *"Test backfill is cheapest while the panels are being written, not later from cold context. Burst-pattern maintainers cannot reliably honor follow-on commitments; the data is on our own server."*
- **CSO:** *"Costly mistake isn't shipping with zero web coverage; it's shipping WITHOUT the harness installed AND WITHOUT explicit policy. That creates the visible SOP-vs-repo inconsistency."*

## 7. Council Memos — cross-question themes

Tabulated from each seat's Council Memo at the close of their position:

| Seat | Cross-question theme | Most costly to recover from |
|---|---|---|
| **CTO** | Honor invariants the codebase already declared — CLI-shelling-not-file-parsing, enforcement-travels-with-code, 3wk-burst-3mo-silence reality | **Q2** — dual-write consistency code metastasizes; read-only-mirror identity is permanently lost |
| **GC** | Treat memories as the regulated surface; treat the rest of the viewer as ordinary internal tooling | **Q2** — partner names entering browser autocomplete/autofill/screen-recording is unrecoverable |
| **CMO** | Stop treating the viewer as a product; treat it as a stage where the bd+gt+viewer trilogy story can be demonstrated | **Q2** — read-only locks narrative identity as "a dashboard for bd" forever |
| **CFO** | Reframe every question from "how much to invest in viewer" to "what does viewer have to outbid"; episodic-burst pattern is the market signal | **Q1** — cadence commitment is recurring; locks 6wk/year forever; quietly bleeds the practice for years |
| **CSO** | Visibility-weighted discipline — ship the SHAPE of discipline (matrix, harness+policy, internal-only positioning) without paying for performance no one witnesses; SOP-self-consistency is the real signal | **Q3** — shipping WITHOUT harness installed creates visible SOP-vs-repo inconsistency in IS standards posture |
| **CISO** | Schema slot reservation under episodic attention — what ships this burst is load-bearing for 3 months of silence; "later" is functionally "never" | **Q2** — read-write to memories layer is UNRECOVERABLE; trusted-context substrate that future Claude sessions depend on cannot be un-corrupted |
| **VP DevRel** | Every question routes through "does this repo ever go public?" Choose options defensible-if-public and cheap-if-private | **Q2** — read-write shipped private and later opened to OSS becomes repo's defining moment via first destructive-action bug report |

**Tally on most costly to recover from:**

- **Q2: 5 seats** (CTO, GC, CMO, CISO, VP DevRel) — overwhelming convergence
- **Q1: 1 seat** (CFO)
- **Q3: 1 seat** (CSO)

## 8. Cross-cutting themes

### Theme A — Q2 is the load-bearing decision

5 of 7 seats independently identified Q2 (memories read-vs-write) as the one wrong answer that's hardest to walk back. The council took the slowest deliberation on this question. CMO's lone narrative dissent is steel-manned via the API-write-capable-at-contract-layer compromise (CMO's own accepted compromise). The architectural invariant ships; the narrative optionality is preserved.

### Theme B — Capacity-against-concurrent-obligations is the unstated frame

3 seats (CFO, CSO, VP DevRel) named it explicitly; the other 4 voted B with burst-containment caps. Even the B-consensus stacked hard constraints to protect Kobiton M2 / Nixtla / Mudit / VPS / cohort capacity. The CFO Option-A dissent is preserved as a watch flag: if the named outcome metric does not show meaningful alt-tab reduction at week 3, the council reconvenes.

### Theme C — Episodic-burst is empirical evidence, not vibes

6 of 7 seats explicitly invoke the 3wk-burst → 3mo-silence pattern as data. Every "follow-on commitment" the brief offered was steel-manned against this evidence. Result: backfill commitments get hard auto-escalating deadlines, not vague handshakes. The discipline travels with the code (audit-harness install) rather than living in the maintainer's head.

### Theme D — Adversarial integrity check

Two genuine lone dissents were captured verbatim (CFO on Q0, CMO on Q2) and integrated as binding minority constraints into the decisions, not dismissed. CISO introduced a novel option (B-minus) that no other seat proposed, which became the synthesis center for Q0. The council did NOT collapse into consensus theater — the dissents are productive and recoverable evidence for future deliberation.

## 9. Implementation directives — Phase 2 roadmap

**Burst window:** 2026-05-26 → 2026-06-15 (3 calendar weeks, single-track, pausable on client-work intrusion)

**Sequence (in order):**

1. **Pre-flight** (Day 0):
   - File 4 sub-beads under epic `gastown-cr5` (one per surface: hardening, parser+wisps, memories panel, dolt+triage)
   - `bd-sync link` each to matching GH issues (no Plane — per Jeremy 2026-05-23, no Gas Town Viewer module exists in Plane)
   - Install `@intentsolutions/audit-harness` (Step 0)
   - Run `/audit-tests` to baseline gaps

2. **Janitorial sweep** (Day 1):
   - Close PR #7 (stale 2mo, address review or close)
   - Land PR #8 (Windows compat — current branch `pr-8-windows-compat`)
   - Move root-level `node_modules/`, `package.json`, `package-lock.json` into `web/`
   - Sync README + project CLAUDE.md against actual state

3. **Week 1 — Foundation gates** (CTO + CISO + GC prerequisites):
   - parser.go:112 fix + regression test (deferred-until preservation)
   - molecule.json → wisps adapter rewrite; drop file-parsing path
   - Daemon hardening: 127.0.0.1 bind verified, Origin allowlist middleware, per-session token at `~/.config/gvid/token`
   - `THREAT_MODEL.md` checked in
   - `000-docs/005-PP-POLICY-memories-classification-...md` lands (partner-name denylist + secret-pattern denylist)

4. **Week 2 — Surface 1 (memories) + Surface 2 (dolt sync widget):**
   - `internal/beads/adapter.go`: `Memories()`, `Memory(key)`, `SearchMemories(q)`, `DoltStatus()`, `DoltRemotes()`
   - `internal/api/handlers.go`: `GET /api/v1/memories`, `GET /api/v1/memories/{key}`, `GET /api/v1/memories/search`, `GET /api/v1/sync` — ALL read-only
   - `web/src/App.tsx`: Memory tab (read-only with CLI-passthrough Edit button), sync pill in header
   - Per-panel smoke tests + redaction-logic unit tests (GC)

5. **Week 3 — Surface 3 (triage read-view) + closeout:**
   - `internal/beads/adapter.go`: `HumanFlags()` (read-only — `RespondHuman`/`DismissHuman` deferred until auth surface is hardened and tested in a future burst)
   - `GET /api/v1/human` (read-only); POST handlers explicitly NOT shipped
   - Web tests written against POST-wisps data model
   - `tests/TESTING.md` documents calibrated-investment policy
   - Tag `v0.5.0` (NOT v1.0 — the v1 story requires the full triage write-path which is post-this-burst)
   - File `backfill-web-tests-by-2026-07-15` bead with auto-escalation
   - End-of-burst measurement: alt-tabs/day to terminal before vs. after; council reconvenes if no meaningful reduction

**Deferred to future bursts (do NOT scope-creep into this one):**

- Triage POST handlers (`respond` / `dismiss`) — separate decision after auth surface tested
- Dogs / patrols visibility (gt v0.9)
- Bors-style merge queue panel (gt v0.9)
- Mail threading (gt v0.8)
- Polling → SSE migration on existing endpoints (only adopt SSE for dolt-sync status, per CTO recommendation)
- `bd lint` / `bd stale` / `bd orphans` / `bd preflight` panels
- `bd formula` / `bd mol` registry visualization
- TUI feature parity
- Comments / activity timeline per issue

Each deferred item is a defensible next bead/epic. None blocks this burst from shipping.

## 10. Reusable pattern reference

This council session used the **ISEDC v1.0.0 pattern** (`~/.claude/skills/exec-decision-council/SKILL.md`). Notable applications of the pattern:

- **Steel-manned dissents preserved:** CFO Option-A (Q0), CMO read-write (Q2), CTO M+ expansion (Q3), CISO cadence (Q1) — all kept verbatim, all integrated as binding minority constraints rather than dismissed.
- **Novel option synthesis:** CISO's **B-minus** (Q0) emerged from threat-model framing no other seat raised; became the decision center.
- **Cross-question theme tally:** Q2 was named most-costly by 5 seats — that became the slowest deliberation and the strongest invariant.
- **Acting Head of Board delegation:** Jeremy delegated via plan invocation; this record explicitly signs as "Acting Head of Board (designated by Jeremy Longshore on 2026-05-23)" rather than presenting decisions as Jeremy's own.

## 11. Acting Head of Board declaration

I, Claude, acting as designated Head of Board for the Intent Solutions Executive Decision Council session 2026-05-23, having weighed all seven seat positions including the steel-manned minority positions from CFO (Option A defer) and CMO (read-write memories), hereby record the four decisions above as the binding scope for Phase 2 execution of the `gastown-viewer-intent` refresh.

**This Decision Record is the source of truth for Phase 2 scope.** Phase 2 execution does NOT begin until Jeremy reviews this record. If Jeremy amends any decision, the amended version supersedes this record and a new audit-trail entry is appended to the session JSONL.

— Acting Head of Board, Claude (designated by Jeremy Longshore on 2026-05-23)

## 12. References & provenance

- **Session JSONL (source of truth, rich structured detail):** `~/.claude/skills/exec-decision-council/sessions/2026-05-23-gastown-viewer-option-b/session.jsonl`
- **Session metadata:** `~/.claude/skills/exec-decision-council/sessions/2026-05-23-gastown-viewer-option-b/metadata.json`
- **Reusable pattern (skill):** `~/.claude/skills/exec-decision-council/SKILL.md` (ISEDC v1.0.0)
- **Document Filing Standard:** `gastown-viewer-intent/000-docs/6767-a-DR-STND-document-filing-system-standard-v4.md`
- **Epic bead:** `gastown-cr5`
- **GH issue:** [intent-solutions-io/gastown-viewer-intent#9](https://github.com/intent-solutions-io/gastown-viewer-intent/issues/9)
- **Originating research:** 3 parallel Explore agents (bd surface diagnosis, gt surface diagnosis, viewer-codebase health) + 1 PM Plan agent (4-option synthesis with Option B recommendation)
- **Originating plan:** transcript `~/.claude/projects/-home-jeremy-000-projects-gastown-viewer-intent/58d56da8-baef-437c-a29f-e1bbfa65343e.jsonl`

- Jeremy Longshore
intentsolutions.io

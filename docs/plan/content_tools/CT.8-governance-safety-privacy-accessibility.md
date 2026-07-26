# CT.8 — Content Tools: Governance, Safety, Privacy & Accessibility Conformance

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.8 |
| **Section** | Content Tools (CT) |
| **Severity** | BLOCKER (gates GA) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Trust & safety + compliance engineering |
| **Depends on** | CT.3, CT.6 |
| **Unblocks** | GA of the whole feature; CT.9 (marketplace cannot open without a review bar) |

---

## 1. Problem Statement

Content Tools takes free-text and behavioural data from students — including minors — stores it as
loosely-structured JSON, sometimes sends it to a third-party model, sometimes shows it to peers, and
will eventually run code written by people outside Lextures. Each of those is a compliance and safety
surface that no individual tool story should be re-deciding. Without one owner, the predictable
outcomes are: a tool that ships inaccessible, student PII in a provider log, a peer-visible tool used
for bullying, an AI feature that is undisclosed in a district that requires disclosure, and a DSAR
that cannot be answered because nobody registered `state_json`. This story is that owner.

## 2. Goals

- Define and enforce the **shipping gate** every tool must pass: accessibility, privacy data map,
  safety review, i18n, and (where AI is involved) disclosure and evals.
- Discharge the platform's obligations for tool data: FERPA, COPPA, GDPR/DSAR, retention, and the
  EU AI Act's transparency duties, with links to the owning standards plans.
- Ship the safety controls peer-visible and free-text tools need: moderation, reporting, blocking of
  identified abuse, and instructor visibility.
- Make AI use in tools disclosed, gated by org policy, budgeted, logged and human-overridable.
- Guarantee accessibility conformance is verified, not asserted, per tool and per release.

## 3. Non-Goals

- Re-implementing platform compliance machinery (DSAR orchestration, consent ledger, retention engine
  exist in `../standards/`) — CT.8 *registers with* them.
- Content moderation of instructor-authored configuration (existing course-content policy applies).
- Legal sign-off itself — CT.8 produces the artefacts legal reviews.
- Marketplace vendor vetting (CT.9 uses this story's bar; it does not define it).

## 4. Personas & User Stories

- **As a district privacy officer**, I want a written data map of what each tool collects and where it
  goes so that I can approve Lextures without a bespoke audit.
- **As a parent**, I want to know if my child's writing is sent to an AI provider and to opt out so
  that I retain control.
- **As a student who uses a screen reader**, I want every interactive element to be operable so that a
  "more engaging" page is not a less accessible one.
- **As an instructor**, I want to be told when a student's response contains a self-harm signal so
  that I can act on it as our policy requires.
- **As a student**, I want to report an abusive peer comment inside a discussion tool so that it stops.
- **As a compliance engineer**, I want a per-tool checklist enforced by CI so that conformance does not
  depend on a reviewer remembering.
- **As an org admin**, I want to disable categories of tools (AI, peer-visible, external network) so
  that policy is enforced by configuration, not by asking teachers.

## 5. Functional Requirements

- **FR-1.** Every tool MUST ship a **Tool Data Sheet**: what it collects, retention, whether data
  leaves the platform, who can see it, AI usage, network egress, and its accessibility statement.
  Registration MUST fail without one.
- **FR-2.** CI MUST enforce a per-tool **conformance gate**: axe clean on the tool's harness stories,
  keyboard-operability test present, i18n keys complete for the default locale, data sheet present,
  state schema present, projection function present (CT.7).
- **FR-3.** The platform MUST support **org-level tool policy**: allow/deny by tool id and by
  capability class (`ai`, `peer_visible`, `network`, `media_capture`, `code_execution`).
- **FR-4.** AI-capable tools MUST be gated by `aigateway` per call (already CT.6) **and** MUST display
  an AI-use disclosure in the tool UI before first interaction, per org disclosure settings.
- **FR-5.** Under-13 (COPPA-flagged) accounts MUST have AI tools disabled unless verifiable parental
  consent is recorded, reusing the shipped COPPA consent mechanism.
- **FR-6.** Student-authored free text sent to a provider MUST be PII-redacted (CT.6 FR-13); the
  platform MUST retain a record of *that a call occurred* even when content retention is disabled.
- **FR-7.** Peer-visible tools MUST support: instructor moderation (hide/remove), student reporting,
  per-course anonymity settings, and a hard block on rendering removed content.
- **FR-8.** Free-text tools MUST run the shipped content filter (`service/contentfilter`) on student
  submissions, with configurable actions: allow, flag-to-instructor, or block-with-guidance.
- **FR-9.** Crisis-signal detection (self-harm, abuse) MUST route through the platform's existing
  escalation path, MUST notify per org configuration, and MUST NOT be silently logged only.
- **FR-10.** `content_tool_states`, `content_tool_state_resets`, `content_tool_events` and
  `content_tool_link_sources` MUST be registered with DSAR export (S01) and the retention/deletion
  engine (S02), with documented default windows.
- **FR-11.** Students (or parents, per jurisdiction) MUST be able to obtain their tool data in the
  standard export format, and deletion requests MUST cascade correctly without orphaning aggregates.
- **FR-12.** A **student opt-out** MUST exist for AI-backed tools where the tool is not the graded
  activity itself; opting out MUST leave a non-AI path (e.g. the question is shown to the instructor
  instead) rather than blocking learning.
- **FR-13.** The platform MUST maintain an **AI transparency record** per AI tool (purpose, model
  family, human oversight, limitations) satisfying EU AI Act transparency duties, surfaced in the
  admin trust centre.
- **FR-14.** Every tool MUST declare its WCAG conformance level and known limitations; the palette MUST
  display an accessibility note for tools with limitations (e.g. drag-based tools and their keyboard
  alternative).
- **FR-15.** Drag-and-drop, canvas, timing-based and media tools MUST provide non-drag, non-timed and
  captioned alternatives respectively; a tool without them MUST NOT pass the gate.
- **FR-16.** The platform MUST provide an **incident kill path**: disable one tool platform-wide, or
  one instance, or all AI tools, without a deploy, with an operator audit record.
- **FR-17.** Retention defaults MUST be: tool state — life of the enrollment plus org record policy;
  reset snapshots — 90 days (CT.4); AI prompt/completion logs — org policy, default 30 days; link
  cache — 7 days.

## 6. Non-Functional Requirements

- **Performance** — Content filtering adds ≤ 40 ms p95 to a free-text submission; policy evaluation is
  an in-memory lookup on cached org policy.
- **Security** — Policy decisions are server-side and re-evaluated per request; a client can never
  assert its own eligibility. Moderation actions are audited and irreversible only through audited
  restore.
- **Privacy & Compliance** — Deliverables: RoPA entries (S05), a DPIA covering the AI tools (S06),
  sub-processor disclosure updates when a tool adds a processor (S07), children's-privacy analysis
  (S08), FERPA analysis (S09), and an EU AI Act transparency record (S13).
- **Accessibility** — Conformance to WCAG 2.1 AA is a *gate*, not a goal: automated axe in CI, a manual
  screen-reader script per tool category, and a documented VPAT update per release (S20).
- **Scalability** — Policy and filter calls scale with submissions; both are cached/batched.
- **Reliability** — If the content filter is unavailable, the configured failure mode applies
  (default: allow + flag for review, never silently drop student work).
- **Observability** — `lextures_content_tool_policy_denials_total{reason}`,
  `…_moderation_actions_total{action}`, `…_content_filter_flags_total{category}`,
  `…_crisis_escalations_total`, `…_a11y_gate_failures_total{tool_id}` (CI-exported). Alert on any
  crisis escalation and on filter unavailability.
- **Maintainability** — One policy service; tools declare capabilities and never implement policy.
- **Internationalization** — Disclosure, moderation and safety copy localized; content filtering
  quality per language documented, with a stated fallback for unsupported languages.
- **Backward compatibility** — Policy defaults preserve current behaviour (nothing enabled until a
  course opts in).

## 7. Acceptance Criteria

- **AC-1.** *Given* a tool registered without a data sheet or a11y declaration, *When* the server
  starts or CI runs, *Then* it fails with the tool named.
- **AC-2.** *Given* an org policy denying capability `ai`, *When* an instructor opens the palette,
  *Then* AI tools are absent, and a direct API call to an AI action returns a typed policy denial.
- **AC-3.** *Given* a COPPA-flagged student without parental consent, *When* they open a page with an
  AI tool, *Then* they see the non-AI alternative path and no provider call is made.
- **AC-4.** *Given* a student submits free text containing a slur, *When* the configured action is
  block-with-guidance, *Then* the submission is refused with guidance, nothing is stored, and the
  instructor sees an aggregate flag rather than the raw text.
- **AC-5.** *Given* a submission containing a self-harm signal, *When* it is processed, *Then* the
  configured escalation fires within 60 s and is recorded.
- **AC-6.** *Given* a peer-visible tool post is reported, *When* the instructor removes it, *Then* it
  is hidden for every viewer, the author is notified per policy, and the action is audited.
- **AC-7.** *Given* a DSAR export for a student, *When* it completes, *Then* it contains their tool
  state, reset snapshots and tool events in the standard format.
- **AC-8.** *Given* a deletion request, *When* it completes, *Then* no `content_tool_*` row referencing
  that subject remains and analytics aggregates remain internally consistent.
- **AC-9.** *Given* a drag-based tool, *When* tested with keyboard only, *Then* every item can be
  placed via the documented keyboard alternative and the tool completes successfully.
- **AC-10.** *Given* the platform-wide AI kill path is used, *When* a student interacts with an AI
  tool, *Then* they see a maintenance state, no provider call occurs, and stored state is untouched.
- **AC-11.** *Given* an AI tool, *When* a student first opens it, *Then* the disclosure is shown per
  org settings and the acknowledgement (where required) is recorded.
- **AC-12.** *Given* a release, *When* the conformance report is generated, *Then* every registered
  tool appears with its axe status, keyboard test status, WCAG level and known limitations.

## 8. Data Model

Migration `server/migrations/456_content_tool_governance.sql` (+ `.down.sql`).

```sql
-- 456_content_tool_governance.sql

-- Org-level policy over tools and tool capabilities.
CREATE TABLE IF NOT EXISTS tenant.content_tool_policies (
    org_id              UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    denied_capabilities TEXT[] NOT NULL DEFAULT '{}',   -- ai|peer_visible|network|media_capture|code_execution
    denied_tool_ids     TEXT[] NOT NULL DEFAULT '{}',
    allowed_tool_ids    TEXT[] NOT NULL DEFAULT '{}',   -- non-empty = strict allowlist
    ai_disclosure_mode  TEXT NOT NULL DEFAULT 'banner'
                          CHECK (ai_disclosure_mode IN ('none','banner','acknowledge')),
    free_text_filter_action TEXT NOT NULL DEFAULT 'flag'
                          CHECK (free_text_filter_action IN ('allow','flag','block')),
    crisis_escalation_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ai_log_retention_days INTEGER NOT NULL DEFAULT 30,
    updated_by          UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Moderation actions on peer-visible tool content.
CREATE TABLE IF NOT EXISTS course.content_tool_moderation (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id   UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    state_id      UUID REFERENCES course.content_tool_states (id) ON DELETE CASCADE,
    content_path  TEXT,                     -- JSON pointer into state_json (e.g. /posts/2)
    action        TEXT NOT NULL CHECK (action IN ('reported','hidden','removed','restored','warned')),
    category      TEXT,                     -- abuse|harassment|off_topic|self_harm|other
    reason        TEXT,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    subject_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ctm_instance ON course.content_tool_moderation (instance_id, created_at DESC);

-- Per-student AI disclosure acknowledgements / opt-outs for tools.
CREATE TABLE IF NOT EXISTS course.content_tool_ai_consents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id     UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    tool_id       TEXT,
    decision      TEXT NOT NULL CHECK (decision IN ('acknowledged','opted_out')),
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, course_id, tool_id)
);

-- Registry mirror of the declarative Tool Data Sheet (for the trust centre + audits).
CREATE TABLE IF NOT EXISTS course.content_tool_data_sheets (
    tool_id         TEXT PRIMARY KEY,
    version         TEXT NOT NULL,
    collects_json   JSONB NOT NULL,     -- field → purpose → retention
    leaves_platform BOOLEAN NOT NULL DEFAULT FALSE,
    processors      TEXT[] NOT NULL DEFAULT '{}',
    visibility      TEXT NOT NULL CHECK (visibility IN ('self','instructor','peers','public')),
    wcag_level      TEXT NOT NULL DEFAULT 'AA',
    a11y_limitations TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Backfill** — data sheets are synced from the registry on boot. **Retention** — per FR-17, enforced by
the shipped retention engine.

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `/api/v1/orgs/{org_id}/content-tool-policy` | org admin |
| `PUT` | `/api/v1/orgs/{org_id}/content-tool-policy` | org admin |
| `GET` | `/api/v1/content-tools/data-sheets` | any authenticated (trust centre) |
| `POST` | `.../content-tools/instances/{instance_id}/report` | student |
| `POST` | `.../content-tools/instances/{instance_id}/moderate` | instructor |
| `GET` | `.../content-tools/instances/{instance_id}/moderation` | instructor |
| `POST` | `.../content-tools/ai-consent` | student / parent |
| `POST` | `/api/v1/admin/content-tools/kill` (`{scope: 'tool'\|'capability'\|'all_ai', target}`) | platform admin |

- **Rate limits** — report 10/min/user; moderate 60/min/user; consent 10/min/user.
- **OpenAPI** — all routes documented; the kill route marked internal and audited.

## 10. UI / UX

**Org admin — Content Tools policy** (in the existing admin policy area): capability toggles with
plain-language consequences, tool allow/deny lists, AI disclosure mode, free-text filter action,
crisis escalation, retention fields. Changes show an impact preview ("42 courses use 3 tools you are
about to deny").

**Instructor** — a moderation queue per peer-visible instance (reported items first), with hide /
remove / restore / warn; a flags panel showing content-filter hits without exposing blocked raw text
beyond what policy allows.

**Student** — AI disclosure banner (or acknowledgement dialog) before first use of an AI tool, with an
opt-out control where FR-12 applies; a **Report** affordance on peer content; clear, non-punitive
guidance when a submission is blocked.

**Trust centre** — public-facing per-tool data sheets: what it collects, who sees it, whether data
leaves the platform, accessibility level and limitations.

**States** — *Denied by policy*: neutral placeholder naming the reason and the policy owner. *Blocked
submission*: inline guidance with an edit affordance (work is never lost). *Removed content*: tombstone
visible to the author and instructor only.

**Accessibility** — dialogs follow the shipped confirm pattern; disclosure is not a modal trap;
guidance messages are announced assertively; the moderation queue is a semantic list with clear labels.

**Copy & i18n** — `contentTools.governance.*`, `contentTools.safety.*`, with a reviewed tone for
minors (non-punitive, plain language).

## 11. AI / ML Considerations

- **Models** — the content filter and crisis detector reuse shipped `service/contentfilter` classifiers;
  no new model is introduced by CT.8.
- **Human oversight** — every automated flag is instructor-reviewable; no automated action affects a
  grade; blocks are reversible by the instructor.
- **Evals** — precision/recall tracked per category on a labelled set, with special attention to false
  positives on non-English text and on legitimate academic content (a biology lesson containing
  clinical terms must not be blocked).
- **Bias** — filter outcomes monitored by demographic aggregate (never per-student) for disparate
  impact, as required by the fairness commitments in the standards folder.
- **Fallback** — filter unavailable → configured failure mode (default allow + flag).
- **Transparency** — the AI transparency record (FR-13) is generated from manifests, not hand-written,
  so it cannot drift from what ships.

## 12. Integration Points

- **Internal** — `service/contentfilter`, `service/coppa`, `service/ferpa`, `service/gdpr`,
  `service/dpa`, `service/research_consent`, `service/aidisclosure` + `service/aigateway`,
  `service/adminaudit`, `service/notifications`, `service/accessibility`,
  `repos/platformconfig` (kill path), `internal/telemetry`.
- **Standards plans** — registers with S01 (DSAR), S02 (retention), S05 (RoPA), S06 (DPIA),
  S07 (sub-processors), S08 (children), S09 (FERPA), S13 (EU AI Act), S20 (accessibility law),
  S21 (continuous evidence).
- **CI** — new job `tools-conformance` producing the per-release conformance report artefact.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.3 (state exists), CT.6 (AI path exists).
- **Must ship before:** GA of Content Tools, any peer-visible tool (CT.22), any AI tool in a K-12
  tenant, and CT.9.
- **Shared infra needed:** content filter, consent records, audit log, notification service.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A tool ships inaccessible because the gate is advisory | H | H | Gate enforced in CI and at registration; startup fails without an a11y declaration; per-release conformance report |
| Over-blocking legitimate academic language | M | M | Default action is flag (not block); per-category thresholds; false-positive review loop with instructors |
| Under-blocking harmful peer content | M | H | Reporting always available; instructor queue; crisis escalation independent of the filter |
| Compliance obligations spread across tool stories and drift | H | H | Single owner (this story); manifest-derived data sheets so documentation cannot lag code |
| Parental opt-out makes a graded activity impossible | M | M | FR-12 requires a non-AI path; a tool that cannot provide one may not be the graded activity |
| Kill path used carelessly and disrupts classes | L | M | Audited, alerting, scoped options (tool/capability/all-AI), documented runbook |

## 15. Rollout Plan

- **Feature flag** — policy defaults are permissive-but-safe (filter=flag, disclosure=banner);
  `CONTENT_TOOLS_AI_KILL_SWITCH` and per-tool disable exist from day one.
- **Sequencing** — migration `456_*` → policy service + org UI → conformance gate in CI → disclosure &
  consent → filter + escalation wiring → moderation UI → trust-centre data sheets → compliance
  artefacts (DPIA, RoPA, VPAT delta) → GA sign-off.
- **Dogfood** — run the conformance gate against CT.10–CT.23 and fix findings before GA.
- **GA criteria** — every shipped tool passes the gate; DPIA and RoPA updated; VPAT delta published;
  escalation path tested end-to-end with the safety team.
- **Rollback** — policy is data; revert to prior policy rows. The conformance gate can be set to
  warn-only for one release in an emergency, with an explicit expiry.

## 16. Test Plan

- **Unit** — policy evaluation matrix (org deny × course allow × tool capability × role); disclosure
  mode selection; consent resolution incl. COPPA; retention window computation; data-sheet validation.
- **Integration** — DSAR export contains tool data; deletion cascades; filter actions (allow/flag/block)
  end-to-end; escalation fires; moderation hides content for every viewer; kill path.
- **End-to-end** — Playwright: student sees disclosure and opts out, gets the alternative path; student
  reports peer content and instructor removes it; blocked submission preserves the student's text in
  the editor.
- **Security** — authz matrix for policy, moderation and kill routes; attempts to bypass policy via
  direct action calls; verification that removed content is not retrievable through any read path.
- **Accessibility** — the conformance gate itself is the test: axe + keyboard scripts per tool, plus
  screen-reader review of disclosure, blocking guidance and moderation.
- **Performance / load** — filter latency under submission bursts; policy cache behaviour on update.
- **Manual exploratory** — multilingual abusive content; borderline academic content; parent-initiated
  opt-out mid-term; a tool disabled while students are mid-interaction.

## 17. Documentation & Training

- **Admin** — Content Tools policy guide; what each capability class means; retention settings.
- **Instructor** — moderation duties, what flags mean, what to do on an escalation, accessibility
  notes when choosing tools.
- **Student/parent** — plain-language explanation of AI use, data collected, opt-out rights.
- **Public trust centre** — per-tool data sheets, AI transparency records, VPAT.
- **Runbook** — escalation handling, kill path, filter outage, conformance-gate failures.

## 18. Open Questions

1. Should crisis detection run on *all* free-text tools or only those the org opts into? Proposed: all
   by default, org-disableable with an explicit acknowledgement of the consequence.
2. Do we need per-tool DPAs when a third-party tool (CT.9) introduces a processor, or does the
   marketplace agreement cover it? Proposed: marketplace agreement plus per-tool disclosure; legal to
   confirm before CT.9 GA.
3. Is `acknowledge` disclosure mode (a dialog before first AI use) too heavy for daily classroom use?
   Proposed: default `banner`; `acknowledge` available for districts that require it.
4. Should peer-visible tools default to anonymous-to-peers in K-12? Proposed: yes for K-12 program
   type, named for HE — confirm with pilot instructors.

## 19. References

- Existing files this work touches: `server/internal/service/contentfilter/`,
  `server/internal/service/coppa/`, `server/internal/service/aidisclosure/`,
  `server/internal/service/aigateway/service.go`, `server/internal/service/accessibility/`,
  `server/internal/service/gdpr/`, `server/migrations/456_content_tool_governance.sql`.
- External standards: WCAG 2.1 AA, EN 301 549, Section 508; FERPA; COPPA; GDPR Arts. 13–15, 17, 30, 35;
  EU AI Act transparency obligations; OWASP LLM Top 10.
- Related plans: [CT.3](CT.3-student-runtime-and-state-persistence.md),
  [CT.6](CT.6-grounded-context-and-link-ingestion.md),
  [CT.9](CT.9-tool-marketplace-and-third-party-tools.md),
  [S01](../standards/S01-unified-data-subject-rights-orchestration.md),
  [S02](../standards/S02-data-retention-deletion-engine.md),
  [S06](../standards/S06-dpia-pia-algorithmic-impact.md),
  [S08](../standards/S08-childrens-privacy-age-assurance-design-codes.md),
  [S09](../standards/S09-ferpa-hardening.md),
  [S13](../standards/S13-eu-ai-act-high-risk.md),
  [S20](../standards/S20-accessibility-legal-mandates.md).

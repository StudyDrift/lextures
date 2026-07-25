# AC.8 — Governance, Safety, Fairness, Privacy & Compliance

> Implementation plan. Source: cross-cutting guardrail owner for ACE; discharges Standards obligations (S06/S08/S13/S01/S02). Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.8 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS (Global) |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Trust & safety + backend platform + legal/DPO |
| **Depends on** | AC.3 (what's generated); cross-cuts AC.2–AC.7 |
| **Unblocks** | GA of the whole ACE feature (gates it) |

---

## 1. Problem Statement

ACE decides, per student, what teaching content they see, using generative AI — squarely "high-risk AI in education" under the EU AI Act and an automated decision under GDPR, on data covered by FERPA/COPPA. Shipping the mechanics (AC.1–AC.7) without a governance layer would be reckless: it could hallucinate into a lesson, systematically disadvantage a demographic through uneven adaptation quality, profile minors without safeguards, or leave students unable to understand or contest a decision about their education. This story is the single owner that makes ACE *defensible*: disclosure, human oversight, fidelity/safety enforcement, **fairness/bias auditing**, minors' safeguards, DSAR/retention coverage, and an incident path — each linked to the relevant Standards plan.

## 2. Goals

- Guarantee every ACE model call is disclosed, consented, and logged; and that no adapted content serves without passing fidelity + safety + a11y gates (enforcement, not just signals).
- Provide a **fairness/bias audit**: monitor adaptation coverage, fidelity, and lift across protected/proxy groups and flag disparities.
- Enforce minors' safeguards (COPPA/age-appropriate design) for adaptive content and profiling.
- Make ACE artifacts (profiles, variants, servings, outcomes) first-class in DSAR, retention, and the data inventory.
- Give students/guardians transparency, opt-out (AC.6), and a **contest/appeal** path; give admins an oversight console and kill-switch.

## 3. Non-Goals

- Re-implementing the platform compliance engines (S01 DSAR, S02 retention, S03 breach, S04 consent) — this story *integrates* ACE with them.
- The generation/fidelity mechanics themselves (AC.3) — this story sets policy and enforcement thresholds and audits outcomes.
- General platform AI governance (10.17 / AI governance panel) — extends it for ACE specifically.

## 4. Personas & User Stories

- **As a student**, I want to understand that content was AI-adapted, why, and how to opt out or ask a human to review it.
- **As a guardian of a minor**, I want assurance my child isn't being profiled or fed AI content without safeguards, and I want control.
- **As a DPO/compliance lead**, I want a DPIA, an automated-decision record, a data inventory entry, and evidence of human oversight for ACE.
- **As an equity officer**, I want proof the adaptation isn't helping some groups more than others — or an alert when it is.
- **As a platform admin**, I want an oversight console: what ACE is doing, disparity flags, incident controls, and a kill-switch.

## 5. Functional Requirements

- **FR-1.** ACE MUST register with the platform AI-disclosure/governance layer (10.17): feature `adaptive_content`, purpose, model, and data categories shown wherever AI features are disclosed; the AI governance panel MUST list ACE with an org-level enable/disable that an admin can use (without being a *required* on-switch for course-level enablement).
- **FR-2.** No variant MAY serve unless it is `approved` (or `auto_served` after passing gates) AND passes fidelity ≥ `min_fidelity` AND passes safety AND has no blocking a11y flag; this is enforced server-side at serve time (belt-and-suspenders over AC.5/AC.6).
- **FR-3.** ACE MUST run a **fairness audit** job computing coverage, mean fidelity, and mean lift grouped by available demographic/proxy attributes (only those already lawfully held, e.g., grade band, section, language, IEP/504 accommodation flag), with small-cell suppression, and MUST flag disparities beyond a configurable threshold to admins.
- **FR-4.** For COPPA-gated minors, ACE MUST default to **no profiling/generation** unless verifiable parental consent + org policy permit; where denied, students transparently get base content (AC.6 FR-7). Age-appropriate design: no manipulative adaptation, no dark patterns, conservative defaults (S08).
- **FR-5.** ACE artifacts (`adaptation_profiles`, `content_variants`, `adaptation_servings`, `adaptation_outcomes`, `adaptive_content_events`, opt-outs, key terms) MUST be included in DSAR export (S01) and deletion (S02), and registered in the RoPA/data inventory (S05).
- **FR-6.** ACE MUST provide a **student/guardian contest path**: a control to say "this adaptation seems wrong" that routes to the instructor (human review, per AC.5) and records the contest; and a documented right to obtain base content (opt-out, AC.6).
- **FR-7.** ACE MUST complete a **DPIA/algorithmic-impact assessment** (S06) and an EU AI Act high-risk conformity checklist (S13) as tracked artifacts before GA, kept current with prompt/model changes.
- **FR-8.** ACE MUST log an immutable oversight trail: every generation (model, prompt version, inputs snapshot, fidelity/safety results), every approval/rejection/override, every serving, every opt-out/contest — queryable for audits (S21).
- **FR-9.** ACE MUST support an **incident response** path: on a safety/fidelity incident, admins can quarantine a unit/course/variant, engage the kill-switch (AC.1), and the breach process (S03) applies if PII is implicated.
- **FR-10.** The system MUST expose an **admin oversight console** summarizing ACE activity, disparity flags, regressing units (from AC.7), cost, and incident controls.

## 6. Non-Functional Requirements

- **Performance** — Serve-time gate re-check is O(1) (flags already computed by AC.3); audits/DPIA are offline jobs, not on the hot path.
- **Security** — Oversight console + fairness data are admin/DPO-gated; demographic attributes accessed under least privilege and never sent to the model. Audit log append-only/tamper-evident.
- **Privacy & Compliance** — This is the story that maps ACE to FERPA (education records), COPPA/AADC (minors, S08), GDPR Art. 22 + EU AI Act (S13), DPIA (S06), DSAR/retention (S01/S02), RoPA (S05), breach (S03), and continuous evidence (S21). Fairness audit uses lawfully-held attributes only, with suppression to prevent re-identification.
- **Accessibility** — Generated content a11y is *enforced*, not optional: blocking a11y flags prevent serving; the contest/opt-out controls meet WCAG 2.1 AA; consistent with accessibility-law obligations (S20).
- **Scalability** — Audit jobs run per course/org on a schedule; suppression and grouping scale with cohort size.
- **Reliability** — Fail-closed: any governance-check error ⇒ serve base (never an ungated variant). Kill-switch is immediate and durable.
- **Observability** — Metrics: `adaptive_content.fairness_disparity_flag`, `.contest_opened`, `.gate_block_served_base`, `.minor_blocked`; dashboards + alerts for disparity and incidents.
- **Maintainability** — Policy thresholds centralized in `service/adaptivecontent/governance.go`; DPIA/AI-Act artifacts live in `docs/` and are referenced from the console.
- **Internationalization** — Disclosure, contest, opt-out copy localized; fairness includes a language-group dimension.
- **Backward compatibility** — Governance is additive; when ACE is off (default), nothing changes. Enforcement only tightens, never loosens, existing behavior.

## 7. Acceptance Criteria

- **AC-1.** *Given* a variant with a blocking a11y flag or fidelity below threshold, *When* any student would be served, *Then* the server serves base and records a `gate_block` — even if the variant row says approved.
- **AC-2.** *Given* a COPPA-gated minor without consent, *When* they open an adaptive unit, *Then* no profile/variant is generated and base is served, transparently.
- **AC-3.** *Given* the fairness audit runs, *When* one language group has materially lower mean fidelity or lift (adequate n), *Then* a disparity flag is raised to admins and the group cells respect small-cell suppression.
- **AC-4.** *Given* a DSAR export for a student, *When* generated, *Then* it includes their adaptation profiles, served variants, servings, outcomes, opt-outs, and events; *and* a deletion request removes them.
- **AC-5.** *Given* a student clicks "this adaptation seems wrong", *When* submitted, *Then* a contest record is created, the instructor is notified for human review, and the student can immediately switch to the original.
- **AC-6.** *Given* an admin quarantines a unit during an incident, *When* done, *Then* serving stops instantly (base only) and the action is audited.
- **AC-7.** *Given* GA readiness review, *When* checked, *Then* a completed DPIA (S06) and EU AI Act high-risk checklist (S13) exist and reference the current prompt/model versions.

## 8. Data Model

Reserves `446_adaptive_content_governance.sql`.

```sql
-- 446_adaptive_content_governance.sql
CREATE TABLE course.adaptive_content_contests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    serving_id UUID REFERENCES course.adaptation_servings (id) ON DELETE SET NULL,
    student_user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    reason TEXT,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','reviewed','resolved','dismissed')),
    resolved_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX idx_ac_contests_unit ON course.adaptive_content_contests (unit_id, status);

-- Fairness audit results (aggregate, suppressed; per course × dimension × group).
CREATE TABLE analytics.adaptive_content_fairness (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    dimension TEXT NOT NULL,               -- 'language' | 'grade_band' | 'section' | 'accommodation'
    group_label TEXT NOT NULL,
    n INTEGER NOT NULL,
    mean_fidelity REAL,
    coverage_pct REAL,
    mean_lift REAL,
    disparity_flag BOOLEAN NOT NULL DEFAULT FALSE,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ac_fairness_course ON analytics.adaptive_content_fairness (course_id, dimension);

-- Org-level ACE governance toggle (admin), plus incident quarantine flags.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS adaptive_content_org_enabled BOOLEAN;   -- admin visibility/disable, NOT a required on-switch
ALTER TABLE course.adaptive_content_units
    ADD COLUMN IF NOT EXISTS quarantined BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS quarantined_reason TEXT;
```

**Backfill:** none. Fairness rows populate on first audit run.

## 9. API Surface

```
POST /api/v1/courses/{course_code}/adaptive-content/units/{id}/contest   student   ({ servingId?, reason? })
GET  /api/v1/courses/{course_code}/adaptive-content/contests             instructor
POST /api/v1/courses/{course_code}/adaptive-content/contests/{id}/resolve instructor
GET  /api/v1/admin/adaptive-content/oversight                            admin  (activity, disparity flags, cost, incidents)
GET  /api/v1/admin/adaptive-content/fairness?course=…                    admin/DPO
POST /api/v1/admin/adaptive-content/quarantine                           admin  ({ unitId|courseId, reason })
POST /api/v1/admin/adaptive-content/kill-switch                          admin  (engage/disengage; == AC.1 kill-switch)
-- DSAR integration: ACE artifacts added to the existing export/delete pipelines (S01/S02), not a new public route.
```

## 10. UI / UX

- **Student:** an unobtrusive "Report this adaptation" link near the AC.6 banner → short form → confirmation + one-tap "show original". No friction to opt out.
- **Guardian (parent portal):** an "Adaptive content" panel showing that it's on for the child's course, the opt-out control, and a plain-language explainer.
- **Instructor:** a "Contests" inbox in the AC.5 workspace; resolving links to the variant review.
- **Admin oversight console** (extends the AI governance panel): ACE activity summary, disparity flags (with suppression), regressing units (AC.7), cost (AC.4), incident controls (quarantine, kill-switch), links to the DPIA/AI-Act docs.
- **States/accessibility:** all governance controls WCAG 2.1 AA; disparity flags icon+text; console tables have exports; destructive incident actions confirm.
- **Mobile:** student report link + guardian panel fully responsive.

## 11. AI / ML Considerations

- **This story governs the AI rather than adding a model surface.** It sets and enforces the thresholds AC.3 computes against (min fidelity, safety, a11y), owns the **prompt/model change-management** process (bump `prompt_version`, re-run DPIA delta, re-eval), and defines the **fairness metrics** (coverage, fidelity, lift by group) that treat the generative system as a monitored, high-risk component (NIST AI RMF: govern/measure/manage).
- **Human-oversight modes.** The platform default is **auto-serve after the gates pass** — the automated fidelity/safety/a11y gates are the primary control, and oversight is discharged by (a) up-front instructor guardrails/config (AC.5), (b) *post-hoc* review, revoke, and the contest path, and (c) continuous fairness/regression monitoring. **AC.8 MUST force `require_instructor_approval=true` (pre-serve human sign-off) for EU AI Act high-risk deployments and for COPPA-gated minors**, and MUST allow a jurisdiction/org policy to require it elsewhere — this is how the auto-serve default remains compatible with EU AI Act Art. 14 and GDPR Art. 22 "meaningful human involvement" where the law demands intervention *before* an automated educational decision takes effect.
- **Bias pathways audited:** uneven fidelity for non-dominant languages/reading levels; uneven coverage (who gets a variant at all); uneven lift; misconception-library gaps for some cohorts. Findings feed back to AC.3 prompt/eval work.

## 12. Integration Points

- `server/internal/aidisclosure/`, `service/aigateway/`, the AI governance panel (`clients/web/src/components/settings/ai-governance-panel.tsx`) — extend for ACE.
- Platform compliance engines: S01 DSAR export/delete, S02 retention, S05 RoPA, S03 breach, S21 evidence — register ACE artifacts.
- `service/adaptivecontent/governance.go`, `fairness.go` (new); serve-time gate re-check in `serve.go` (AC.6).
- COPPA service (`service/coppa`) + age gating already in `aigateway`.
- notifications — contest + disparity + incident alerts.
- `server/migrations/446_adaptive_content_governance.sql` (+ down).
- Docs: `docs/plan/standards/S06`, `S08`, `S13`, `S01`, `S02`, `S05`, `S20`, `S21`; new DPIA + AI-Act artifacts under `docs/`.

## 13. Dependencies & Sequencing

- **Must ship after:** AC.3 (defines what's generated); integrates AC.2–AC.7 artifacts.
- **Must ship before:** **GA of ACE** — this story gates general availability; pilots may run under a documented interim risk acceptance with the core gates (fidelity/safety/disclosure/opt-out/minors) already enforced.
- **Shared infra:** compliance engines (S01/S02/S03/S05/S21), aigateway, notifications, admin console.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Hallucinated content reaches a learner despite gates | L | **H** | Server-side serve-time re-check; human approval default; incident quarantine + kill-switch; breach process if applicable |
| Systematic disadvantage to a group | M | **H** | Fairness audit + disparity alerts; per-language fidelity targets; misconception-library coverage review; ability to disable adaptation for affected cohorts |
| Minor profiled/served AI without consent | L | **H** | Fail-closed COPPA gate; conservative defaults; guardian controls; AADC alignment |
| Regulator finds no DPIA/human-oversight evidence | M | H | DPIA (S06) + AI-Act checklist (S13) as GA gate; immutable oversight log (S21) |
| Opt-out/contest ignored operationally | L | M | Server-enforced opt-out; contest SLA + instructor inbox; audited |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1) remains the course-level gate. The org-level `adaptive_content_org_enabled` is an **admin visibility/disable** control for governance — **not** a required global on-switch (an org that leaves it null does not block a course from enabling ACE; setting it false is an affirmative org-wide disable). This preserves the "course-level, not platform-level" mandate while giving admins a lawful override.
- **Sequencing:** enforce core gates from the first pilot (fidelity/safety/disclosure/opt-out/minors) → add fairness audit + oversight console → complete DPIA + AI-Act checklist → GA.
- **Pilot cohort:** same as AC.6, under interim risk acceptance signed by the DPO.
- **GA criteria:** all gates enforced server-side; DSAR/retention cover ACE artifacts; fairness audit live with alerting; DPIA + AI-Act checklist complete and current; incident runbook tested.
- **Rollback:** kill-switch (halt generation+serving), quarantine (unit/course), org disable, course flag off — layered, each immediate.

## 16. Test Plan

- **Unit** — serve-time gate re-check truth table; minor gating; suppression logic; disparity threshold; contest state machine.
- **Integration** — DSAR export/delete includes+removes ACE artifacts; opt-out honored end-to-end; quarantine stops serving; org-disable vs. course-flag interaction.
- **Security** — oversight/fairness admin-gated; demographic attrs never in prompts (assertion); audit log append-only.
- **Fairness/eval** — synthetic multi-group cohorts → correct disparity flags + suppression; per-language fidelity regression fixtures.
- **Accessibility** — contest/opt-out/guardian panels WCAG 2.1 AA (axe + SR); blocking a11y flag prevents serve.
- **Compliance** — DPIA + AI-Act checklist artifacts exist and are linked; RoPA entry present; retention job reaches ACE tables.
- **Manual exploratory** — incident drill: inject a bad variant, quarantine, verify base fallback + audit + (if PII) breach process trigger.

## 17. Documentation & Training

- DPIA / algorithmic-impact assessment for ACE (S06) and EU AI Act high-risk conformity checklist (S13) — living docs.
- Student/guardian transparency notices; opt-out + contest help.
- Admin governance runbook: reading fairness/oversight, quarantine, kill-switch, incident + breach handoff.
- Trust center update: "How Lextures governs adaptive content."

## 18. Open Questions

1. Which demographic/proxy attributes are lawful to use for the fairness audit per jurisdiction, and which require aggregation-only? (Legal to define per market; default to language/section/grade-band + accommodation flag.)
2. Do we need per-jurisdiction adaptation limits (e.g., stricter minor rules in EU/UK)? (Likely; coordinate with S08/S13/S18.)
3. Should contests auto-pause a variant after N reports pending review? (Lean yes with a threshold.)
4. What is the SLA for resolving a contest and for acting on a disparity flag? (Define with trust & safety.)

## 19. References

- Existing files: `server/internal/aidisclosure/`, `service/aigateway/service.go`, `service/coppa/`, `clients/web/src/components/settings/ai-governance-panel.tsx`, `server/migrations/281_ai_usage_logs.sql`.
- Related plans: [AC.3](../../completed/adaptive/AC.3-content-generation-engine.md), [AC.6](../../completed/adaptive/AC.6-student-runtime-and-transparency.md), [AC.7](../../completed/adaptive/AC.7-post-assessment-and-effectiveness.md); Standards `../standards/S06-dpia-pia-algorithmic-impact.md`, `S08-childrens-privacy-age-assurance-design-codes.md`, `S13-eu-ai-act-high-risk.md`, `S01-unified-data-subject-rights-orchestration.md`, `S02-data-retention-deletion-engine.md`, `S20-accessibility-legal-mandates.md`, `S21-compliance-evidence-continuous-monitoring.md`.
- External: EU AI Act (Annex III, Arts. 9/13/14/52); GDPR Art. 22; FERPA; COPPA 16 CFR Part 312; NIST AI RMF; UK AADC.

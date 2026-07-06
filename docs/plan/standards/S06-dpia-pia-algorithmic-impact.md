# S06 — DPIA / PIA & Algorithmic Impact Assessment Automation

> Implementation plan. Hardens: [10.3 GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md) (Art 35), [10.17 AI disclosure](../../completed/10-compliance-privacy-security/10.17-ai-usage-disclosure.md), [10.10 ISO 27701](../../completed/10-compliance-privacy-security/10.10-iso-27001-27701.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S06 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | EU/UK · US · Global |
| **Status (today)** | MISSING — no structured DPIA/PIA process; AI features (adaptive paths, AI grading, proctoring, at-risk scoring) ship without documented risk assessments that GDPR Art 35, several US state laws, and the EU AI Act require |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | DPO + AI/ML Lead |
| **Depends on** | 10.3, 10.17, S05 (data inventory), S04 (purposes) |
| **Unblocks** | S13 (AI Act conformity reuses AIA), S08 (kids' DPIA), S11 (state AIA duties) |

---

## 1. Problem Statement

High-risk processing must be risk-assessed *before* it ships. GDPR Art 35 requires a DPIA for large-scale/systematic/vulnerable-subject processing — which describes a school platform's analytics, proctoring, and AI grading exactly. A growing set of US state laws (e.g. Colorado, Connecticut, Virginia) require data-protection assessments for profiling/targeted-processing, and several add **algorithmic impact assessment** duties for automated decisions affecting people; the EU AI Act layers a fundamental-rights impact assessment on high-risk AI. Lextures has shipped multiple AI/profiling features with **no documented assessment**, meaning we cannot demonstrate we weighed the risks — an automatic finding and, for the AI Act, a market-access blocker. We need a repeatable, evidence-producing DPIA/PIA/AIA workflow wired to our actual features.

## 2. Goals

- A **DPIA/PIA workflow**: screening threshold → full assessment → mitigations → residual-risk sign-off → periodic review.
- An **algorithmic impact assessment (AIA)** variant for automated decisions/profiling (accuracy, bias, human oversight, contestability, explainability).
- **Screening automation**: new features/purposes (from S04/S05) that cross risk thresholds auto-raise a required assessment.
- A **register** of assessments with review dates and links to the features/purposes they cover.
- Reusable outputs that feed the EU AI Act technical file (S13) and state AIA filings (S11).

## 3. Non-Goals

- The AI feature implementations themselves; this assesses them.
- Model evaluation tooling (accuracy/bias metrics) — this consumes their results as evidence, it doesn't build the eval harness.
- Consultation with a supervisory authority (a manual legal step when residual risk stays high).

## 4. Personas & User Stories

- **As a DPO**, I want any high-risk processing to require a completed DPIA before launch so that we meet Art 35.
- **As a product manager**, I want a screening questionnaire that tells me whether my feature needs a DPIA/AIA so that I don't guess.
- **As an ML lead**, I want to attach bias/accuracy evidence and a human-oversight design to an AIA so that automated decisions are defensible and contestable.
- **As a compliance officer**, I want a register with review dates so that assessments don't go stale as features change.
- **As a regulator/auditor**, I want to see the DPIA behind a profiling feature so that accountability is demonstrable.

## 5. Functional Requirements

- **FR-1.** The system MUST provide a **screening** questionnaire; a positive result creates a required assessment linked to the feature/purpose (S04/S05).
- **FR-2.** A **DPIA** MUST capture: description, necessity/proportionality, data flows (from S05), risks to subjects, mitigations, residual risk, and DPO sign-off.
- **FR-3.** An **AIA** MUST additionally capture: decision logic summary, training-data provenance, accuracy + **bias/fairness** metrics, human-oversight design, contestability/appeal path, and explainability approach.
- **FR-4.** The system MUST **block launch** (via a release gate) of a feature flagged high-risk until its assessment is signed off (integrates with the feature-flag/release process).
- **FR-5.** The system MUST maintain a **review schedule**; material change to a feature/model re-opens its assessment.
- **FR-6.** Assessments MUST be exportable to feed the AI Act technical documentation (S13) and state assessment filings (S11).
- **FR-7.** When residual risk remains high, the system MUST flag that **prior consultation** with a supervisory authority is required and record its outcome.
- **FR-8.** All assessment activity MUST be audit-logged (10.11).

## 6. Non-Functional Requirements

- **Performance** — Assessment CRUD is low-volume; no hot-path concerns.
- **Security** — Assessments may reference sensitive design details; access gated by `privacy:dpia_author` / `privacy:dpo`.
- **Privacy & Compliance** — GDPR Art 35–36; EU AI Act FRIA (Art 27) inputs; CO/CT/VA/etc. data-protection-assessment duties; ISO 27701.
- **Accessibility** — Authoring UI WCAG 2.1 AA.
- **Scalability** — Dozens–hundreds of assessments; linkable to many features.
- **Reliability** — Release gate is fail-closed for high-risk features (no assessment → no launch).
- **Observability** — `dpia_overdue_review_total`, `high_risk_features_without_signoff`; alert on either > 0.
- **Maintainability** — Service `server/internal/service/dpia/`; templates/questionnaires are versioned data.
- **Internationalization** — N/A for internal authoring; exports may be translated for regulators.
- **Backward compatibility** — Retroactively creates assessments for already-shipped AI/profiling features (backlog list seeded from `server/internal/service` AI modules).

## 7. Acceptance Criteria

- **AC-1.** *Given* a new feature that profiles students (e.g. at-risk scoring), *when* screening is completed, *then* a required DPIA+AIA is created and the release gate blocks launch until sign-off.
- **AC-2.** *Given* an AIA for AI grading, *when* it is authored, *then* accuracy + bias metrics, human-oversight design, and a contestability path are mandatory fields that must be non-empty to sign off.
- **AC-3.** *Given* a model is retrained materially, *when* the change is recorded, *then* the linked AIA re-opens for review and appears in the overdue register if not revisited within the review window.
- **AC-4.** *Given* a completed DPIA with high residual risk, *when* it is finalised, *then* it flags "prior consultation required" and cannot be marked "cleared" until the consultation outcome is recorded.
- **AC-5.** *Given* an already-shipped AI feature with no assessment, *when* the backlog seeder runs, *then* it appears in `high_risk_features_without_signoff` until assessed.
- **AC-6.** *Given* S13 builds the AI Act technical file, *when* it imports assessments, *then* the AIA content populates the fundamental-rights and risk-management sections.

## 8. Data Model

New migration `362_dpia_assessments.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.dpia_assessments (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind          TEXT NOT NULL CHECK (kind IN ('dpia','pia','aia')),
  title         TEXT NOT NULL,
  feature_key   TEXT NOT NULL,                 -- links to feature flag / module
  purpose_keys  TEXT[],                        -- S04 purposes covered
  status        TEXT NOT NULL DEFAULT 'draft'
                  CHECK (status IN ('screening','required','draft','in_review','signed_off','cleared','reopened')),
  necessity     TEXT,
  risks         JSONB,                          -- [{risk, likelihood, severity, mitigation, residual}]
  residual_risk TEXT CHECK (residual_risk IN ('low','medium','high')),
  consultation_required BOOLEAN NOT NULL DEFAULT FALSE,
  consultation_outcome TEXT,
  -- AIA-specific
  decision_logic TEXT,
  accuracy_metrics JSONB,
  bias_metrics   JSONB,
  human_oversight TEXT,
  contestability TEXT,
  explainability TEXT,
  author_id     UUID REFERENCES "user".users(id),
  signed_off_by UUID REFERENCES "user".users(id),
  signed_off_at TIMESTAMPTZ,
  next_review_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dpia_overdue ON compliance.dpia_assessments(next_review_at)
  WHERE status IN ('signed_off','cleared');
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/compliance/dpia/screen` | `privacy:dpia_author` | Screening → creates required assessment if triggered |
| `GET/POST/PATCH` | `/api/v1/compliance/dpia` | `privacy:dpia_author` | Author assessments |
| `POST` | `/api/v1/compliance/dpia/{id}/signoff` | `privacy:dpo` | Sign off / clear |
| `GET` | `/api/v1/compliance/dpia/register` | `privacy:dpo` | Register + overdue list |
| `GET` | `/api/v1/compliance/dpia/{id}/export` | `privacy:dpo` | Export (feeds S13/S11) |
| internal | release-gate check | — | Blocks high-risk feature launch without sign-off |

## 10. UI / UX

- **Screening wizard:** short questionnaire → verdict (needs DPIA/AIA or not) with rationale.
- **Assessment editor:** sectioned form (DPIA base + AIA extension), risk table builder, evidence attachments, sign-off flow.
- **Register dashboard:** all assessments, statuses, next-review dates, overdue highlights, unassessed high-risk features.
- States: empty, draft autosave, blocked-launch banner, overdue-review red flag.
- Accessibility: form landmarks, error summaries, keyboard risk-table entry; i18n keys `dpia.*` (internal, en-first).

## 11. AI / ML Considerations

This plan is the connective tissue for responsible AI: it consumes bias/accuracy evidence from the model-eval pipeline, documents human-oversight and contestability for every automated decision (adaptive paths, AI grading, at-risk scoring, proctoring), and its AIAs are the primary input to the EU AI Act technical file (S13) and to AI-transparency disclosures (10.17). PII redaction (10.14) applies to any examples embedded in an assessment.

## 12. Integration Points

- `server/internal/service/dpia/` (new); release-gate hook in the feature-flag system; model-eval outputs from AI modules (`adaptivepath`, `assignmentrubricai`, `atriskscoring`, `quizlockdown`/proctoring).
- S04 (purposes), S05 (data flows), S13 (technical file), S11 (state filings), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: S05, S04.
- Must ship before: S13 (AI Act reuses AIAs), and before launching any *new* high-risk AI feature.
- Shared infra: feature-flag/release pipeline, object storage for evidence.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Assessments become box-ticking, not real analysis | M | H | DPO sign-off with mandatory mitigation/residual fields; periodic quality review |
| Release gate blocks shipping and gets bypassed | M | H | Gate is code-enforced on high-risk flags; bypass requires DPO override logged |
| Backlog of unassessed shipped features never cleared | M | M | Seed register + track `high_risk_features_without_signoff` to zero with deadlines |
| Bias metrics unavailable for some models | M | M | Require at least a documented fairness approach + plan when metrics are pending |

## 15. Rollout Plan

- Flag `dpia_workflow_enabled`. Phase 1: assessment model + editor + register. Phase 2: screening + release gate (warn). Phase 3: gate enforce for high-risk flags; seed backlog. GA when backlog assessed. Rollback: gate to warn-only (process reverts to manual).

## 16. Test Plan

- **Unit** — screening threshold logic; AIA mandatory-field validation; overdue computation.
- **Integration** — release gate blocks a high-risk flag without sign-off; retrain reopens AIA; export shape matches S13 importer.
- **E2E** — screen → author DPIA+AIA → sign off → feature launches; skip sign-off → launch blocked.
- **Security** — authz on authoring/sign-off; override audit.
- **Accessibility** — axe on editor; keyboard risk-table.
- **Performance** — register renders with hundreds of assessments.
- **Manual** — DPO reviews a real feature's AIA end-to-end.

## 17. Documentation & Training

- Guide: "Do I need a DPIA/AIA?" with the screening criteria.
- Template library (DPIA, PIA, AIA) with worked examples for grading/proctoring/at-risk.
- Runbook: prior-consultation process when residual risk is high.

## 18. Open Questions

1. Exact screening thresholds per jurisdiction (Art 35 lists vs. state AIA triggers) — harmonise to the strictest?
2. Where do bias metrics come from for third-party (OpenRouter/provider) models we don't train?
3. Does the release gate live in CI, the feature-flag admin, or both?
4. Cadence for periodic review (annual vs. risk-tiered)?

## 19. References

- `server/internal/service/{adaptivepath,assignmentrubricai,atriskscoring,quizlockdown,aigateway}`
- GDPR Arts 35–36; EU AI Act Art 27 (FRIA); CO Privacy Act, CT DPA, VA CDPA assessment duties; ISO 27701
- Related: [S13](S13-eu-ai-act-high-risk.md), [S11](S11-us-state-privacy-expansion.md), [10.17](../../completed/10-compliance-privacy-security/10.17-ai-usage-disclosure.md)

# S13 — EU AI Act: Education as a High-Risk AI System

> Implementation plan. Hardens: [10.17 AI usage disclosure](../../completed/10-compliance-privacy-security/10.17-ai-usage-disclosure.md); builds on [S06 DPIA/AIA](S06-dpia-pia-algorithmic-impact.md), [S12 GDPR Art 22](S12-gdpr-uk-swiss-accountability-hardening.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S13 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER (EU) |
| **Markets** | EU (all), spillover to global buyers referencing it |
| **Status (today)** | MISSING — AI is pervasive (adaptive paths, AI grading/rubrics, quiz generation, proctoring/lockdown, at-risk scoring, tutor/study-buddy) with disclosure (10.17) but **no** AI-Act risk-management system, technical documentation, logging, human-oversight, or transparency-to-deployer artifacts |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | AI/ML Lead + DPO + Legal |
| **Depends on** | 10.17, S06 (AIA), S12 (Art 22), S05 (data), S04 (consent) |
| **Unblocks** | EU market access for AI features; global RFP "AI governance" answers |

---

## 1. Problem Statement

Under Regulation (EU) 2024/1689 (the **AI Act**), **Annex III** classifies AI systems used in **education and vocational training** as **high-risk** when they (a) determine access/admission, (b) evaluate learning outcomes (incl. steering the learning process), (c) assess the appropriate level of education, or (d) monitor and detect prohibited behaviour during tests (proctoring). Lextures does **all four**: adaptive paths steer learning, AI grading evaluates outcomes, placement/diagnostics assess level, and quiz-lockdown/proctoring monitors test behaviour. High-risk systems carry a heavy provider obligation set — risk-management system, data governance, technical documentation, record-keeping/logging, transparency & instructions to deployers, human oversight, accuracy/robustness/cybersecurity, quality-management system, conformity assessment + CE marking, and registration — phasing in through **2026–2027**. Additionally, **certain uses are outright prohibited** (e.g. emotion recognition in educational settings, some biometric categorisation), and **GPAI/transparency** duties apply to generative features. Shipping AI into the EU without this is not a fine risk — it's a **market-access ban**.

## 2. Goals

- **Classify** every AI feature against the AI Act (prohibited / high-risk / limited-risk / minimal) and **remove or gate** any prohibited use (notably **emotion recognition** in education).
- Stand up a **risk-management system** and **technical documentation** (Annex IV) per high-risk system, reusing S06 AIAs.
- Implement **record-keeping/logging** (Art 12) for high-risk AI: automatic, tamper-evident event logs sufficient for traceability.
- Implement **human oversight** (Art 14) and **transparency/instructions for deployers** (Art 13) plus **user-facing AI disclosure** (Art 50) for generative/interactive features.
- Establish the **quality-management system**, **conformity-assessment** readiness, EU-database **registration**, and **post-market monitoring** + serious-incident reporting.

## 3. Non-Goals

- Building the AI features themselves (they exist) — this governs them.
- General GDPR accountability (S12) and AIAs (S06) — reused, not rebuilt.
- Non-EU AI regulation nuances (referenced in jurisdiction plans) — this is the AI Act specifically.

## 4. Personas & User Stories

- **As a compliance officer**, I want each AI feature classified and prohibited uses removed so that we can lawfully sell AI in the EU.
- **As an EU institutional deployer**, I want instructions-for-use and human-oversight guidance so that our own deployer obligations are met.
- **As a teacher (human overseer)**, I want to review, override, and understand AI grades/flags so that meaningful human oversight is real.
- **As an ML lead**, I want a technical-documentation file and event logs per high-risk system so that a conformity assessment can pass.
- **As a student**, I want to know when I'm interacting with AI and that a human can review consequential decisions so that I'm treated fairly.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain an **AI system register** classifying each feature (prohibited / high-risk / limited / minimal) with its Annex III basis, and MUST **disable prohibited uses** (emotion recognition in education; social scoring; untargeted scraping) — verified by policy + code gate.
- **FR-2.** Each high-risk system MUST have a **risk-management system** (continuous, iterative) and **technical documentation** per Annex IV, generated from and linked to its S06 AIA.
- **FR-3.** High-risk systems MUST produce **automatic, tamper-evident logs** (Art 12): inputs, model/version, output, confidence, human-override events, timestamps — retained for the statutory period and queryable.
- **FR-4.** The system MUST implement **human oversight** (Art 14): a human can understand, monitor, override, and disregard the AI output for grading, placement, at-risk, and proctoring decisions; oversight is enforced, not optional.
- **FR-5.** The system MUST provide **instructions for use / transparency to deployers** (Art 13): intended purpose, accuracy levels, known limitations, oversight measures.
- **FR-6.** Generative/interactive features MUST meet **Art 50 transparency**: users are told they're interacting with AI, and AI-generated content is marked/detectable where required (ties to 10.17).
- **FR-7.** The system MUST support **accuracy, robustness, and cybersecurity** evidence (Art 15): documented metrics, adversarial/robustness testing, and security controls per high-risk system.
- **FR-8.** The system MUST establish **post-market monitoring** and **serious-incident reporting** (Arts 72–73) wired to the incident engine (S03), plus **EU-database registration** metadata and a **conformity-assessment** checklist toward CE marking.
- **FR-9.** A **data-governance** record (Art 10) MUST document training/validation/test data provenance, representativeness, and bias examination for each high-risk system.

## 6. Non-Functional Requirements

- **Performance** — High-risk logging must not materially slow inference (async, buffered); target < 10 ms overhead.
- **Security** — Logs are tamper-evident (hash-chained) and access-controlled; technical docs contain sensitive IP (gated `ai:governance`).
- **Privacy & Compliance** — Regulation (EU) 2024/1689 Arts 5, 8–15, 16–21, 26, 43, 49, 50, 61, 72–73, Annex III(3), Annex IV; interplay with GDPR (S12) and DPIA/AIA (S06).
- **Accessibility** — Human-oversight and disclosure UIs WCAG 2.1 AA; AI disclosures in plain language.
- **Scalability** — Logging volume for all AI calls across all EU tenants; partitioned, retained per statute.
- **Reliability** — Oversight override path always available; if logging fails, high-risk inference degrades safely (fail-safe policy defined per feature).
- **Observability** — `ai_highrisk_inferences_total{system}`, `ai_human_overrides_total{system}`, `ai_prohibited_use_blocked_total`, `ai_logging_failures_total`; alert on any prohibited-use hit or logging failure.
- **Maintainability** — Governance layer in `server/internal/service/aigovernance/`; per-system config drives obligations; reuses `aigateway`/`aidisclosure`.
- **Internationalization** — Disclosures + deployer instructions in EU languages.
- **Backward compatibility** — Existing AI features keep working; high-risk obligations gate EU availability per feature via flags.

## 7. Acceptance Criteria

- **AC-1.** *Given* the AI register, *when* it's reviewed, *then* every AI feature is classified with its Annex III basis, and any emotion-recognition capability in an educational context is disabled and code-gated (`ai_prohibited_use_blocked_total` proves the gate).
- **AC-2.** *Given* an AI grading decision in the EU, *when* it runs, *then* a tamper-evident Art 12 log entry records input/model/output/confidence, and a teacher can view, override, and record a reason (Art 14).
- **AC-3.** *Given* a high-risk system, *when* the technical-documentation file is generated, *then* it contains Annex IV elements (purpose, data governance, accuracy, oversight, risk management) sourced from the AIA (S06).
- **AC-4.** *Given* a student uses the AI tutor, *when* the session starts, *then* an Art 50 disclosure states they're interacting with AI, and generated content is marked.
- **AC-5.** *Given* a serious AI malfunction affecting outcomes, *when* it's detected, *then* an S03 incident opens with AI-Act serious-incident-reporting obligations and deadlines.
- **AC-6.** *Given* an EU deployer onboards, *when* they access instructions-for-use, *then* purpose, accuracy, limitations, and required oversight measures are provided per high-risk system.

## 8. Data Model

New migration `369_ai_act_governance.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.ai_systems (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key           TEXT NOT NULL UNIQUE,          -- 'adaptive_path','ai_grading','placement','proctoring','ai_tutor'
  name          TEXT NOT NULL,
  risk_class    TEXT NOT NULL CHECK (risk_class IN ('prohibited','high','limited','minimal')),
  annex_iii_basis TEXT,                         -- '3a_admission','3b_evaluation','3c_level','3d_proctoring'
  aia_assessment_id UUID,                        -- S06
  data_governance JSONB,                         -- Art 10 provenance/representativeness/bias
  accuracy_metrics JSONB,                        -- Art 15
  oversight_design TEXT,                          -- Art 14
  instructions_for_use TEXT,                      -- Art 13
  eu_db_registration_id TEXT,                     -- Art 49 registration
  conformity_status TEXT NOT NULL DEFAULT 'pending'
                  CHECK (conformity_status IN ('pending','assessed','ce_marked','withdrawn')),
  eu_available  BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS compliance.ai_event_logs (
  id            BIGSERIAL PRIMARY KEY,
  system_key    TEXT NOT NULL REFERENCES compliance.ai_systems(key),
  subject_id    UUID,
  model_ref     TEXT NOT NULL,
  input_hash    TEXT NOT NULL,                   -- hashed/redacted (10.14)
  output_summary JSONB,
  confidence    NUMERIC,
  human_override BOOLEAN NOT NULL DEFAULT FALSE,
  override_reason TEXT,
  prev_hash     TEXT,                            -- tamper-evidence chain
  at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_event_logs_system ON compliance.ai_event_logs(system_key, at DESC);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/ai-act/systems` | `ai:governance` | Manage AI register + classifications |
| `GET` | `/api/v1/compliance/ai-act/systems/{key}/tech-doc` | `ai:governance` | Generate Annex IV technical file |
| `GET` | `/api/v1/compliance/ai-act/systems/{key}/logs` | `ai:governance` | Art 12 event logs |
| `POST` | `/api/v1/ai/decisions/{id}/override` | teacher / overseer | Human oversight override (Art 14) |
| `GET` | `/api/v1/public/ai-act/instructions/{key}` | deployer | Instructions for use (Art 13) |
| internal | prohibited-use gate + logging middleware | — | Blocks prohibited uses; logs high-risk inference |

## 10. UI / UX

- **AI governance console:** register with risk class + conformity status + EU availability toggle; technical-doc generator; log viewer.
- **Human-oversight surfaces:** in grading/proctoring/at-risk views, clear AI-suggestion labelling with override + reason capture (extends existing grading UIs).
- **Deployer instructions page** (public/tenant): purpose, accuracy, limitations, oversight.
- **Student AI disclosure** at point of interaction (tutor/study-buddy/quiz-gen) with content marking.
- States: EU-unavailable banner for a non-conformant high-risk feature; prohibited-use hard-off; override recorded.
- Accessibility: plain-language disclosures, localised; i18n keys `aiact.*`.

## 11. AI / ML Considerations

This is the core AI story. It touches every AI module: `adaptivepath`, `assignmentrubricai`/`grading`/`gradingagent`, `quizgenerationai`, `quizlockdown` (proctoring), `atriskscoring`, `aitutor`/`studybuddy`, routed via `aigateway`/`openrouter`. Key stances: **disable emotion recognition** in educational contexts (prohibited); ensure **meaningful human oversight** on all outcome-affecting decisions (no fully-automated final grades without human confirmation in the EU); mark generative content; keep tamper-evident logs; carry accuracy/robustness evidence from the model-eval pipeline into each system's record. GPAI transparency duties apply to the underlying foundation models via provider terms (S07).

## 12. Integration Points

- `server/internal/service/aigovernance/` (new); middleware in `aigateway`; disclosure via `aidisclosure` (10.17); logging redaction via `logredaction` (10.14).
- S06 (AIA → technical file), S12 (Art 22 overlap), S05 (data governance), S03 (serious-incident reporting), S07 (GPAI provider terms), all AI feature modules.

## 13. Dependencies & Sequencing

- Must ship after: S06, S12, 10.17.
- Must ship before: any EU launch/continuation of high-risk AI features past the applicable AI-Act deadlines.
- Shared infra: model-eval pipeline, incident engine, object storage for technical files.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A prohibited use (emotion recognition) is live | M | H (ban/fines) | Audit all AI features; hard code-gate + zero-tolerance metric |
| Fully-automated grading violates human-oversight | M | H | Enforce human confirmation on EU outcome decisions; override path mandatory |
| Technical documentation incomplete at conformity assessment | M | H | Generate Annex IV from AIA; gap checklist; legal review |
| Logging overhead/failure degrades UX | M | M | Async buffered logging; fail-safe policy; failure alerting |
| Deadlines misjudged per phase-in | M | H | Track per-obligation effective dates; conservative EU-availability gating |

## 15. Rollout Plan

- Flag `ai_act_governance_enabled` (EU tenants). Phase 1: AI register + classification + **prohibited-use removal** + Art 50 disclosures. Phase 2: Art 12 logging + Art 14 human oversight enforcement + Annex IV tech docs. Phase 3: conformity-assessment prep, EU-database registration, post-market monitoring + S03 serious-incident wiring. Gate EU availability per feature to its readiness. GA per feature as it reaches conformity. Rollback: EU-availability off for a non-conformant feature (fail-safe = withhold in EU).

## 16. Test Plan

- **Unit** — classification/register logic; prohibited-use gate; log hash-chain integrity; oversight-override recording.
- **Integration** — high-risk inference emits Art 12 log; EU grading requires human confirmation; serious incident opens S03 case.
- **E2E** — teacher overrides an AI grade with reason; student sees AI disclosure; deployer views instructions.
- **Security** — tamper-evidence of logs; authz on governance console; prohibited-use cannot be re-enabled.
- **Accessibility** — axe + localised disclosures.
- **Performance** — logging overhead < 10 ms; high-volume log ingestion.
- **Manual** — mock conformity-assessment walkthrough against Annex IV.

## 17. Documentation & Training

- Annex IV technical-documentation template per high-risk system.
- Deployer instructions-for-use library (Art 13).
- Human-overseer training: how to review/override AI grading, proctoring, at-risk.
- Runbook: serious-incident reporting timelines (Arts 72–73).

## 18. Open Questions

1. Are we the "provider," "deployer," or both for each feature, and does that change per tenant configuration?
2. Which features can remain **fully automated** vs. require human-in-the-loop in the EU (final grade = human-confirm; recommendations = automated)?
3. Retention period + storage tier for Art 12 logs at full EU volume?
4. Timing of conformity assessment + CE marking vs. the 2026/2027 phase-in for our specific Annex III uses?
5. Foundation-model GPAI obligations — reliance on provider documentation vs. our own?

## 19. References

- `server/internal/service/{aigateway,aidisclosure,adaptivepath,assignmentrubricai,gradingagent,quizgenerationai,quizlockdown,atriskscoring,aitutor,studybuddy}`
- Regulation (EU) 2024/1689 (AI Act) Arts 5, 8–15, 13–14, 26, 43, 49, 50, 61, 72–73; Annex III(3); Annex IV
- Related: [S06](S06-dpia-pia-algorithmic-impact.md), [S12](S12-gdpr-uk-swiss-accountability-hardening.md), [10.17](../../completed/10-compliance-privacy-security/10.17-ai-usage-disclosure.md)

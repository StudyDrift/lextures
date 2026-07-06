# S10 — PPRA (Protection of Pupil Rights Amendment)

> Implementation plan. New coverage (pairs with [10.1 FERPA](../../completed/10-compliance-privacy-security/10.1-ferpa-workflow.md) / [S09](S09-ferpa-hardening.md)). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S10 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER (K12) |
| **Markets** | K12 |
| **Status (today)** | MISSING — no PPRA controls at all. The platform runs surveys, SEL/wellbeing check-ins, and can surface targeted content, all of which PPRA regulates |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Compliance Lead + Product |
| **Depends on** | 10.1, S04 (consent), S08 (minors), survey/SEL features |
| **Unblocks** | S11 (state survey laws mirror PPRA), K12 procurement |

---

## 1. Problem Statement

PPRA (20 U.S.C. § 1232h) is FERPA's under-appreciated sibling and a standard item on K-12 privacy checklists. It gives parents rights over **surveys** funded by the U.S. Department of Education, requires **consent or opt-out** for surveys probing eight protected categories (political affiliation, mental/psychological problems, sex behaviour/attitudes, illegal/anti-social behaviour, critical appraisals of family, privileged relationships, religious practices, income), regulates **collection of data for marketing** to students, and requires parents be able to **inspect** instructional/survey materials and any instrument used to collect personal information for marketing. Lextures runs surveys, SEL and wellbeing check-ins, and could surface targeted/sponsored content — all squarely in PPRA's scope — with **zero** PPRA controls. Any district counsel will flag this immediately.

## 2. Goals

- Classify survey/instrument content against the **eight PPRA protected categories** and gate accordingly (consent for DoE-funded; opt-out otherwise).
- Provide parents the right to **inspect** surveys, instructional materials tied to them, and any marketing-data instruments.
- Prohibit (by default) and control the **collection of student data for marketing/sale**, with the narrow educational-product exceptions documented.
- Require districts to adopt/publish a **PPRA policy** and give **direct notification** of applicable activities with opt-out.
- Keep an auditable record of PPRA classifications, notices, consents, and opt-outs.

## 3. Non-Goals

- General survey authoring/engine features (owned by the assessment/survey area) — this adds the PPRA compliance layer over them.
- FERPA record rights (S09).
- Advertising infrastructure — PPRA marketing controls default to prohibition.

## 4. Personas & User Stories

- **As a parent**, I want to inspect a survey before my child takes it and opt them out if it probes protected categories so that my rights are honoured.
- **As a district admin**, I want to publish our PPRA policy and have the platform enforce consent/opt-out on protected surveys so that we comply.
- **As a survey author**, I want the tool to flag when my questions hit a protected category so that I attach the right consent/opt-out.
- **As a compliance officer**, I want proof of PPRA notices, consents, and opt-outs so that I can answer a complaint.

## 5. Functional Requirements

- **FR-1.** Surveys/instruments MUST be classifiable against the **eight protected categories**; authors are prompted and a reviewer confirms the classification.
- **FR-2.** A survey hitting a protected category MUST require **prior written parental consent** if DoE-funded, or offer **opt-out** otherwise, before any student can respond — enforced, not advisory.
- **FR-3.** Parents MUST be able to **inspect** the full survey/instrument and related instructional materials on request (§1232h(c)(1)(C)/(F)).
- **FR-4.** The system MUST default to **prohibiting collection/use/sale of student personal information for marketing**, with any permitted educational-product use explicitly configured and disclosed.
- **FR-5.** Districts MUST be able to store a **PPRA policy** and trigger **direct notification** to parents of the specific/approximate dates of protected activities, with opt-out capture.
- **FR-6.** The system MUST record classifications, notices, consents, opt-outs, and inspection requests for audit.
- **FR-7.** Physical exams/screenings scheduling (non-emergency) MUST support opt-out where the district uses that feature.

## 6. Non-Functional Requirements

- **Performance** — Classification checks at survey-publish time; parent inspection loads on demand.
- **Security** — Only authorised staff classify; parent inspection auth-gated to their child's cohort.
- **Privacy & Compliance** — 20 U.S.C. § 1232h; 34 CFR Part 98; aligns with FERPA (S09) and state student-survey laws (S11).
- **Accessibility** — Notices, consent/opt-out forms, and inspection views WCAG 2.1 AA + translated.
- **Scalability** — District-wide notification via queue.
- **Reliability** — A protected survey cannot open to students until its consent/opt-out gate is satisfied (fail-closed).
- **Observability** — `ppra_protected_surveys_total`, `ppra_optouts_total`, `ppra_consent_pending`; alert on a protected survey opened without a satisfied gate (should be zero).
- **Maintainability** — PPRA layer in `server/internal/service/ppra/`; category taxonomy is versioned data.
- **Internationalization** — Notices/forms localised.
- **Backward compatibility** — Existing surveys default to "unclassified → must classify before next run."

## 7. Acceptance Criteria

- **AC-1.** *Given* a survey with a question about family income, *when* the author publishes, *then* it's flagged as protected and cannot open to students until consent/opt-out is configured.
- **AC-2.** *Given* a DoE-funded protected survey, *when* a student without parental consent attempts it, *then* they are blocked and the parent consent status is shown.
- **AC-3.** *Given* a parent requests to inspect a survey, *when* they open it, *then* the full instrument and related materials are viewable and the request is logged.
- **AC-4.** *Given* default settings, *when* any feature attempts to use student data for marketing, *then* it is blocked and requires explicit district configuration + disclosure.
- **AC-5.** *Given* a district schedules a protected activity, *when* direct notification runs, *then* parents receive dates + opt-out, and opt-outs are captured and enforced.
- **AC-6.** *Given* a compliance officer exports PPRA records, *when* the report runs, *then* classifications, notices, consents, opt-outs, and inspections are all present.

## 8. Data Model

New migration `366_ppra.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.ppra_classifications (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES org.organizations(id),
  instrument_id UUID NOT NULL,                  -- survey/assessment id
  protected_categories TEXT[] NOT NULL DEFAULT '{}',  -- subset of the 8
  doe_funded    BOOLEAN NOT NULL DEFAULT FALSE,
  gate          TEXT NOT NULL DEFAULT 'none' CHECK (gate IN ('none','consent','opt_out')),
  classified_by UUID REFERENCES "user".users(id),
  classified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.ppra_parent_actions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  classification_id UUID NOT NULL REFERENCES compliance.ppra_classifications(id) ON DELETE CASCADE,
  student_id    UUID NOT NULL REFERENCES "user".users(id),
  parent_id     UUID NOT NULL REFERENCES "user".users(id),
  action        TEXT NOT NULL CHECK (action IN ('consent','opt_out','inspect')),
  at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.ppra_policies (
  org_id        UUID PRIMARY KEY REFERENCES org.organizations(id),
  policy_text   TEXT NOT NULL,
  marketing_use TEXT NOT NULL DEFAULT 'prohibited'
                  CHECK (marketing_use IN ('prohibited','educational_only')),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/compliance/ppra/classify` | `survey:author` / `records:admin` | Classify an instrument |
| `GET` | `/api/v1/compliance/ppra/inspect/{instrument_id}` | parent | Inspect survey + materials |
| `POST` | `/api/v1/compliance/ppra/parent-action` | parent | Consent / opt-out |
| `GET/PUT` | `/api/v1/compliance/ppra/policy` | `records:admin` | District PPRA policy + marketing setting |
| `POST` | `/api/v1/compliance/ppra/notify` | `records:admin` | Direct notification of protected activities |
| internal | survey-publish gate | — | Blocks protected survey without satisfied gate |

## 10. UI / UX

- **Author classification step:** category checklist during survey publish; reviewer confirmation.
- **Parent inspection + action:** view instrument, consent or opt-out, see status.
- **District PPRA policy editor + marketing toggle** (default prohibited).
- **Direct-notification composer** with dates + opt-out.
- States: unclassified-blocking-publish; protected-gate-pending; opt-out recorded.
- Accessibility: translated notices, plain language; i18n keys `ppra.*`.

## 11. AI / ML Considerations

AI-generated survey questions (via quiz/SEL AI) MUST pass PPRA classification before publish; the classifier MAY use an internal model to *suggest* protected categories, but a human confirms. Marketing-prohibition default means no minor survey data feeds ad/engagement models (aligns with S08).

## 12. Integration Points

- `server/internal/service/ppra/` (new); survey/SEL features (publish gate); S04 (consent records), S08 (minor policy), S09 (parent identity), mail (6.2), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: 10.1, S04.
- Must ship before: S11 (state survey laws reference PPRA), K12 GA.
- Shared infra: email, survey engine hook.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Protected survey slips out unclassified | M | H | Publish gate fail-closed; unclassified = blocked; zero-tolerance metric |
| SEL/wellbeing check-ins overlooked as "surveys" | H | H | Classify SEL instruments explicitly; default protected for mental-health category |
| Marketing use enabled without disclosure | L | H | Default prohibited; enabling requires policy text + disclosure |
| Author over/under-classifies | M | M | Reviewer confirmation + AI suggestion assist |

## 15. Rollout Plan

- Flag `ppra_enabled` (K12 tenants). Phase 1: classification + publish gate + policy. Phase 2: parent inspection + consent/opt-out. Phase 3: direct notification + marketing controls. Pilot one district. GA for K12. Rollback: flag off (surveys revert to pre-gate behaviour) — but recommend keeping SEL classification on.

## 16. Test Plan

- **Unit** — category classification; gate selection (consent vs opt-out vs none); marketing-default enforcement.
- **Integration** — publish gate blocks protected survey; parent action satisfies gate; SEL flagged mental-health.
- **E2E** — author classifies → parent inspects → opts out → student blocked.
- **Security** — parent inspection scoped to own child; author authz.
- **Accessibility** — axe + translation on notices/forms.
- **Performance** — district notification fan-out.
- **Manual** — counsel checklist against § 1232h and Part 98.

## 17. Documentation & Training

- District guide: adopting a PPRA policy + classifying SEL/surveys.
- Parent help: inspection + opt-out rights.
- Author guide: recognising protected categories.

## 18. Open Questions

1. Are all SEL/wellbeing check-ins treated as protected by default, or classified case-by-case?
2. How do we detect "DoE-funded" status — district attestation per survey?
3. Marketing "educational_only" exception scope — what qualifies and who approves?
4. Retention of PPRA consent/opt-out records (ties to S02 + FERPA).

## 19. References

- Survey/SEL features under `server/internal/service`, `server/internal/service/ppra/` (new)
- PPRA 20 U.S.C. § 1232h; 34 CFR Part 98
- Related: [S04](S04-unified-consent-preference-ledger.md), [S08](S08-childrens-privacy-age-assurance-design-codes.md), [S09](S09-ferpa-hardening.md), [S11](S11-us-state-privacy-expansion.md)

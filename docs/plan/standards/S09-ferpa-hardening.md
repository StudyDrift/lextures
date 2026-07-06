# S09 — FERPA Hardening (deep 34 CFR Part 99)

> Implementation plan. Hardens: [10.1 FERPA workflow](../../completed/10-compliance-privacy-security/10.1-ferpa-workflow.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S09 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | K12 · HE |
| **Status (today)** | PARTIAL — 10.1 shipped directory opt-out, LEI gating, record requests, consent, and a disclosure log (`server/internal/service/ferpa`, migration `179_ferpa`). Missing the deeper §99 provisions that a district/university counsel audits for |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Compliance Lead + Backend |
| **Depends on** | 10.1, 10.11 (audit log), S01 (rights orchestration), S02 (retention) |
| **Unblocks** | S10 (PPRA), S11 (state ed-privacy), K12/HE procurement sign-off |

---

## 1. Problem Statement

The shipped FERPA workflow (10.1) covers the headline rights, but "bullet-proof" means the whole of 34 CFR Part 99 — and the parts we skipped are exactly what an experienced district/university counsel checks: the **annual notification** of rights (§99.7), the **record of disclosures** that must be kept *and made available to the parent for inspection* (§99.32), the **redisclosure** limitation and its tracking (§99.33), the specific **exceptions** including the health-or-safety emergency (§99.36), the studies/audit-evaluation exceptions (§99.31(a)(6)/(3)) with their required written agreements, the **de-identification** standard for research/analytics (§99.31(b)), the treatment of **sole-possession** and **law-enforcement-unit** records (excluded from "education records"), and the **directory-information annual public notice with opt-out window**. Each gap is a finding that stalls a K-12 or Title-IV deal.

## 2. Goals

- Implement the **annual FERPA notification** generator and delivery tracking (§99.7), including the directory-information public notice with an opt-out window.
- Build the **§99.32 record of disclosures** as a parent-inspectable log (distinct from our internal audit log), covering to-whom, what, legitimate interest, and dates.
- Enforce and track **redisclosure** limitations (§99.33): recipients bound to purpose, redisclosure recorded on the originating institution's behalf.
- Wire the FERPA **exceptions** (§99.31) as explicit, logged authority claims — especially the **health-or-safety emergency** (§99.36) and **studies/audit** exceptions with their written-agreement requirements.
- Apply the FERPA **de-identification** standard (§99.31(b)) to any analytics/research/AI use of education records, and correctly **exclude** sole-possession and law-enforcement-unit records from education-record disclosures.

## 3. Non-Goals

- Re-implementing directory opt-out / LEI / basic record requests (10.1 owns those; this extends them).
- SIS/registrar integration (4.x) and grade passback (14.5).
- Parent-portal UI shell (13.1) — this feeds it.

## 4. Personas & User Stories

- **As a district compliance officer**, I want the platform to generate our annual FERPA notice and directory opt-out window so that we meet §99.7/§99.37 without bespoke work.
- **As a parent**, I want to inspect the record of who my child's education records were disclosed to so that I can exercise my §99.32 right.
- **As a registrar**, I want each disclosure tagged with its §99.31 exception (or consent) so that no disclosure is unauthorised.
- **As a safety officer**, I want to invoke the health-or-safety emergency exception with a recorded justification so that urgent disclosures are lawful and documented.
- **As a researcher/analyst**, I want education-record data de-identified to the FERPA standard before analysis so that no consent is required.

## 5. Functional Requirements

- **FR-1.** The system MUST generate a configurable **annual notification** of FERPA rights per tenant and track its delivery to parents/eligible students (§99.7).
- **FR-2.** The system MUST publish a **directory-information notice** listing designated directory fields and open a tenant-configurable **opt-out window** (§99.37), integrating the existing opt-out flag.
- **FR-3.** The system MUST maintain a **§99.32 disclosure record per student**, parent-inspectable, capturing recipient, data, legitimate interest/exception, and date — for every non-exempt disclosure.
- **FR-4.** The system MUST tag every disclosure with an explicit **authority basis**: consent (§99.30), school official + LEI (§99.31(a)(1)), other-school-enrollment (§99.31(a)(2)), audit/evaluation (§99.31(a)(3)), studies (§99.31(a)(6)), health/safety emergency (§99.36), directory info (§99.37), or statutory.
- **FR-5.** For **studies** and **audit/evaluation** exceptions, the system MUST require and store the **written agreement** metadata (purpose, IRB/authority, destruction date) before allowing disclosure.
- **FR-6.** The system MUST enforce **redisclosure** rules (§99.33): mark recipients' permitted purpose; where the institution rediscloses on behalf, record it in the §99.32 log.
- **FR-7.** The system MUST provide a FERPA **de-identification** routine (§99.31(b)) — removing direct + indirect identifiers and applying small-cell suppression — for analytics/research/AI exports.
- **FR-8.** The system MUST let records be flagged **sole-possession** or **law-enforcement-unit** so they are **excluded** from education-record disclosures and access requests.
- **FR-9.** The **health-or-safety emergency** disclosure MUST require a recorded articulable-threat justification and be time-bounded and reviewed.

## 6. Non-Functional Requirements

- **Performance** — Annual-notice generation and de-identified exports run as jobs; parent §99.32 log loads < 1 s.
- **Security** — Exception invocation (esp. §99.36) gated by role + reason; de-identification validated to prevent re-identification; step-up auth on record inspection (per 10.1).
- **Privacy & Compliance** — 34 CFR §§ 99.7, 99.30–99.39; alignment with state ed-privacy (S11) and PPRA (S10); NIST 800-53 AC/AU/IP.
- **Accessibility** — Notices and parent log WCAG 2.1 AA; notices in plain language + translated.
- **Scalability** — Whole-district annual-notice runs (tens of thousands) via queue.
- **Reliability** — Disclosure logging is transactional with the disclosure itself (no disclosure without a log row).
- **Observability** — `ferpa_disclosures_total{authority}` (extends 10.1), `ferpa_annual_notices_sent`, `ferpa_emergency_disclosures_total`; alert on emergency-exception spikes.
- **Maintainability** — Extends `server/internal/service/ferpa/service.go`; exceptions modelled as an enum + policy, not ad-hoc branches.
- **Internationalization** — Annual notice localised (districts serve multilingual families).
- **Backward compatibility** — Extends `179_ferpa` schema additively; existing disclosure-log rows map to authority-tagged records.

## 7. Acceptance Criteria

- **AC-1.** *Given* a tenant's school year start, *when* the annual-notice job runs, *then* every parent/eligible student receives the FERPA rights notice + directory notice, delivery is tracked, and the opt-out window opens.
- **AC-2.** *Given* a parent opens the §99.32 record, *when* it loads, *then* they see every non-exempt disclosure of their child's records with recipient, data, basis, and date.
- **AC-3.** *Given* a researcher requests education-record data under the studies exception, *when* no written agreement is on file, *then* the disclosure is blocked until the agreement metadata is recorded.
- **AC-4.** *Given* a safety officer invokes §99.36, *when* they disclose, *then* an articulable-threat justification is required, the disclosure is logged as `health_safety_emergency`, and it surfaces in the emergency-disclosure metric and review queue.
- **AC-5.** *Given* an analytics export of education records, *when* de-identification runs, *then* direct/indirect identifiers are removed, small cells are suppressed, and re-identification testing fails.
- **AC-6.** *Given* a record flagged sole-possession, *when* a parent files an access request (S01), *then* it is excluded and the exclusion basis is documented.

## 8. Data Model

New migration `365_ferpa_hardening.sql` (+ `.down.sql`), extending `179_ferpa`:

```sql
ALTER TABLE compliance.ferpa_disclosure_log
  ADD COLUMN authority_section TEXT,             -- '99.30','99.31_a1',...,'99.36','99.37'
  ADD COLUMN redisclosure_permitted BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN written_agreement_id UUID,
  ADD COLUMN emergency_justification TEXT,
  ADD COLUMN parent_inspectable BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS compliance.ferpa_annual_notices (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES org.organizations(id),
  school_year   TEXT NOT NULL,
  directory_fields TEXT[] NOT NULL,
  optout_opens_at  DATE NOT NULL,
  optout_closes_at DATE NOT NULL,
  content_version INT NOT NULL DEFAULT 1,
  generated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.ferpa_written_agreements (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES org.organizations(id),
  exception_type TEXT NOT NULL CHECK (exception_type IN ('studies','audit_evaluation')),
  recipient     TEXT NOT NULL,
  purpose       TEXT NOT NULL,
  irb_ref       TEXT,
  destruction_date DATE,
  signed_at     DATE
);

ALTER TABLE "user".users
  ADD COLUMN ferpa_record_class TEXT NOT NULL DEFAULT 'education'
    CHECK (ferpa_record_class IN ('education','sole_possession','law_enforcement_unit'));
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/compliance/ferpa/annual-notice` | `records:admin` | Generate + send annual + directory notice |
| `GET` | `/api/v1/compliance/ferpa/disclosures/{student_id}` | parent / eligible student | §99.32 inspectable record |
| `POST` | `/api/v1/compliance/ferpa/written-agreements` | `records:admin` | Register studies/audit agreement |
| `POST` | `/api/v1/compliance/ferpa/emergency-disclosure` | `records:safety` | §99.36 with justification |
| `POST` | `/api/v1/compliance/ferpa/deidentify` | `records:admin` | De-identified export job |
| `GET` | `/api/v1/compliance/ferpa/emergency-review` | `records:admin` | Emergency-disclosure review queue |

## 10. UI / UX

- **Annual-notice builder:** directory-field designation, opt-out window dates, preview, translated versions, send + delivery tracking.
- **Parent §99.32 record viewer:** chronological disclosures with basis explained in plain language.
- **Exception console:** written-agreement registry; emergency-disclosure form with mandatory justification; review queue.
- States: opt-out window open/closed banner; emergency-disclosure flagged for review; empty ("no disclosures on record").
- Accessibility: plain-language notices, translated; i18n keys `ferpa.*`.

## 11. AI / ML Considerations

Any AI/analytics use of education records MUST pass through the §99.31(b) de-identification routine first, or hold consent (S04) / a studies agreement. The AI gateway MUST treat education records as non-disclosable to external models absent one of these bases (extends 10.17/10.1 FR).

## 12. Integration Points

- `server/internal/service/ferpa/service.go` (extend); S01 (record class exclusions), S02 (agreement destruction dates → retention), S04 (consent basis), S10/S11.
- Mail (6.2) for annual notices; job queue for bulk send + de-identified exports; `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: 10.1, 10.11, and at least S01's request model.
- Must ship before: S11 (state laws build on FERPA disclosure records), and any HE/K12 GA push.
- Shared infra: email, job queue.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Emergency exception abused for routine disclosure | M | H | Mandatory justification + review queue + metric alerting |
| De-identification insufficient (re-identification) | M | H | Direct+indirect identifier removal, small-cell suppression, re-id testing |
| Annual notice not actually delivered to all families | M | H | Delivery tracking + retries; multilingual; fallback channels |
| Sole-possession/LE-unit misclassification hides real records | L | M | Restrict who can classify; audit classification changes |

## 15. Rollout Plan

- Flag `ferpa_hardening_enabled` (default off; requires 10.1 on). Phase 1: schema + authority tagging of disclosures. Phase 2: annual notice + §99.32 parent viewer. Phase 3: exceptions (written agreements, §99.36) + de-identification. Pilot: one K12 district + one HE tenant. GA for K12/HE tenant types. Rollback: flag off (additive schema).

## 16. Test Plan

- **Unit** — authority-basis enum coverage; de-identification (identifier removal + suppression); opt-out window math.
- **Integration** — no disclosure without a log row; studies disclosure blocked without agreement; emergency disclosure recorded + queued.
- **E2E** — generate annual notice → parent inspects §99.32 record; de-identified export.
- **Security** — authz on emergency/exception routes; re-auth on record viewing; re-identification attempt fails.
- **Accessibility** — axe + reading-level on notices; translated rendering.
- **Performance** — district-wide annual send via queue; parent log < 1 s.
- **Manual** — counsel checklist mapping each control to a §99 citation.

## 17. Documentation & Training

- Admin runbook: configuring annual notice + directory fields + opt-out window.
- Parent help: "Your right to see who accessed your child's records."
- Registrar guide: choosing the correct disclosure exception; studies-agreement onboarding.

## 18. Open Questions

1. Default directory-information field set per tenant type (K12 vs HE differ)?
2. Emergency-disclosure review — who reviews, and what's the SLA?
3. De-identification method sign-off — do we require an expert-determination review for research exports?
4. Delivery-tracking channel for annual notice when we lack a parent email (paper fallback responsibility)?

## 19. References

- `server/internal/service/ferpa/service.go`, `server/migrations/179_ferpa.sql`, `server/internal/repos/ferpa`
- FERPA 34 CFR §§ 99.7, 99.30, 99.31 (incl. (a)(1)–(6), (b)), 99.32, 99.33, 99.34, 99.35, 99.36, 99.37; NIST 800-53 AC-3, AU-2/12, IP-1
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S10](S10-ppra-pupil-rights.md), [S11](S11-us-state-privacy-expansion.md), [10.1](../../completed/10-compliance-privacy-security/10.1-ferpa-workflow.md)

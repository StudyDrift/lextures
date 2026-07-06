# S20 — Accessibility Legal Mandates (ADA Title II/III · §508 · EAA/EN 301 549 · AODA)

> Implementation plan. Hardens: [10.7 WCAG conformance program](../../completed/10-compliance-privacy-security/10.7-wcag-conformance-program.md), [10.8 VPAT](../../completed/10-compliance-privacy-security/10.8-vpat.md), [docs/accessibility](../../accessibility/), [docs/vpat](../../vpat/). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S20 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | US · EU · Canada (K12 · HE · SL) |
| **Status (today)** | PARTIAL — a WCAG conformance program (10.7) and VPAT (10.8) exist, but they treat accessibility as a *guideline*. The binding **legal deadlines** — ADA Title II 2024 rule, the European Accessibility Act, Section 508, AODA — are not tracked as compliance obligations with dates and evidence |
| **Estimated effort** | M (2–4w for the legal/evidence layer; remediation effort tracked separately) |
| **Owner (proposed)** | Accessibility Lead + Compliance |
| **Depends on** | 10.7, 10.8, S21 (evidence) |
| **Unblocks** | US public-entity (Title II) + EU (EAA) + Ontario (AODA) sales; RFP accessibility sections |

---

## 1. Problem Statement

Accessibility is no longer just "conform to WCAG" — it is now hard law with **dates that have arrived**. The **DOJ ADA Title II final rule (2024)** requires state/local government entities (which includes public schools, districts, and public universities — our core buyers) to meet **WCAG 2.1 AA**, with compliance dates in **April 2026 and April 2027** depending on entity size. The **European Accessibility Act** obligations began **28 June 2025**, pulling in **EN 301 549** for products/services sold in the EU. **Section 508** binds federal-funded contexts, and Ontario's **AODA** requires WCAG conformance with public reporting. Our WCAG program (10.7) produces conformance work but does not track these as **legal obligations with deadlines, jurisdiction scoping, and defensible evidence** — which is what a procurement office, a plaintiff's demand letter, or an EU market-surveillance authority actually asks for. That gap turns an accessibility bug into legal exposure.

## 2. Goals

- Track accessibility as **legal obligations** (ADA Title II/III, §508, EAA/EN 301 549, AODA) with **jurisdiction scope, effective dates, and required conformance level**, not just a WCAG backlog.
- Maintain **defensible conformance evidence** per surface: current ACR/VPAT (10.8), automated + manual + assistive-tech test results, and a dated conformance statement.
- Operate an **accessibility-conformance register** linking each product surface to its obligations, status, and known issues (with remediation SLAs).
- Provide an **accessibility statement** and a **feedback/complaint** mechanism (required by EAA and good practice everywhere).
- Feed the compliance-evidence dashboard (S21) with live accessibility posture.

## 3. Non-Goals

- The WCAG remediation work itself (10.7 / per-surface tickets) — this is the legal/evidence overlay that prioritises and proves it.
- Building a new testing harness (reuse existing axe/lighthouse pipelines in `docs/lighthouse`, `docs/accessibility`).

## 4. Personas & User Stories

- **As a district/university procurement officer**, I want a current ACR/VPAT and a conformance statement so that we can lawfully adopt under ADA Title II.
- **As an EU customer**, I want an EN 301 549 accessibility statement + feedback channel so that the product meets the EAA.
- **As an accessibility lead**, I want each surface mapped to its legal obligations and deadlines so that we remediate in priority order.
- **As a user with a disability**, I want an accessibility statement and a way to report barriers so that issues get fixed.
- **As legal counsel**, I want dated evidence of conformance so that we can answer a demand letter.

## 5. Functional Requirements

- **FR-1.** An **obligations registry** MUST encode ADA Title II (WCAG 2.1 AA, tiered 2026/2027 dates), ADA Title III, §508, EAA + EN 301 549 (from 2025), and AODA, each with jurisdiction scope and required level.
- **FR-2.** A **conformance register** MUST map each product surface (web pages, mobile screens, PDFs/exports, emails) to obligations, current status, last-tested date, and open issues with severity + remediation SLA.
- **FR-3.** The system MUST generate a dated **conformance statement / ACR (VPAT 2.x)** per obligation from the register (extends 10.8), and expose a public **accessibility statement** (EAA-compliant) with a **feedback mechanism**.
- **FR-4.** CI MUST run **automated** checks (axe/lighthouse) and record results into the register; **manual + assistive-tech** test results MUST be recordable and required for a "conformant" claim (automated alone is insufficient).
- **FR-5.** The register MUST **flag surfaces that will miss a deadline** and escalate, and MUST block a "fully conformant" claim where required manual/AT evidence is missing.
- **FR-6.** Accessibility issues reported via the feedback channel MUST enter the register and be trackable to resolution.
- **FR-7.** The register MUST feed S21 (evidence dashboard) and be exportable for procurement/legal.

## 6. Non-Functional Requirements

- **Performance** — CI checks within pipeline budgets; register queries fast.
- **Security** — Evidence exports gated by `a11y:admin`; public statement is read-only.
- **Privacy & Compliance** — ADA Title II (28 CFR Part 35, 2024 rule) + Title III; Section 508 (29 U.S.C. §794d); EAA (Directive (EU) 2019/882) + EN 301 549; AODA (O. Reg. 191/11); WCAG 2.1 AA (baseline), tracking WCAG 2.2.
- **Accessibility** — The accessibility tooling and statement are themselves WCAG 2.1 AA (dogfood).
- **Scalability** — Hundreds of surfaces tracked.
- **Reliability** — Conformance claims require complete evidence (fail-closed on the claim).
- **Observability** — `a11y_surfaces_conformant`, `a11y_deadline_at_risk`, `a11y_open_issues{severity}`; alert on deadline-at-risk.
- **Maintainability** — Register in `server/internal/service/accessibility` (extends existing) or a compliance overlay; obligations are versioned data.
- **Internationalization** — Accessibility statement localised.
- **Backward compatibility** — Builds on 10.7/10.8 outputs; additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* the obligations registry, *when* reviewed, *then* ADA Title II with the 2026/2027 tiered dates, EAA (2025), §508, and AODA each appear with scope + required level.
- **AC-2.** *Given* a web surface with passing axe but no manual/AT evidence, *when* a conformant claim is attempted, *then* it's blocked until manual/AT results are recorded.
- **AC-3.** *Given* a surface at risk of missing the April 2026 Title II date, *when* the register evaluates, *then* it's flagged and escalated with its open issues.
- **AC-4.** *Given* a procurement request, *when* the ACR/VPAT is generated, *then* it's dated, per-surface, and reflects real test evidence.
- **AC-5.** *Given* an EU user, *when* they view the accessibility statement, *then* it's EN 301 549-aligned with a working feedback mechanism, and reported issues enter the register.
- **AC-6.** *Given* S21, *when* it aggregates posture, *then* live accessibility conformance is included.

## 8. Data Model

New migration `376_accessibility_conformance.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.a11y_obligations (
  key           TEXT PRIMARY KEY,              -- 'ada_title_ii','ada_title_iii','section_508','eaa','aoda'
  jurisdiction  TEXT NOT NULL,
  required_level TEXT NOT NULL DEFAULT 'wcag21aa',
  effective_dates JSONB NOT NULL,              -- tiered dates (e.g. Title II 2026/2027)
  standard_ref  TEXT NOT NULL                  -- 'EN 301 549','28 CFR 35',...
);

CREATE TABLE IF NOT EXISTS compliance.a11y_surfaces (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,                 -- 'web:course-detail','mobile:quiz','export:report-pdf'
  platform      TEXT NOT NULL,                 -- 'web','ios','android','pdf','email'
  obligations   TEXT[] NOT NULL,               -- keys from a11y_obligations
  status        TEXT NOT NULL DEFAULT 'unknown'
                  CHECK (status IN ('unknown','non_conformant','partial','conformant')),
  automated_pass BOOLEAN,
  manual_verified BOOLEAN NOT NULL DEFAULT FALSE,
  at_verified   BOOLEAN NOT NULL DEFAULT FALSE, -- assistive-tech tested
  last_tested_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS compliance.a11y_issues (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  surface_id    UUID NOT NULL REFERENCES compliance.a11y_surfaces(id) ON DELETE CASCADE,
  wcag_ref      TEXT,                          -- '1.4.3','2.1.1'
  severity      TEXT NOT NULL CHECK (severity IN ('low','medium','high','critical')),
  source        TEXT NOT NULL,                 -- 'axe','manual','at','feedback'
  remediation_due DATE,
  resolved_at   TIMESTAMPTZ
);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/a11y/surfaces` | `a11y:admin` | Register + status |
| `POST` | `/api/v1/compliance/a11y/test-results` | CI / `a11y:admin` | Ingest automated/manual/AT results |
| `GET` | `/api/v1/compliance/a11y/acr` | `a11y:admin` | Generate ACR/VPAT (per obligation) |
| `GET` | `/api/v1/public/accessibility-statement` | public | EAA-aligned statement |
| `POST` | `/api/v1/public/accessibility-feedback` | public | Report a barrier (→ register) |
| `GET` | `/api/v1/compliance/a11y/at-risk` | `a11y:admin` | Deadline-at-risk surfaces |

## 10. UI / UX

- **Conformance register console:** surfaces × obligations grid, status, evidence completeness, deadline countdowns, open issues.
- **ACR/VPAT generator** (extends 10.8).
- **Public accessibility statement** page + **feedback form**.
- States: unknown/untested, at-risk (deadline), evidence-incomplete (blocks conformant claim), issue-open.
- Accessibility: the console/statement are exemplary WCAG 2.1 AA; i18n keys `a11y.*`.

## 11. AI / ML Considerations

AI-generated content (alt text via `alttextai`/`imagealt`, captions via `captions`, TTS) is in scope: auto-generated alt text/captions must be quality-gated, and AI outputs surfaced to users must meet the same conformance bar. Reuse existing accessibility AI modules; do not claim conformance on unreviewed AI output.

## 12. Integration Points

- Extends `server/internal/service/accessibility`, `accommodations`, `captions`, `alttextai`, `imagealt`; CI pipelines in `docs/lighthouse`/`docs/accessibility`; 10.8 VPAT; S21 (evidence).

## 13. Dependencies & Sequencing

- Must ship after: 10.7, 10.8.
- Must ship before: US public-entity + EU GA claims; ideally before the April 2026 Title II date for large entities.
- Shared infra: CI test pipeline, object storage for ACRs.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Automated-only testing over-claims conformance | H | H | Require manual + AT evidence before "conformant"; fail-closed claim |
| Deadlines (2025 EAA, 2026/2027 Title II) missed | M | H | Deadline tracking + at-risk escalation; prioritise buyer-critical surfaces |
| PDFs/exports/emails overlooked | M | H | Explicitly registered as surfaces; not just web |
| AI-generated alt/captions low quality | M | M | Quality gate + human review before conformance claim |

## 15. Rollout Plan

- Flag `a11y_conformance_program_enabled`. Phase 1: obligations + surface register + CI ingestion. Phase 2: ACR generation + public statement + feedback. Phase 3: deadline tracking + at-risk escalation + S21 feed. GA aligned to Title II/EAA dates. Rollback: register becomes reporting-only (non-blocking).

## 16. Test Plan

- **Unit** — obligation date logic; conformant-claim gating; deadline-at-risk computation.
- **Integration** — CI results ingest; feedback → issue; ACR reflects evidence.
- **E2E** — public statement + feedback flow; ACR generation.
- **Security** — authz on evidence; public endpoints read-only/rate-limited.
- **Accessibility** — the tooling + statement pass axe + AT review (dogfood).
- **Performance** — register with hundreds of surfaces.
- **Manual** — screen-reader scripts on top buyer surfaces; procurement dry-run.

## 17. Documentation & Training

- Accessibility statement + ACR/VPAT library (public trust surface).
- Runbook: recording manual/AT evidence; responding to a demand letter with evidence.
- Engineering guide: accessibility acceptance criteria in the definition of done.

## 18. Open Questions

1. Which surfaces are "buyer-critical" for the April 2026 Title II priority wave?
2. Do we adopt WCAG 2.2 AA as the internal bar ahead of legal minimums?
3. Cadence for manual/AT testing per surface (per release vs. quarterly)?
4. Ownership of PDF/export/email accessibility (often the weakest link)?

## 19. References

- `server/internal/service/{accessibility,accommodations,captions,alttextai,imagealt}`, `docs/accessibility`, `docs/vpat`, `docs/lighthouse`
- ADA Title II (28 CFR Part 35, 2024 rule) + Title III; Section 508 (29 U.S.C. §794d); EAA (Directive (EU) 2019/882); EN 301 549; AODA (O. Reg. 191/11); WCAG 2.1/2.2 AA
- Related: [10.7](../../completed/10-compliance-privacy-security/10.7-wcag-conformance-program.md), [10.8](../../completed/10-compliance-privacy-security/10.8-vpat.md), [S21](S21-compliance-evidence-continuous-monitoring.md)

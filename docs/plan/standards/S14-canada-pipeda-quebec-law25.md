# S14 — Canada: PIPEDA + Quebec Law 25 + Provincial PIPA

> Implementation plan. New coverage; configures the cross-cutting engines ([S01](S01-unified-data-subject-rights-orchestration.md)–[S07](S07-cross-border-transfer-subprocessor-governance.md)) for Canada. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S14 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR (BLOCKER for Quebec/Canada public-sector deals) |
| **Markets** | Canada (K12 · HE · SL) |
| **Status (today)** | MISSING — no Canadian privacy handling |
| **Estimated effort** | S–M (1–3w on top of the engines) |
| **Owner (proposed)** | Compliance Lead |
| **Depends on** | S01, S02, S03, S04, S05, S07 |
| **Unblocks** | Canadian market entry |

---

## 1. Problem Statement

Canada layers federal **PIPEDA** (private-sector, consent-centric, with a breach-of-security-safeguards reporting duty) over strong provincial regimes — most notably **Quebec's Law 25**, which is the strictest in North America (mandatory privacy officer, privacy impact assessments for systems/transfers, confidentiality-by-default, data-portability, a right to de-indexing, tightened consent, and significant fines), plus **BC and Alberta PIPA** and public-sector FIPPA/education-record rules. Canadian districts, colleges, and Quebec public bodies will not procure a platform that cannot demonstrate PIPEDA + Law 25 compliance and Canadian data-handling. Nothing today addresses this.

## 2. Goals

- Configure S01 rights, S04 consent, and S03 breach reporting for **PIPEDA** (real-risk-of-significant-harm breach test; record-keeping of all breaches).
- Implement **Quebec Law 25** specifics: privacy officer designation, PIA for systems/cross-border transfers (via S06/S07), confidentiality-by-default settings, portability, de-indexing/right-to-cease-dissemination, and consent tightening (esp. minors under 14).
- Support **BC/Alberta PIPA** and public-sector education-record expectations.
- Provide Canadian **data-residency** posture and transfer disclosure (10.12/S07), including Quebec's transfer-PIA duty.
- Bilingual (English/French) notices and subject-facing flows (Quebec French-language obligations).

## 3. Non-Goals

- Rebuilding rights/consent/breach engines (S01/S04/S03) — this configures them.
- Provincial public-sector FIPPA record-access build-out beyond alignment (handled like FERPA via S09 patterns where applicable).

## 4. Personas & User Stories

- **As a Quebec public body**, I want a designated privacy officer, transfer PIAs, and French notices so that Law 25 procurement clears.
- **As a Canadian data subject**, I want access/correction and portability under PIPEDA/Law 25 so that my rights are honoured.
- **As a Quebec parent**, I want consent rules for my under-14 respected so that my child is protected.
- **As a compliance officer**, I want PIPEDA breach record-keeping + real-risk assessment so that we report correctly.

## 5. Functional Requirements

- **FR-1.** S01 MUST offer PIPEDA/Law 25 rights (access, correction, withdrawal, **portability**, Quebec **de-indexing/cease-dissemination**) with correct handling and no fixed federal deadline beyond "reasonable"/30-day norms; Quebec timelines applied where stricter.
- **FR-2.** S03 MUST apply the **PIPEDA real-risk-of-significant-harm** test, notify the OPC + affected individuals when met, and **record all breaches** regardless (record-keeping duty), plus Quebec's CAI notification.
- **FR-3.** The system MUST support a **privacy officer** designation (Law 25) surfaced as a contact and case owner.
- **FR-4.** **Transfer PIAs** MUST be required (S06/S07) for cross-border flows out of Quebec/Canada, with the factors Law 25 enumerates.
- **FR-5.** Consent (S04) MUST reflect Canadian norms: meaningful consent, **confidentiality-by-default** (Law 25), and **minors under 14** require parental consent (Quebec).
- **FR-6.** Notices and key subject flows MUST be available in **French** (Quebec) as well as English.
- **FR-7.** Canadian **data-residency** options and transfer disclosures MUST be configurable (10.12/S07).

## 6. Non-Functional Requirements

- **Performance** — Reuses engine performance budgets.
- **Security** — Confidentiality-by-default enforced in profile/visibility defaults for Quebec tenants.
- **Privacy & Compliance** — PIPEDA; Quebec Law 25 (Act respecting the protection of personal information in the private sector, as amended); BC PIPA; Alberta PIPA; provincial education-record norms.
- **Accessibility** — WCAG 2.1 AA; bilingual EN/FR.
- **Scalability** — Engine-backed.
- **Reliability** — Breach record-keeping is durable and complete.
- **Observability** — `ca_rights_requests_total`, `ca_breaches_recorded_total`, `quebec_transfer_pias_total`.
- **Maintainability** — Canada as regime config in `stateprivacy`-style registry + S01/S03 strategies; no fork.
- **Internationalization** — Full French localisation (Quebec French).
- **Backward compatibility** — Additive regime.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Quebec subject requests portability + de-indexing, *when* S01 processes it, *then* both are supported and completed within Quebec timelines.
- **AC-2.** *Given* a breach meeting the real-risk test, *when* S03 assesses it, *then* OPC + individuals (+ CAI for Quebec) are notified and the breach is recorded; a sub-threshold breach is still recorded.
- **AC-3.** *Given* a Quebec tenant, *when* a cross-border transfer is configured, *then* a transfer PIA is required before it's allowed (S07).
- **AC-4.** *Given* a Quebec user under 14, *when* they register, *then* parental consent is required (S08/S04).
- **AC-5.** *Given* a Quebec deployment, *when* notices render, *then* they're available in French and confidentiality-by-default settings apply.
- **AC-6.** *Given* an auditor, *when* they review, *then* the privacy officer, breach register, and transfer PIAs are all present.

## 8. Data Model

New migration `370_canada_privacy.sql` (+ `.down.sql`) — mostly registry rows + a privacy-officer record:

```sql
INSERT INTO compliance.state_privacy_rules /* reused registry, region-namespaced */ ;
-- (Canadian regimes seeded as rows: 'CA_PIPEDA','CA_QC_L25','CA_BC_PIPA','CA_AB_PIPA')

CREATE TABLE IF NOT EXISTS compliance.privacy_officers (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID REFERENCES org.organizations(id),
  regime       TEXT NOT NULL,                  -- 'ca_qc_l25','global_dpo',...
  name         TEXT NOT NULL,
  contact      TEXT NOT NULL,
  designated_at DATE NOT NULL DEFAULT CURRENT_DATE
);
```

Reuses S01 `rights_requests`, S03 `incidents`, S06 `dpia_assessments` (Law 25 PIA = kind 'pia'), S07 transfer records.

## 9. API Surface

Reuses S01/S03/S04/S06/S07 endpoints with Canadian regime codes. New:

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/privacy-officer` | `privacy:admin` | Designate officer (Law 25) |
| `GET` | `/api/v1/compliance/ca/breach-register` | `privacy:dpo` | PIPEDA breach record export |

## 10. UI / UX

- Canadian regime toggles in the compliance admin; privacy-officer designation; French notice variants; confidentiality-by-default indicators for Quebec tenants; transfer-PIA prompts. Reuses S01/S04 subject surfaces. i18n keys reuse `rights.*`/`consent.*` with `fr-CA` locale.
- Accessibility: bilingual, WCAG 2.1 AA.

## 11. AI / ML Considerations

Law 25 requires informing individuals of decisions **based exclusively on automated processing** and offering review — align with S12 Art 22 patterns + S13. AI transfers out of Canada trigger transfer PIAs (S07).

## 12. Integration Points

- Registry (`stateprivacy`), S01, S03, S04, S06, S07, 10.12 (residency), l10n (`fr-CA`).

## 13. Dependencies & Sequencing

- Must ship after: S01, S03, S04, S06, S07.
- Must ship before: Canadian GA.
- Shared infra: French localisation, residency options.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Law 25 stricter than assumed | M | H | Treat Quebec as the strictest Canadian baseline; legal review |
| French-language obligations missed | M | M | Full fr-CA localisation gated before Quebec GA |
| Transfer PIA skipped | M | H | S07 gate blocks Quebec transfers without PIA |

## 15. Rollout Plan

- Flag `ca_privacy_enabled`. Phase 1: regime rows + rights/consent config + fr-CA notices. Phase 2: PIPEDA breach register + Quebec CAI. Phase 3: transfer PIAs + confidentiality-by-default. Pilot with a Canadian tenant. GA. Rollback: flag off.

## 16. Test Plan

- **Unit** — Canadian rights set; real-risk breach test; under-14 consent.
- **Integration** — Quebec transfer blocked without PIA; breach register completeness.
- **E2E** — Quebec subject portability + de-indexing; fr-CA notice render.
- **Security/Accessibility/Performance** — per engine defaults; bilingual axe pass.
- **Manual** — Law 25 checklist review by counsel.

## 17. Documentation & Training

- Canadian compliance runbook; privacy-officer duties; French notice management.

## 18. Open Questions

1. Do we need Canadian data residency as a hard requirement or a configurable option per tenant?
2. Quebec's automated-decision review scope for grading?
3. Provincial public-sector (FIPPA) education-record access — in scope now or later?

## 19. References

- PIPEDA; Quebec Law 25; BC PIPA; Alberta PIPA
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S03](S03-global-breach-notification-incident-response.md), [S06](S06-dpia-pia-algorithmic-impact.md), [S07](S07-cross-border-transfer-subprocessor-governance.md), [10.12](../../completed/10-compliance-privacy-security/10.12-data-residency.md)

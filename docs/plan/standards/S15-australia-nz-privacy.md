# S15 — Australia (Privacy Act / APPs / NDB) + New Zealand

> Implementation plan. New coverage; configures the cross-cutting engines for AU/NZ. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S15 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | Australia · New Zealand (K12 · HE · SL) |
| **Status (today)** | MISSING — no AU/NZ privacy handling |
| **Estimated effort** | S–M (1–3w on top of the engines) |
| **Owner (proposed)** | Compliance Lead |
| **Depends on** | S01, S03, S04, S05, S07 |
| **Unblocks** | AU/NZ market entry |

---

## 1. Problem Statement

Australia's **Privacy Act 1988** and the **13 Australian Privacy Principles (APPs)** govern collection, use, disclosure, cross-border disclosure (APP 8), and access/correction (APP 12/13), backed by the **Notifiable Data Breaches (NDB)** scheme with its own "eligible data breach"/"likely serious harm" test and OAIC + individual notification duty. Recent reforms increase penalties and tighten children's protections (a forthcoming Children's Online Privacy Code). **New Zealand's Privacy Act 2020** adds its own 13 IPPs, a mandatory breach-notification regime, and cross-border rules (IPP 12). Australian and NZ schools, universities, and TAFEs require APP/NDB alignment and often local data residency. Nothing today addresses this.

## 2. Goals

- Configure S01 access/correction for **APP 12/13** and NZ IPP access/correction, S04 for APP collection/consent norms.
- Implement **NDB** (AU) and NZ breach notification in S03, each with its own harm test, OAIC/OPC (NZ) + individual notice.
- Apply **APP 8 / NZ IPP 12** cross-border disclosure accountability (S07), with AU/NZ data-residency options (10.12).
- Prepare for the AU **Children's Online Privacy Code** (align with S08).
- Support AU/NZ government/education vendor expectations (e.g. IRAP-style posture references via SOC2/ISO evidence — S21).

## 3. Non-Goals

- Rebuilding engines; AU/NZ are regime configs.
- Full IRAP certification (referenced; certification is a separate program).

## 4. Personas & User Stories

- **As an Australian university**, I want APP 12 access + NDB compliance + APP 8 transfer accountability so that procurement clears.
- **As a NZ school**, I want IPP access/correction and NZ breach notification so that we meet the Privacy Act 2020.
- **As an AU data subject**, I want access/correction and to know about overseas disclosures so that my APP rights are honoured.
- **As a compliance officer**, I want the NDB harm assessment recorded so that we notify correctly and on time.

## 5. Functional Requirements

- **FR-1.** S01 MUST offer APP 12/13 (AU) and IPP 6/7 (NZ) **access/correction**, handled to the "reasonable"/statutory timeframes.
- **FR-2.** S03 MUST implement the **NDB** eligible-breach/likely-serious-harm assessment → OAIC + individual notification, and the NZ notification test → OPC + individuals.
- **FR-3.** **APP 8 / IPP 12** cross-border disclosure accountability MUST be enforced via S07 (the discloser remains accountable for overseas recipients), with residency options (10.12).
- **FR-4.** Consent/collection (S04) MUST reflect APP 3/5 (collection notice) and NZ IPP 3 (collection from the individual, purpose notice).
- **FR-5.** Children's protections MUST align with S08 and track the forthcoming AU **Children's Online Privacy Code**.
- **FR-6.** Notices/flows MUST support en-AU/en-NZ conventions; Māori-language support considered for NZ public sector.

## 6. Non-Functional Requirements

- **Performance/Security/Reliability** — engine defaults.
- **Privacy & Compliance** — AU Privacy Act 1988 + APPs 1–13 + NDB scheme; NZ Privacy Act 2020 + IPPs + breach regime; APP 8 / IPP 12 transfers.
- **Accessibility** — WCAG 2.1 AA.
- **Observability** — `au_ndb_assessments_total`, `nz_breach_notifications_total`, `apac_rights_requests_total{country}`.
- **Maintainability** — AU/NZ regime rows + S01/S03 strategies.
- **Internationalization** — en-AU/en-NZ; te reo Māori consideration.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* an AU subject requests access, *when* S01 processes it, *then* APP 12 handling and timeframe apply.
- **AC-2.** *Given* an eligible data breach (AU), *when* S03 assesses likely serious harm, *then* OAIC + affected individuals are notified with the NDB statement.
- **AC-3.** *Given* a NZ breach with notifiable harm, *when* assessed, *then* the OPC + individuals are notified per the Privacy Act 2020.
- **AC-4.** *Given* an AU tenant disclosing data overseas, *when* configured, *then* APP 8 accountability is recorded (S07) and residency options are offered.
- **AC-5.** *Given* AU children's-code readiness, *when* minors use the platform, *then* S08 protections apply.
- **AC-6.** *Given* an auditor, *when* they review, *then* NDB assessments and transfer records are present.

## 8. Data Model

New migration `371_anz_privacy.sql` (+ `.down.sql`): regime rows (`AU_PRIVACY_ACT`, `NZ_PRIVACY_ACT`) in the registry; NDB assessment stored as an S03 incident risk-assessment variant. No new tables beyond registry seeds + an optional `overseas_recipients` view over S07 for APP 8 reporting.

## 9. API Surface

Reuses S01/S03/S04/S07 with AU/NZ regime codes. New:

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET` | `/api/v1/compliance/au/ndb-register` | `privacy:dpo` | NDB assessment/notification register |
| `GET` | `/api/v1/compliance/apac/overseas-disclosures` | `privacy:dpo` | APP 8 / IPP 12 recipient list (from S07) |

## 10. UI / UX

- AU/NZ regime toggles; NDB assessment workflow inside the S03 case; APP 8 overseas-disclosure disclosures on collection notices; residency selection. Reuses S01/S04 surfaces. i18n `en-AU`/`en-NZ`.
- Accessibility: WCAG 2.1 AA.

## 11. AI / ML Considerations

AI-driven overseas disclosures (e.g. LLM providers offshore) invoke APP 8 / IPP 12 accountability via S07; align automated-decision transparency with S12/S13 where AU/NZ reforms introduce ADM duties.

## 12. Integration Points

- Registry, S01, S03, S04, S07, 10.12; l10n en-AU/en-NZ.

## 13. Dependencies & Sequencing

- Must ship after: S01, S03, S04, S07.
- Must ship before: AU/NZ GA.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| NDB harm test misapplied | M | H | Documented assessment in S03; legal review of criteria |
| APP 8 accountability overlooked for offshore AI | M | H | S07 records all offshore recipients; collection notice discloses |
| AU reforms (children's code, ADM) land mid-build | M | M | Track reform timeline; S08/S12 hooks ready |

## 15. Rollout Plan

- Flag `anz_privacy_enabled`. Phase 1: regime rows + rights/collection notices. Phase 2: NDB + NZ breach in S03. Phase 3: APP 8/IPP 12 + residency. Pilot AU + NZ tenant. GA. Rollback: flag off.

## 16. Test Plan

- **Unit** — AU/NZ rights sets; NDB/NZ harm tests.
- **Integration** — offshore disclosure recorded (APP 8); breach notification flows.
- **E2E** — AU access request; NDB notification draft.
- **Security/Accessibility/Performance** — engine defaults; en-AU/en-NZ axe.
- **Manual** — APP/IPP checklist review.

## 17. Documentation & Training

- AU/NZ compliance runbook; NDB assessment guide; APP 8 transfer disclosures.

## 18. Open Questions

1. AU/NZ data residency — mandatory for government/education tenants?
2. Timing of the AU Children's Online Privacy Code vs. our build?
3. te reo Māori localisation scope for NZ public sector?

## 19. References

- AU Privacy Act 1988; Australian Privacy Principles 1–13; NDB scheme; NZ Privacy Act 2020 + IPPs
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S03](S03-global-breach-notification-incident-response.md), [S07](S07-cross-border-transfer-subprocessor-governance.md), [S08](S08-childrens-privacy-age-assurance-design-codes.md)

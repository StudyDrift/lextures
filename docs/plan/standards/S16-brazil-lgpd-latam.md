# S16 — Brazil LGPD (+ LATAM)

> Implementation plan. New coverage; configures the cross-cutting engines for Brazil and the wider LATAM region. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S16 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | Brazil · LATAM (K12 · HE · SL) |
| **Status (today)** | MISSING — no LGPD handling |
| **Estimated effort** | S–M (1–3w on top of the engines) |
| **Owner (proposed)** | Compliance Lead |
| **Depends on** | S01, S02, S03, S04, S05, S07 |
| **Unblocks** | Brazil + Spanish-speaking LATAM market entry |

---

## 1. Problem Statement

Brazil's **LGPD** (Lei Geral de Proteção de Dados, Law 13.709/2018) is a GDPR-style regime enforced by the **ANPD**: ten legal bases, an extensive data-subject rights set (confirmation, access, correction, anonymisation/blocking/deletion, portability, information about sharing, and review of automated decisions), a **DPO ("encarregado")** requirement, breach notification to the ANPD and data subjects, and special protection for **children and adolescents** (best-interest standard, parental consent for children). Around it, the LATAM region has active regimes — Mexico's LFPDPPP (with a recent reform), Argentina, Chile's modernised law, Colombia, and others — that Portuguese/Spanish-language education buyers expect. Nothing today addresses Brazil or LATAM.

## 2. Goals

- Configure S01 rights for the **LGPD Art 18** set (incl. anonymisation/blocking and **review of automated decisions**), S04 consent + legal bases (Art 7/11).
- Implement **ANPD** breach notification (+ data-subject notice) in S03.
- Designate a **DPO/encarregado** (Art 41) surfaced as a contact.
- Apply **children/adolescent** protections (Art 14 — best interest, parental consent) aligned with S08.
- Provide **Portuguese (pt-BR)** and **Spanish (es-419)** notices/flows; support LATAM regime rows (Mexico, Argentina, Chile, Colombia) for expansion.
- Cross-border transfer accountability (LGPD Ch. V) via S07.

## 3. Non-Goals

- Rebuilding engines; Brazil/LATAM are regime configs.
- Full per-country LATAM build in v1 (Brazil first; others seeded as registry rows for staged enablement).

## 4. Personas & User Stories

- **As a Brazilian school/university**, I want LGPD rights, an encarregado contact, and ANPD breach handling so that we comply.
- **As a Brazilian data subject**, I want access, anonymisation/deletion, portability, and review of automated decisions so that my Art 18 rights are honoured.
- **As a Brazilian parent**, I want my child's data handled in their best interest with my consent so that Art 14 is met.
- **As a compliance officer**, I want ANPD notification handled by the breach engine so that timelines are met.

## 5. Functional Requirements

- **FR-1.** S01 MUST offer the LGPD Art 18 rights, including **anonymisation/blocking**, **portability**, information about **data sharing**, and **review of decisions taken solely on automated processing**.
- **FR-2.** S04 MUST record the applicable **legal basis** (Art 7 general; Art 11 sensitive) per purpose; consent must be specific and highlighted.
- **FR-3.** S03 MUST notify the **ANPD** and affected subjects on qualifying breaches within the expected timeframe, recording the risk assessment.
- **FR-4.** The system MUST designate an **encarregado (DPO)** (Art 41) with a public contact.
- **FR-5.** **Children/adolescents** (Art 14) MUST be handled in their best interest with parental consent for children (S08).
- **FR-6.** Cross-border transfers MUST use an LGPD-valid mechanism (adequacy, SCCs, specific consent, etc.) via S07.
- **FR-7.** Notices/flows MUST be available in **pt-BR** (and es-419 for LATAM expansion).

## 6. Non-Functional Requirements

- **Privacy & Compliance** — LGPD (Law 13.709/2018) Arts 7, 9, 11, 14, 18, 33–36 (transfers), 37–41, 48 (breach); ANPD regulations; LATAM (LFPDPPP, Argentina PDPA, Chile, Colombia) as registry rows.
- **Accessibility** — WCAG 2.1 AA; pt-BR/es-419.
- **Observability** — `br_rights_requests_total`, `br_anpd_notifications_total`, `latam_rights_requests_total{country}`.
- **Maintainability** — Regime rows + S01/S03 strategies; Portuguese/Spanish localisation.
- **Reliability/Security/Performance** — engine defaults.
- **Internationalization** — pt-BR primary; es-419 for LATAM.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Brazilian subject requests anonymisation and portability, *when* S01 processes them, *then* both are supported (anonymisation via S02) and completed.
- **AC-2.** *Given* an automated grade, *when* a Brazilian subject requests review, *then* an Art 20 review is provided (reuses S12/S13 human-review path).
- **AC-3.** *Given* a qualifying breach, *when* S03 assesses it, *then* the ANPD + affected subjects are notified and the assessment is recorded.
- **AC-4.** *Given* a Brazilian tenant, *when* configured, *then* an encarregado contact is published.
- **AC-5.** *Given* a Brazilian child registers, *when* onboarding runs, *then* best-interest handling + parental consent apply (S08).
- **AC-6.** *Given* pt-BR locale, *when* notices render, *then* they're in Brazilian Portuguese.

## 8. Data Model

New migration `372_lgpd_latam.sql` (+ `.down.sql`): regime rows (`BR_LGPD`, `MX_LFPDPPP`, `AR_PDPA`, `CL_LPDP`, `CO_L1581`) in the registry; encarregado stored via `compliance.privacy_officers` (from S14) with `regime='br_lgpd'`. Reuses S01/S02/S03/S07 tables.

## 9. API Surface

Reuses S01/S03/S04/S07 with LATAM regime codes. New:

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/br/encarregado` | `privacy:admin` | Designate DPO (Art 41) |
| `GET` | `/api/v1/compliance/br/anpd-register` | `privacy:dpo` | ANPD breach register |

## 10. UI / UX

- Brazil/LATAM regime toggles; encarregado designation; pt-BR/es-419 notices; Art 18 rights in the S01 portal (anonymisation/blocking added); automated-decision review link. i18n `pt-BR`, `es-419`.
- Accessibility: WCAG 2.1 AA.

## 11. AI / ML Considerations

LGPD Art 20 review of automated decisions maps to the S12/S13 human-review + explanation path. Offshore AI providers invoke LGPD transfer rules via S07.

## 12. Integration Points

- Registry, S01, S02 (anonymisation), S03, S04, S07, l10n pt-BR/es-419, S14 privacy-officer table.

## 13. Dependencies & Sequencing

- Must ship after: S01, S02, S03, S04, S07.
- Must ship before: Brazil GA (then LATAM rollout).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| ANPD timelines/regulations evolve | M | M | Configurable timeframe; track ANPD guidance |
| LATAM per-country nuance underestimated | M | M | Brazil-first; others staged as reviewed rows |
| pt-BR localisation incomplete | M | M | Gate Brazil GA on full pt-BR notice set |

## 15. Rollout Plan

- Flag `lgpd_enabled`. Phase 1: BR_LGPD row + rights/consent + pt-BR notices + encarregado. Phase 2: ANPD breach in S03 + Art 20 review. Phase 3: transfers + LATAM rows (es-419). Pilot Brazilian tenant. GA Brazil, then LATAM. Rollback: flag off.

## 16. Test Plan

- **Unit** — LGPD rights set; legal-basis capture; anonymisation via S02.
- **Integration** — ANPD notification; Art 20 review path; offshore transfer via S07.
- **E2E** — Brazilian subject anonymisation + portability; pt-BR notice render.
- **Security/Accessibility/Performance** — engine defaults; pt-BR axe.
- **Manual** — LGPD checklist review.

## 17. Documentation & Training

- Brazil/LATAM compliance runbook; encarregado duties; pt-BR/es-419 notice management.

## 18. Open Questions

1. Brazilian data residency expectations for education buyers?
2. Which LATAM countries to enable first after Brazil?
3. ANPD breach-notification concrete deadline as guidance firms up?

## 19. References

- LGPD (Law 13.709/2018); Mexico LFPDPPP; Argentina PDPA; Chile LPDP; Colombia Law 1581
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S02](S02-data-retention-deletion-engine.md), [S03](S03-global-breach-notification-incident-response.md), [S07](S07-cross-border-transfer-subprocessor-governance.md), [S08](S08-childrens-privacy-age-assurance-design-codes.md)

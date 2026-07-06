# S18 — China PIPL & Data-Localization

> Implementation plan. New coverage; configures the cross-cutting engines for China. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S18 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR (BLOCKER to operate in mainland China) |
| **Markets** | China (HE · SL; K12 heavily restricted) |
| **Status (today)** | MISSING — no PIPL/localization handling |
| **Estimated effort** | M (2–4w on top of the engines; localization infra is the long pole) |
| **Owner (proposed)** | Compliance Lead + Infra |
| **Depends on** | S01, S03, S04, S05, S07, 10.12 (residency) |
| **Unblocks** | Mainland-China deployments (with caveats) |

---

## 1. Problem Statement

China's **PIPL** (Personal Information Protection Law), alongside the **Data Security Law (DSL)** and **Cybersecurity Law (CSL)**, imposes a strict regime: consent (often **separate consent** for sensitive PI, cross-border transfer, and third-party sharing), data-subject rights, a **local representative** for offshore processors, **data-localization** expectations, and **cross-border transfer** mechanisms (CAC security assessment, certification, or SCCs) that are among the world's most demanding. Sensitive PI includes minors' data (under-14 requires guardian consent + special rules). K-12 online tutoring is heavily regulated in China; even HE/self-learner operation requires PIPL alignment and, realistically, in-country data handling. Operating in China without this is unlawful; this plan defines what "China-ready" means and flags where a separate in-country deployment is required.

## 2. Goals

- Configure S04 for PIPL consent, including **separate consent** for sensitive PI, cross-border transfer, and third-party sharing.
- Configure S01 for PIPL rights (access, correction, deletion, portability, explanation of automated decisions).
- Implement **cross-border transfer** controls in S07 (CAC security assessment / certification / SCC filing) and **data-localization** posture via 10.12.
- Handle **minors under 14** as sensitive PI with guardian consent + specific rules (S08), aligned with the strict Chinese standard.
- Define a **local representative / entity** requirement and the boundary where a **separate in-country deployment** is mandatory.

## 3. Non-Goals

- Standing up China in-country infrastructure (a major infra program; this plan specifies the requirement and gates behaviour).
- K-12 online-tutoring market entry decisions (regulatory/business call flagged here, not resolved).

## 4. Personas & User Stories

- **As a Chinese university/institution**, I want PIPL-aligned consent, rights, and localized data handling so that deployment is lawful.
- **As a Chinese data subject**, I want separate consent for sensitive data and transfers, and explanation of automated decisions, so that my PIPL rights are honoured.
- **As a guardian**, I want under-14 data treated as sensitive with my consent so that minors are protected.
- **As legal/compliance**, I want cross-border transfers gated behind the correct CAC mechanism so that we don't transfer unlawfully.

## 5. Functional Requirements

- **FR-1.** S04 MUST capture **separate, explicit consent** for (a) sensitive PI, (b) each cross-border transfer, and (c) third-party sharing — distinct from general consent.
- **FR-2.** S01 MUST offer PIPL rights (access, copy/**portability**, correction, **deletion**, and **explanation** of automated decision-making).
- **FR-3.** S07 MUST gate cross-border transfers behind a valid PIPL mechanism (**CAC security assessment**, **certification**, or **standard contract filing**) with a transfer impact/PIA (S06), and MUST block transfers lacking one.
- **FR-4.** Data-localization posture (10.12) MUST allow configuring **in-China storage/processing**; the system MUST detect and flag when operating in China without localized handling.
- **FR-5.** **Minors under 14** MUST be treated as sensitive PI with **guardian consent** and dedicated rules (S08).
- **FR-6.** A **local representative/entity** contact MUST be configurable and surfaced (PIPL Art 53).
- **FR-7.** Notices/flows MUST be available in **Simplified Chinese (zh-CN)**.

## 6. Non-Functional Requirements

- **Privacy & Compliance** — PIPL (esp. Arts 13–14, 23, 28–31 sensitive PI, 38–41 cross-border, 44–50 rights, 53 representative); DSL; CSL; Minors' cyber-protection rules.
- **Accessibility** — WCAG 2.1 AA; zh-CN.
- **Observability** — `cn_rights_requests_total`, `cn_crossborder_blocked_total`, `cn_localization_violations_total` (should be 0 in-market).
- **Security/Reliability** — transfer gate fail-closed.
- **Maintainability** — Regime row + strict sensitive-PI/child profile; zh-CN localisation.
- **Internationalization** — Simplified Chinese.
- **Backward compatibility** — additive; China gated behind explicit enablement + localization readiness.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Chinese subject, *when* sensitive PI or a transfer is involved, *then* separate explicit consent is captured distinctly from general consent.
- **AC-2.** *Given* a cross-border transfer without a valid CAC mechanism, *when* it's attempted, *then* it's blocked and `cn_crossborder_blocked_total` increments.
- **AC-3.** *Given* an automated decision, *when* a Chinese subject requests explanation, *then* it's provided (reuses S12/S13).
- **AC-4.** *Given* operation in China without localized handling, *when* detected, *then* it's flagged and the deployment is gated.
- **AC-5.** *Given* an under-14 user, *when* they register, *then* guardian consent + sensitive-PI handling apply (S08).
- **AC-6.** *Given* zh-CN locale, *when* notices render, *then* they appear in Simplified Chinese, with the local representative contact shown.

## 8. Data Model

New migration `374_china_pipl.sql` (+ `.down.sql`): regime row `CN_PIPL` (sensitive-by-default for minors under 14, transfer-gated); `consent_records` (S04) already supports per-purpose "separate consent" (each transfer/sensitive purpose is its own purpose_key). Local representative via `compliance.privacy_officers` (`regime='cn_pipl_rep'`). Transfer mechanism enum in S07 extended with `cac_assessment`, `cac_certification`, `cn_standard_contract`.

## 9. API Surface

Reuses S01/S03/S04/S07 with China regime codes. New:

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/cn/representative` | `privacy:admin` | Local representative (Art 53) |
| `GET` | `/api/v1/compliance/cn/transfer-status` | `privacy:dpo` | Cross-border mechanism status |
| internal | localization guard | — | Flags/gates un-localized China operation |

## 10. UI / UX

- China regime toggle; separate-consent prompts (sensitive/transfer/sharing shown distinctly); local representative contact; zh-CN notices; localization-readiness banner. Strict under-14 experience. i18n `zh-CN`.
- Accessibility: WCAG 2.1 AA, zh-CN.

## 11. AI / ML Considerations

Offshore AI providers are cross-border transfers of PI → blocked unless a CAC mechanism + separate consent exist; in practice, China deployments should route AI to compliant/in-country providers or disable offshore AI. Automated-decision explanation reuses S12/S13.

## 12. Integration Points

- Registry, S01, S03, S04 (separate consent), S07 (CAC mechanisms), 10.12 (localization), S08, l10n zh-CN, S14 officer table, `aigateway` (block offshore transfer without mechanism).

## 13. Dependencies & Sequencing

- Must ship after: S01, S03, S04, S07, 10.12.
- Must ship before: any mainland-China deployment.
- Shared infra: **data-localization** capability is the gating prerequisite.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Cross-border transfer without CAC mechanism | M | H | S07 fail-closed gate; block offshore AI without mechanism |
| Operating without localization | M | H | Localization guard + deployment gate; flag as business/legal decision |
| Regulatory complexity underestimated | H | H | Treat China as separate-deployment by default; legal counsel required pre-launch |
| K-12 tutoring restrictions | M | H | Scope to HE/SL; explicit legal review before any K-12 offering |

## 15. Rollout Plan

- Flag `pipl_enabled` (off by default; requires localization readiness). Phase 1: CN_PIPL row + separate-consent + zh-CN + representative. Phase 2: transfer gate + localization guard. Phase 3: in-country deployment integration (separate infra program). No GA without localized handling + legal sign-off. Rollback: flag off; China disabled.

## 16. Test Plan

- **Unit** — separate-consent capture; transfer-mechanism gate; under-14 sensitive handling.
- **Integration** — offshore transfer blocked without mechanism; localization guard flags un-localized op.
- **E2E** — Chinese subject sensitive-consent + rights; zh-CN notice render.
- **Security/Accessibility/Performance** — engine defaults; zh-CN axe.
- **Manual** — PIPL/DSL/CSL checklist + counsel review; go/no-go on China.

## 17. Documentation & Training

- China compliance runbook (with the explicit "separate deployment" decision framework); representative duties; zh-CN notice management.

## 18. Open Questions

1. Do we operate China at all, and if so via a separate in-country deployment/partner?
2. Which cross-border mechanism is realistic for our data volumes (assessment vs. standard contract)?
3. AI providers acceptable in-country vs. disabling offshore AI for China?
4. K-12 exclusion — confirm scope to HE/SL only.

## 19. References

- PIPL; Data Security Law; Cybersecurity Law; Provisions on the Cyber Protection of Minors
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S04](S04-unified-consent-preference-ledger.md), [S07](S07-cross-border-transfer-subprocessor-governance.md), [10.12](../../completed/10-compliance-privacy-security/10.12-data-residency.md)

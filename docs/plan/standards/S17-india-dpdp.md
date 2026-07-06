# S17 — India DPDP Act 2023

> Implementation plan. New coverage; configures the cross-cutting engines for India. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S17 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | India (K12 · HE · SL) |
| **Status (today)** | MISSING — no DPDP handling |
| **Estimated effort** | S–M (1–3w on top of the engines) |
| **Owner (proposed)** | Compliance Lead |
| **Depends on** | S01, S02, S03, S04, S07, S08 |
| **Unblocks** | India market entry |

---

## 1. Problem Statement

India's **Digital Personal Data Protection Act, 2023** (with its implementing Rules) is a consent-first regime with features that materially affect an ed-tech platform: a mandated **Consent Manager** ecosystem and clear notice, data-principal rights (access, correction, erasure, grievance redressal, nomination), a **Data Protection Officer / grievance officer** requirement for Significant Data Fiduciaries, breach notification to the **Data Protection Board** and affected principals, and — critically for us — **heightened child protections**: **verifiable parental consent for anyone under 18** and a **prohibition on tracking, behavioural monitoring, and targeted advertising directed at children**. India is one of the largest education markets in the world; without DPDP alignment we cannot responsibly operate there.

## 2. Goals

- Configure S04 for **DPDP consent + clear notice** (itemised, withdrawable, with a Consent Manager integration path).
- Configure S01 for DPDP rights (access, correction, **erasure**, grievance redressal, **nomination**).
- Implement **breach notification** to the Data Protection Board + principals in S03.
- Enforce **under-18 verifiable parental consent** and **no child tracking/behavioural monitoring/targeted ads** (S08) — the strictest child bar in this folder.
- Designate a **grievance/DPO** contact and grievance-redressal workflow.
- Support **English + major Indian languages** for notices (the Act emphasises language accessibility).

## 3. Non-Goals

- Rebuilding engines; India is a regime config with a notably strict child profile.
- Building a Consent Manager product (we integrate with the ecosystem; internal ledger is S04).

## 4. Personas & User Stories

- **As an Indian school/university**, I want DPDP consent, grievance redressal, and breach handling so that we comply.
- **As an Indian data principal**, I want access/correction/erasure, a nominee, and a grievance path so that my DPDP rights are honoured.
- **As an Indian parent**, I want to consent for my under-18 and be assured no behavioural tracking/ads target my child so that DPDP's child rules are met.
- **As a compliance officer**, I want Data Protection Board notification handled so that timelines are met.

## 5. Functional Requirements

- **FR-1.** S04 MUST present a **clear, itemised notice** and capture **specific, withdrawable consent** per purpose, with an integration path to registered **Consent Managers**.
- **FR-2.** S01 MUST offer DPDP rights: **access, correction, erasure, grievance redressal, and nomination** (designating someone to exercise rights on death/incapacity).
- **FR-3.** S03 MUST notify the **Data Protection Board of India** and affected principals on a personal-data breach.
- **FR-4.** For all users **under 18**, the system MUST obtain **verifiable parental consent** and MUST NOT undertake **tracking, behavioural monitoring, or targeted advertising** directed at them (S08) — enforced, not advisory.
- **FR-5.** The system MUST provide a **grievance-redressal** workflow and a published **grievance officer/DPO** contact; Significant-Data-Fiduciary obligations (DPO, DPIA, audit) MUST be supportable.
- **FR-6.** Notices/flows MUST be available in **English and major Indian languages** (Eighth Schedule languages as feasible).

## 6. Non-Functional Requirements

- **Privacy & Compliance** — DPDP Act 2023 §§ 5–9 (notice/consent, children), 8 (fiduciary duties incl. breach + erasure), 10 (SDF), 11–14 (rights), + DPDP Rules; alignment with S08.
- **Accessibility** — WCAG 2.1 AA; multilingual.
- **Observability** — `in_rights_requests_total`, `in_dpb_notifications_total`, `in_child_tracking_blocked_total` (must be 0).
- **Maintainability** — Regime row + strict child profile; multilingual notices.
- **Security/Reliability/Performance** — engine defaults; child-tracking block fail-safe.
- **Internationalization** — English + Indian languages.
- **Backward compatibility** — additive; India child profile is the strictest and overrides looser defaults.

## 7. Acceptance Criteria

- **AC-1.** *Given* an Indian user under 18, *when* they register, *then* verifiable parental consent is required and behavioural tracking/targeted ads are disabled and code-gated.
- **AC-2.** *Given* an Indian principal requests erasure + nomination, *when* S01 processes them, *then* both are supported.
- **AC-3.** *Given* a breach, *when* S03 assesses it, *then* the Data Protection Board + principals are notified.
- **AC-4.** *Given* a grievance is filed, *when* it's submitted, *then* the grievance-redressal workflow tracks it to resolution with the grievance officer.
- **AC-5.** *Given* a Consent Manager integration, *when* consent is captured/withdrawn there, *then* it reconciles with the S04 ledger.
- **AC-6.** *Given* an Indian-language locale, *when* notices render, *then* they appear in that language.

## 8. Data Model

New migration `373_india_dpdp.sql` (+ `.down.sql`): regime row `IN_DPDP` (child_min_age=18, strict) in the registry; grievance-redressal reuses S01 `rights_requests` with `request_type` extended by a `grievance` classification and a nominee field:

```sql
ALTER TABLE compliance.rights_requests
  ADD COLUMN nominee JSONB;                     -- DPDP nomination details

INSERT INTO compliance.privacy_officers (regime, name, contact)
  VALUES ('in_dpdp_grievance', '<configured>', '<configured>');
```

Reuses S03/S04/S08 tables; India seeds `minor_policies` with `age_band` treated as minor up to 18.

## 9. API Surface

Reuses S01/S03/S04 with India regime codes. New:

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/compliance/in/grievance` | data principal | File grievance |
| `GET` | `/api/v1/compliance/in/dpb-register` | `privacy:dpo` | Data Protection Board notice register |
| `POST` | `/api/v1/compliance/consent/consent-manager-sync` | integration | Reconcile Consent Manager events (S04) |

## 10. UI / UX

- India regime toggle; clear itemised consent notice; grievance-redressal form + status; nominee capture; multilingual notices. Strict child experience: no tracking controls even present for under-18. i18n: English + Indian languages.
- Accessibility: WCAG 2.1 AA, multilingual.

## 11. AI / ML Considerations

Under-18 users get the strictest AI posture (no behavioural monitoring, no ad targeting, no training on their data) — S08 enforced for India up to 18. Offshore AI providers subject to DPDP transfer restrictions via S07 (government may restrict certain countries).

## 12. Integration Points

- Registry, S01 (grievance/nomination), S03, S04 (Consent Manager), S07, S08, l10n (Indian languages), S14 privacy-officer table.

## 13. Dependencies & Sequencing

- Must ship after: S01, S03, S04, S07, S08.
- Must ship before: India GA.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Under-18 VPC + no-tracking hard to enforce at scale | M | H | S08 strict child profile as the India default; zero-tolerance tracking metric |
| DPDP Rules still finalising specifics | M | M | Configurable timeframes/thresholds; track Rules |
| Consent Manager ecosystem integration immature | M | M | Internal S04 ledger authoritative; CM as sync path |
| Transfer restrictions to specific countries | M | M | S07 residency/allowlist controls |

## 15. Rollout Plan

- Flag `dpdp_enabled`. Phase 1: IN_DPDP row + strict child profile + clear-notice consent + multilingual. Phase 2: rights (erasure/nomination) + grievance workflow + DPB breach. Phase 3: Consent Manager sync + transfer controls. Pilot Indian tenant. GA. Rollback: flag off (keep strict child protections on).

## 16. Test Plan

- **Unit** — India rights set; under-18 gating; grievance state machine.
- **Integration** — DPB notification; Consent Manager reconciliation; no-tracking enforcement for under-18.
- **E2E** — Indian under-18 registration with VPC; grievance filed → resolved.
- **Security/Accessibility/Performance** — engine defaults; multilingual axe.
- **Manual** — DPDP checklist review.

## 17. Documentation & Training

- India compliance runbook; grievance-officer duties; multilingual notice management; SDF obligations checklist.

## 18. Open Questions

1. Which Consent Managers to integrate first, and is integration required at launch?
2. Data-localisation/transfer-restriction posture as Rules finalise?
3. Do we meet the Significant Data Fiduciary threshold (extra DPO/DPIA/audit duties)?
4. Which Indian languages to prioritise for notices?

## 19. References

- Digital Personal Data Protection Act, 2023 (§§ 5–14) + DPDP Rules
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S03](S03-global-breach-notification-incident-response.md), [S04](S04-unified-consent-preference-ledger.md), [S07](S07-cross-border-transfer-subprocessor-governance.md), [S08](S08-childrens-privacy-age-assurance-design-codes.md)

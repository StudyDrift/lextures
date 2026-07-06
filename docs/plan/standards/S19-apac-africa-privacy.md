# S19 — APAC & Africa (Japan APPI · Korea PIPA · South Africa POPIA · Nigeria NDPA)

> Implementation plan. New coverage; configures the cross-cutting engines for additional APAC + African regimes. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S19 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MINOR (MAJOR per-country on entry) |
| **Markets** | Japan · South Korea · South Africa · Nigeria (+ template for further countries) |
| **Status (today)** | MISSING — no coverage for these regimes |
| **Estimated effort** | M (2–4w for all four as registry configs) |
| **Owner (proposed)** | Compliance Lead |
| **Depends on** | S01, S03, S04, S07 |
| **Unblocks** | APAC + African market entry; a repeatable "add a country" pattern |

---

## 1. Problem Statement

Beyond the headline regimes, several countries with real education markets have mature or fast-maturing privacy laws that a global platform must honour: **Japan's APPI** (with its data-subject rights, cross-border transfer rules, and breach reporting to the PPC), **South Korea's PIPA** (one of Asia's strictest — consent-heavy, unique-identifier limits, breach reporting to the PIPC, strong child rules), **South Africa's POPIA** (eight conditions, an Information Officer requirement, Information Regulator breach notification), and **Nigeria's NDPA** (data-controller registration, DPO, breach notice). Rather than a bespoke build each time, this plan proves the "add a country as configuration" pattern the cross-cutting engines were designed for, delivering these four and a template for the next.

## 2. Goals

- Add **Japan (APPI)**, **South Korea (PIPA)**, **South Africa (POPIA)**, and **Nigeria (NDPA)** as regime configurations over S01/S03/S04/S07.
- Cover each regime's **rights**, **consent** norms, **breach notification** (PPC / PIPC / Information Regulator / NITDA), and **transfer** rules.
- Provide each regime's **officer role** (Information Officer for POPIA, DPO for NDPA, etc.) and **breach register**.
- Localise notices (ja, ko; en for ZA/NG) and handle child rules (esp. Korea's strict under-14).
- Establish a documented **"onboard a new country"** runbook so future regimes are days, not months.

## 3. Non-Goals

- Rebuilding engines; these are configs.
- Exhaustive coverage of every remaining country in one pass (this delivers four + a template).

## 4. Personas & User Stories

- **As an institution in JP/KR/ZA/NG**, I want local-law-aligned privacy handling so that we can procure lawfully.
- **As a data subject in these countries**, I want my local rights and breach notice honoured so that I'm protected.
- **As a compliance officer**, I want a repeatable process to add a country so that expansion isn't a rebuild.
- **As a Korean guardian**, I want strict under-14 protections so that PIPA's child rules are met.

## 5. Functional Requirements

- **FR-1.** Each regime MUST be a **registry row** encoding rights, SLA, consent norms, breach authority + timeline, transfer rules, officer role, and child age.
- **FR-2.** S01 MUST resolve local rights (JP APPI access/correction/cessation; KR PIPA access/correction/deletion/suspension; ZA POPIA access/correction/deletion; NG NDPA access/rectification/erasure).
- **FR-3.** S03 MUST notify the correct authority (**PPC** JP, **PIPC** KR, **Information Regulator** ZA, **NITDA** NG) and subjects, per each timeline.
- **FR-4.** S04 MUST reflect consent norms (KR's separate/explicit consent + unique-identifier limits; POPIA conditions; APPI notice).
- **FR-5.** S07 MUST enforce each regime's **cross-border transfer** rules; officer roles registered via the shared officer table.
- **FR-6.** Notices MUST be localised (**ja**, **ko**; **en** for ZA/NG) and child rules applied (S08; KR under-14 strict).
- **FR-7.** A documented **country-onboarding runbook** MUST exist and be validated by adding at least one of the four via it.

## 6. Non-Functional Requirements

- **Privacy & Compliance** — Japan APPI (as amended); Korea PIPA; South Africa POPIA (8 conditions, Info Officer); Nigeria NDPA 2023 + NDPR.
- **Accessibility** — WCAG 2.1 AA; ja/ko localisation.
- **Observability** — `apac_africa_rights_requests_total{country}`, `apac_africa_breach_notifications_total{authority}`.
- **Maintainability** — Pure registry configs; no per-country code branches.
- **Security/Reliability/Performance** — engine defaults.
- **Internationalization** — ja, ko, en.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Korean subject requests suspension of processing, *when* S01 resolves it, *then* PIPA's suspension right is honoured with its timeline.
- **AC-2.** *Given* a POPIA breach, *when* S03 assesses it, *then* the Information Regulator + data subjects are notified and the Information Officer owns the case.
- **AC-3.** *Given* a Japanese transfer offshore, *when* configured, *then* APPI transfer rules are enforced via S07.
- **AC-4.** *Given* a Korean under-14, *when* they register, *then* strict child consent applies (S08).
- **AC-5.** *Given* Nigeria, *when* enabled, *then* a DPO is registered and NITDA breach notification is wired.
- **AC-6.** *Given* the onboarding runbook, *when* a compliance analyst follows it, *then* a new country is added as config without a code deploy.

## 8. Data Model

New migration `375_apac_africa_privacy.sql` (+ `.down.sql`): regime rows `JP_APPI`, `KR_PIPA`, `ZA_POPIA`, `NG_NDPA` in the registry; officers via `compliance.privacy_officers` (`za_popia_info_officer`, `ng_ndpa_dpo`). Reuses S01/S03/S04/S07 tables. Transfer enum extended as needed per regime.

## 9. API Surface

Reuses S01/S03/S04/S07 with the four regime codes. New:

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/officers` | `privacy:admin` | Register Info Officer / DPO per regime (shared) |
| `GET` | `/api/v1/compliance/breach-register` | `privacy:dpo` | Per-authority breach register (shared, filtered) |

## 10. UI / UX

- Country regime toggles; officer registration; localized notices (ja/ko/en); child-strict experience for KR. Reuses S01/S04 subject surfaces. i18n `ja`, `ko`, `en`.
- Accessibility: WCAG 2.1 AA.

## 11. AI / ML Considerations

Offshore AI as cross-border transfer handled per each regime via S07; KR/JP automated-decision transparency aligns with S12/S13 where applicable.

## 12. Integration Points

- Registry, S01, S03, S04, S07, S08, l10n (ja/ko), shared officer table (S14).

## 13. Dependencies & Sequencing

- Must ship after: S01, S03, S04, S07.
- Must ship before: entry into any of these markets.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Korea PIPA stricter than the generic config supports | M | M | KR-specific fields (separate consent, ID limits); legal review |
| Localisation (ja/ko) incomplete | M | M | Gate per-country GA on notice localisation |
| "Config-only" assumption breaks for a nuance | M | M | Escape-hatch strategy hook; document exceptions |

## 15. Rollout Plan

- Flag `apac_africa_privacy_enabled` (per-country sub-flags). Phase 1: registry rows + rights/consent + localisation. Phase 2: breach authorities + officers. Phase 3: transfers + onboarding-runbook validation. Pilot per country on entry. GA per country. Rollback: per-country flag off.

## 16. Test Plan

- **Unit** — each regime's rights/consent/breach config; KR child strictness.
- **Integration** — POPIA breach → Info Regulator; APPI transfer; onboarding runbook adds a country.
- **E2E** — KR suspension request; ja/ko notice render.
- **Security/Accessibility/Performance** — engine defaults; ja/ko axe.
- **Manual** — per-country checklist review.

## 17. Documentation & Training

- **Country-onboarding runbook** (the reusable pattern) — the key deliverable.
- Per-regime officer duties; localisation notes.

## 18. Open Questions

1. Priority order among JP/KR/ZA/NG based on go-to-market?
2. Korea adequacy/representative requirements for our setup?
3. Which additional countries queue next behind these four?

## 19. References

- Japan APPI; South Korea PIPA; South Africa POPIA; Nigeria NDPA 2023 / NDPR
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S03](S03-global-breach-notification-incident-response.md), [S04](S04-unified-consent-preference-ledger.md), [S07](S07-cross-border-transfer-subprocessor-governance.md)

# S08 — Children's Privacy, Age Assurance & Design Codes

> Implementation plan. Hardens: [10.2 COPPA](../../completed/10-compliance-privacy-security/10.2-coppa-workflow.md), [10.1 FERPA](../../completed/10-compliance-privacy-security/10.1-ferpa-workflow.md), plus [S04 consent](S04-unified-consent-preference-ledger.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S08 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | K12 (Global) · SL (minors) |
| **Status (today)** | PARTIAL — COPPA VPC flow exists (`server/internal/service/coppa`); no robust age assurance, no adaptation to the 2025 amended COPPA rule, and no conformance to children's *design* codes (UK AADC, Ireland Fundamentals, US state kids' codes) |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Compliance + Product + Web/Mobile |
| **Depends on** | 10.2, S04 (consent ledger), S06 (children's DPIA), 10.4 age flags |
| **Unblocks** | S11 (state kids' codes), S13 (AI + minors) |

---

## 1. Problem Statement

Children are the platform's core users and the most heavily protected data subjects everywhere. COPPA governs under-13s in the US (with a materially amended rule effective 2025 tightening consent, retention, and data-security), GDPR sets a digital-consent age of 13–16 by member state, the UK Age Appropriate Design Code and Ireland's "Fundamentals" impose **design** duties (high-privacy defaults, no nudging, data minimisation, no profiling of children by default), and a wave of US **state kids' codes** (starting with California's AADC) extends this. Our current COPPA flow handles parental consent but does not do reliable **age assurance**, has not been updated for the 2025 rule, and there is no evidence of **age-appropriate design** — high-privacy defaults, restraint on profiling/notifications for minors, or a children's DPIA. Each is a first-tier enforcement priority (COPPA and AADC carry the largest ed-tech fines), so this is squarely a blocker.

## 2. Goals

- **Age assurance** proportionate to risk: establish a user's age band (under-13 / 13–15 / 16–17 / adult) without over-collecting, and gate features accordingly.
- **COPPA 2025-rule conformance**: verifiable parental consent (VPC) refresh, separate consent for third-party/AI sharing, data-retention limits, and heightened data-security representations.
- **Children's design-code conformance**: high-privacy defaults for minors, no behavioural profiling/targeted nudges by default, minimised data, and a Children's DPIA (via S06).
- **Age-band-driven policy**: a single mechanism the rest of the app consults to adjust defaults, AI use, notifications, and data sharing for minors.
- Parent/guardian transparency and control aligned with FERPA (S09) and the consent ledger (S04).

## 3. Non-Goals

- Building a biometric age-estimation vendor in-house (we integrate/attest, and prefer school-context assurance).
- FERPA record-access mechanics (S09) — this focuses on age + design, not education-record rights.
- The consent *ledger* itself (S04) — this defines the child-specific consent rules recorded there.

## 4. Personas & User Stories

- **As a parent**, I want to give verifiable consent and control my under-13's data sharing so that I stay in charge.
- **As a district**, I want school-authorised consent (the ed-tech COPPA pathway) to work so that we can deploy without per-parent friction where the law allows.
- **As a 15-year-old**, I want privacy-protective defaults and no behavioural profiling so that the product is safe by design.
- **As a product manager**, I want one age-band signal to adjust feature defaults so that we don't scatter age logic across the codebase.
- **As a DPO**, I want a Children's DPIA and evidence of age-appropriate design so that we can prove AADC conformance.

## 5. Functional Requirements

- **FR-1.** The system MUST establish an **age band** per user via layered signals (school-provided grade/DOB, self-declared DOB with neutral age-gate, or assurance vendor), storing the band and its assurance level — not raw DOB where avoidable.
- **FR-2.** For under-13 (COPPA) users outside the school-consent pathway, the system MUST obtain **VPC** and record it in the consent ledger (S04); AI/third-party sharing requires **separate** consent (2025 rule).
- **FR-3.** The system MUST support the **school-authorised consent** pathway (school consents on parents' behalf for educational use) with the required limitations documented.
- **FR-4.** The system MUST apply **high-privacy defaults** for all minor age bands: profiling/behavioural targeting off, geolocation precise-off, public visibility off, engagement-nudge notifications minimised, data collection minimised.
- **FR-5.** The system MUST NOT use a minor's data for **behavioural profiling, targeted advertising, or engagement optimisation** without a lawful, documented basis (default: never).
- **FR-6.** COPPA data-retention limits MUST be enforced via S02 (retain child data only as long as necessary; auto-dispose).
- **FR-7.** The system MUST complete and maintain a **Children's DPIA** (S06) covering all features minors use, including AI features (S13).
- **FR-8.** Parental controls MUST allow review, deletion (S01), and consent withdrawal (S04) for the child's data.
- **FR-9.** Age-band changes (e.g. a user turns 13/18) MUST re-evaluate defaults and consents and be audit-logged.

## 6. Non-Functional Requirements

- **Performance** — Age-band lookup is cached and on the hot path for default-gating; < 5 ms.
- **Security** — Assurance artifacts (if any DOB/ID) are minimised, encrypted, and disposed after verification (S02); parental-identity data isolated.
- **Privacy & Compliance** — COPPA (16 CFR Part 312, 2025 amendments); GDPR Art 8; UK AADC (15 standards); Ireland "Fundamentals"; CA AADC + successor state codes; FERPA alignment.
- **Accessibility** — Age-gates and parental flows WCAG 2.1 AA; age-appropriate, plain-language copy.
- **Scalability** — Whole-district onboarding via the school-consent pathway (bulk).
- **Reliability** — Fail-safe: unknown age ⇒ treat as a minor (most protective) until assured.
- **Observability** — `minors_by_age_band`, `vpc_pending_total`, `minor_profiling_blocked_total`; alert if any profiling occurs on a minor band.
- **Maintainability** — Age policy in `server/internal/service/ageassurance/`; a single `AgeBand`/`MinorPolicy` consulted app-wide (existing age-appropriate UI work in the mobile plan is a consumer).
- **Internationalization** — Consent age varies by country (GDPR 13–16); the digital-consent-age table is data, localised copy.
- **Backward compatibility** — Existing COPPA records migrate into the ledger; default existing minors to protective settings with parent re-confirmation.

## 7. Acceptance Criteria

- **AC-1.** *Given* a self-registering user who declares an under-13 DOB, *when* they proceed, *then* a neutral age-gate routes them to VPC and no account is fully activated until consent is recorded.
- **AC-2.** *Given* a district using the school-consent pathway, *when* students are provisioned, *then* educational-use processing is permitted without per-parent VPC, and non-educational/AI sharing still requires separate consent.
- **AC-3.** *Given* any minor age band, *when* their profile loads, *then* profiling, targeted content, precise geolocation, and public visibility are off by default and cannot be silently enabled.
- **AC-4.** *Given* an attempt to use a minor's data for engagement optimisation, *when* it is evaluated, *then* it is blocked and `minor_profiling_blocked_total` increments.
- **AC-5.** *Given* a user turns 18, *when* the birthday job runs, *then* their band updates, adult defaults become available (opt-in), and the change is logged.
- **AC-6.** *Given* a DPO reviews compliance, *when* they open the Children's DPIA, *then* it covers every minor-facing feature including AI, with high-privacy-default evidence.

## 8. Data Model

New migration `364_age_assurance.sql` (+ `.down.sql`):

```sql
ALTER TABLE "user".users
  ADD COLUMN age_band TEXT NOT NULL DEFAULT 'unknown'
    CHECK (age_band IN ('unknown','under_13','13_15','16_17','adult')),
  ADD COLUMN age_assurance_level TEXT NOT NULL DEFAULT 'none'
    CHECK (age_assurance_level IN ('none','self_declared','school_provided','verified')),
  ADD COLUMN age_assured_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS compliance.minor_policies (
  age_band      TEXT PRIMARY KEY,
  profiling_allowed BOOLEAN NOT NULL DEFAULT FALSE,
  targeted_content BOOLEAN NOT NULL DEFAULT FALSE,
  precise_geo   BOOLEAN NOT NULL DEFAULT FALSE,
  public_visibility BOOLEAN NOT NULL DEFAULT FALSE,
  engagement_nudges BOOLEAN NOT NULL DEFAULT FALSE,
  requires_vpc  BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS compliance.digital_consent_ages (
  country_code  TEXT PRIMARY KEY,             -- GDPR member-state variance 13–16
  consent_age   INT NOT NULL
);

-- school-authorised consent pathway record
CREATE TABLE IF NOT EXISTS compliance.school_consent_authorizations (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES org.organizations(id),
  authorized_by UUID NOT NULL REFERENCES "user".users(id),
  scope         TEXT NOT NULL,                -- 'educational_use'
  limitations   TEXT,
  authorized_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Backfill: seed `minor_policies` (all minor bands high-privacy), `digital_consent_ages` per member state; compute `age_band` from existing DOB/grade where available, else `unknown` (protective).

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/compliance/age/declare` | self | Neutral age-gate declaration |
| `GET` | `/api/v1/compliance/age/band` | self / internal | Current band + assurance level |
| `POST` | `/api/v1/compliance/coppa/vpc` | parent | Verifiable parental consent (into S04 ledger) |
| `POST` | `/api/v1/compliance/coppa/school-consent` | school admin | School-authorised pathway |
| `GET/PUT` | `/api/v1/compliance/minor-policies` | `privacy:child_admin` | Manage band defaults |
| internal | `MinorPolicy` gate | — | Consulted app-wide for defaults/feature gating |

## 10. UI / UX

- **Neutral age-gate:** non-leading DOB entry (no "must be 13+" nudge to lie).
- **VPC flow:** parent verification per an approved COPPA method; consent + separate AI/third-party toggle recorded in the preference center (S04).
- **Parent dashboard:** review/delete child data, withdraw consent, see defaults.
- **Minor experience:** high-privacy defaults visibly on; profiling/targeting controls absent (not just off); minimal nudges (aligns with mobile age-appropriate UI plan).
- States: pending-VPC (limited account), assured, band-transition prompt at 13/18.
- Accessibility + age-appropriate language; i18n keys `age.*`, `coppa.*`.

## 11. AI / ML Considerations

AI features for minors require the **most restrictive** posture: no training on minors' content, no profiling, separate parental consent for any AI processing beyond core educational use, and inclusion in the Children's DPIA (S06) and AI Act high-risk analysis (S13). The AI gateway MUST consult `MinorPolicy` and block non-consented AI processing for minor bands.

## 12. Integration Points

- `server/internal/service/ageassurance/` (new) + `server/internal/service/coppa` (VPC into S04); `MinorPolicy` consumed by feed/recommendations/notifications/`aigateway`.
- Mobile "age-appropriate UI mode" (M10.4, already shipped) becomes a consumer of the age band.
- S01 (parental deletion), S02 (child retention limits), S04 (consent), S06 (Children's DPIA), S09 (FERPA), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: S04 (ledger), S02 (retention), S06 (DPIA).
- Must ship before: S11 (state kids' codes lean on this), S13 (AI + minors).
- Shared infra: consent ledger, retention engine, feature-flag defaults.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Age assurance over-collects (ID/biometrics) creating new risk | M | H | Prefer school-context assurance; minimise + dispose artifacts (S02); proportionate to feature risk |
| Kids lie at the age-gate | H | M | Neutral gate; school-provided age as authoritative; protective default for `unknown` |
| Design-code duties scattered → some feature profiles a minor | M | H | Single `MinorPolicy` gate; profiling blocked at gateway; monitored metric = 0 |
| 2025 COPPA rule deltas missed | M | H | Explicit checklist mapping each rule change to a control; legal sign-off |

## 15. Rollout Plan

- Flag `age_assurance_enabled`. Phase 1: age band + `MinorPolicy` + protective defaults for `unknown`/minor bands. Phase 2: VPC refresh into S04 + school-consent pathway + 2025-rule deltas. Phase 3: design-code defaults enforced app-wide + Children's DPIA. Pilot with one K12 district. GA globally. Rollback: keep protective defaults (safe direction); flag off reverts consent-flow changes only.

## 16. Test Plan

- **Unit** — band derivation from signals; digital-consent-age lookup; `MinorPolicy` gating; birthday transitions.
- **Integration** — VPC recorded in ledger with separate AI consent; school-consent pathway permits educational-only; profiling blocked for minors.
- **E2E** — under-13 self-register → age-gate → VPC → limited then full account; turn-18 transition.
- **Security** — assurance-artifact minimisation + disposal; parental-flow authz.
- **Accessibility** — axe + reading-level check on child/parent copy.
- **Performance** — band gate < 5 ms on hot path.
- **Manual** — AADC self-assessment against the 15 standards; COPPA 2025-rule checklist.

## 17. Documentation & Training

- Help center: parent guide to consent + controls; student-facing plain-language privacy.
- AADC/Fundamentals conformance statement (public trust surface).
- Runbook: school-authorised consent onboarding.

## 18. Open Questions

1. Which age-assurance methods do we accept per jurisdiction (school-provided vs. vendor vs. self-declared) and at what feature-risk thresholds?
2. Do we localise the digital-consent age strictly per member state, or apply the highest (16) EU-wide for simplicity?
3. How do we treat mixed-age classes where some students are 12 and some 13+?
4. Retention of VPC artifacts — minimum needed to prove consent vs. privacy of parent data.

## 19. References

- `server/internal/service/{coppa,aigateway}`, mobile age-appropriate UI plan (M10.4), `server/internal/service/coppa/ai_blocked.go`
- COPPA 16 CFR Part 312 (2025 amendments); GDPR Art 8; UK Age Appropriate Design Code; Ireland "Fundamentals"; CA AADC (AB 2273)
- Related: [S02](S02-data-retention-deletion-engine.md), [S04](S04-unified-consent-preference-ledger.md), [S06](S06-dpia-pia-algorithmic-impact.md), [S09](S09-ferpa-hardening.md), [S13](S13-eu-ai-act-high-risk.md)

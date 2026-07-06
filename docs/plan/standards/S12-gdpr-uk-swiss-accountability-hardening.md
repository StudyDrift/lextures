# S12 — GDPR / UK GDPR / Swiss FADP Accountability Hardening

> Implementation plan. Hardens: [10.3 GDPR/UK-GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S12 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | EU/UK · CH |
| **Status (today)** | PARTIAL — 10.3 shipped DSAR export, erasure, a consent module, a DPA artifact, and a static RoPA (`server/internal/service/gdpr`, migration `182_gdpr`). The *accountability* spine — Art 22 automated-decision safeguards, DPO function, lawful-basis-per-purpose enforcement, transparency completeness — is thin, and Switzerland is uncovered |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | DPO + Backend |
| **Depends on** | 10.3, S01, S04, S05, S06, S07 |
| **Unblocks** | EU/UK enterprise + institutional sales, S13 (AI Act builds on GDPR base) |

---

## 1. Problem Statement

The GDPR baseline (10.3) delivered the *rights* but not the *accountability principle* (Art 5(2)) that regulators actually audit: demonstrable lawful basis for **each** processing purpose, enforced **Article 22** safeguards on automated decisions with legal/significant effects (grade decisions, proctoring flags, at-risk interventions), a defined **DPO** function and contact, complete and layered **transparency** (Arts 13–14), and coverage of **Switzerland's revised FADP** which mirrors GDPR but is a distinct regime EU coverage does not satisfy. Without these, our GDPR posture is "we can respond to a DSAR" rather than "we can prove we process lawfully" — the difference between passing and failing a supervisory-authority audit, and a barrier to EU institutional deals.

## 2. Goals

- Enforce **lawful-basis-per-purpose** at processing time using the S04 ledger + S05 RoPA (no purpose runs without a valid basis).
- Implement **Article 22** safeguards: identify decisions producing legal/significant effects, ensure a lawful ground, and provide **human review, explanation, and contestation**.
- Formalise the **DPO** function: contact endpoint, internal escalation, and record-keeping obligations.
- Complete **transparency**: layered privacy notices (Arts 13–14) generated from the RoPA/purposes, including recipients (S07) and retention (S02).
- Add **Swiss FADP** as a first-class regime (rights, records, controller/representative, transfer specifics) sharing the engines.

## 3. Non-Goals

- Cross-border transfer mechanics (S07 owns SCCs/DPF/TIAs) — referenced here.
- The consent banner/ledger build (S04) and RoPA build (S05) — this consumes them.
- The AI Act (S13) — Art 22 here is the GDPR overlap, not the AI Act conformity assessment.

## 4. Personas & User Stories

- **As an EU data subject**, I want to contest an automated grade/flag and get human review with an explanation so that Art 22 is real for me.
- **As a DPO**, I want lawful basis enforced per purpose so that I can attest to Art 5(2) accountability.
- **As an EU institutional customer**, I want a complete, layered privacy notice and a named DPO contact so that our own compliance is satisfied.
- **As a Swiss controller/customer**, I want FADP-specific handling so that Swiss deployments are lawful.
- **As a supervisory authority**, I want to see records demonstrating lawful, transparent processing so that an audit resolves quickly.

## 5. Functional Requirements

- **FR-1.** Every processing purpose (S04/S05) MUST carry a lawful basis; the runtime MUST **refuse** processing for a purpose lacking a valid basis for that subject, logging the refusal.
- **FR-2.** The system MUST maintain a registry of **automated decisions with legal/significant effect** (e.g. final AI grade, proctoring integrity flag, at-risk escalation) and, for each: ensure an Art 22 lawful ground, and expose **human-review + explanation + contestation** via S01.
- **FR-3.** The system MUST provide a **DPO contact** channel and route DSARs/complaints/breach queries to it (S01/S03), keeping the required records.
- **FR-4.** The system MUST generate **layered privacy notices** (Arts 13–14) from RoPA/purposes — identity, purposes, bases, recipients (S07), retention (S02), rights, transfers, and automated-decision info.
- **FR-5.** The system MUST support **Switzerland (revFADP)** as a regime: subject rights, processing records, EU/Swiss representative info, and FADP transfer rules, reusing S01/S05/S07.
- **FR-6.** The system MUST handle **legitimate-interest** purposes with a stored **LIA (balancing test)** and an easy **objection** path (Art 21) wired to S01.
- **FR-7.** The system MUST honour **restriction** (Art 18) and **portability** (Art 20, structured/machine-readable) via S01.
- **FR-8.** All accountability artifacts (LIAs, notices, DPO records) MUST be versioned and audit-logged.

## 6. Non-Functional Requirements

- **Performance** — Lawful-basis check on the hot path via S04 cache (< 5 ms); Art 22 review flows are async.
- **Security** — Automated-decision registry + LIAs gated by `privacy:dpo`; contestation records protect subject data.
- **Privacy & Compliance** — GDPR Arts 5, 6, 9, 12–22, 30, 35; UK GDPR + DPA 2018; Swiss revFADP; EDPB guidance on Art 22 and transparency.
- **Accessibility** — Notices + contestation UI WCAG 2.1 AA, localised across EU languages incl. RTL where applicable.
- **Scalability** — EU/UK/CH subject volumes; notice generation cached.
- **Reliability** — Basis enforcement fails closed; Art 22 contestation cannot be silently dropped (tracked to resolution in S01).
- **Observability** — `processing_refused_no_basis_total{purpose}`, `art22_contestations_total`, `dpo_contacts_total`; alert on refusal spikes (mis-config) or unresolved contestations.
- **Maintainability** — Extends `server/internal/service/gdpr`; Swiss as a regime variant, not a fork.
- **Internationalization** — Full EU-language notice set.
- **Backward compatibility** — Existing 10.3 DSAR/erasure endpoints continue; basis enforcement rolls out per purpose behind flags.

## 7. Acceptance Criteria

- **AC-1.** *Given* a purpose whose basis is consent and a subject who withdrew it, *when* that processing is invoked, *then* it is refused, the refusal is logged, and any non-AI fallback runs instead.
- **AC-2.** *Given* a final AI grade (automated decision with significant effect), *when* a student contests it, *then* a human review is created (S01), an explanation is provided, and the outcome is recorded.
- **AC-3.** *Given* an EU subject views the privacy notice, *when* it renders, *then* it contains all Art 13–14 elements populated from the live RoPA (recipients from S07, retention from S02).
- **AC-4.** *Given* a Swiss deployment, *when* a subject exercises rights, *then* FADP rules/timelines apply and the Swiss representative info is shown.
- **AC-5.** *Given* a legitimate-interest purpose, *when* a subject objects (Art 21), *then* processing stops unless a documented overriding LIA applies, and the decision is recorded.
- **AC-6.** *Given* a supervisory-authority audit, *when* the DPO exports accountability evidence, *then* bases, LIAs, notices, Art 22 registry, and RoPA are all produced.

## 8. Data Model

New migration `368_gdpr_accountability.sql` (+ `.down.sql`), extending `182_gdpr`:

```sql
CREATE TABLE IF NOT EXISTS compliance.automated_decisions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key           TEXT NOT NULL UNIQUE,          -- 'final_ai_grade','proctoring_flag','at_risk_escalation'
  description   TEXT NOT NULL,
  significant_effect BOOLEAN NOT NULL DEFAULT TRUE,
  lawful_ground TEXT NOT NULL,                 -- 'explicit_consent','contract','authorised_by_law'
  human_review  BOOLEAN NOT NULL DEFAULT TRUE,
  explanation_template TEXT,
  aia_assessment_id UUID                        -- S06 link
);

CREATE TABLE IF NOT EXISTS compliance.legitimate_interest_assessments (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  purpose_key   TEXT NOT NULL,                 -- S04 purpose
  interest      TEXT NOT NULL,
  necessity     TEXT NOT NULL,
  balancing     TEXT NOT NULL,                 -- outcome of the balancing test
  outcome       TEXT NOT NULL CHECK (outcome IN ('passes','fails')),
  version       INT NOT NULL DEFAULT 1,
  reviewed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.privacy_notices (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  regime        TEXT NOT NULL,                 -- 'eu_gdpr','uk_gdpr','ch_fadp'
  locale        TEXT NOT NULL,
  version       INT NOT NULL,
  body          TEXT NOT NULL,                 -- generated from RoPA/purposes
  published_at  TIMESTAMPTZ
);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/gdpr/automated-decisions` | `privacy:dpo` | Manage Art 22 registry |
| `POST` | `/api/v1/compliance/gdpr/contest` | subject | Contest an automated decision (→ S01 human review) |
| `GET/PUT` | `/api/v1/compliance/gdpr/lia` | `privacy:dpo` | Legitimate-interest assessments |
| `GET` | `/api/v1/compliance/gdpr/notice` | public | Layered privacy notice (regime + locale) |
| `POST` | `/api/v1/compliance/gdpr/dpo-contact` | subject | Contact DPO |
| internal | basis-enforcement middleware | — | Refuse purpose without valid basis |

## 10. UI / UX

- **Privacy notice pages** (extend `privacy-centre-page.tsx`): layered, per-regime/locale, generated content.
- **Contestation UI:** "Request human review of this decision," explanation shown, status tracked (S01).
- **DPO console:** Art 22 registry, LIAs, notice versions, accountability export.
- **Objection controls** in the preference center (S04) for legitimate-interest purposes.
- States: processing-blocked (no basis) explained to admins; contestation pending; regime-specific notice selection.
- Accessibility: multilingual, plain-language; i18n keys `gdpr.*`, `fadp.*`.

## 11. AI / ML Considerations

Art 22 is the GDPR hook for our AI: every AI decision with significant effect must have a lawful ground, human oversight, and explanation — reusing S06 AIAs and feeding S13. Explanations must be meaningful (logic + significance + consequences), not just "a model decided." Basis enforcement (FR-1) blocks AI processing when consent is the basis and it's absent.

## 12. Integration Points

- `server/internal/service/gdpr` (extend) + repo `server/internal/repos/gdpr`; S01 (rights/contestation), S04 (basis/consent), S05 (notice source), S06 (AIA), S07 (recipients/transfers), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: S04, S05, S06, S07.
- Must ship before: S13 (AI Act layers on GDPR accountability), EU/UK enterprise GA.
- Shared infra: consent ledger, RoPA, rights engine.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Basis enforcement misconfig blocks legitimate processing | M | H | Staged per-purpose rollout; refusal metric + alert; non-AI fallbacks |
| Art 22 decisions missed in the registry | M | H | Derive candidates from S06 AIAs; DPO review; default significant-effect = true |
| Notice drifts from actual processing | M | M | Generate from RoPA; regenerate on RoPA change |
| Swiss nuances treated as identical to GDPR | L | M | Distinct regime row + representative + transfer rules; legal review |

## 15. Rollout Plan

- Flag `gdpr_accountability_enabled`. Phase 1: automated-decision registry + LIAs + generated notices. Phase 2: Art 22 contestation via S01 + DPO channel. Phase 3: basis-enforcement middleware per purpose; add Swiss regime. GA EU/UK/CH. Rollback: enforcement to warn-only; artifacts remain.

## 16. Test Plan

- **Unit** — basis-enforcement decisions; LIA outcome gating; notice generation from RoPA; regime selection.
- **Integration** — contest → S01 human review created; consent-withdrawal blocks purpose; Swiss timeline differs.
- **E2E** — student contests AI grade → human review + explanation; EU notice renders complete.
- **Security** — authz on DPO console; contestation data protection.
- **Accessibility** — axe + multilingual notice rendering (incl. RTL).
- **Performance** — basis check < 5 ms.
- **Manual** — DPO mock supervisory-authority audit using the export.

## 17. Documentation & Training

- DPO handbook: Art 22 registry, LIAs, accountability evidence.
- Help center: "Request human review of an automated decision."
- Public: DPO contact + layered notices per regime.

## 18. Open Questions

1. Which decisions truly meet the Art 22 "legal or similarly significant effect" bar (final grades yes; recommendations?)?
2. Do we appoint an EU/UK Art 27 representative, and is that surfaced in-product?
3. Explanation depth for third-party-model decisions we can't fully introspect?
4. FADP: do we need a Swiss-based representative for certain deployments?

## 19. References

- `server/internal/service/gdpr`, `server/internal/repos/gdpr`, `server/migrations/182_gdpr.sql`, `clients/web/src/pages/privacy-centre-page.tsx`
- GDPR Arts 5, 6, 9, 12–22, 30, 35; UK GDPR + DPA 2018; Swiss revFADP; EDPB Art 22 & transparency guidelines
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S05](S05-ropa-data-inventory-mapping.md), [S06](S06-dpia-pia-algorithmic-impact.md), [S13](S13-eu-ai-act-high-risk.md)

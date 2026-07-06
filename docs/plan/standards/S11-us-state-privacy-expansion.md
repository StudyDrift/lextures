# S11 — US State Privacy-Law Coverage Expansion

> Implementation plan. Hardens: [10.4 CCPA/CPRA](../../completed/10-compliance-privacy-security/10.4-ccpa-cpra.md), [10.6 state student-privacy (CA/NY/IL)](../../completed/10-compliance-privacy-security/10.6-state-specific-laws.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S11 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | K12 · HE · SL (US) |
| **Status (today)** | PARTIAL — CCPA/CPRA (10.4) and three state student-privacy laws (10.6: CA SOPIPA, NY 2-d, IL SOPPA) exist (`server/internal/service/{ccpa,stateprivacy}`). The 20+ comprehensive consumer-privacy acts and other states' student-data laws are uncovered |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Compliance Lead + Legal |
| **Depends on** | 10.4, 10.6, S01 (rights), S04 (consent/GPC), S06 (assessments) |
| **Unblocks** | US-wide K12/HE/SL sales confidence |

---

## 1. Problem Statement

Since CCPA, most US states have enacted **comprehensive consumer-privacy acts** (Virginia, Colorado, Connecticut, Utah, Texas, Oregon, Montana, Florida, and a growing list), each with its own rights, opt-out signals, data-protection-assessment duties, and sensitive-data rules. In parallel, many states have **student-data-privacy laws** modelled on SOPIPA beyond the three we cover, plus new **children's design codes**. Handling this as one-law-at-a-time (as 10.4/10.6 did) does not scale and leaves gaps that a multi-state customer or a plaintiff's bar will find. We need a **state-law rules engine** driven by the subject's state, layered on the cross-cutting engines (S01/S04/S06), rather than more bespoke per-state code.

## 2. Goals

- A **state-rules registry**: per state, the applicable rights, thresholds, opt-out mechanisms (incl. GPC), sensitive-data categories, assessment triggers, and student-data obligations.
- Drive **S01** (rights), **S04** (opt-out/consent), and **S06** (assessments) from that registry so a new state is largely configuration.
- Extend student-data controls to **all states with SOPIPA-style laws** (no targeted ads, no sale, deletion on district request, security requirements).
- Maintain **prohibition attestations** and per-state DPA addenda (extends 10.5/10.6).
- Honour **universal opt-out signals** (GPC) uniformly across recognising states.

## 3. Non-Goals

- Re-implementing CCPA/CPRA core (10.4) or the three existing student laws (10.6) — this generalises and extends them.
- Federal law (S09/S10) — referenced, not duplicated.
- CIPA filtering technology itself (covered as an attestation here; filtering is a network/school concern).

## 4. Personas & User Stories

- **As a resident of any US state**, I want my state's privacy rights honoured so that I'm protected wherever I live.
- **As a multi-state district**, I want per-state student-data attestations and DPA addenda so that procurement clears in every state.
- **As a compliance officer**, I want to add a newly-effective state law as configuration so that we're compliant on day one.
- **As a data subject with GPC enabled**, I want automatic opt-out in every recognising state so that I don't click per-site.
- **As a DPO**, I want the state assessment duties (profiling/targeted-processing) to auto-trigger a DPIA (S06) so that we don't miss them.

## 5. Functional Requirements

- **FR-1.** A `state_privacy_rules` registry MUST encode, per state: rights offered, applicability thresholds, sensitive-data list, opt-out signals honoured, assessment triggers, and student-data prohibitions.
- **FR-2.** S01 MUST resolve a US subject's rights from the registry (right set + SLA + verification), replacing per-state branching.
- **FR-3.** S04 MUST honour **GPC** in states that recognise it and apply state-specific consent rules for sensitive data.
- **FR-4.** For states with SOPIPA-style student laws, the system MUST enforce: no targeted advertising to students, no sale of student data, deletion on district request, and required security controls — regardless of which state, driven by the registry.
- **FR-5.** Registry-flagged **assessment triggers** (profiling, targeted advertising, sensitive-data processing) MUST auto-create a required S06 assessment.
- **FR-6.** The system MUST generate **per-state attestations** (prohibition of ads/sale) and **DPA addenda** for districts (extends 10.5/10.6).
- **FR-7.** The system MUST provide a **parent/subject disclosure feed** per state where required (extends 10.6), showing data used/shared.
- **FR-8.** Adding/enabling a state MUST be config + review, not a code deploy, and MUST be audit-logged.

## 6. Non-Functional Requirements

- **Performance** — Rule resolution cached per (state, subject-type); < 10 ms.
- **Security** — Registry edits gated by `privacy:state_admin`; attestations signed.
- **Privacy & Compliance** — CCPA/CPRA; VCDPA; CPA; CTDPA; UCPA; TDPSA; OCPA; MTCDPA; FDBR; and the ongoing wave; SOPIPA-family student laws; state kids' codes; CIPA attestations.
- **Accessibility** — Disclosure feeds + opt-out UIs WCAG 2.1 AA.
- **Scalability** — 50-state registry; disclosure feeds via queue.
- **Reliability** — Unknown/edge state defaults to the **most protective** applicable rule set.
- **Observability** — `state_rights_requests_total{state}`, `gpc_optouts_total{state}`, `state_rules_version`; alert on a request in a state with no registry entry.
- **Maintainability** — Rules as versioned data in `server/internal/service/stateprivacy/`; extend the existing module + repo.
- **Internationalization** — English-first; Spanish where states require bilingual notices.
- **Backward compatibility** — Fold 10.4/10.6 behaviour into the registry (CA/NY/IL become registry rows); no regression.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Texas resident files an access request, *when* S01 resolves rights, *then* it uses the TDPSA rule row (rights, thresholds, SLA) from the registry — no code change was needed to support Texas.
- **AC-2.** *Given* a subject with GPC in Colorado, *when* a request is processed, *then* they're opted out of targeted advertising/sale automatically and it's logged.
- **AC-3.** *Given* a student in any SOPIPA-family state, *when* the platform runs, *then* no targeted ads/sale occur and district-requested deletion works — uniformly.
- **AC-4.** *Given* we enable profiling-based recommendations in a state with an assessment trigger, *when* it's configured, *then* a required S06 DPIA is auto-created and blocks launch until signed off.
- **AC-5.** *Given* a multi-state district, *when* they request DPA addenda, *then* per-state addenda + prohibition attestations generate.
- **AC-6.** *Given* a request arrives from a state with no registry entry, *when* resolved, *then* the most-protective default applies and an alert fires to add the state.

## 8. Data Model

New migration `367_state_privacy_expansion.sql` (+ `.down.sql`), extending `188_state_privacy`:

```sql
CREATE TABLE IF NOT EXISTS compliance.state_privacy_rules (
  state_code      TEXT PRIMARY KEY,             -- 'CA','VA','CO','TX',...
  law_name        TEXT NOT NULL,
  rights          TEXT[] NOT NULL,              -- {'access','delete','correct','opt_out_sale','opt_out_profiling',...}
  sla_days        INT NOT NULL,
  honors_gpc      BOOLEAN NOT NULL DEFAULT FALSE,
  sensitive_categories TEXT[] NOT NULL DEFAULT '{}',
  assessment_triggers TEXT[] NOT NULL DEFAULT '{}',
  student_law     TEXT,                          -- 'sopipa_family' | null
  bilingual_required BOOLEAN NOT NULL DEFAULT FALSE,
  effective_date  DATE,
  version         INT NOT NULL DEFAULT 1,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.state_attestations (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID NOT NULL REFERENCES org.organizations(id),
  state_code    TEXT NOT NULL REFERENCES compliance.state_privacy_rules(state_code),
  attestation_type TEXT NOT NULL,               -- 'no_targeted_ads','no_sale','dpa_addendum'
  document_path TEXT,
  generated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Backfill: seed CA/NY/IL from 10.6, CA CCPA from 10.4, plus the current comprehensive-act states.

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/state-rules` | `privacy:state_admin` | Manage the registry |
| `GET` | `/api/v1/compliance/state-rules/resolve` | internal (S01/S04) | Rights/SLA/opt-out for a state |
| `POST` | `/api/v1/compliance/state-attestations` | `records:admin` | Generate attestations/DPA addenda |
| `GET` | `/api/v1/compliance/state-disclosure-feed/{student_id}` | parent | Per-state disclosure feed |

## 10. UI / UX

- **State-rules admin console:** registry grid, version history, effective-date scheduling, "most-protective default" indicator.
- **Attestation generator:** per-state prohibition attestations + DPA addenda (extends 10.5/10.6 flows).
- **Subject rights + opt-out UIs** reuse S01/S04 surfaces, state-aware.
- States: unknown-state fallback banner; upcoming-effective-date reminders.
- Accessibility: bilingual rendering where required; i18n keys `stateprivacy.*`.

## 11. AI / ML Considerations

Profiling/automated-decision recommendations are gated by state assessment triggers (auto-DPIA via S06) and by opt-out-of-profiling rights (S01/S04). Student data is excluded from ad/engagement models everywhere (SOPIPA-family + S08).

## 12. Integration Points

- `server/internal/service/stateprivacy/` (extend) + repo `server/internal/repos/stateprivacy`; `ccpa` service folded into the registry model.
- S01 (rights resolution), S04 (GPC/opt-out), S06 (assessment triggers), S07 (DPA addenda), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: 10.4, 10.6, S01, S04, S06.
- Must ship before: US-wide GA marketing claims of "50-state compliant."
- Shared infra: rights engine, consent ledger, assessment workflow.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Registry drifts behind newly-effective laws | H | H | Legal review cadence; effective-date scheduling; alert on unknown-state requests |
| Per-state nuance lost in generalisation | M | H | Registry carries state-specific fields; escape hatch for true one-offs |
| GPC honoured inconsistently | M | M | Driven by `honors_gpc` flag per state; tested |
| Attestation says something untrue | L | H | Attestations generated from enforced controls, not free text |

## 15. Rollout Plan

- Flag `state_privacy_registry_enabled`. Phase 1: registry + seed CA/NY/IL/CCPA + fold existing behaviour. Phase 2: add comprehensive-act states + GPC + assessment triggers. Phase 3: attestations/DPA addenda + disclosure feeds. GA rolling per state as reviewed. Rollback: fall back to per-law modules (kept during transition).

## 16. Test Plan

- **Unit** — rule resolution per state; most-protective default; GPC flag handling; assessment-trigger firing.
- **Integration** — S01 uses registry for TX/VA/CO; SOPIPA-family enforcement uniform; auto-DPIA on trigger.
- **E2E** — multi-state district generates per-state attestations; GPC opt-out in a recognising state.
- **Security** — authz on registry edits; attestation integrity.
- **Accessibility** — axe + bilingual rendering.
- **Performance** — resolution < 10 ms cached.
- **Manual** — legal review of 5 sampled state rows against statute.

## 17. Documentation & Training

- Legal runbook: adding a state to the registry (fields → statute mapping).
- District guide: obtaining per-state attestations/addenda.
- Public trust page: "Where we comply" state coverage list.

## 18. Open Questions

1. Cadence and ownership for tracking newly-enacted state laws (in-house vs. external monitoring service)?
2. Do we key applicability on subject residency, tenant location, or both?
3. Which states' universal-opt-out recognition is mandatory vs. optional right now?
4. CIPA: attestation-only, or do we add filtering-status reporting hooks?

## 19. References

- `server/internal/service/{stateprivacy,ccpa}`, `server/internal/repos/stateprivacy`, `server/migrations/188_state_privacy.sql`
- CCPA/CPRA; VCDPA; CPA; CTDPA; UCPA; TDPSA; OCPA; MTCDPA; FDBR; SOPIPA-family student laws; state kids' codes; CIPA 47 U.S.C. § 254(h)
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S04](S04-unified-consent-preference-ledger.md), [S06](S06-dpia-pia-algorithmic-impact.md), [10.6](../../completed/10-compliance-privacy-security/10.6-state-specific-laws.md)

# S07 — Cross-Border Transfer & Subprocessor Governance

> Implementation plan. Hardens: [10.3 GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md) (**explicitly deferred** SCC/BCR transfer mechanisms), [10.5 DPA template](../../completed/10-compliance-privacy-security/10.5-sdpc-national-dpa-template.md), [10.12 data residency](../../completed/10-compliance-privacy-security/10.12-data-residency.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S07 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | EU/UK · Global |
| **Status (today)** | MISSING — GDPR plan 10.3 declared transfer mechanisms "legal document review only, not an engineering deliverable"; there is no subprocessor register, no transfer-mechanism tracking, no DPA lifecycle, no customer-facing subprocessor list |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | DPO + Vendor/Procurement |
| **Depends on** | 10.3, 10.5, 10.12, S05 (which stores are external) |
| **Unblocks** | S03 (controller notification contacts), S12–S19 (each cites its transfer basis) |

---

## 1. Problem Statement

Every personal-data flow leaving a subject's jurisdiction needs a lawful **transfer mechanism** (EU SCCs, UK IDTA/Addendum, EU-US Data Privacy Framework, adequacy, or a derogation) plus a **transfer impact assessment** where required. Every third party that touches personal data on our behalf is a **subprocessor** requiring a signed DPA with GDPR Art 28 flow-down terms, and customers must be given notice of them with a right to object. Lextures ships with numerous subprocessors (LLM providers, storage, email/SMS, analytics) and cross-region flows, yet has **no register, no transfer-mechanism record, no DPA lifecycle, and no published subprocessor list** — the GDPR plan explicitly punted this. That is a direct Chapter V violation, a DPA breach with every institutional customer, and the reason enterprise/EU deals stall in security review.

## 2. Goals

- A **subprocessor register**: every third party processing personal data, the categories/purposes it handles (from S05), its location, DPA status, and transfer mechanism.
- **Transfer-mechanism tracking** per flow: SCC module + version, UK IDTA, DPF certification status, adequacy, or documented derogation — with expiry/renewal.
- **Transfer Impact Assessments (TIAs)** where a mechanism requires one, reusing S06.
- A **published, versioned subprocessor list** with customer notification + objection workflow (as promised in DPAs).
- **DPA lifecycle** management (incoming customer DPAs + outgoing subprocessor DPAs) with renewal alerts.

## 3. Non-Goals

- Data-residency *routing/storage* itself (10.12) — S07 documents and governs it, not the infra.
- Negotiating contract terms (legal function) — S07 tracks their state and surfaces obligations.
- Breach notification (S03) — S07 supplies the contacts S03 uses.

## 4. Personas & User Stories

- **As a DPO**, I want every subprocessor and its transfer mechanism in one register so that I can prove Chapter V compliance.
- **As an institutional customer's security reviewer**, I want a current subprocessor list and to be notified of additions so that I can exercise objection rights.
- **As procurement/vendor management**, I want DPA renewal alerts so that no processor operates under a lapsed agreement.
- **As a security responder (S03)**, I want the notification contact for a breached subprocessor so that I can meet controller-notice duties.
- **As a data subject**, I want to know which categories of recipients my data is shared with (transparency) so that disclosures are honest.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain a `subprocessors` register (name, purpose, categories from S05, location, sub-sub-processors, contact, status active/retired).
- **FR-2.** Each register entry MUST record its **transfer mechanism** (`scc_2021` + module, `uk_idta`, `dpf`, `adequacy`, `derogation`) with effective/expiry dates.
- **FR-3.** The system MUST require a **TIA** (S06 variant) for mechanisms/regions that mandate one, and block "active" status until it's complete for those.
- **FR-4.** The system MUST publish a **customer-facing subprocessor list** with versioning, and on additions, **notify subscribed customers** and open an objection window.
- **FR-5.** The system MUST track **DPA agreements** (customer-inbound and subprocessor-outbound): signed status, version, Art 28 clause coverage, expiry, and renewal alerts.
- **FR-6.** The register MUST link to S05 data stores so that "external store" ⇒ "must have a subprocessor entry + mechanism," enforced by a consistency check.
- **FR-7.** The system MUST expose subprocessor contacts to S03 for breach notifications.
- **FR-8.** Changes MUST be audit-logged (10.11) and reflected in the RoPA recipients (S05).

## 6. Non-Functional Requirements

- **Performance** — Register is low-volume; the public list is cached/static-generated.
- **Security** — Register edits gated by `vendor:admin`; DPA documents stored encrypted; public list exposes only what DPAs promise (no internal contacts).
- **Privacy & Compliance** — GDPR Chapter V (Arts 44–49), Art 28; UK IDTA; EU-US DPF; Schrems II TIA expectations; DPA contractual commitments.
- **Accessibility** — Admin + public subprocessor pages WCAG 2.1 AA.
- **Scalability** — Dozens of subprocessors; customer notification fan-out via queue.
- **Reliability** — Renewal/expiry alerts are reliable and idempotent; objection windows tracked to closure.
- **Observability** — `subprocessors_active`, `transfer_mechanisms_expiring_30d`, `dpas_lapsed`; alert on lapse or missing mechanism for an active external store.
- **Maintainability** — Service `server/internal/service/transfergovernance/`; extends `server/internal/service/dpa`.
- **Internationalization** — Public list + notices localised.
- **Backward compatibility** — Seed the register from current vendors (LLM providers via `openrouter`/`aiprovider`, storage, `mail`, `sms`, analytics); additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* an active external store in S05 (e.g. an LLM provider), *when* the consistency check runs, *then* it fails unless a matching subprocessor entry with a valid transfer mechanism exists.
- **AC-2.** *Given* a new subprocessor is added, *when* it is published, *then* subscribed customers are notified and a 30-day objection window opens and is tracked.
- **AC-3.** *Given* a subprocessor in a non-adequate country requiring a TIA, *when* an admin tries to mark it active without a completed TIA, *then* the system blocks activation.
- **AC-4.** *Given* a subprocessor DPA expiring in 30 days, *when* the alert job runs, *then* `transfer_mechanisms_expiring_30d` fires and vendor management is notified.
- **AC-5.** *Given* a breach at a subprocessor (S03), *when* responders open the case, *then* the subprocessor's notification contact is available from the register.
- **AC-6.** *Given* the RoPA is generated (S05), *when* recipients are listed, *then* they match the active subprocessor register.

## 8. Data Model

New migration `363_transfer_governance.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.subprocessors (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  purpose       TEXT NOT NULL,
  category_keys TEXT[] NOT NULL,               -- S05 categories handled
  location      TEXT NOT NULL,                 -- country/region of processing
  sub_processors TEXT[],
  contact_email TEXT,                          -- breach/notice contact (internal)
  status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','active','retired')),
  published     BOOLEAN NOT NULL DEFAULT FALSE,
  data_store_ids UUID[],                        -- links to S05 data_stores
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  retired_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS compliance.transfer_mechanisms (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subprocessor_id UUID NOT NULL REFERENCES compliance.subprocessors(id) ON DELETE CASCADE,
  mechanism     TEXT NOT NULL
                  CHECK (mechanism IN ('scc_2021','uk_idta','dpf','adequacy','derogation')),
  scc_module    TEXT,
  tia_assessment_id UUID,                        -- FK to S06 dpia_assessments (kind=pia)
  effective_at  DATE,
  expires_at    DATE
);

CREATE TABLE IF NOT EXISTS compliance.dpa_agreements (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  direction     TEXT NOT NULL CHECK (direction IN ('customer_inbound','subprocessor_outbound')),
  counterparty  TEXT NOT NULL,
  org_id        UUID REFERENCES org.organizations(id),   -- for customer DPAs
  subprocessor_id UUID REFERENCES compliance.subprocessors(id),
  version       TEXT NOT NULL,
  art28_covered BOOLEAN NOT NULL DEFAULT FALSE,
  signed_at     DATE,
  expires_at    DATE,
  document_path TEXT
);

CREATE TABLE IF NOT EXISTS compliance.subprocessor_objections (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subprocessor_id UUID NOT NULL REFERENCES compliance.subprocessors(id),
  org_id        UUID NOT NULL REFERENCES org.organizations(id),
  raised_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolution    TEXT
);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/POST/PATCH` | `/api/v1/compliance/subprocessors` | `vendor:admin` | Manage register |
| `GET` | `/api/v1/public/subprocessors` | public | Published, versioned list |
| `POST` | `/api/v1/compliance/subprocessors/{id}/publish` | `vendor:admin` | Publish + notify customers |
| `POST` | `/api/v1/compliance/subprocessors/{id}/object` | customer admin | Raise objection |
| `GET/POST` | `/api/v1/compliance/dpas` | `vendor:admin` | DPA lifecycle |
| `GET` | `/api/v1/compliance/transfers/consistency` | `privacy:dpo` | External-store ↔ mechanism check |

## 10. UI / UX

- **Subprocessor admin console:** register grid, transfer-mechanism editor (with TIA link), publish action, objection tracker, consistency-check panel.
- **DPA manager:** inbound/outbound agreements, expiry calendar, renewal alerts.
- **Public subprocessor page** (`clients/web`): current list, subscribe-for-updates, version history.
- **Customer notification + objection UI** in the tenant admin area.
- States: empty, pending-TIA blocking activation, expiring-soon warnings, objection-open badge.
- Accessibility: table semantics, calendar keyboard nav; i18n keys `subprocessor.*`.

## 11. AI / ML Considerations

LLM/model providers are the highest-scrutiny subprocessors: each must be registered with its data-handling terms (training-use, retention, region), a transfer mechanism, and a TIA. The AI gateway (`aigateway`) SHOULD refuse routing to a provider that is not an `active` registered subprocessor (defence-in-depth with 10.17/S13).

## 12. Integration Points

- `server/internal/service/transfergovernance/` (new); extends `server/internal/service/dpa` + repo `server/internal/repos/dpa`.
- `server/internal/service/{openrouter,aiprovider,aigateway,mail,sms,filestorage}` — seed sources + optional enforcement.
- S05 (external stores + RoPA recipients), S03 (breach contacts), S06 (TIA), 10.12 (residency), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: S05 (external stores), 10.5 (DPA template), 10.12 (residency).
- Must ship before: S03 GA (breach contacts), S12–S19 (transfer bases).
- Shared infra: object storage (DPA docs), email (customer notices), scheduler (expiry alerts).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| External data flow with no mechanism ships silently | H | H | S05 consistency check gates + alerts; aigateway refuses unregistered providers |
| Subprocessor addition without customer notice breaches DPA | M | H | Publish action *is* the notify action; can't activate published entry without notice sent |
| DPA lapses unnoticed | M | M | Expiry calendar + alerts at 60/30/7 days |
| TIA treated as a formality post-Schrems II | M | M | Require documented supplementary measures; DPO sign-off |

## 15. Rollout Plan

- Flag `transfer_governance_enabled`. Phase 1: register + mechanisms + DPA lifecycle, seeded from current vendors. Phase 2: public list + customer notification/objection. Phase 3: consistency check (warn → enforce) + aigateway enforcement. GA when every active external store has a valid mechanism. Rollback: enforcement to warn-only; register is documentation (non-destructive).

## 16. Test Plan

- **Unit** — mechanism/expiry validation; consistency-check logic; objection-window tracking.
- **Integration** — publish → customers notified → objection recorded; RoPA recipients match register; TIA gate blocks activation.
- **E2E** — add subprocessor → TIA → publish → objection → resolution; DPA expiry alert.
- **Security** — authz on register/DPA edits; public list leaks no internal contacts.
- **Accessibility** — axe on console + public page.
- **Performance** — customer notification fan-out to many tenants via queue.
- **Manual** — DPO verifies register vs. actual vendor inventory.

## 17. Documentation & Training

- Public "Subprocessors" page + change-log.
- Runbook: onboarding a new subprocessor (register → mechanism → TIA → publish → notify).
- DPO evidence pack: Chapter V compliance bundle.

## 18. Open Questions

1. DPF reliance vs. SCCs as primary EU-US mechanism given ongoing legal challenges — belt-and-suspenders both?
2. Objection resolution: does an unresolved objection block go-live of the subprocessor for that tenant, or trigger contract off-ramp?
3. Do we enforce aigateway provider-registration in v1 or ship it as warn-only first?
4. Sub-subprocessor depth — one level published, or full chain?

## 19. References

- `server/internal/service/{dpa,openrouter,aiprovider,aigateway,mail,sms,filestorage}`, `server/internal/repos/dpa`
- GDPR Chapter V (Arts 44–49), Art 28; UK IDTA + Addendum; EU-US DPF; Schrems II (C-311/18)
- Related: [S03](S03-global-breach-notification-incident-response.md), [S05](S05-ropa-data-inventory-mapping.md), [S06](S06-dpia-pia-algorithmic-impact.md), [10.12](../../completed/10-compliance-privacy-security/10.12-data-residency.md)

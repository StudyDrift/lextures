# S04 — Unified Consent & Preference Management Ledger

> Implementation plan. Hardens: [10.2 COPPA](../../completed/10-compliance-privacy-security/10.2-coppa-workflow.md), [10.3 GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md) (consent module + deferred cookie banner), [10.4 CCPA](../../completed/10-compliance-privacy-security/10.4-ccpa-cpra.md), [10.17 AI disclosure](../../completed/10-compliance-privacy-security/10.17-ai-usage-disclosure.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S04 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL (Global) |
| **Status (today)** | PARTIAL — COPPA parental consent, GDPR consent, and research consent each keep their own state; no single ledger, no cookie/ePrivacy banner, no Global Privacy Control handling, no lawful-basis registry |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Backend/Compliance + Web |
| **Depends on** | 10.2, 10.3, 10.4, 10.17, 270 research_consent, S05 (purposes tie to processing records) |
| **Unblocks** | S08 (kids' consent), S11 (GPC/opt-out), S13 (AI-use consent), S14–S19 |

---

## 1. Problem Statement

We ask users to consent in at least four disconnected places — COPPA verifiable parental consent (`server/internal/service/coppa`), GDPR AI-processing consent (`server/internal/service/gdpr`), research participation (`server/internal/service/research_consent`), and implicit "you accepted the ToS" — and we have **no cookie/tracking consent banner at all** (GDPR plan 10.3 deferred it to "a separate UI ticket"). There is no single record that answers "what is the lawful basis for processing X, did this person consent to purpose Y, at what version, and did they withdraw?" Regulators (and the GDPR accountability principle) require exactly that ledger; ePrivacy/PECR require prior consent for non-essential cookies; CPRA requires honouring the Global Privacy Control browser signal. Each missing piece is independently actionable.

## 2. Goals

- A single **consent/preference ledger**: append-only records of every consent grant/withdrawal, keyed by subject × purpose × version, with the capture context.
- A **lawful-basis registry**: for each processing purpose, the declared basis (consent, contract, legitimate interest, legal obligation, vital interest, public task) linked to S05's RoPA.
- A **cookie/tracking consent** manager (banner + preference center) with granular categories, prior-consent enforcement, and re-prompt on policy change.
- **Global Privacy Control (GPC)** + "Do Not Sell/Share" opt-out honoured automatically.
- One **preference center** unifying marketing, notifications, cookies, AI-processing, and research consents.

## 3. Non-Goals

- The *mechanism* of verifiable parental consent (card/ID/knowledge checks) — that hardening is S08; this ledger records its outcome.
- The rights-request workflow (S01) — though "withdraw consent" filed there resolves against this ledger.
- Notification *delivery* preferences plumbing already in the notifications service — this centralises the *consent* view over them.

## 4. Personas & User Stories

- **As a data subject**, I want one preference center to see and change every consent I've given so that I'm in control.
- **As an EU visitor**, I want non-essential cookies to load only after I opt in so that my privacy is respected by default.
- **As a Californian**, I want my browser's GPC signal to automatically opt me out of sale/share so that I don't have to click anything.
- **As a DPO**, I want a defensible record of when/what/which-version each subject consented to so that I can prove lawful basis.
- **As a parent**, I want my consent decisions for my child recorded and revocable so that I retain authority.

## 5. Functional Requirements

- **FR-1.** The ledger MUST store append-only `consent_records` (subject, purpose, decision, policy_version, method, ip/ua context, timestamp); withdrawals are new records, never edits.
- **FR-2.** A `processing_purposes` registry MUST enumerate purposes with a declared `lawful_basis` and link to the S05 RoPA entry.
- **FR-3.** Non-essential cookies/trackers MUST NOT be set before consent (prior-consent / opt-in model) in consent-required jurisdictions; essential ones are exempt and documented.
- **FR-4.** The system MUST present a compliant banner (accept-all / reject-all with **equal prominence**, plus granular choices) and a persistent preference center.
- **FR-5.** The system MUST detect and honour the **GPC** header/signal as a valid opt-out of sale/share, and record it as a consent event.
- **FR-6.** On a material privacy-policy/purpose version change, the system MUST **re-solicit** consent for affected purposes and record the new version.
- **FR-7.** Consent state MUST be enforceable at processing time: a purpose without a valid basis for a subject MUST block that processing (e.g. no AI enrichment without AI-consent where consent is the basis).
- **FR-8.** The ledger MUST support export of a subject's full consent history for S01 access requests and be the source of truth for "withdraw consent."
- **FR-9.** All changes MUST be written to `admin_audit_log` (10.11) and reflected in metrics.

## 6. Non-Functional Requirements

- **Performance** — Consent check on the hot path served from cache; p95 < 5 ms; banner script is lightweight and non-blocking.
- **Security** — Consent records are integrity-protected (append-only, hash-chained); only the subject (or authorised parent/agent) can change their own; admin reads gated by `privacy:consent_read`.
- **Privacy & Compliance** — GDPR Arts 6, 7, 9; ePrivacy Directive / PECR; CPRA §1798.135 + GPC; COPPA VPC; LGPD; DPDP. Consent must be freely given, specific, informed, unambiguous, and as easy to withdraw as to give.
- **Accessibility** — Banner + preference center WCAG 2.1 AA; no dark patterns; reject as reachable as accept.
- **Scalability** — Ledger append-only and partitioned by time; consent cache invalidated on write.
- **Reliability** — Consent writes are durable before the corresponding processing proceeds; cache-miss fails closed (treat as no-consent).
- **Observability** — `consent_events_total{purpose,decision}`, `gpc_signals_honored_total`, `consent_reconsent_pending`; alert if enforcement cache error-rate rises.
- **Maintainability** — Ledger in `server/internal/service/consentledger/`; existing consent services become writers to it, not separate stores.
- **Internationalization** — Banner/preference-center copy fully localised incl. RTL (`clients/web/src/i18n/rtl-locales.ts`).
- **Backward compatibility** — Migrate existing COPPA/GDPR/research consent rows into the ledger; keep their read APIs working via a view.

## 7. Acceptance Criteria

- **AC-1.** *Given* an EU visitor's first page load, *when* they have not consented, *then* no non-essential cookie/tracker fires and the banner offers reject-all with equal prominence to accept-all.
- **AC-2.** *Given* a request carrying a GPC signal, *when* it is processed, *then* the subject is recorded as opted-out of sale/share without any UI interaction.
- **AC-3.** *Given* the privacy policy version increments for the "AI tutoring" purpose, *when* an affected subject next visits, *then* they are re-prompted and processing under the old version is paused until they re-consent.
- **AC-4.** *Given* a subject withdraws AI-processing consent, *when* an AI feature is invoked for them, *then* the processing is blocked (or falls back to a non-AI path) and the block is logged.
- **AC-5.** *Given* an S01 access request, *when* the consent provider runs, *then* the subject's full grant/withdrawal history with versions and timestamps is included.
- **AC-6.** *Given* a parent withdraws COPPA consent, *when* the child next uses the product, *then* previously-consented processing stops and the ledger shows the withdrawal record.

## 8. Data Model

New migration `360_consent_ledger.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.processing_purposes (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key           TEXT NOT NULL UNIQUE,          -- 'ai_tutoring','analytics','marketing_email','cookies_ads'
  description   TEXT NOT NULL,
  lawful_basis  TEXT NOT NULL
                  CHECK (lawful_basis IN ('consent','contract','legitimate_interest',
                          'legal_obligation','vital_interest','public_task')),
  requires_consent BOOLEAN NOT NULL,
  ropa_ref      UUID,                           -- FK to S05 records_of_processing
  current_version INT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS compliance.consent_records (
  id            BIGSERIAL PRIMARY KEY,          -- append-only
  subject_id    UUID REFERENCES "user".users(id),
  subject_key   TEXT,                           -- anon cookie id for pre-auth cookie consent
  purpose_key   TEXT NOT NULL REFERENCES compliance.processing_purposes(key),
  decision      TEXT NOT NULL CHECK (decision IN ('granted','withdrawn','denied')),
  policy_version INT NOT NULL,
  method        TEXT NOT NULL,                  -- 'banner','preference_center','vpc','gpc','api'
  captured_by   UUID REFERENCES "user".users(id), -- parent/agent if not self
  context       JSONB,                          -- ip, ua, locale (redaction-aware)
  prev_hash     TEXT,                           -- chain to previous record for tamper-evidence
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_records_subject ON compliance.consent_records(subject_id, purpose_key, created_at DESC);
CREATE INDEX idx_consent_records_key ON compliance.consent_records(subject_key, purpose_key, created_at DESC);
```

Backfill: migrate `coppa`, `gdpr` consent, and `research_consent` (270) records into `consent_records` with `method` set accordingly; seed `processing_purposes` from S05.

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET` | `/api/v1/compliance/consent/purposes` | public | Purpose catalog + current versions |
| `GET` | `/api/v1/compliance/consent` | self / parent | Subject's current consent state |
| `POST` | `/api/v1/compliance/consent` | self / parent / agent | Record grant/withdrawal (banner/pref-center) |
| `GET` | `/api/v1/compliance/consent/history` | self / `privacy:consent_read` | Full ledger history |
| `GET/PUT` | `/api/v1/compliance/consent/purposes/admin` | `privacy:consent_admin` | Manage purposes/basis/version |
| internal | consent-check middleware | — | Enforce at processing time; honours GPC header |

## 10. UI / UX

- **Cookie banner:** first-load, granular categories (essential/functional/analytics/advertising), accept-all & reject-all equally prominent, "manage" link.
- **Preference center** (extends `privacy-centre-page.tsx`): toggles for cookies, marketing, notifications, AI-processing, research; each shows lawful basis + last-changed.
- **Re-consent prompt** on version bump for affected purposes.
- **Admin purpose console:** manage purposes, bases, and versions (bumping a version triggers re-consent).
- States: pre-consent (banner blocking non-essential), loading, error (fail-closed = treat as no consent), offline (queued locally, synced).
- Accessibility: keyboard reachable, no pre-ticked boxes, focus trap in banner, ARIA `dialog`; i18n keys `consent.*`.

## 11. AI / ML Considerations

The `ai_tutoring`, `ai_grading`, and `ai_analytics` purposes are first-class ledger entries; the AI gateway (`server/internal/service/aigateway`) MUST consult the consent-check before sending subject content to any model where consent is the basis, and fall back to a non-AI path or block otherwise (ties to 10.17 / S13).

## 12. Integration Points

- `server/internal/service/consentledger/` (new); writers: `coppa`, `gdpr`, `research_consent`.
- `server/internal/service/aigateway` (enforcement), notifications service (marketing/notification prefs), `clients/web` banner + `privacy-centre-page.tsx`.
- S05 (purpose ↔ RoPA), S01 (consent history + withdraw), `adminaudit`.

## 13. Dependencies & Sequencing

- Must ship after: S05 (purposes reference processing records).
- Must ship before: S08 (kids' consent UX), S13 (AI-consent enforcement), S11 (GPC/opt-out).
- Shared infra: cache, web bundle for banner.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Banner tagged a "dark pattern" | M | H | Reject-all equal prominence; no pre-ticks; UX legal review; A/B forbidden on consent |
| Consent-check on hot path adds latency | M | M | Cache with fail-closed; async warmers; sub-5ms budget with tests |
| GPC mis-detection over/under opts-out | M | M | Conservative parse; log signal; user can override in pref-center |
| Fragmented legacy consents lost in migration | M | H | Reversible backfill with reconciliation report; keep source tables read-only during cutover |

## 15. Rollout Plan

- Flag `consent_ledger_enabled` + `cookie_banner_enabled` (region-gated). Phase 1: ledger + purpose registry + backfill. Phase 2: preference center + enforcement middleware. Phase 3: cookie banner (EU/UK first), then GPC honouring, then re-consent-on-version. Pilot on one region. GA per region. Rollback: flags off; legacy consent services still authoritative during transition.

## 16. Test Plan

- **Unit** — hash-chain integrity; lawful-basis validation; version-bump → re-consent trigger; GPC parse.
- **Integration** — enforcement blocks AI processing without consent; backfill reconciliation; cache fail-closed.
- **E2E** — EU visitor: no non-essential cookie pre-consent; reject-all persists; GPC auto opt-out; withdraw AI consent → feature disabled.
- **Security** — tamper-evidence of ledger; authz on admin purpose edits; no consent write without durable persist.
- **Accessibility** — axe on banner/pref-center; dark-pattern checklist; keyboard-only reject.
- **Performance** — 10k rps consent checks < 5 ms p95.
- **Manual** — regulator-style review of banner against EDPB dark-pattern guidance.

## 17. Documentation & Training

- Help center: "Manage your privacy preferences."
- Runbook: adding a new processing purpose (basis choice + version + re-consent implications).
- Cookie inventory doc (every cookie, category, purpose, provider) maintained alongside S07 subprocessors.

## 18. Open Questions

1. Cookie-consent scope for embedded LTI/third-party tools — do we consent on their behalf or defer?
2. For contract/legitimate-interest purposes, do we still surface an informational toggle (transparency) or hide non-consent bases?
3. GPC: honour globally or only in states/countries that legally recognise it?
4. Retention of the consent ledger itself vs. S02 (it is evidence of lawful basis — likely long-lived).

## 19. References

- `server/internal/service/{coppa,gdpr,research_consent,aigateway}`, `clients/web/src/pages/privacy-centre-page.tsx`, `clients/web/src/i18n/`
- GDPR Arts 6–7, 9; ePrivacy Directive; PECR; CPRA §1798.135 + GPC spec; COPPA 16 CFR §312.5
- Related: [S01](S01-unified-data-subject-rights-orchestration.md), [S05](S05-ropa-data-inventory-mapping.md), [S08](S08-childrens-privacy-age-assurance-design-codes.md), [S13](S13-eu-ai-act-high-risk.md)

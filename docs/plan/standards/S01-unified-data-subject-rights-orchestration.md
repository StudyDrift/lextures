# S01 — Unified Data-Subject Rights (DSAR) Orchestration

> Implementation plan. Hardens: [10.3 GDPR](../../completed/10-compliance-privacy-security/10.3-gdpr-uk-gdpr.md), [10.4 CCPA](../../completed/10-compliance-privacy-security/10.4-ccpa-cpra.md), [10.1 FERPA](../../completed/10-compliance-privacy-security/10.1-ferpa-workflow.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S01 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL (Global) |
| **Status (today)** | PARTIAL — per-law request flows exist (FERPA, GDPR, CCPA) but are siloed, with divergent SLAs, no identity-proofing, and no single audit trail |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Backend/Compliance Lead |
| **Depends on** | 10.1, 10.3, 10.4, 10.11 (audit log), S02 (retention), S05 (data inventory) |
| **Unblocks** | S11, S14–S19 (each jurisdiction reuses this orchestrator instead of building its own) |

---

## 1. Problem Statement

Today a data subject's rights are served by three independent code paths: FERPA record requests (`server/internal/service/ferpa`), GDPR DSAR export/erasure (`server/internal/service/gdpr`), and CCPA requests (`server/internal/service/ccpa`). Each has its own table, status enum, SLA clock, and admin queue. Adding Canada, Brazil, India, or Australia would mean cloning that logic a fourth through ninth time, and no one can answer "show me every open rights request across every law, with proof we met the deadline." That fragmentation is itself an audit finding, and a missed statutory deadline in any one silo is a regulator-reportable failure. We need one orchestrator that classifies a request against the requester's applicable jurisdiction(s), routes it to the right fulfilment workers, enforces the correct clock, and produces one defensible audit trail.

## 2. Goals

- One request intake + case-management model that all rights requests (access, portability, correction, erasure/deletion, restriction, objection, opt-out of sale/share, automated-decision review, consent withdrawal) flow through.
- Deterministic **jurisdiction resolution**: given a requester, compute which laws apply and therefore which rights + SLA + verification standard bind us.
- **Identity proofing** proportional to request sensitivity, satisfying CCPA §999.323 and GDPR Art 12(6) "reasonable measures."
- A single **SLA engine** with per-law clocks, extension records, and escalation before breach.
- One immutable audit trail spanning intake → verify → fulfil → deliver → close, exportable per-regulator.

## 3. Non-Goals

- The underlying data export/erasure *mechanics* — those remain in S02 (deletion) and each domain repo; this plan orchestrates and records them.
- Breach handling (see S03) and consent capture (see S04).
- Replacing law-specific *statutory language* (e.g. FERPA's amendment-hearing wording) — those templates stay with their `10.x` plans.

## 4. Personas & User Stories

- **As a data subject (any jurisdiction)**, I want one place to exercise my rights so that I don't have to know which law protects me.
- **As a parent/guardian**, I want to file on behalf of my minor child with proof of authority so that the request is honoured, not rejected.
- **As a Privacy Operations analyst**, I want a single queue with each case's law, deadline, and required verification so that nothing breaches SLA.
- **As a DPO / compliance officer**, I want to export "all rights requests for regulator X in period Y with fulfilment evidence" so that I can answer an inquiry in hours, not weeks.
- **As an authorized agent (CCPA §1798.135)**, I want to submit on a consumer's behalf with a signed permission so that the request is verified and processed.

## 5. Functional Requirements

- **FR-1.** The system MUST expose a single intake (authenticated in-app + an unauthenticated public form) that creates a `rights_request` regardless of governing law.
- **FR-2.** The system MUST resolve applicable jurisdictions from the subject's residency/tenant/citizenship signals and stamp the resulting `law_basis[]` and binding `sla_days` (min across applicable laws) on the request.
- **FR-3.** The system MUST verify requester identity before fulfilment, with tiered assurance: self (session) < email/step-up < document/attestation for high-risk (erasure, full access) — configurable per law.
- **FR-4.** The system MUST support requests **by an authorized agent or parent/guardian**, capturing and storing proof of authority.
- **FR-5.** The system MUST fan a request out to registered **fulfilment providers** (one per data domain: grades, submissions, messages, media, analytics, AI logs) and aggregate their results, so new domains self-register rather than being hard-coded.
- **FR-6.** The system MUST run an SLA clock per applicable law, record any permitted extension (with reason + notice to subject), and raise escalation alerts at 75% and 90% of the window.
- **FR-7.** The system MUST record refusals with a lawful ground (e.g. manifestly unfounded, legal-hold under S02, conflicting rights) and notify the subject with appeal instructions.
- **FR-8.** The system SHOULD deduplicate concurrent requests from the same verified subject and MUST honour a legal hold from S02 by suppressing erasure while still logging the deferral.
- **FR-9.** Every state transition MUST be written to `admin_audit_log` (10.11) with actor, law basis, and evidence pointers.

## 6. Non-Functional Requirements

- **Performance** — Intake p95 < 300 ms; access-package assembly runs async via the job queue (17.3) with progress; deadline computations are pure and cached.
- **Security** — Public intake is rate-limited and CAPTCHA-gated; delivered packages are encrypted at rest and served via short-lived, single-use, re-auth-gated links. Authz: only `privacy:rights_admin` may action a case; subjects see only their own.
- **Privacy & Compliance** — Satisfies GDPR Arts 12, 15–22; UK GDPR; CCPA/CPRA §§1798.100–130; and is the shared substrate for S14–S19. Verification data collected for a request MUST NOT be repurposed and MUST be deleted per S02 once the case closes.
- **Accessibility** — Public + in-app forms and the admin console conform to WCAG 2.1 AA (see 10.7 / S20).
- **Scalability** — Tens of thousands of requests/tenant; queue-backed fan-out; provider timeouts isolated so one slow domain cannot stall a case.
- **Reliability** — Fulfilment is idempotent and resumable; a re-run never double-deletes and reuses cached export artifacts within 24 h.
- **Observability** — Counters `rights_requests_total{law,type,outcome}`, histogram `rights_request_fulfilment_seconds`, and a gauge `rights_requests_sla_at_risk` (see 17.7).
- **Maintainability** — Orchestrator in `server/internal/service/rightsorchestrator/`; law specifics live in small strategy structs, not `if law == …` branches.
- **Internationalization** — All subject-facing copy externalised (see 11.1); deadline math is timezone- and locale-correct.
- **Backward compatibility** — Existing FERPA/GDPR/CCPA endpoints continue to work but internally create a unified case; a migration links historical rows to the new model.

## 7. Acceptance Criteria

- **AC-1.** *Given* a subject resident in the EU **and** California, *when* they file an access request, *then* the case carries both `gdpr` and `ccpa` bases and the SLA is the shorter (30 days) with both regulators' evidence captured.
- **AC-2.** *Given* an unverified public erasure request, *when* identity proofing fails twice, *then* the case is held in `awaiting_verification`, no data is deleted, and the subject receives resubmission guidance.
- **AC-3.** *Given* an open access case at 90% of its SLA window, *when* the escalation job runs, *then* a `rights_requests_sla_at_risk` alert fires and the assigned analyst is notified.
- **AC-4.** *Given* an erasure request for a student under an active legal hold (S02), *when* fulfilment runs, *then* deletion is deferred, the deferral + ground are logged, and the subject is told which data is retained and why.
- **AC-5.** *Given* a parent files on behalf of a minor with uploaded proof of authority, *when* an analyst approves, *then* the case proceeds and the authority document is retained on the case, not in the child's profile.
- **AC-6.** *Given* a DPO requests "all GDPR requests in Q3," *when* they export, *then* every case appears with intake, verification, fulfilment, delivery timestamps, and deadline-met/-missed status.

## 8. Data Model

New migration `357_rights_orchestration.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.rights_requests (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID REFERENCES org.organizations(id),          -- null for platform-level (SL) subjects
  subject_id    UUID REFERENCES "user".users(id),               -- null until matched to an account
  subject_email TEXT NOT NULL,
  filed_by      UUID REFERENCES "user".users(id),               -- agent/parent if not self
  filed_capacity TEXT NOT NULL DEFAULT 'self'
                  CHECK (filed_capacity IN ('self','parent','guardian','authorized_agent')),
  request_type  TEXT NOT NULL
                  CHECK (request_type IN ('access','portability','correction','erasure',
                          'restriction','objection','opt_out','adm_review','consent_withdraw')),
  law_basis     TEXT[] NOT NULL,                                 -- e.g. {'gdpr','ccpa'}
  status        TEXT NOT NULL DEFAULT 'received'
                  CHECK (status IN ('received','awaiting_verification','verified','in_progress',
                          'awaiting_subject','fulfilled','delivered','refused','withdrawn')),
  verification_tier TEXT NOT NULL DEFAULT 'unverified',
  sla_days      INT  NOT NULL,
  received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  due_at        TIMESTAMPTZ NOT NULL,
  extended_to   TIMESTAMPTZ,
  extension_reason TEXT,
  refusal_ground TEXT,
  legal_hold_ref UUID,                                           -- FK to S02 legal_holds
  closed_at     TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.rights_request_tasks (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id    UUID NOT NULL REFERENCES compliance.rights_requests(id) ON DELETE CASCADE,
  provider      TEXT NOT NULL,          -- 'grades','submissions','messages','ai_logs',...
  status        TEXT NOT NULL DEFAULT 'pending',
  artifact_path TEXT,                    -- encrypted object key
  result_meta   JSONB,
  completed_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS compliance.rights_request_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id  UUID NOT NULL REFERENCES compliance.rights_requests(id) ON DELETE CASCADE,
  actor_id    UUID REFERENCES "user".users(id),
  event       TEXT NOT NULL,
  detail      JSONB,
  at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rights_requests_open ON compliance.rights_requests(status, due_at)
  WHERE status NOT IN ('delivered','refused','withdrawn');
```

Backfill: a data migration links existing `ferpa_record_requests`, `gdpr_*`, `ccpa_*` rows to new `rights_requests` rows (status mapped) so history is queryable in one place.

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/compliance/rights-requests` | public (rate-limited) or self | Create; body includes type + declared residency |
| `POST` | `/api/v1/compliance/rights-requests/{id}/verify` | requester | Submit verification proof |
| `GET` | `/api/v1/compliance/rights-requests` | `privacy:rights_admin` | Queue with filters (law, status, at-risk) |
| `GET` | `/api/v1/compliance/rights-requests/{id}` | admin or owning subject | Case detail + timeline |
| `PATCH` | `/api/v1/compliance/rights-requests/{id}` | `privacy:rights_admin` | Verify/extend/fulfil/refuse/close |
| `GET` | `/api/v1/compliance/rights-requests/{id}/package` | owning subject (re-auth) | Download delivered archive |
| `GET` | `/api/v1/compliance/rights-requests/export` | `privacy:rights_admin` | Regulator evidence export (CSV/JSON) |

Rate limit: 10 req/hour/IP on public intake; OpenAPI documented; all mutations audited.

## 10. UI / UX

- **Public "Privacy Rights" page** (extends `clients/web/src/pages/privacy-centre-page.tsx`): pick a right → declare residency → identity step → confirmation with reference number.
- **In-app "Your data & rights" tab** in account settings: file, track status, download packages.
- **Privacy Ops console** (new admin page): unified queue, SLA countdown chips, verification review, provider task board, refuse/extend actions, evidence export.
- States: empty ("No open requests"); loading skeletons on package build; error ("Verification could not be completed"); offline banner on mobile.
- Accessibility: labelled steps, focus moves to each step heading, ARIA `status` on the SLA countdown; i18n keys `rights.*`.

## 11. AI / ML Considerations

Access packages MUST include the subject's **AI interaction logs** (prompts/outputs tied to them) sourced from `server/internal/service/aidisclosure`; erasure MUST propagate to those logs and to any vector/embedding stores (see S06). No rights-request content is itself sent to an LLM; the AI-log fulfilment provider reads records, it does not generate.

## 12. Integration Points

- `server/internal/service/{ferpa,gdpr,ccpa}` — refactored to register as fulfilment providers behind the orchestrator.
- `server/internal/service/rightsorchestrator/` — new module (resolver, SLA engine, verifier, fan-out).
- `server/internal/repos/rightsrequests/` — new repo.
- Job queue (17.3) for async fulfilment; mail service (6.2) for acknowledgements/delivery; `adminaudit` for the trail; S02 for legal-hold checks; S05 for the provider registry (which domains hold PII).

## 13. Dependencies & Sequencing

- Must ship after: 10.11 (audit log), S05 (inventory tells the orchestrator which providers exist), and at least S02's legal-hold table.
- Must ship before: S14–S19 (they configure this orchestrator rather than reimplementing intake).
- Shared infra: job queue, encrypted object storage, email.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Over-collecting verification data creates new PII risk | M | M | Minimise fields; auto-delete verification artifacts at close (S02) |
| Jurisdiction resolver mis-classifies subject → wrong SLA | M | H | Default to strictest applicable law; log signals used; manual override with reason |
| A slow/broken domain provider stalls all cases | M | M | Per-provider timeout + retry; partial-fulfilment status; queue isolation |
| Public intake abused for enumeration/DoS | M | M | Rate limit, CAPTCHA, no account-existence disclosure |

## 15. Rollout Plan

- Flag `rights_orchestrator_enabled` (default off). Phase 1: migration + backfill (flag off). Phase 2: orchestrator + providers behind flag, existing endpoints shim into it. Phase 3: unified console + public page. Pilot: one HE + one K12 tenant, 30-day soak. GA: enable globally; deprecate direct per-law admin queues. Rollback: flag off restores legacy paths (data is additive).

## 16. Test Plan

- **Unit** — resolver truth table (residency combos → law_basis + sla_days); SLA/extension math across DST; refusal-ground validation.
- **Integration** — multi-provider fan-out with one failing provider; legal-hold suppression of erasure; backfill correctness.
- **E2E** — Playwright: public access request → verify → analyst fulfils → subject downloads; parent-on-behalf-of-minor.
- **Security** — authz matrix (subject vs other subject vs admin), single-use link expiry, rate-limit + CAPTCHA on public intake, OWASP ASVS §4.
- **Accessibility** — axe on public/in-app/admin surfaces; keyboard-only case action.
- **Performance/load** — 10k concurrent open cases; escalation job completes < 2 min.
- **Manual** — DPO evidence-export walkthrough matched to a mock regulator inquiry.

## 17. Documentation & Training

- Help center: "Exercise your privacy rights" (subject-facing, multi-jurisdiction).
- Runbook: "Handling a rights request end-to-end," including verification decision tree and extension criteria.
- API reference + OpenAPI update; DPO playbook: producing regulator evidence exports.

## 18. Open Questions

1. Which residency signal is authoritative when tenant, profile, and IP geolocation disagree?
2. Do we accept authorized-agent requests for K12 minors, or restrict those to verified parents/guardians only?
3. Verification-artifact retention: fixed 90 days post-close, or per-law minimum?
4. Should opt-out-of-sale/share be instant (no verification) per CPRA guidance, bypassing the standard verify step?

## 19. References

- `server/internal/service/{ferpa,gdpr,ccpa}/service.go`; `server/internal/httpserver/{ccpa_http,coppa_http,admin_audit_log_http}.go`
- `server/internal/repos/{ferpa,gdpr,ccpa,adminaudit}`
- GDPR Arts 12, 15–22; UK GDPR; CCPA/CPRA §§1798.100–130, 999.323; FERPA 34 CFR §99.10, §99.20
- Related: [S02](S02-data-retention-deletion-engine.md), [S03](S03-global-breach-notification-incident-response.md), [S05](S05-ropa-data-inventory-mapping.md), [10.11](../../completed/10-compliance-privacy-security/10.11-admin-audit-log.md)

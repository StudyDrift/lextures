# S03 — Global Data-Breach Notification & Incident Response

> Implementation plan. Hardens: [10.9 SOC 2](../../completed/10-compliance-privacy-security/10.9-soc2-type-ii.md), [10.10 ISO 27001/27701](../../completed/10-compliance-privacy-security/10.10-iso-27001-27701.md), [10.11 audit log](../../completed/10-compliance-privacy-security/10.11-admin-audit-log.md), ISMS [incident-response](../../isms/incident-response.md). Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S03 |
| **Section** | Standards & Legal Hardening |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL (Global) |
| **Status (today)** | THIN — an ISMS incident-response *policy* exists on paper; no system to track a breach, compute per-jurisdiction notification deadlines, or generate regulator/affected-party notices |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Security/Compliance Lead |
| **Depends on** | 10.11 (audit log), 10.14 (PII redaction), S05 (data inventory tells us who/what was affected) |
| **Unblocks** | S12–S19 (each names its regulator + deadline; this engine executes them) |

---

## 1. Problem Statement

Breach-notification duties are a minefield of conflicting clocks and thresholds: GDPR requires supervisory-authority notice within **72 hours** and affected-individual notice "without undue delay" for high-risk breaches; US state laws range from "most expedient time possible" to hard day-counts and vary on AG/consumer-reporting thresholds; PIPEDA requires notice for "real risk of significant harm"; Australia's NDB scheme has its own eligibility test; FERPA and state ed-privacy laws add school-notification duties. Missing any one deadline converts a security incident into a separate regulatory violation with its own fine. We currently have a policy document and nothing that operationally tracks an incident, tells responders *which regulators must be told by when*, and produces the notices. That gap is uninsurable and a guaranteed audit finding.

## 2. Goals

- An **incident case model** covering detection → triage → containment → assessment → notification → closure.
- A **notification decision + deadline engine**: given affected data categories and data-subject jurisdictions (from S05), compute every regulator/individual notice obligation and its deadline.
- **Notice generation**: templated, per-jurisdiction regulator filings and affected-party communications, populated from the case.
- A defensible, immutable **incident record** for auditors and cyber-insurers.
- Tabletop/drill support to prove the process works before it's needed.

## 3. Non-Goals

- Threat detection / SIEM (upstream security tooling feeds this via an intake, but detection is out of scope).
- The forensic investigation itself; this tracks findings, it doesn't perform IR forensics.
- General customer status-page comms (that is `server/internal/service/statuspage`), though this can trigger it.

## 4. Personas & User Stories

- **As an incident commander**, I want one case that shows every notification obligation and its countdown so that nothing is missed under pressure.
- **As a DPO**, I want the 72-hour GDPR clock to start when we became aware and to be reminded before it expires so that we file on time.
- **As a security engineer**, I want to record affected data categories and record counts so that the engine determines who must be notified.
- **As legal counsel**, I want draft regulator filings and affected-party notices generated from case facts so that we only edit, not author from scratch.
- **As a customer's compliance officer**, I want timely, accurate breach notice per our DPA so that we can meet our own downstream duties.

## 5. Functional Requirements

- **FR-1.** The system MUST create an `incident` with an "awareness" timestamp that starts all statutory clocks.
- **FR-2.** The system MUST capture affected **data categories**, **subject populations** (with jurisdictions from S05), and estimated record counts.
- **FR-3.** The **obligation engine** MUST derive the full set of notification obligations (regulator + individuals + affected customer/controllers + ed-authority) with each obligation's legal deadline and risk-threshold determination.
- **FR-4.** The system MUST track each obligation's status (required / not-required-with-reason / drafted / filed) and alert before any deadline.
- **FR-5.** The system MUST generate notice drafts from per-jurisdiction templates, populated from case fields, and record what was actually sent and when.
- **FR-6.** For processor-role breaches, the system MUST notify affected **controllers (customers)** "without undue delay" and surface which DPAs/contacts apply (from S07).
- **FR-7.** The system MUST maintain a documented **risk-assessment** per breach (harm likelihood/severity) to justify notify/no-notify decisions.
- **FR-8.** The system MUST support drills that create non-production incidents flagged `is_drill = true`, excluded from real metrics but exercising the full flow.
- **FR-9.** Every action MUST be written to the immutable incident timeline and `admin_audit_log` (10.11).

## 6. Non-Functional Requirements

- **Performance** — Obligation computation < 2 s; the case UI updates deadlines live.
- **Security** — Incident data is highly sensitive; access limited to `security:incident_responder` / `privacy:dpo`; case content is redaction-aware (10.14) so raw PII of victims isn't over-exposed.
- **Privacy & Compliance** — GDPR Arts 33–34; UK GDPR; US state breach laws (all 50 + DC); PIPEDA breach-of-security-safeguards; Australia NDB; LGPD Art 48; DPDP §8(6); FERPA/state ed-breach duties.
- **Accessibility** — Responder console WCAG 2.1 AA (responders may work under stress on any device).
- **Scalability** — Handles an incident affecting an entire tenant (100k+ subjects) without enumerating each individual in the UI; notification batches run via the queue.
- **Reliability** — Deadline clocks are monotonic and survive restarts; reminders are idempotent; a partially-sent affected-party batch resumes.
- **Observability** — `incidents_open`, `breach_obligations_at_risk`, `breach_notices_sent_total{jurisdiction}`; page on any obligation < 6 h to deadline.
- **Maintainability** — Engine in `server/internal/service/incidentresponse/`; jurisdiction rules and templates are data, not code.
- **Internationalization** — Affected-party notices localised to the recipient's locale.
- **Backward compatibility** — Additive; links to existing `statuspage` and `securityreports` services.

## 7. Acceptance Criteria

- **AC-1.** *Given* an incident affecting EU + California + Australia subjects, *when* obligations compute, *then* the case lists the lead EU supervisory authority (72 h), CA AG/consumer thresholds, and the AU OAIC NDB assessment, each with its own deadline.
- **AC-2.** *Given* an open GDPR obligation at 60 h since awareness, *when* the reminder job runs, *then* responders and the DPO are paged and `breach_obligations_at_risk` increments.
- **AC-3.** *Given* a processor-role breach touching three customer tenants, *when* obligations compute, *then* each affected controller's DPA notification contact (from S07) is listed with the "without undue delay" duty.
- **AC-4.** *Given* a low-risk breach where the engine's risk assessment concludes no individual notice is required, *when* the responder records that decision, *then* the justification is stored and the obligation is closed as "not-required-with-reason."
- **AC-5.** *Given* a drill incident, *when* it runs end-to-end, *then* the full flow executes, no real notices are sent, and it is excluded from production breach metrics.
- **AC-6.** *Given* a closed incident, *when* an auditor exports it, *then* the timeline shows awareness time, every obligation, every notice sent (with content hash + timestamp), and the risk assessment.

## 8. Data Model

New migration `359_incident_response.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS security.incidents (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ref           TEXT NOT NULL UNIQUE,                 -- human ref e.g. INC-2026-014
  severity      TEXT NOT NULL CHECK (severity IN ('low','medium','high','critical')),
  status        TEXT NOT NULL DEFAULT 'triage'
                  CHECK (status IN ('triage','contained','assessing','notifying','closed')),
  role          TEXT NOT NULL CHECK (role IN ('controller','processor')),
  aware_at      TIMESTAMPTZ NOT NULL,                 -- starts statutory clocks
  affected_categories TEXT[] NOT NULL,
  affected_jurisdictions TEXT[] NOT NULL,
  affected_count INT,
  risk_assessment JSONB,                              -- harm likelihood/severity + rationale
  is_drill      BOOLEAN NOT NULL DEFAULT FALSE,
  commander_id  UUID REFERENCES "user".users(id),
  opened_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  closed_at     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS security.incident_obligations (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id   UUID NOT NULL REFERENCES security.incidents(id) ON DELETE CASCADE,
  obligation    TEXT NOT NULL,                        -- 'gdpr_sa','ca_ag','au_oaic','controller_notice',...
  recipient     TEXT,
  deadline_at   TIMESTAMPTZ,
  status        TEXT NOT NULL DEFAULT 'required'
                  CHECK (status IN ('required','not_required','drafted','filed')),
  not_required_reason TEXT,
  notice_hash   TEXT,
  filed_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS security.incident_timeline (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id UUID NOT NULL REFERENCES security.incidents(id) ON DELETE CASCADE,
  actor_id    UUID REFERENCES "user".users(id),
  entry       TEXT NOT NULL,
  detail      JSONB,
  at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incident_obligations_open ON security.incident_obligations(deadline_at)
  WHERE status IN ('required','drafted');
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `POST` | `/api/v1/security/incidents` | `security:incident_responder` | Open case (sets aware_at) |
| `PATCH` | `/api/v1/security/incidents/{id}` | responder | Update scope/status/risk |
| `POST` | `/api/v1/security/incidents/{id}/recompute-obligations` | responder | Re-run engine after scope change |
| `PATCH` | `/api/v1/security/incidents/{id}/obligations/{oid}` | responder / dpo | Mark drafted/filed/not-required |
| `POST` | `/api/v1/security/incidents/{id}/notices/{oid}/draft` | responder | Generate notice draft |
| `GET` | `/api/v1/security/incidents/{id}/export` | `privacy:dpo` | Auditor/insurer export |

## 10. UI / UX

- **Incident console:** case header with live deadline chips, obligation checklist, timeline feed, notice draft editor, "recompute" after scope edits.
- **Obligation board:** each notice with countdown, required/not-required toggle (reason mandatory), file action.
- States: empty (no open incidents), at-risk banner (red countdown < 6 h), drill watermark, error on notice-send with retry.
- Accessibility: countdowns announced via ARIA live regions; high-contrast severity badges; i18n keys `incident.*`.

## 11. AI / ML Considerations

If a breach involves the AI subsystem (leaked prompts/outputs, model-store exposure), the affected-category set MUST include AI-log categories, and notices MUST describe AI-processed data specifically. No incident PII is sent to external LLMs; an optional internal LLM may *draft* notice prose from structured fields only, behind the redaction proxy (10.14 / S06).

## 12. Integration Points

- `server/internal/service/incidentresponse/` (new); `server/internal/service/statuspage` (public comms trigger); `server/internal/service/securityreports`.
- S05 (affected-category/jurisdiction lookup), S07 (controller/subprocessor notification contacts), mail service (6.2), `adminaudit`, scheduler (deadline reminders).

## 13. Dependencies & Sequencing

- Must ship after: S05 (needs data inventory), 10.11 (audit log), 10.14 (redaction).
- Must ship before: S12–S19 GA (each cites this engine for its breach clock); recommended before any external pilot expansion.
- Shared infra: scheduler, email, status page.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Wrong `aware_at` mis-starts the 72 h clock | M | H | Require explicit awareness entry; log who/when; disallow silent edits |
| Obligation rules drift from law | M | H | Rules as versioned data with legal review cadence; annual tabletop validates |
| Responder overwhelmed → missed obligation | M | H | Live countdowns + escalation paging; default to "required" until explicitly cleared |
| Over-notification (notify when not required) erodes trust | L | M | Documented risk-assessment gate before individual notices |

## 15. Rollout Plan

- Flag `incident_response_enabled` (default on for internal responders only). Phase 1: case model + timeline. Phase 2: obligation engine + reminders. Phase 3: notice generation + customer/controller notifications. Validate with a full tabletop drill before relying on it. GA is internal (not customer-flagged). Rollback: revert to manual runbook (policy stays valid).

## 16. Test Plan

- **Unit** — obligation derivation truth tables per jurisdiction combo; deadline math incl. weekends/holidays where laws specify; risk-assessment gating.
- **Integration** — scope change recomputes obligations; controller-notice contacts resolve via S07; reminder paging fires.
- **E2E** — full drill from open → obligations → drafts → "file" → export.
- **Security** — authz (only responders/DPO), redaction of victim PII in case views, export integrity (content hashes).
- **Accessibility** — axe + live-region announcement of countdowns.
- **Performance** — tenant-wide incident (100k subjects) computes obligations < 2 s.
- **Manual** — annual tabletop scored against real statutory deadlines.

## 17. Documentation & Training

- Runbook: incident-commander playbook + this tool's operation.
- Regulator-filing template library (per jurisdiction) with legal sign-off.
- Customer-facing DPA breach-notice commitments cross-referenced here.

## 18. Open Questions

1. Who has authority to set `aware_at` and does resetting it require dual control?
2. Do we pre-stage regulator portal API integrations (e.g. some SAs accept electronic filing) or keep filings manual for v1?
3. How do drills avoid contaminating cyber-insurance reporting while still proving readiness?
4. Threshold for auto-triggering the public status page vs. targeted notice only.

## 19. References

- `server/internal/service/{statuspage,securityreports}`, `server/internal/scheduler`, ISMS `docs/isms/incident-response.md`
- GDPR Arts 33–34; PIPEDA breach reporting; Australia NDB scheme; US state breach statutes; LGPD Art 48; DPDP §8(6)
- Related: [S05](S05-ropa-data-inventory-mapping.md), [S07](S07-cross-border-transfer-subprocessor-governance.md), [10.14](../../completed/10-compliance-privacy-security/10.14-pii-redaction-logs.md)

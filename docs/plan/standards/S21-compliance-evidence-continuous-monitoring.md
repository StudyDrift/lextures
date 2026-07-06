# S21 — Compliance Evidence, Continuous Control Monitoring & Audit Readiness

> Implementation plan. Hardens: [10.9 SOC 2](../../completed/10-compliance-privacy-security/10.9-soc2-type-ii.md), [10.10 ISO 27001/27701](../../completed/10-compliance-privacy-security/10.10-iso-27001-27701.md), [10.11 audit log](../../completed/10-compliance-privacy-security/10.11-admin-audit-log.md); aggregates every S0x control. Source landscape: [standards/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | S21 |
| **Section** | Standards & Legal Hardening |
| **Severity** | MAJOR |
| **Markets** | Global (K12 · HE · SL) |
| **Status (today)** | THIN — SOC 2 (10.9) and ISO (10.10) produce point-in-time artifacts; there is no **continuous** control-monitoring layer that proves, on any given day, that all the S0x controls are actually operating |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Compliance Lead + Platform |
| **Depends on** | 10.9, 10.10, 10.11, and every S01–S20 (their control signals) |
| **Unblocks** | Fast audit response; a public/customer **Trust Center**; RFP security-questionnaire automation |

---

## 1. Problem Statement

A pile of well-designed controls (S01–S20) is worthless in an audit if you cannot **prove they are operating right now**. SOC 2 Type II and ISO 27001 both hinge on **evidence over time**, and every enterprise deal ships a security questionnaire that asks the same 200 questions. Today, answering an auditor or a customer means an engineer manually screenshotting dashboards and grepping logs — slow, error-prone, and impossible to keep current. This plan adds the **continuous control-monitoring and evidence layer**: a control catalog mapping each framework/law to the concrete S0x mechanism that satisfies it, automated **evidence collectors** that check those controls on a schedule, a live **compliance posture** view, and a **Trust Center** that turns all of it into customer-facing assurance and a questionnaire-answer library.

## 2. Goals

- A **control catalog** mapping frameworks/laws (SOC 2 TSC, ISO 27001 Annex A, ISO 27701, GDPR, FERPA, and the S0x controls) to the concrete implementing mechanism and its owner.
- **Automated evidence collectors** that periodically test each control (e.g. "erasure certificates are generated," "breach obligations never breached SLA," "no PII in logs," "MFA enforced") and record pass/fail with artifacts.
- A **live compliance posture** dashboard: per-framework readiness, failing controls, and drift.
- A **Trust Center** (public/gated) exposing certifications, subprocessors (S07), status (statuspage), and a **security-questionnaire answer library**.
- **Audit-package export**: one click produces the evidence bundle for an auditor or customer.

## 3. Non-Goals

- Replacing the auditors or the SOC 2/ISO certifications themselves (10.9/10.10) — this feeds them.
- Building the underlying controls (S01–S20 own those) — this monitors and evidences them.

## 4. Personas & User Stories

- **As a compliance officer**, I want a live view of which controls are passing so that I know our real posture, not last quarter's.
- **As an auditor**, I want an evidence package with timestamped artifacts so that fieldwork is fast.
- **As an enterprise security reviewer**, I want a Trust Center + questionnaire answers so that I can approve the vendor quickly.
- **As a control owner**, I want to be alerted when my control fails its automated check so that I fix drift before an audit.
- **As a customer**, I want to see current certifications, subprocessors, and status so that I trust the platform.

## 5. Functional Requirements

- **FR-1.** A **control catalog** MUST map each framework requirement/law article to its implementing S0x mechanism, owner, and evidence source.
- **FR-2.** **Evidence collectors** MUST run on a schedule, executing automated checks against real system state (DB, logs via 10.11, metrics via 17.7) and recording pass/fail + artifact.
- **FR-3.** A **posture dashboard** MUST show per-framework readiness %, failing/at-risk controls, and trend, with drill-down to evidence.
- **FR-4.** Control failures MUST alert the owner and open a remediation task; repeated failure escalates.
- **FR-5.** An **audit-package export** MUST bundle selected controls' evidence over a date range (for SOC 2 Type II period coverage).
- **FR-6.** A **Trust Center** MUST publish certifications, the subprocessor list (S07), current status, accessibility conformance (S20), and a **questionnaire answer library** (mapped to CAIQ/SIG-style questions).
- **FR-7.** Evidence and posture MUST be **tamper-evident** and reference immutable audit-log entries (10.11).
- **FR-8.** The catalog MUST show **coverage gaps** (framework requirements with no mapped control) so nothing is unaccounted for.

## 6. Non-Functional Requirements

- **Performance** — Collectors run off-peak; dashboard reads from materialised posture snapshots.
- **Security** — Evidence may reveal control internals; gated `compliance:evidence_read`; Trust Center exposes only approved content; questionnaire library reviewed before publish.
- **Privacy & Compliance** — SOC 2 TSC; ISO 27001 Annex A + 27701; GDPR accountability (Art 5(2)); supports every S0x plan's audit needs.
- **Accessibility** — Dashboard + Trust Center WCAG 2.1 AA (dogfood; see S20).
- **Scalability** — Hundreds of controls × periodic runs; evidence retained per audit-period needs (via S02).
- **Reliability** — A collector failure is itself a signal (fail = "cannot evidence" ≠ silent pass).
- **Observability** — `controls_passing_ratio{framework}`, `controls_failing_total`, `evidence_collector_errors_total`; alert on failing critical controls.
- **Maintainability** — Service `server/internal/service/compliancemonitor/`; collectors are pluggable per control; catalog is versioned data.
- **Internationalization** — Trust Center localised.
- **Backward compatibility** — Aggregates existing SOC 2/ISO artifacts (`server/internal/service/{soc2,iso}`); additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* the control catalog, *when* reviewed, *then* each SOC 2 TSC / ISO Annex A / key GDPR-FERPA requirement maps to an implementing S0x control + owner, and unmapped requirements show as coverage gaps.
- **AC-2.** *Given* PII appears in a log (violating 10.14), *when* the collector runs, *then* the "no PII in logs" control fails, the owner is alerted, and a remediation task opens.
- **AC-3.** *Given* a SOC 2 Type II period, *when* the audit package is exported, *then* it contains each control's evidence across the full date range with timestamps + audit-log references.
- **AC-4.** *Given* an enterprise reviewer opens the Trust Center, *then* they see current certifications, subprocessors (S07), status, accessibility posture (S20), and answered questionnaire items.
- **AC-5.** *Given* a control fails repeatedly, *when* escalation triggers, *then* it's surfaced on the posture dashboard as critical and paged.
- **AC-6.** *Given* a new S0x control ships, *when* it's added to the catalog with a collector, *then* posture updates automatically.

## 8. Data Model

New migration `377_compliance_monitor.sql` (+ `.down.sql`):

```sql
CREATE TABLE IF NOT EXISTS compliance.control_catalog (
  key           TEXT PRIMARY KEY,              -- 'no_pii_in_logs','erasure_certificates','breach_sla',...
  title         TEXT NOT NULL,
  frameworks    JSONB NOT NULL,                -- {'soc2':['CC6.1'],'iso27001':['A.8.12'],'gdpr':['Art17']}
  implementing_plan TEXT NOT NULL,             -- 'S02','S03','10.14',...
  owner_role    TEXT NOT NULL,
  collector     TEXT,                          -- collector id, null = manual evidence
  critical      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS compliance.control_evidence (
  id            BIGSERIAL PRIMARY KEY,
  control_key   TEXT NOT NULL REFERENCES compliance.control_catalog(key),
  result        TEXT NOT NULL CHECK (result IN ('pass','fail','error','manual')),
  artifact      JSONB,                          -- metrics/query result/screenshot ref
  audit_log_ref UUID,                            -- immutable 10.11 reference
  collected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.trust_questionnaire (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  standard      TEXT NOT NULL,                  -- 'CAIQ','SIG','custom'
  question      TEXT NOT NULL,
  answer        TEXT NOT NULL,
  control_keys  TEXT[],                          -- controls that substantiate the answer
  published     BOOLEAN NOT NULL DEFAULT FALSE,
  reviewed_at   TIMESTAMPTZ
);

CREATE INDEX idx_control_evidence_recent ON compliance.control_evidence(control_key, collected_at DESC);
```

## 9. API Surface

| Method | Path | Auth Scope | Notes |
|---|---|---|---|
| `GET/PUT` | `/api/v1/compliance/controls` | `compliance:evidence_admin` | Manage control catalog |
| `GET` | `/api/v1/compliance/posture` | `compliance:evidence_read` | Live posture per framework |
| `POST` | `/api/v1/compliance/controls/{key}/run` | `compliance:evidence_admin` | Trigger a collector |
| `GET` | `/api/v1/compliance/audit-package` | `compliance:evidence_read` | Export evidence bundle (date range) |
| `GET` | `/api/v1/public/trust` | public/gated | Trust Center |
| `GET/PUT` | `/api/v1/compliance/questionnaire` | `compliance:evidence_admin` | Answer library |

## 10. UI / UX

- **Posture dashboard:** framework cards (readiness %), failing/critical controls, trend, drill-down to evidence + audit-log link, coverage-gap list.
- **Control catalog editor:** map requirements → S0x controls → collectors → owners.
- **Audit-package builder:** select framework + period → export bundle.
- **Trust Center** (public/gated): certifications, subprocessors, status, accessibility, questionnaire library, document downloads.
- States: collector-error (distinct from fail), coverage-gap warning, at-risk-before-audit banner.
- Accessibility: WCAG 2.1 AA (dogfood); i18n keys `trust.*`, `compliance.*`.

## 11. AI / ML Considerations

Include AI-governance controls (S13/S06/10.17) in the catalog — e.g. "every high-risk AI system has a signed AIA," "prohibited AI uses = 0," "AI event logs intact." An internal LLM MAY draft questionnaire answers from mapped controls, but a human reviews before publish (no unreviewed claims).

## 12. Integration Points

- `server/internal/service/compliancemonitor/` (new); collectors query 10.11 audit log, 17.7 metrics/telemetry, and each S0x service; `server/internal/service/{soc2,iso}` artifacts; `statuspage` + S07 + S20 feed the Trust Center.

## 13. Dependencies & Sequencing

- Must ship after: the controls it monitors (S01–S20) exist (can ship incrementally, monitoring controls as they land).
- Must ship before: the next SOC 2 Type II audit + enterprise-sales push benefit most.
- Shared infra: scheduler, metrics/telemetry (17.7), audit log (10.11), object storage.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Collectors give false "pass" (checking the wrong thing) | M | H | Peer-reviewed collectors; negative tests; collector-error ≠ pass |
| Trust Center over-claims | M | H | Publish gated on human review; claims traced to passing controls |
| Evidence volume unmanageable | M | M | Retention via S02; summarise + keep artifacts for audit period only |
| Coverage gaps hidden | M | H | Catalog surfaces unmapped requirements explicitly |

## 15. Rollout Plan

- Flag `compliance_monitor_enabled`. Phase 1: catalog + a few high-value collectors (no-PII-in-logs, breach-SLA, erasure-certificates, MFA-enforced) + posture dashboard. Phase 2: audit-package export + questionnaire library. Phase 3: Trust Center (gated → public) + full collector coverage. GA before the next audit cycle. Rollback: dashboard reporting-only; no external claims.

## 16. Test Plan

- **Unit** — posture computation; coverage-gap detection; collector pass/fail/error semantics.
- **Integration** — inject a control violation (e.g. PII in a log) → collector fails → task + alert; audit-package spans a period.
- **E2E** — reviewer opens Trust Center; compliance officer exports SOC 2 package.
- **Security** — authz on evidence/catalog; Trust Center exposes only approved content.
- **Accessibility** — axe on dashboard + Trust Center.
- **Performance** — hundreds of controls; dashboard from snapshots.
- **Manual** — auditor dry-run against a real framework mapping.

## 17. Documentation & Training

- Control catalog as living documentation (the master mapping).
- Runbook: responding to an audit with the evidence package.
- Trust Center content-governance process (review before publish).

## 18. Open Questions

1. Build vs. buy for the monitoring layer (GRC tools exist) — do we integrate one or keep it in-house for control?
2. Which collectors are highest ROI to build first?
3. Trust Center: fully public vs. gated behind NDA/click-through?
4. Evidence retention period to satisfy Type II look-back without unbounded storage?

## 19. References

- `server/internal/service/{soc2,iso,adminaudit}`, `server/internal/telemetry` (17.7 observability layer), `docs/soc2`, `docs/isms`
- SOC 2 Trust Services Criteria; ISO/IEC 27001 Annex A; ISO/IEC 27701; GDPR Art 5(2); CAIQ / SIG questionnaires
- Related: every [S01](S01-unified-data-subject-rights-orchestration.md)–[S20](S20-accessibility-legal-mandates.md) plan; [10.9](../../completed/10-compliance-privacy-security/10.9-soc2-type-ii.md), [10.11](../../completed/10-compliance-privacy-security/10.11-admin-audit-log.md)

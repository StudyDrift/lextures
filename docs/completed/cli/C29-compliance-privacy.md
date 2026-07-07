# C29 — Compliance, privacy & trust

> CLI parity plan. Source: `compliance/*` (67 routes: `ferpa`, `gdpr`, `coppa`, `ccpa`, `iso`, `dpa`, `state`, `security-reports`, `data-inventory`, `ai-inference-log`), `registerPIIRedactionRoutes`, `registerTrustRoutes`, `registerLegalRoutes`, `registerResearchConsentRoutes`, `ai_disclosure_http.go`, `soc2_http.go`. Baseline: `clients/cli/cmd/compliance_privacy.go`, `compliance_privacy_logic.go`, `compliance_privacy_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C29 |
| **Section** | Compliance & trust |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Compliance / CLI |
| **Depends on** | C19, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

The platform has an extensive compliance surface (FERPA, GDPR, COPPA, CCPA, ISO, DPA, state privacy, SOC 2, PII redaction, data inventory, AI-inference logs, research consent) — none reachable from the CLI. Privacy officers cannot fulfill Data Subject Access/erasure Requests (DSAR), pull data inventories, or export compliance evidence via automation, all of which are time-boxed legal obligations.

## 2. Goals

- Fulfill DSARs: export a subject's data (access) and process erasure (right-to-be-forgotten).
- Pull data inventory and compliance evidence (SOC 2 / ISO) for audits.
- Manage consent (COPPA parental, research) and AI-disclosure records.
- Run PII redaction jobs and pull security reports.

## 3. Non-Goals

- Legal adjudication (human/process).
- Being the system of record for compliance (server owns state); CLI drives and exports.

## 4. Personas & User Stories

- **As a privacy officer**, I want `gdpr export --subject U --out d` to fulfill a DSAR within deadline.
- **As a privacy officer**, I want `gdpr erase --subject U` (right to be forgotten) with confirmation.
- **As a compliance lead**, I want `compliance data-inventory export` for a records map.
- **As an auditor**, I want `soc2 evidence export` and `iso controls list`.
- **As a K12 admin**, I want `coppa consent list --pending` to chase parental consent.

## 5. Functional Requirements

- **FR-1.** MUST add `gdpr export|erase|status --subject <u>` and `ccpa`/`state` equivalents (DSAR access/delete/do-not-sell).
- **FR-2.** MUST add `ferpa disclosures list|log`, `ferpa consent` where applicable.
- **FR-3.** MUST add `coppa consent list|grant|revoke` (parental consent) and `research-consent` management.
- **FR-4.** MUST add `compliance data-inventory export`, `compliance audit-log export` (shared with C19), `ai-inference-log export`.
- **FR-5.** SHOULD add `soc2 evidence|status`, `iso controls list`, `dpa list|get`, `security-reports list|get`.
- **FR-6.** SHOULD add `pii redact submit|status` (`registerPIIRedactionRoutes`) and `ai-disclosure get|set` (per course/org).

## 6. Non-Functional Requirements

- **Performance** — DSAR export is async/job-backed; `--wait` streams progress.
- **Security** — compliance-officer scope (highly privileged); all actions audited; erasure requires `--yes` + `--confirm-subject <id>` double-confirmation.
- **Privacy & Compliance** — this IS the compliance surface: exports are legally sensitive; encrypted-at-rest output recommended; deletion is irreversible and logged.
- **Reliability** — export/erase idempotent by request id; resumable.
- **Observability** — every command references a compliance request id for the audit trail.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a subject, *When* `gdpr export --wait`, *Then* a data package downloads and a request id prints.
- **AC-2.** *Given* an erasure, *When* `gdpr erase` without `--confirm-subject`, *Then* it refuses.
- **AC-3.** *Given* an audit, *When* `soc2 evidence export`, *Then* evidence artifacts download.

## 8. Data Model

- Client stores nothing sensitive; writes exports to caller-specified `--out` only.

## 9. API Surface

- `compliance/{gdpr,ferpa,coppa,ccpa,iso,dpa,state,security-reports,data-inventory,ai-inference-log,audit-log}`; `registerPIIRedactionRoutes`; `soc2_http.go`; `ai_disclosure_http.go`; `registerResearchConsentRoutes`; `registerTrustRoutes`; `registerLegalRoutes`.

## 10. UI / UX

- `lextures gdpr|ccpa|ferpa|coppa|iso|soc2|dpa|compliance|pii ...`.
- Destructive ops require double confirmation and print the audit request id.

## 11. AI / ML Considerations

- `ai-inference-log` and `ai-disclosure` support AI-governance/EU-AI-Act obligations; CLI exports logs and manages disclosure text.

## 12. Integration Points

- Server compliance handlers; audit log (C19); jobs (`--wait`, C18).

## 13. Dependencies & Sequencing

- After: C19 (audit export), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accidental irreversible erasure | M | H | Double-confirm; dry-run preview of scope; audit |
| Sensitive export left on disk | M | H | Warn; recommend `--out` to encrypted volume; no stdout dumps |

## 15. Rollout Plan

- Ship DSAR export + data-inventory + audit export first (most-requested), then erasure, then SOC2/ISO/consent.
- Rollback: additive; erasure is not reversible — extra guards.

## 16. Test Plan

- **Unit** — confirmation gating; request-id handling.
- **Integration** — DSAR export job; consent list.
- **Security** — erasure double-confirm; scope; audit emission.
- **E2E** — export a test subject → verify package; (staging only) erase → verify.

## 17. Documentation & Training

- "Fulfill a DSAR within deadline" and "Pull SOC 2 evidence for an audit" runbooks.

## 18. Open Questions

1. Is DSAR export a single unified endpoint or per-regulation?
2. Does erasure cascade across all data stores in one call?

## 19. References

- `compliance/*` handlers; `soc2_http.go`, `ai_disclosure_http.go`, `registerPIIRedactionRoutes`.
- Related: [C19](C19-audit-impersonation-search.md), [C21](C21-platform-settings.md), [C39](C39-profile-account-personas.md).

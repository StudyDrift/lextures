# C26 — xAPI / LRS / SCORM runtime & engagement

> CLI parity plan. Source: `/api/v1/xapi/statements`, `admin/lrs-config`, `admin/lrs-dead-letter`, `/api/v1/scorm/rte/{registration_id}/commit`, `registerEngagementRoutes`, `/api/v1/recommendations/event`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C26 |
| **Section** | Integrations & interoperability |
| **Severity** | MINOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Analytics / CLI |
| **Depends on** | C18, C40 |
| **Unblocks** | C28 |

---

## 1. Problem Statement

Learning-record (xAPI/LRS) and engagement-event surfaces are UI/API-only with no CLI. Analytics engineers cannot post/query xAPI statements for testing, configure the LRS, drain the dead-letter queue, or emit engagement/recommendation events — all needed to validate a learning-analytics pipeline.

## 2. Goals

- Post and query xAPI statements for pipeline testing.
- Configure the LRS and manage its dead-letter queue.
- Emit engagement/recommendation events for testing downstream analytics.

## 3. Non-Goals

- Being a full LRS query console; this is a scripting/diagnostic aid.
- Analytics dashboards (see C28).

## 4. Personas & User Stories

- **As an analytics engineer**, I want `xapi post --file statement.json` to test ingestion.
- **As an engineer**, I want `lrs dead-letter list|retry` to drain failed statements.
- **As a QA engineer**, I want `engagement emit --file event.json` to validate signals.

## 5. Functional Requirements

- **FR-1.** MUST add `xapi post --file <json>` and `xapi query [--actor|--verb|--activity|--since]`.
- **FR-2.** MUST add `lrs config get|set` and `lrs dead-letter list|retry|purge` (`admin/lrs-config`, `admin/lrs-dead-letter`).
- **FR-3.** SHOULD add `engagement emit --file <json>` (`registerEngagementRoutes`) and `recommendations event`.
- **FR-4.** MAY add `scorm commit <registration_id> --file <json>` for SCORM RTE testing.

## 6. Non-Functional Requirements

- **Performance** — batch `xapi post` supports newline-delimited JSON.
- **Security** — LRS/analytics scope; LRS credentials via file/stdin.
- **Privacy & Compliance** — xAPI statements carry learner identity (FERPA); query results gated by `--yes` for export.
- **Reliability** — post idempotent by statement id; dead-letter retry safe.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a statement file, *When* `xapi post`, *Then* the statement id is returned.
- **AC-2.** *Given* failed statements, *When* `lrs dead-letter retry`, *Then* they re-process.
- **AC-3.** *Given* filters, *When* `xapi query --verb completed`, *Then* matching statements print.

## 8. Data Model

- None client-side. Accept standard xAPI statement JSON.

## 9. API Surface

- `/api/v1/xapi/statements` (post/query); `admin/lrs-config`; `admin/lrs-dead-letter`; `registerEngagementRoutes`; `/api/v1/recommendations/event`; `/api/v1/scorm/rte/{id}/commit`.

## 10. UI / UX

- `lextures xapi ...`, `lextures lrs ...`, `lextures engagement ...`.

## 11. AI / ML Considerations

- Recommendation events feed AI recommenders server-side; CLI only emits.

## 12. Integration Points

- Server LRS/engagement/recommendation handlers; dead-letter (C18); analytics (C28).

## 13. Dependencies & Sequencing

- After: C18 (dead-letter/jobs), C40.
- Before: C28 (events feed insights).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Malformed xAPI rejected opaquely | M | L | Client-side schema hint before post |

## 15. Rollout Plan

- Ship xapi post/query + lrs config/dead-letter first, then engagement/recommendation emit.
- Rollback: additive.

## 16. Test Plan

- **Unit** — statement validation; ND-JSON batch.
- **Integration** — post/query; dead-letter retry.
- **E2E** — post → query round-trip.

## 17. Documentation & Training

- "Test your learning-analytics pipeline" recipe.

## 18. Open Questions

1. Does the LRS support the full xAPI query API or a subset?

## 19. References

- xAPI/SCORM RTE routes in `server.go`; `registerEngagementRoutes`.
- Related: [C18](C18-jobs-scheduler-backups.md), [C25](C25-integrations-webhooks-bots.md), [C28](C28-insights-at-risk.md).

# C28 — Insights, at-risk & classroom signals

> CLI parity plan. Source: `at_risk.go` (`courses/{id}/at-risk`, `admin/at-risk`), `registerInsightsRoutes` (`insights`), `classroom_signals.go`, `registerEngagementRoutes`, `courses/{id}/leaderboard`. Baseline: `clients/cli/cmd/at_risk.go`, `insights.go`, `signals.go`, `insights_at_risk_logic.go`, `insights_at_risk_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C28 |
| **Section** | Reporting & insights |
| **Severity** | MAJOR |
| **Markets** | K12 / HE |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Analytics / CLI |
| **Depends on** | C27, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

At-risk detection, instructor insights and classroom signals — a headline differentiator — are UI-only. Student-success teams cannot pull at-risk cohorts into their outreach/CRM tooling or trigger recomputation, so early-alert workflows can't be automated.

## 2. Goals

- Export at-risk student lists per course/org for outreach automation.
- Pull instructor insights and classroom signals programmatically.
- Trigger recomputation of at-risk models where supported.

## 3. Non-Goals

- Building the risk model (server/ML owns it).
- Case-management/outreach workflow (CRM concern).

## 4. Personas & User Stories

- **As a success coach**, I want `at-risk list --org O --threshold high` to feed my outreach queue.
- **As an instructor**, I want `insights course <course>` for a weekly summary.
- **As an admin**, I want `at-risk recompute --course C` after a grade import.

## 5. Functional Requirements

- **FR-1.** MUST add `at-risk list <course>` and `at-risk list --org <org>` with `--threshold`/`--factor` filters (`at_risk.go`).
- **FR-2.** MUST add `insights course|student <id>` (`registerInsightsRoutes`) and `signals course <course>` (`classroom_signals.go`).
- **FR-3.** SHOULD add `at-risk recompute` / `at-risk factors <course> --user U` (explanation of risk drivers).
- **FR-4.** SHOULD add `--export`/`--out` for at-risk cohorts (CSV for CRM import).

## 6. Non-Functional Requirements

- **Performance** — org-wide at-risk paginated; p95 < 2 s per page.
- **Security** — insights/at-risk scope; 403 → exit 2.
- **Privacy & Compliance** — at-risk data is highly sensitive FERPA (and reputationally) → export gated by `--yes`; access audited.
- **Reliability** — recompute idempotent; safe to re-run.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course, *When* `at-risk list --threshold high --json`, *Then* flagged students emit with scores.
- **AC-2.** *Given* an org, *When* `at-risk list --org O --export --out d`, *Then* a CSV lands (gated by `--yes`).
- **AC-3.** *Given* a course, *When* `insights course`, *Then* an insight summary prints.

## 8. Data Model

- None client-side.

## 9. API Surface

- `at_risk.go` (course + admin); `registerInsightsRoutes`; `classroom_signals.go`; `leaderboard`.

## 10. UI / UX

- `lextures at-risk ...`, `lextures insights ...`, `lextures signals ...`.

## 11. AI / ML Considerations

- Risk scores are ML-derived server-side; CLI reads scores + factor explanations. Surface model/version if provided (ties to C29 AI disclosure).

## 12. Integration Points

- Server at-risk/insights/signals handlers; report exports (C27); events (C26).

## 13. Dependencies & Sequencing

- After: C27 (shared export plumbing), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Sensitive labels mishandled | M | H | `--yes` gate; audit; document responsible-use guidance |
| Recompute expensive | M | M | Async + `--wait`; rate-limit |

## 15. Rollout Plan

- Ship at-risk + insights read/export first, then recompute/factors.
- Rollback: additive.

## 16. Test Plan

- **Unit** — threshold/factor filters; export gating.
- **Integration** — at-risk list; insights shape.
- **Security** — `--yes` gate; scope.
- **E2E** — export at-risk cohort → verify CSV.

## 17. Documentation & Training

- "Feed at-risk cohorts to your outreach tool" recipe + responsible-use note.

## 18. Open Questions

1. Does the server expose risk-factor explanations per student?

## 19. References

- `at_risk.go`, `registerInsightsRoutes`, `classroom_signals.go`.
- Related: [C12](C12-attendance-behavior.md), [C27](C27-reports-exports.md), [C29](C29-compliance-privacy.md).

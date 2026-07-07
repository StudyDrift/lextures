# C10 — Plagiarism & originality

> CLI parity plan. Source: `courses/{id}/plagiarism-settings` (3), `originality_http.go`, `webhooks_originality.go`, `registerFERPARoutes` (integrity adjacency). Baseline: `clients/cli/cmd/plagiarism.go`, `originality.go`, `plagiarism_originality_logic.go`, `plagiarism_originality_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C10 |
| **Section** | Assessment & grading |
| **Severity** | MAJOR |
| **Markets** | HE / K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Integrity / CLI |
| **Depends on** | C03, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Originality/plagiarism configuration and report retrieval are UI-only. Institutions with academic-integrity policies cannot standardize plagiarism settings across courses via script, nor pull originality scores/reports for audit.

## 2. Goals

- Configure per-course plagiarism settings consistently across many courses.
- Trigger originality checks and retrieve scores/reports for submissions.

## 3. Non-Goals

- Implementing a detector (third-party providers do this).
- Adjudicating integrity cases (a human/process concern).

## 4. Personas & User Stories

- **As an integrity officer**, I want `plagiarism settings set --file policy.json` across courses.
- **As an instructor**, I want `originality get <assignment> --user U` to pull a report link/score.
- **As an auditor**, I want `originality list <assignment>` to export scores for a cohort.

## 5. Functional Requirements

- **FR-1.** MUST add `plagiarism settings get|set <course>` (`plagiarism-settings`, `--file`).
- **FR-2.** MUST add `originality status|get <assignment> --user <u>` and `originality list <assignment>`.
- **FR-3.** SHOULD add `originality submit <assignment> --user <u>` to (re)trigger a check.
- **FR-4.** MAY add `originality export <assignment> --out file.csv`.

## 6. Non-Functional Requirements

- **Performance** — list paginated.
- **Security** — integrity/report scope; provider credentials never printed.
- **Privacy & Compliance** — originality reports are FERPA records; export gated by `--yes`.
- **Reliability** — (re)submit idempotent while a check is pending.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a policy file, *When* `plagiarism settings set`, *Then* `get` reflects it.
- **AC-2.** *Given* a checked submission, *When* `originality get`, *Then* score + report reference print.
- **AC-3.** *Given* `originality export` without `--yes`, *Then* it refuses.

## 8. Data Model

- None client-side. Policy JSON schema (PATCH body):

```json
{
  "plagiarismChecksEnabled": true,
  "plagiarismProvider": "turnitin",
  "plagiarismAlertThresholdPct": 25
}
```

`plagiarismProvider` accepts `none`, `turnitin`, `copyleaks`, or `gptzero` (validated client- and server-side).

## 9. API Surface

- `GET|PATCH /api/v1/courses/{c}/plagiarism-settings`
- `GET .../submissions/{id}/originality` (full reports)
- `GET .../originality/summary` (status)
- `GET .../originality/embed-url` (report link)
- `POST .../originality/retry` (submit/retry)
- Inbound `webhooks_originality.go` is server-side, not CLI.

## 10. UI / UX

- `lextures plagiarism settings ...`, `lextures originality ...`.
- Human tables for list; `--json` passthrough for scripting.

## 11. AI / ML Considerations

- Some originality is AI-detection; CLI only reads provider results.

## 12. Integration Points

- Server originality + plagiarism-settings handlers; third-party provider (Turnitin/etc.) via server.
- Internal: `clients/cli/cmd/plagiarism.go`, `originality.go`, `plagiarism_originality_logic.go`.

## 13. Dependencies & Sequencing

- After: C03 (submissions), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Provider variance in report shape | M | M | Normalize to score + report URL; passthrough raw in `--json` |

## 15. Rollout Plan

- Shipped settings get/set, originality status/get/list/submit/export.
- Rollback: additive.

## 16. Test Plan

- **Unit** — policy parse; `--yes` gate (`plagiarism_originality_test.go`).
- **Integration** — httptest for originality get/list shapes.
- **E2E** — set policy → submit → get score (manual / future stack test).

## 17. Documentation & Training

- "Standardize plagiarism policy across a term" recipe:

```bash
# 1. Author a shared policy file in git
cat > policy.json <<'EOF'
{
  "plagiarismChecksEnabled": true,
  "plagiarismProvider": "turnitin",
  "plagiarismAlertThresholdPct": 25
}
EOF

# 2. Apply to each course in a term
for code in CS101 CS102 CS103; do
  lextures plagiarism settings set "$code" --file policy.json
done

# 3. Audit originality scores for an assignment cohort
lextures originality list <item-id> --course CS101
lextures originality export <item-id> --course CS101 --out scores.csv --yes
```

## 18. Open Questions

1. Providers enabled today: `none`, `turnitin`, `copyleaks`, `gptzero`. Report shapes differ per provider; the CLI normalizes to similarity/AI score + report URL and passes raw `reports` in `--json`.

## 19. References

- `originality_http.go`, `webhooks_originality.go`, `course_plagiarism_settings.go`.
- Related: [C03](C03-assignments.md), [C29](../../plan/cli/C29-compliance-privacy.md).
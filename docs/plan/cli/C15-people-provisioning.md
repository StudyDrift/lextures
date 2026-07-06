# C15 — People, provisioning & bulk import

> CLI parity plan. Source: `admin/people` (5), `admin/provisioning` (8), `admin/users`, `admin/students`, `admin/teachers`, `admin-console/imports`, `admin_import.go`, `admin_custom_fields.go`, `platform/people`. Baseline: `users list/get/create/enroll`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C15 |
| **Section** | Admin & governance |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL (users list/get/create/enroll) |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Admin / CLI |
| **Depends on** | C14, C40 |
| **Unblocks** | C11, C16 |

---

## 1. Problem Statement

The CLI can create one user at a time but has no bulk provisioning, no update/deactivate, no custom-field management, and no import-job orchestration. Onboarding a district/university (thousands of users) requires the bulk import pipeline the server already has — which is entirely UI-driven today.

## 2. Goals

- Bulk create/update/deactivate users from CSV with a per-row result summary.
- Drive the server's import-job pipeline (submit → poll → report) from the CLI.
- Manage custom profile fields and user metadata.
- Update/suspend/reactivate individual accounts.

## 3. Non-Goals

- SIS/SCIM continuous sync (see C22) — this is the manual/imperative import layer.
- Enrollment into courses (see C11), though `--enroll` convenience may bridge.

## 4. Personas & User Stories

- **As an admin**, I want `users import --file users.csv` to onboard thousands with a summary.
- **As an admin**, I want `imports submit --file batch.csv` then `imports status <id> --wait` for large jobs.
- **As a help-desk admin**, I want `users update/suspend/reactivate <id>`.
- **As a data admin**, I want `custom-fields create/list` to define profile schema.

## 5. Functional Requirements

- **FR-1.** MUST add `users import --file users.csv` (bulk upsert; `--dry-run`, per-row summary).
- **FR-2.** MUST add `users update <id>`, `users suspend|reactivate <id>`, `users delete <id>`.
- **FR-3.** MUST add `imports submit|status|list <id>` bridging `admin-console/imports` + `admin_import.go` (async jobs, `--wait` via C40).
- **FR-4.** MUST add `custom-fields list|create|update|delete` (`admin_custom_fields.go`).
- **FR-5.** SHOULD add `users search` (server-side) and `people list` (admin/people, platform/people) with rich filters.
- **FR-6.** SHOULD support `--enroll course=role` convenience delegating to C11.

## 6. Non-Functional Requirements

- **Performance** — import chunks + async job path for >N rows; progress via `--wait`.
- **Security** — provisioning scope (elevated); PII handled per policy; secrets (temp passwords) printed only to a `--secrets-out` file, not stdout logs.
- **Privacy & Compliance** — user PII is FERPA/COPPA/GDPR; bulk export gated by `--yes`; import supports minor-consent flags.
- **Reliability** — import idempotent by email/SIS id; resumable via job id.
- **Observability** — summary of created/updated/skipped/failed with row errors.
- **Backward compatibility** — existing `users create` unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a CSV, *When* `users import --dry-run`, *Then* a diff prints and nothing changes.
- **AC-2.** *Given* a large CSV, *When* `imports submit` then `imports status --wait`, *Then* the job completes and a report prints.
- **AC-3.** *Given* a user, *When* `users suspend`, *Then* `get` shows suspended state.

## 8. Data Model

- None client-side. Document user CSV schema (email, name, role, sis_id, org_unit, custom fields).

## 9. API Surface

- `admin/users` CRUD/state; `admin/provisioning`; `admin-console/imports` + `admin_import.go` job endpoints; `admin_custom_fields.go`; `admin/people`/`platform/people` search.

## 10. UI / UX

- Extend `usersCmd`; new `imports` and `custom-fields` groups.
- Import prints summary; `--wait` streams job progress; temp secrets never logged to stdout.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server provisioning/import/custom-field handlers; job queue (`admin_jobs.go`, C18).
- Internal: `clients/cli/cmd/users.go`; new command files.

## 13. Dependencies & Sequencing

- After: C14 (org/org-unit target), C40 (`--wait`, `--file`, `--dry-run`).
- Before: C11 (users exist before enroll), C16.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Temp password exposure | M | H | `--secrets-out` file only; never stdout; redact in `--json` |
| Sync vs async import ambiguity | M | M | Small imports inline; large ones auto-route to job path |

## 15. Rollout Plan

- Ship user update/state + custom-fields, then bulk import, then job orchestration.
- Rollback: additive.

## 16. Test Plan

- **Unit** — CSV parse; secret redaction; idempotency key.
- **Integration** — import job submit/poll; custom-field CRUD.
- **Security** — secrets not in stdout/`--json`; scope 403 → exit 2.
- **E2E** — onboard 1k test users via job path.

## 17. Documentation & Training

- "Bulk onboard users from CSV" and "Run a large import job" recipes.

## 18. Open Questions

1. Threshold where import auto-routes to the async job pipeline?
2. How are temporary credentials returned by the server?

## 19. References

- `clients/cli/cmd/users.go`; `admin_import.go`, `admin_custom_fields.go`, provisioning handlers.
- Related: [C14](C14-org-administration.md), [C11](C11-enrollments-sections.md), [C16](C16-roles-permissions.md), [C22](C22-sis-scim-oneroster.md), [C40](C40-cli-framework.md).

# C11 — Enrollments & sections

> CLI parity plan. Source: `courses/{id}/enrollments` (25), `courses/{id}/sections`, `self-enroll`, `registerCatalogRoutes`, `enrollments` top-level (4), `orgs/{orgId}/cross-list-groups`. Baseline: `users enroll` only.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C11 |
| **Section** | Roster & classroom |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL (`users enroll` single-user) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / CLI |
| **Depends on** | C01, C15, C40 |
| **Unblocks** | C06, C12, C22 |

---

## 1. Problem Statement

Roster management is the #1 CLI use case for an LMS, but the CLI only supports enrolling one user at a time (`users enroll`). There is no bulk enroll/unenroll, no section management, no cross-listing, no roster export, and no enrollment-state changes (conclude, deactivate). Registrars cannot sync rosters from a SIS export or manage sections at scale.

## 2. Goals

- Bulk enroll/unenroll from a CSV roster.
- Full section lifecycle (create, list, move students, cross-list).
- Roster export per course/section for reconciliation.
- Enrollment state transitions (active/invited/concluded/deactivated).

## 3. Non-Goals

- Automated SIS sync scheduling (see C22 SIS/SCIM) — this is the manual/imperative layer.
- User account creation (see C15) — enrollment assumes users exist or references by email.

## 4. Personas & User Stories

- **As a registrar**, I want `enrollments import --file roster.csv` to enroll a whole section.
- **As a registrar**, I want `enrollments export --course C` to reconcile against the SIS.
- **As an admin**, I want `sections create/cross-list` to manage multi-section courses.
- **As an instructor**, I want `enrollments conclude --user U` to drop a student.

## 5. Functional Requirements

- **FR-1.** MUST add `enrollments list <course>` with filters (`--role`, `--section`, `--state`) and `enrollments export`.
- **FR-2.** MUST add `enrollments import <course> --file roster.csv` (bulk create; upsert by email/SIS id; `--role`).
- **FR-3.** MUST add `enrollments add|remove <course> --user <u> --role <r>` and `enrollments set-state` (conclude/deactivate/reactivate).
- **FR-4.** MUST add `sections list|create|update|delete <course>` and `sections move --user U --to <section>`.
- **FR-5.** SHOULD add `sections cross-list` / `orgs cross-list-groups` for shared rosters.
- **FR-6.** SHOULD add `enrollments self-enroll <course>` (enable/generate self-enroll link).

## 6. Non-Functional Requirements

- **Performance** — bulk import chunks requests; handles 1000+ rows with progress.
- **Security** — enroll/manage scope; role escalation prevented (server-enforced).
- **Privacy & Compliance** — roster contains PII (names, emails, SIS ids) → export gated by `--yes`; FERPA note.
- **Reliability** — import idempotent (re-import = no dup enrollments); per-row error summary.
- **Observability** — import prints added/updated/skipped/failed counts.
- **Backward compatibility** — keep `users enroll` as an alias.

## 7. Acceptance Criteria

- **AC-1.** *Given* a 200-row CSV, *When* `enrollments import`, *Then* enrollments are created and a summary prints; re-running creates 0 new.
- **AC-2.** *Given* a course, *When* `enrollments export --json`, *Then* an array of enrollment DTOs is emitted.
- **AC-3.** *Given* a student, *When* `enrollments set-state --state concluded`, *Then* `list` shows concluded.

## 8. Data Model

- None client-side. Document roster CSV schema (email/sis_id, role, section).

## 9. API Surface

- `courses/{c}/enrollments` CRUD/state; `courses/{c}/sections` CRUD + move; `enrollments` top-level; `self-enroll`; cross-list-groups under orgs.

## 10. UI / UX

- New `lextures enrollments ...` and `lextures sections ...` groups; keep `users enroll`.
- CSV interchange; `--json` for automation; progress on bulk.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server enrollment/section handlers; SIS import (C22) shares the CSV schema.
- Internal: new command files.

## 13. Dependencies & Sequencing

- After: C01, C15 (users may need to exist), C40.
- Before: C06/C12/C22 (roster is the substrate).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Enroll-by-email when user absent | H | M | `--create-missing` flag delegating to C15 provisioning |
| Section move semantics vs re-enroll | M | M | Verify server supports move vs drop+add |

## 15. Rollout Plan

- Ship list/export + single add/remove, then bulk import, then sections/cross-list.
- Rollback: additive; `users enroll` preserved.

## 16. Test Plan

- **Unit** — CSV parse; idempotency keys.
- **Integration** — bulk import partial-failure summary; state transitions.
- **E2E** — import roster → export → diff.

## 17. Documentation & Training

- "Sync a section roster from CSV" recipe; note relationship to C22.

## 18. Open Questions

1. Does enroll accept email/SIS id or only user UUID? (Affects `--create-missing`.)
2. Is cross-listing course-level or org-level only?

## 19. References

- `clients/cli/cmd/users.go` (`usersEnrollCmd`); enrollment/section handlers.
- Related: [C15](C15-people-provisioning.md), [C22](C22-sis-scim-oneroster.md), [C06](C06-gradebook-final-grades.md).

# C11 — Enrollments & sections

> CLI parity plan. Source: `courses/{id}/enrollments`, `courses/{id}/sections`, `self-enroll`, `enrollments` top-level, `orgs/{orgId}/cross-list-groups`. Baseline: `clients/cli/cmd/enrollments.go`, `sections.go`, `enrollments_sections_logic.go`, `orgs_cross_list.go`, `enrollments_sections_test.go`; `users enroll` preserved.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C11 |
| **Section** | Roster & classroom |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
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

- **Unit** — CSV parse; idempotency keys; state aliases; FERPA gate (`enrollments_sections_test.go`).
- **Integration** — httptest for list/export/import/set-state/section move.
- **E2E** — import roster → export → diff (manual / future stack test).

## 17. Documentation & Training

- "Sync a section roster from CSV" recipe:

```bash
# 1. Author a roster CSV (email required; sis_id reserved for C22)
cat > roster.csv <<'EOF'
email,role,section
alice@uni.edu,student,SEC-01
bob@uni.edu,student,SEC-01
ta@uni.edu,ta,
EOF

# 2. Bulk enroll (re-run is idempotent: skipped=already enrolled)
lextures enrollments import CS101 --file roster.csv --role student

# 3. Export for SIS reconciliation (FERPA-gated)
lextures enrollments export CS101 --format json --yes

# 4. Conclude a student enrollment
lextures enrollments set-state CS101 --user alice@uni.edu --state concluded

# 5. Cross-list sections under an org
lextures sections cross-list CS101 --org <org-uuid> --primary SEC-01 --name "Combined lecture"
lextures orgs cross-list-groups add-member <org-uuid> <group-uuid> --section <section-uuid>
```

Note relationship to [C22](../../plan/cli/C22-sis-scim-oneroster.md) for automated SIS sync.

## 18. Open Questions

1. Bulk enroll API accepts **email only** (not user UUID). CLI `enrollments add` resolves UUID → email before POST. SIS id column is parsed but rejected until C22 lookup exists.
2. Cross-listing is **org-level** (`/api/v1/orgs/{orgId}/cross-list-groups`); `sections cross-list` is a convenience wrapper.

## 19. References

- `clients/cli/cmd/users.go` (`usersEnrollCmd`); `enrollments.go`, `sections.go`, `enrollments_sections_logic.go`.
- Server: `course_enrollments_http.go`, `course_sections.go`, `enrollment_state_http.go`, `cross_list_http.go`.
- Related: [C15](../../plan/cli/C15-people-provisioning.md), [C22](../../plan/cli/C22-sis-scim-oneroster.md), [C06](C06-gradebook-final-grades.md).

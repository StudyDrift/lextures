# C13 — Groups & collaboration

> CLI parity plan. Source: `courses/{id}/groups` (6), `my-groups`, `collab-docs` (7), `whiteboards` (5), `forums`/`discussion-threads`/`discussion-posts`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C13 |
| **Section** | Roster & classroom |
| **Severity** | MINOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Collaboration / CLI |
| **Depends on** | C11, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Course groups (for group assignments/spaces) and collaborative artifacts (docs, whiteboards, forums) can't be managed from the CLI. Instructors cannot bulk-create groups or auto-assign students, nor export discussion/forum content for archival.

## 2. Goals

- Create and populate course groups in bulk (including random/auto assignment).
- Read/export collaborative artifacts and discussion content for archival.

## 3. Non-Goals

- Real-time collaborative editing (browser).
- Moderation workflows beyond read/export and basic post/lock.

## 4. Personas & User Stories

- **As an instructor**, I want `groups create --set "Project Teams" --auto --size 4` to form teams.
- **As an instructor**, I want `groups add --group G --user U` to adjust membership.
- **As an archivist**, I want `discussions export --course C` for records retention.

## 5. Functional Requirements

- **FR-1.** MUST add `groups list|create|delete <course>` and group-set creation with `--auto`/`--size`.
- **FR-2.** MUST add `groups add|remove <course> --group <g> --user <u>` and `groups members <g>`.
- **FR-3.** SHOULD add `discussions list|export <course>` (forums/threads/posts) and `discussions post|lock`.
- **FR-4.** MAY add `collab-docs list|export` and `whiteboards list|export` (read/backup).

## 6. Non-Functional Requirements

- **Performance** — export streamed for large forums.
- **Security** — group/discussion manage scope.
- **Privacy & Compliance** — student-authored content is FERPA → export gated by `--yes`.
- **Reliability** — auto-assignment deterministic with `--seed` for reproducibility.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* 20 students, *When* `groups create --auto --size 4`, *Then* 5 groups of 4 are formed.
- **AC-2.** *Given* a group, *When* `groups add --user U`, *Then* `members` lists U.
- **AC-3.** *Given* a course, *When* `discussions export`, *Then* threads/posts are written.

## 8. Data Model

- None client-side.

## 9. API Surface

- `courses/{c}/groups` CRUD + membership; `forums`/`discussion-threads`/`discussion-posts`; `collab-docs`/`whiteboards` read.

## 10. UI / UX

- `lextures groups ...`, `lextures discussions ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server group/discussion handlers.

## 13. Dependencies & Sequencing

- After: C11 (membership from roster), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Auto-assign algorithm mismatch with server | M | L | Prefer server-side auto-assign endpoint; CLI just triggers |

## 15. Rollout Plan

- Ship groups CRUD + membership first, then discussion export.
- Rollback: additive.

## 16. Test Plan

- **Unit** — auto-assign sizing with `--seed`.
- **Integration** — membership add/remove; export shape.
- **E2E** — form groups → verify.

## 17. Documentation & Training

- "Auto-form project teams" recipe.

## 18. Open Questions

1. Is auto-assignment server-side or must the CLI compute pairings?

## 19. References

- Group/forum/collab handlers in `server/internal/httpserver`.
- Related: [C11](C11-enrollments-sections.md), [C34](C34-messaging-broadcasts.md).

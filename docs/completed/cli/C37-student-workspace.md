# C37 — Student workspace, notebooks, goals & gamification

> CLI parity plan. Source: `registerStudentNotebookRoutes` + `registerNotebookTaskRoutes` (`me/notebooks`, `me/notebook-tasks`), `registerStudentTodoRoutes` (`me/student-todo-board`), `registerSelfReflectionRoutes` (`me/reflection-journal`), `registerStudyReminderRoutes` (`me/reminder-config`, `me/study-goal`, `me/goals`), `me/reading-preferences`, `registerGamificationRoutes` (`me/gamification`, `courses/{id}/leaderboard`, `me/coaching-tips`, `vibe-activities`). Baseline: `clients/cli/cmd/student_workspace.go`, `student_workspace_logic.go`, `student_workspace_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C37 |
| **Section** | Student experience |
| **Severity** | MINOR |
| **Markets** | SL / HE / K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Learner XP / CLI |
| **Depends on** | C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

The self-directed learner workspace — notebooks, tasks, to-do board, reflection journal, study goals/reminders, reading preferences and gamification — is UI/mobile-only. Power self-learners who live in the terminal, and researchers studying self-regulated learning, cannot capture notes/tasks or read gamification state programmatically.

## 2. Goals

- CRUD notebooks/notes and notebook tasks from the terminal.
- Manage to-do board, reflection journal, study goals and reminders.
- Read gamification state (points, badges, leaderboard) and coaching tips.

## 3. Non-Goals

- Rich note editor UX.
- Designing gamification rules (admin/config concern).

## 4. Personas & User Stories

- **As a self-learner**, I want `notebooks add --file notes.md` to save study notes.
- **As a self-learner**, I want `todo add "read chapter 3" --due tomorrow`.
- **As a self-learner**, I want `goals set --target ...` and `reminders set`.
- **As a learner**, I want `gamification status` and `leaderboard --course C`.

## 5. Functional Requirements

- **FR-1.** MUST add `notebooks list|get|add|update|delete` and `notebook-tasks list|add|complete` (`me/notebooks`, `me/notebook-tasks`).
- **FR-2.** MUST add `todo list|add|complete|remove` (`me/student-todo-board`).
- **FR-3.** SHOULD add `journal list|add` (`me/reflection-journal`), `goals get|set` (`me/goals`, `me/study-goal`), `reminders get|set` (`me/reminder-config`).
- **FR-4.** SHOULD add `gamification status` (`me/gamification`), `leaderboard <course>`, `coaching-tips list`.
- **FR-5.** MAY add `reading-preferences get|set` (`me/reading-preferences`).

## 6. Non-Functional Requirements

- **Performance** — trivial payloads.
- **Security** — self-scope (`me/*`); a user only touches their own workspace.
- **Privacy & Compliance** — personal learner data (FERPA for minors/COPPA); no cross-user access.
- **Reliability** — add/complete idempotent by client id.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a note file, *When* `notebooks add --file`, *Then* a note id returns and `list` shows it.
- **AC-2.** *Given* a task, *When* `todo complete <id>`, *Then* it's marked done.
- **AC-3.** *Given* a learner, *When* `gamification status --json`, *Then* points/badges emit.

## 8. Data Model

- None client-side.

## 9. API Surface

- `me/notebooks`, `me/notebook-tasks`, `me/student-todo-board`, `me/reflection-journal`, `me/goals`/`study-goal`, `me/reminder-config`, `me/gamification`, `courses/{id}/leaderboard`, `me/coaching-tips`, `me/reading-preferences`.

## 10. UI / UX

- `lextures notebooks|todo|journal|goals|reminders|gamification ...`.

## 11. AI / ML Considerations

- Coaching tips may be AI-generated server-side; CLI reads them.

## 12. Integration Points

- Server `me/*` learner-workspace handlers.

## 13. Dependencies & Sequencing

- After: C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Low CLI demand for student UX | M | L | Keep surface thin; prioritize notebooks/todo which suit terminal workflows |

## 15. Rollout Plan

- Ship notebooks + todo first, then journal/goals/reminders, then gamification read.
- Rollback: additive.

## 16. Test Plan

- **Unit** — due-date parsing; idempotency.
- **Integration** — notebook CRUD; todo complete.
- **E2E** — add note + task → verify.

## 17. Documentation & Training

- "Capture study notes from your terminal" recipe.

## 18. Open Questions

1. Are notebooks Markdown or structured blocks on the wire?

## 19. References

- `registerStudentNotebookRoutes`, `registerStudentTodoRoutes`, `registerGamificationRoutes`.
- Related: [C36](C36-tutor-study-buddy.md), [C38](C38-portfolios.md), [C39](C39-profile-account-personas.md).

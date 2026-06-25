# GA-M5 — Section / group / student-scoped runs

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M5 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | MAJOR |
| **Markets** | HE / K12 |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | TA workflows |

## 1. Problem Statement

A run scope is exactly one of `current`, `ungraded`, or `all` (`resolveGraderAgentSubmissions`).
There is no way to grade a **subset** — e.g., "just the submissions in my discussion section," "just
this student group," or "these 5 selected students." In large Higher-Ed courses, grading is divided
across TAs by section. Today a section TA must either grade the whole class (stepping on other TAs) or
grade one submission at a time via "current." This makes the agent impractical for the exact teams
that most need batch grading.

## 2. Goals

- Scope a run to a section, a group set/group, or an explicit selection of submissions.
- Keep the existing scopes; add the subset as an orthogonal filter.
- Respect section/role visibility so a TA only acts on their own students.

## 3. Non-Goals

- Inventing new section/group data — reuse existing sections ([5.4]) and group spaces ([6.6]).
- Per-TA assignment of the agent itself (the agent config stays per-assignment).

## 4. Personas & User Stories

- **As a section TA**, I want to run the agent on just my section, so that I grade only my students.
- **As an instructor**, I want to grade one project group at a time, so that group work is handled coherently.
- **As an instructor**, I want to multi-select submissions and run on those, so that I can re-grade a specific set.

## 5. Functional Requirements

- **FR-1.** The run request MUST accept an optional `filter` of `{ sectionId? , groupId?, submissionIds?[] }` combined with the existing `scope` (ungraded/all act *within* the filter).
- **FR-2.** `resolveGraderAgentSubmissions` MUST intersect the chosen scope with the filter and the caller's visibility.
- **FR-3.** A section TA MUST only be able to target sections/students they can see; the server MUST enforce this, not just the UI.
- **FR-4.** The run summary MUST state the effective target ("Ungraded in Section B: 24 submissions").
- **FR-5.** Selecting zero matching submissions MUST return the existing clear "no submissions matched" error.

## 6. Non-Functional Requirements

- **Performance** — filter pushed into the submission query (no client-side filtering of large sets).
- **Security** — server-side visibility enforcement; section/group membership checked.
- **Privacy & Compliance** — respects blind grading labels in the picker.
- **Accessibility** — scope + filter controls fully keyboard operable; clear current-target summary.
- **Reliability** — deterministic target resolution; idempotent with [GA-B2](bug-2-requeue-double-grading.md) fixes.
- **Observability** — record `filter` on the run for audit.
- **Internationalization** — `gradingAgent.run.filter.*`.
- **Backward compatibility** — no filter ⇒ today's behavior.

## 7. Acceptance Criteria

- **AC-1.** *Given* I pick Section B + ungraded, *when* I run, *then* only ungraded Section B submissions are queued.
- **AC-2.** *Given* I am a TA without access to Section A, *when* I attempt to target Section A, *then* the server rejects it.
- **AC-3.** *Given* I multi-select 5 submissions, *when* I run, *then* exactly those 5 are graded.
- **AC-4.** *Given* a group set, *when* I pick one group, *then* only that group's submissions are graded.
- **AC-5.** *Given* a filter matches nothing, *then* I get the standard no-match message and no run is created.

## 8. Data Model

- No new tables. Reuse sections and group membership repos.
- Persist the run filter on `grading_agent_runs` as `filter JSONB NULL` for audit/repeatability.
- Migration: `server/migrations/NNN_grading_agent_run_filter.sql`.

## 9. API Surface

- `POST …/grader-agent/runs` body gains optional `filter`.
- Submission listing for the picker reuses existing section/group-aware endpoints.

## 10. UI / UX

- Run popover (`run-agent-popover.tsx`) gains a target picker: All / Section ▾ / Group ▾ / Selected.
- A live "target summary" line shows the resolved count before running.
- Submission picker supports multi-select for the "Selected" mode.
- Copy/i18n under `gradingAgent.run.filter.*`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `server/internal/httpserver/grading_agent_http.go` (`resolveGraderAgentSubmissions`, run body, visibility checks).
- `moduleassignmentsubmissions` repo (section/group-filtered listing).
- Sections ([5.4]) and group spaces ([6.6]) repos.
- `clients/web/src/components/annotation/grader-agent/run-agent-popover.tsx`.

## 13. Dependencies & Sequencing

- Independent; pairs naturally with [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md) (review queue can also filter by section).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| TA targets students outside their section | M | H | Server-side visibility enforcement + tests |
| Filter + scope semantics confuse users | M | M | Explicit target summary string before run |

## 15. Rollout Plan

- Flag: `graderAgentRunFilters`.
- Sequence: filtered submission query → run API → picker UI → flip flag.
- Pilot: a multi-TA course.
- Rollback: hide picker; default unfiltered.

## 16. Test Plan

- **Unit** — scope×filter intersection; visibility enforcement.
- **Integration** — section/group/explicit targeting queues exactly the right submissions.
- **Security** — TA cannot target other sections (server-enforced).
- **E2E** — section TA grades only their section.

## 17. Documentation & Training

- Help-center: "Running the grading agent for your section or a group."

## 18. Open Questions

1. Should "Selected" mode persist the selection for repeat runs?
2. Do we expose instructor-only "all sections" explicitly vs the existing `all`?

## 19. References

- `server/internal/httpserver/grading_agent_http.go` (`resolveGraderAgentSubmissions`, `gradableSubmissionsForAgent`).
- `clients/web/src/components/annotation/grader-agent/run-agent-popover.tsx`.
- Related plans: `../completed/05-multi-tenancy-org-roles/5.4-sections.md`, `../completed/06-communication-collaboration/6.6-group-spaces.md`.

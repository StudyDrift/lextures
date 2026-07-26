# CT.4 — Content Tools: Instructor State Console & Per-Enrollment Reset

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.4 |
| **Section** | Content Tools (CT) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend platform + web |
| **Depends on** | CT.1, CT.3 |
| **Unblocks** | CT.7 (shares the roster projection), safe classroom use of every tool story |

---

## 1. Problem Statement

Once tools store per-enrollment state, that state becomes sticky in ways that block teaching. A
student answers a check wrong on the first try and the tool locks; a class rehearses a poll before
the real discussion; a teacher fixes a typo in the correct answer and every prior submission is now
scored against the old key; a student asks to "start the reflection over". Today the only recovery
would be deleting the block, which destroys everyone's work. Instructors need to **see** what a
learner produced in a tool and **reset** it — for one learner, for a group, or for the whole class —
safely, reversibly and auditably.

## 2. Goals

- Give instructors a per-tool roster view: who started, who finished, what they produced, what they scored.
- Ship **reset** at five scopes — one learner × one tool, one learner × one page, one learner × the
  course, all learners × one tool, all learners × one page — each with a dry-run preview.
- Make reset **non-destructive by default**: the prior document is snapshotted and restorable for a
  retention window before it is purged.
- Audit every reset (who, what, when, why, how many rows) and make grade side-effects explicit.
- Let a course optionally allow **students** to reset their own state on tools that permit it.

## 3. Non-Goals

- Editing a learner's state by hand (an instructor can reset, not forge student work). A future story
  may add an explicit, audited override for accessibility cases.
- Class-wide analytics and struggle detection (CT.7) — CT.4 shows facts, not insight.
- Bulk cross-course administration (an admin tool, deliberately out of scope).
- Undo of a *permanent purge* after the retention window.

## 4. Personas & User Stories

- **As an instructor**, I want to reset one student's inline check so that a learner who mis-clicked
  can try again without me deleting the activity.
- **As an instructor**, I want to reset a tool for my whole class after I fixed a wrong answer key so
  that nobody is judged against a mistake I made.
- **As an instructor**, I want to see what a student wrote in a reflection tool so that I can respond
  to it in class.
- **As an instructor**, I want to know before I press the button how many rows a reset will clear so
  that I never destroy work by accident.
- **As a TA**, I want the same reset ability on the sections I teach, and no ability outside them.
- **As a homeschool parent-instructor**, I want to let my learner redo an activity themselves so that
  I am not the bottleneck for a second attempt.
- **As a compliance officer**, I want every reset attributable to a person with a reason so that a
  grade dispute can be reconstructed.
- **As a student**, I want to know my work was reset by my teacher so that I am not confused by a
  suddenly empty activity.

## 5. Functional Requirements

- **FR-1.** The system MUST expose an instructor-only roster read:
  `GET .../content-tools/instances/{instance_id}/states` returning one row per enrolled learner —
  enrollment, display name, status, score, interaction count, last interaction, reset count.
- **FR-2.** The roster MUST include learners with **no** state row (as `not_started`), so "who hasn't
  engaged" is answerable without a client-side join.
- **FR-3.** The system MUST expose a detail read
  `GET .../content-tools/instances/{instance_id}/states/{enrollment_id}` returning the full
  `state_json` plus a tool-provided human-readable summary rendering.
- **FR-4.** The system MUST support reset at these scopes, all through one endpoint with a `scope`
  discriminator: `instance_enrollment`, `instance_all`, `item_enrollment`, `item_all`,
  `course_enrollment`.
- **FR-5.** Every reset MUST accept `dryRun: true` and return the exact affected count and a sample
  of affected learners **without mutating anything**.
- **FR-6.** Reset MUST snapshot the prior state document into `course.content_tool_state_resets`
  before clearing, together with actor, reason, scope and timestamp.
- **FR-7.** Reset MUST clear `state_json` to the tool's declared initial state (or `{}`), set
  `status='not_started'`, null the scores and timestamps, increment `revision` and `reset_count`, and
  set `last_reset_at` / `last_reset_by`.
- **FR-8.** Reset MUST be restorable from a snapshot within the retention window via
  `POST .../state-resets/{reset_id}/restore`, which reinstates the snapshotted document and is itself
  audited.
- **FR-9.** Snapshots MUST be purged after the org's tool-state retention window (default 90 days,
  configurable, floor 7 days) by the nightly sweeper.
- **FR-10.** Reset MUST require the same permission as grading the host item; TAs limited to their
  sections MUST only affect enrollments in those sections.
- **FR-11.** A reset that changes a score which has been passed to the gradebook (CT.7) MUST either
  revert the passed score or refuse with a clear explanation, per the tool's `scoring.mode`; the
  behaviour MUST be stated in the confirmation dialog before the action.
- **FR-12.** Every reset MUST write an `adminaudit` entry and a `content_tool_events` row per affected
  enrollment.
- **FR-13.** Bulk resets over 200 rows MUST execute asynchronously with a job record and progress,
  returning `202` with a poll URL; below the threshold they execute synchronously.
- **FR-14.** Affected learners MUST be notified when their work is reset — in-app notification by
  default, with the instructor able to suppress notification for pre-class housekeeping.
- **FR-15.** When `content_tool_settings.student_reset_allowed` is true **and** the tool's manifest
  sets `allowsSelfReset`, a student MUST be able to reset **their own** state for that instance, with
  the same snapshot and audit path.
- **FR-16.** Resets MUST be idempotent under retry via `idempotencyKey` so a double-clicked bulk reset
  does not create two snapshot generations.
- **FR-17.** The instance-level view MUST offer **export** of all learner state for a tool as CSV/JSON
  (using the tool's summary projection), for gradebook reconciliation and record-keeping.

## 6. Non-Functional Requirements

- **Performance** — Roster read p95 ≤ 200 ms for a 300-learner course (one query, paginated 50/page).
  Sync reset ≤ 200 rows in ≤ 500 ms; async path processes ≥ 2,000 rows/s.
- **Security** — Every route is instructor-gated and course-scoped; section-limited TAs are filtered
  in SQL. Reading `state_json` for another learner requires the grade-read permission, is logged, and
  is denied for observers/parents outside their own child's enrollment.
- **Privacy & Compliance** — Reading student work is a FERPA-relevant disclosure: access logs feed the
  existing FERPA access-log surface. Snapshots are education records and are included in DSAR export
  (S01) and deletion (S02). Retention default 90 days is documented in the DPIA (S06).
- **Accessibility** — Roster is a semantic table with sortable headers and screen-reader-announced
  sort state; destructive dialogs follow the shipped confirm pattern with focus trap and explicit
  labels; progress for async resets is announced politely.
- **Scalability** — Snapshots are append-only and partitionable by month if volume demands; the reset
  job reuses the existing background job queue.
- **Reliability** — Snapshot-then-clear runs in one transaction per batch; a failed batch rolls back
  and the job resumes from the last committed batch. Restore is transactional.
- **Observability** — `lextures_content_tool_resets_total{tool_id,scope,actor_role}`,
  `…_reset_rows_total{scope}`, `…_reset_restores_total`, `…_reset_job_duration_seconds`. Alert on any
  single actor resetting > 1,000 rows in an hour (accident or abuse signal).
- **Maintainability** — One reset service function with a scope enum; no per-tool reset code. A tool
  needing special cleanup (external side effects) implements an optional `OnReset` hook.
- **Internationalization** — All dialog copy, notification templates and CSV headers localized.
- **Backward compatibility** — Additive. `reset_count` already exists from CT.1.

## 7. Acceptance Criteria

- **AC-1.** *Given* a 30-learner course where 12 have state, *When* the instructor opens the tool
  roster, *Then* 30 rows are returned with 18 marked `not_started`.
- **AC-2.** *Given* a dry-run reset at scope `instance_all`, *When* it returns, *Then* the response
  reports 12 affected rows and `SELECT count(*) FROM content_tool_state_resets` is unchanged.
- **AC-3.** *Given* a real reset at scope `instance_enrollment`, *When* it completes, *Then* that
  learner's state is the tool's initial document, `reset_count` is 1, a snapshot row exists, and no
  other learner's row changed.
- **AC-4.** *Given* a reset snapshot inside the retention window, *When* the instructor restores it,
  *Then* the learner's prior document, status and score are reinstated and the restore is audited.
- **AC-5.** *Given* a TA limited to section B, *When* they attempt `instance_all`, *Then* only section
  B enrollments are affected and the response states the applied scope narrowing.
- **AC-6.** *Given* a tool with `scoring.mode = "auto"` whose score was passed to the gradebook, *When*
  a reset runs, *Then* the passed score is reverted in the same transaction and the gradebook history
  shows the reversal.
- **AC-7.** *Given* a bulk reset over 200 rows, *When* it is requested, *Then* the API returns 202 with
  a job id and the job completes with a per-enrollment event row for every affected learner.
- **AC-8.** *Given* the same `idempotencyKey` is submitted twice, *Then* the second call returns the
  first result and creates no additional snapshots.
- **AC-9.** *Given* notifications are not suppressed, *When* a learner's state is reset, *Then* that
  learner receives one in-app notification naming the activity.
- **AC-10.** *Given* `student_reset_allowed=false`, *When* a student calls the self-reset route,
  *Then* the API returns 403 and nothing changes.
- **AC-11.** *Given* an instructor opens another learner's tool detail, *Then* a FERPA access-log
  entry is written naming the viewer, the learner and the activity.
- **AC-12.** *Given* snapshots older than the retention window, *When* the nightly sweeper runs,
  *Then* they are deleted and the corresponding audit entries are retained.

## 8. Data Model

Migration `server/migrations/452_content_tool_resets.sql` (+ `.down.sql`).

```sql
-- 452_content_tool_resets.sql

-- Snapshot of a learner's tool state immediately before a reset. Restorable within retention.
CREATE TABLE IF NOT EXISTS course.content_tool_state_resets (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id      UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    enrollment_id    UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    course_id        UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    tool_id          TEXT NOT NULL,
    scope            TEXT NOT NULL CHECK (scope IN
                       ('instance_enrollment','instance_all','item_enrollment','item_all',
                        'course_enrollment','self')),
    reason           TEXT,
    prior_state_json JSONB NOT NULL,
    prior_status     TEXT NOT NULL,
    prior_score_raw  NUMERIC(10,4),
    prior_score_max  NUMERIC(10,4),
    prior_revision   BIGINT NOT NULL,
    batch_id         UUID,                    -- groups one bulk operation
    reset_by         UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reset_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    restored_at      TIMESTAMPTZ,
    restored_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    purge_after      TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ctsr_instance_enrollment
    ON course.content_tool_state_resets (instance_id, enrollment_id, reset_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctsr_batch ON course.content_tool_state_resets (batch_id);
CREATE INDEX IF NOT EXISTS idx_ctsr_purge ON course.content_tool_state_resets (purge_after);

-- Async bulk-reset jobs (mirrors the shipped job-record pattern).
CREATE TABLE IF NOT EXISTS course.content_tool_reset_jobs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    requested_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    scope          TEXT NOT NULL,
    target_json    JSONB NOT NULL,            -- {instanceId?, itemId?, enrollmentId?, sectionIds?}
    reason         TEXT,
    notify         BOOLEAN NOT NULL DEFAULT TRUE,
    status         TEXT NOT NULL DEFAULT 'queued'
                     CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
    total_rows     INTEGER NOT NULL DEFAULT 0,
    processed_rows INTEGER NOT NULL DEFAULT 0,
    error          TEXT,
    idempotency_key TEXT UNIQUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ctrj_course_created ON course.content_tool_reset_jobs (course_id, created_at DESC);

-- Org-level retention for snapshots (default 90 days).
ALTER TABLE tenant.organizations
    ADD COLUMN IF NOT EXISTS content_tool_state_retention_days INTEGER NOT NULL DEFAULT 90;
COMMENT ON COLUMN tenant.organizations.content_tool_state_retention_days IS
    'Days a Content Tools reset snapshot remains restorable before nightly purge (plan CT.4 FR-9).';
```

**Backfill** — none. **Purge** — nightly sweeper deletes rows where `purge_after < NOW()`.

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `.../content-tools/instances/{instance_id}/states?page=&status=&sectionId=` | instructor / grade-read |
| `GET` | `.../content-tools/instances/{instance_id}/states/{enrollment_id}` | instructor / grade-read |
| `GET` | `.../content-tools/instances/{instance_id}/states/export?format=csv\|json` | instructor |
| `POST` | `.../content-tools/state-resets` | instructor (scope-checked) |
| `GET` | `.../content-tools/state-resets?instanceId=&enrollmentId=` | instructor |
| `POST` | `.../content-tools/state-resets/{reset_id}/restore` | instructor |
| `GET` | `.../content-tools/reset-jobs/{job_id}` | instructor |
| `POST` | `.../content-tools/instances/{instance_id}/self-reset` | student (when permitted) |

```ts
type ResetRequest = {
  scope: 'instance_enrollment' | 'instance_all' | 'item_enrollment' | 'item_all' | 'course_enrollment'
  instanceId?: string
  itemId?: string
  enrollmentId?: string
  sectionIds?: string[]
  reason?: string
  notify?: boolean          // default true
  dryRun?: boolean          // default false
  idempotencyKey?: string
}

type ResetResponse = {
  dryRun: boolean
  affectedCount: number
  sample: Array<{ enrollmentId: string; displayName: string; status: string; score: number | null }>
  batchId?: string          // present on real, synchronous resets
  jobId?: string            // present when the operation was queued (202)
  gradeEffects: Array<{ enrollmentId: string; action: 'reverted' | 'unchanged' | 'blocked'; reason?: string }>
}
```

- **Rate limits** — resets 20/min/user; dry-runs 60/min/user; export 5/min/user.
- **OpenAPI** — all routes documented including the 202 job shape.

## 10. UI / UX

**New components** under `clients/web/src/components/content-tools/instructor/`:
`tool-roster-table.tsx`, `tool-state-detail-drawer.tsx`, `tool-reset-dialog.tsx`,
`reset-scope-picker.tsx`, `reset-history-list.tsx`, `reset-job-progress.tsx`.

**Entry points** (three, because instructors arrive from three different mental models):

1. **From the page** — while viewing a content page as an instructor, each `ToolFrame` gains a
   *Responses* affordance opening the roster for that instance.
2. **From the course** — *Course → Insights → Content Tools* lists every instance with engagement
   counts and a reset action.
3. **From the learner** — *People → {student} → Activity* lists that learner's tool state across the
   course with a per-row reset and a "reset everything in this course" action.

**Reset dialog flow** — pick scope → dry-run runs automatically and shows "*This will clear work for
**12 students**. Their prior answers can be restored for 90 days.*" → optional reason → notification
toggle → grade-effects list when scoring is involved → type-to-confirm for scopes affecting > 25
learners → execute → success toast with **Undo** (calls restore for the batch) for 30 s → batch also
listed in reset history.

**States** — *Empty*: "No one has started this activity yet." *Loading*: table skeleton. *Error*:
inline retry that does not close the dialog. *Async*: progress bar with cancel. *Partial failure*:
per-row error list with retry-failed-only.

**Mobile / responsive** — roster collapses to cards; the dialog becomes a bottom sheet; type-to-confirm
retained on all breakpoints.

**Accessibility annotations** — table with `<caption>`, `scope="col"`, `aria-sort`; dialog with
`role="alertdialog"`, focus trap, initial focus on the cancel action for destructive scopes; progress
with `role="progressbar"` and polite updates; Undo toast is focusable and keyboard-dismissible.

**Copy & i18n** — `contentTools.instructor.*` and `contentTools.reset.*`; notification templates in
the mail/notification catalogue.

## 11. AI / ML Considerations

No model calls. One deliberate interaction: resetting an AI tool (CT.10, CT.20) clears the stored
transcript, which is also the record of what was sent to a provider. The snapshot preserves it for the
retention window so a safety investigation is still possible after a teacher resets a conversation —
a requirement CT.8 depends on. Purging the snapshot also purges that copy, which is the intended
privacy behaviour.

## 12. Integration Points

- **Internal** — `service/contenttools/reset.go`, `repos/contenttools/`,
  `httpserver/content_tools_instructor.go`, `service/adminaudit`, `service/ferpa` (access logging),
  `service/notifications`, `internal/background` (job worker), `service/grading` (score reversal),
  `service/gdpr` (DSAR inclusion of snapshots).
- **Course roles** — `internal/courseroles` for section-limited TA scoping.
- **Events** — one `content_tool_events` row per affected enrollment (`event_type='state_reset'`).

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1, CT.3 (state must exist to reset).
- **Must ship before:** classroom pilots of any tool; CT.7 reuses the roster projection.
- **Shared infra needed:** background job queue, notification service, audit log.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Instructor mass-resets a class by accident | M | H | Mandatory dry-run preview, type-to-confirm above 25 learners, 30 s Undo, restorable snapshots |
| Snapshots become a second uncontrolled copy of student data | M | M | Fixed retention with org control, DSAR/deletion inclusion, documented in the DPIA |
| Reset diverges from the gradebook | M | H | Score reversal in the same transaction; blocked with an explanation when the item is locked/posted |
| Bulk reset locks rows during class | L | M | Batched (500 rows) with short transactions, async above 200 rows |
| TA scope leakage across sections | M | H | Scoping enforced in SQL, covered by an explicit authz matrix test |
| Students confused by silently emptied activities | H | M | Notification by default; the tool shows "Reset by your instructor on {date}" until next interaction |

## 15. Rollout Plan

- **Feature flag** — inherits `content_tools_enabled`. Async path behind
  `CONTENT_TOOLS_ASYNC_RESET_THRESHOLD` (default 200) for tuning without deploys.
- **Sequencing** — migration `452_*` → reset service + routes → roster UI → dialog + Undo → job worker
  → notifications.
- **Dogfood** — instructors on the pilot course perform at least one of each scope before GA.
- **GA criteria** — zero unintended-reset reports in dogfood; restore verified end-to-end; audit
  entries complete for 100% of resets in a sampled week.
- **Rollback** — hide the reset UI (flag), keep roster reads; snapshots remain valid for restore.

## 16. Test Plan

- **Unit** — scope resolution to enrollment sets; TA narrowing; grade-effect classification; retention
  computation; idempotency handling; initial-state derivation from a manifest.
- **Integration** — dry-run mutates nothing; each of the five scopes affects exactly the right rows;
  snapshot/restore round-trip; cascade behaviour when an instance or enrollment is deleted mid-job;
  purge sweeper.
- **End-to-end** — Playwright: roster → detail → reset one → verify student sees an empty tool and a
  notification; class reset with type-to-confirm and Undo; async job with progress.
- **Security** — authz matrix (student/TA-in-section/TA-out-of-section/instructor/observer/parent ×
  every route); self-reset when disallowed; cross-course instance ids; FERPA access-log assertions.
- **Accessibility** — axe on roster and dialog; screen-reader script for the destructive flow; keyboard
  path for Undo before the toast expires.
- **Performance / load** — reset of 5,000 rows completes < 10 s and holds no transaction > 200 ms.
- **Manual exploratory** — reset while the student is mid-interaction (their next save must 409 and
  recover); reset during an async job; restore after the instance was archived.

## 17. Documentation & Training

- **End-user (student)** — "Why did my activity reset?" help article.
- **Instructor** — reset scopes explained with a decision table; when to reset vs. archive vs. delete;
  how Undo and restore differ; what students are told.
- **Admin** — configuring `content_tool_state_retention_days`; where reset audit entries appear.
- **API reference** — reset, restore and job routes.
- **Runbook** — cancelling a runaway job; restoring a batch after the Undo window; investigating an
  alert for a large-volume reset.

## 18. Open Questions

1. Should `course_enrollment` scope (reset *everything* for one learner) also clear AI transcripts in
   tools whose value is cumulative (CT.10)? Proposed: yes, but call it out explicitly in the dialog.
2. Should the Undo window be 30 s (toast) or until the instructor leaves the page? Proposed: 30 s
   toast plus permanent restore from reset history — two affordances, one mechanism.
3. Does a posted/locked gradebook item block reset entirely, or reset the tool while leaving the grade?
   Proposed: block with an explanation and a link to unlock — silent divergence is worse.
4. Should students always be notified, or only when the reset was not initiated in class? Proposed:
   always notify by default; the suppression toggle covers the in-class case.

## 19. References

- Existing files this work touches: `server/internal/service/adminaudit/`,
  `server/internal/service/ferpa/`, `server/internal/service/notifications/`,
  `server/internal/courseroles/`, `server/internal/background/`,
  `server/migrations/452_content_tool_resets.sql`.
- Precedents followed: quiz attempt reset / "allow another attempt" semantics
  (`service/quizattempt`); moderated-grading audit patterns (`service/moderatedgrading`).
- External standards: FERPA §99.31 access logging expectations; RFC 2119.
- Related plans: [CT.1](CT.1-foundations-registry-and-data-model.md),
  [CT.3](CT.3-student-runtime-and-state-persistence.md),
  [CT.7](../plan/content_tools/CT.7-analytics-insights-and-gradebook.md),
  [CT.8](../plan/content_tools/CT.8-governance-safety-privacy-accessibility.md),
  [S01 DSAR](../../plan/standards/S01-unified-data-subject-rights-orchestration.md),
  [S02 retention](../../plan/standards/S02-data-retention-deletion-engine.md).

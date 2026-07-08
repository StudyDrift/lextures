# IC02 — Automatic Student Enrollment on Account Creation + Backfill

> Implementation plan. Source: product direction — *"the first course that all new users are
> automatically enrolled in as a student."* Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC02 |
| **Section** | Intro Course |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | IC01 (course + system instructor + flag) |
| **Unblocks** | IC05 (completion), IC06 (surfaces have someone to show) |

---

## 1. Problem Statement

The intro course only matters if learners are actually in it. Lextures creates users through
many paths — password signup, magic link, SAML/OIDC SSO, Clever/ClassLink rostering,
Canvas-import provisioning, admin bulk CSV import — and there is no single place that guarantees
a new user becomes a student in the intro course. This plan adds one idempotent enrollment hook,
invoked from every user-creation path, plus a one-time backfill for users who predate the flag,
all gated by `intro_course_enabled` and safe to re-run.

## 2. Goals

- Enroll **every newly created user** as a `student` in the intro course, regardless of creation
  path, the moment the account is committed.
- Do it **idempotently** — re-running never creates duplicate enrollments (rely on the existing
  `UNIQUE (course_id, user_id)` constraint).
- **Backfill** existing users once when the flag is first enabled, via a resumable scheduled job.
- Never block or fail account creation because of intro-course enrollment (best-effort, retried).
- Respect the flag and role rules: skip parents/observers, system users, and instructors-only
  where a student enrollment is inappropriate (see FR-4).

## 3. Non-Goals

- Provisioning the course or system instructor (IC01).
- Course content, grading, or completion (IC03/IC04/IC05).
- Un-enrolling users when the flag is disabled (disable stops *new* enrollment; existing rows
  are retained — IC01 FR-6).
- Per-org targeting / cohorting of who gets enrolled (IC08 open question).

## 4. Personas & User Stories

- **As a new student**, I want the intro course already waiting on my dashboard when I first log
  in, so I have an obvious first thing to do.
- **As an admin who bulk-imports 500 students**, I want them all enrolled in the intro course
  without any extra step, so onboarding is uniform.
- **As an existing user** (account predates the feature), I want to be enrolled once when the
  feature turns on, so I'm not excluded.
- **As a platform engineer**, I want one enrollment function called everywhere, so no creation
  path is silently missed.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `introcourse.EnsureEnrollment(ctx, exec, userID)` that, when
  the flag is on and the course exists, inserts a `course_enrollments` row
  (`role='student'`) for the user, `ON CONFLICT (course_id, user_id) DO NOTHING`. It MUST accept
  a `pgx.Tx` so it can join the user-creation transaction where callers want atomicity.
- **FR-2.** Every user-creation path MUST call it: password `Signup`
  (`authservice/credentials.go`), magic-link first provisioning, SAML/OIDC provisioning,
  Clever/ClassLink provisioning (`repos/user/clever.go`), Canvas-import user provisioning
  (`InsertUserInOrgTx`), admin bulk CSV import, and any parent/child account creation.
- **FR-3.** Enrollment MUST NOT abort account creation. Where it runs in the signup transaction it
  MUST be non-fatal (log + metric on failure, commit the user anyway) — preferably enqueue a job
  rather than fail the commit. Where the path is async (import), enroll inline or enqueue.
- **FR-4.** The system MUST NOT enroll: the IC01 system instructor(s); `account_type='system'`;
  users the deployment marks as `parent`/observer (configurable — default skip parents); and MUST
  NOT downgrade a user who is somehow already the course instructor.
- **FR-5.** A **backfill** MUST run once when `intro_course_enabled` transitions to on (or on the
  first deploy with the flag defaulting on): a resumable batched job enrolling all existing
  eligible users, idempotent and safe to re-run, throttled to avoid gradebook/DB spikes.
- **FR-6.** Enrollment MUST also grant whatever per-enrollment course grants the normal
  student-enroll path grants (mirror `handleCourseEnrollmentsSelfStudent` / the enrollment repo),
  so intro-course students have identical capabilities to any other student.
- **FR-7.** The backfill and hook MUST be observable: counts of enrolled/skipped/failed, and a
  backfill progress cursor persisted so a crash resumes rather than restarts.
- **FR-8.** When the flag is **off**, both the hook and the backfill MUST be no-ops.

## 6. Non-Functional Requirements

- **Performance** — Inline hook adds ≤ 5 ms to account creation (single indexed insert). Backfill
  processes ≥ 5 k users/min in batches of ~500 with short transactions; MUST NOT hold long locks.
- **Security** — Enrollment is `student` only; no privilege escalation. Backfill runs as a system
  job with row-scoped writes. No cross-tenant enrollment (course is in `default` org; users in
  other orgs handled per IC08 OQ).
- **Privacy & Compliance** — Enrollment is an education record; it is included in the user's
  FERPA/GDPR export and removed on erase via the existing `ON DELETE CASCADE` on
  `course_enrollments.user_id`. No new PII.
- **Accessibility** — No UI.
- **Scalability** — Backfill is batched, resumable, throttled; hook is O(1) per user.
- **Reliability** — At-least-once with idempotent upsert (unique constraint) — a retried job or a
  double-fired hook is harmless. Failure of enrollment never rolls back the user.
- **Observability** — `intro_course_enroll_total{path,result}` (path = signup|sso|clever|canvas|
  admin_import|backfill), `intro_course_backfill_progress` gauge, `intro_course_backfill_remaining`
  gauge. Log at INFO on backfill start/finish, DEBUG per batch.
- **Maintainability** — One function, called at each site; no per-path re-implementation. Hook
  sites listed in a doc comment so new creation paths are reminded to call it.
- **Internationalization** — n/a (no user-facing strings).
- **Backward compatibility** — Additive; existing enrollments untouched. Backfill guarded to run
  at most once per flag-enable via a persisted marker.

## 7. Acceptance Criteria

- **AC-1.** *Given* the flag is on, *when* a user signs up with email/password, *then* a
  `course_enrollments` row with `role='student'` for the intro course exists immediately after
  signup.
- **AC-2.** *Given* the same user, *when* the hook fires again (retry, re-login provisioning),
  *then* no duplicate enrollment is created (unique constraint upsert).
- **AC-3.** *Given* users created via SSO, Clever, Canvas import, and admin bulk import, *when*
  each account is created, *then* each is enrolled as a student in the intro course.
- **AC-4.** *Given* the intro-course enrollment insert fails (e.g. transient DB error) during
  signup, *when* signup completes, *then* the user account is still created (enrollment is
  retried via job; failure recorded in metrics).
- **AC-5.** *Given* a database of N existing eligible users and the flag turning on, *when* the
  backfill runs, *then* all N are enrolled exactly once; re-running the backfill enrolls 0 more.
- **AC-6.** *Given* the system instructor and a parent account, *when* creation/backfill runs,
  *then* neither is enrolled as a student.
- **AC-7.** *Given* the flag is off, *when* a user is created, *then* no intro-course enrollment
  is made.

## 8. Data Model

No new tables. Uses `course.course_enrollments` (IC01 course id, `role='student'`).

```sql
-- server/migrations/371_intro_course_backfill_state.sql  (renumber on merge)
-- Tracks one-time backfill so a flag flip doesn't re-scan every deploy and a crash resumes.
CREATE TABLE settings.intro_course_backfill (
    id             BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),   -- singleton row
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    last_user_id   UUID,            -- resume cursor (ordered by user id)
    enrolled_count BIGINT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Backfill query shape (batched, ordered by id for a stable cursor):

```sql
SELECT id FROM "user".users
WHERE id > $cursor
  AND account_type NOT IN ('system')          -- and 'parent' per FR-4 config
  AND id NOT IN (SELECT user_id FROM course.course_enrollments WHERE course_id = $introId)
ORDER BY id LIMIT 500;
```

## 9. API Surface

No new learner endpoints. One admin/ops trigger (defined in IC08, referenced here):

```
POST /api/v1/admin/intro-course/backfill    Auth: platform-admin
  → starts/resumes backfill; 202 { startedAt, remaining }
GET  /api/v1/admin/intro-course/backfill    Auth: platform-admin
  → { startedAt, completedAt, enrolledCount, remaining }
```

Internal:

```go
// server/internal/service/introcourse/enroll.go
func EnsureEnrollment(ctx context.Context, exec Execer, userID uuid.UUID) error // idempotent, flag-gated
func RunBackfill(ctx context.Context, pool *pgxpool.Pool) error                  // resumable, throttled
```

`Execer` is satisfied by both `*pgxpool.Pool` and `pgx.Tx` so callers choose atomicity.

## 10. UI / UX

None directly. The *result* — the intro course appearing on the dashboard — is IC06. Optionally,
IC06 shows a "You've been enrolled in Welcome to Lextures" toast on first login after backfill.

## 11. AI / ML Considerations

None.

## 12. Integration Points

- **Creation paths (call sites):** `server/internal/service/authservice/credentials.go`
  (`Signup`, magic-link, SSO provisioning), `server/internal/repos/user/clever.go`,
  `server/internal/repos/user/user.go` (`InsertUserInOrgTx` — Canvas import), admin bulk import
  (`server/internal/httpserver/admin_import.go` / `userimport`), parent account creation.
- **Enrollment repo:** `server/internal/repos/course/` + `coursegrants/` (mirror
  `handleCourseEnrollmentsSelfStudent` grant grants).
- **Job queue / scheduler:** `server/internal/background/` (`jobqueue_*`, `scheduled_jobs.go`) for
  the retry job and the backfill runner.
- **Flag / course identity:** `introcourse.Service` from IC01.
- **Telemetry:** `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **After:** IC01 (needs course id + flag + system-user exclusion).
- **Before:** IC05 (completion needs students), IC06 (surfaces need enrolled students).
- **Shared infra:** job queue, scheduler — present (see modified `background/` files in the
  current working tree).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A creation path is missed → some users never enrolled | M | M | Single function + doc-commented call-site list; backfill is a safety net that also catches gaps on next run |
| Backfill spikes DB / gradebook recompute | M | M | Batched (500), throttled, short txns, off-peak schedule, resumable cursor |
| Enrollment failure rolls back signup | L | H | Non-fatal in tx / enqueue job; AC-4 test asserts user still created |
| Double-enroll from concurrent hook + backfill | M | L | `ON CONFLICT DO NOTHING` on unique constraint |
| Parents/observers wrongly enrolled as students | M | L | Explicit account_type skip (FR-4), configurable |

## 15. Rollout Plan

- **Flag:** reuse `intro_course_enabled` (IC01); no separate flag.
- **Sequencing:** land hook (inline + job) → verify per-path enrollment on staging → run backfill
  on staging → enable in prod → run/monitor prod backfill off-peak.
- **Dogfood:** internal org signups + a synthetic bulk import verify each path.
- **GA criteria:** all creation paths covered by tests; backfill completes on a full-size staging
  DB within window; zero duplicate enrollments.
- **Rollback:** disable flag → hook/backfill become no-ops. Existing enrollments retained. To
  fully undo, a targeted delete of intro-course student enrollments (documented, not automatic).

## 16. Test Plan

- **Unit** — `EnsureEnrollment` idempotency; flag-off no-op; account_type skip; grant parity.
- **Integration (DB)** — each creation path enrolls exactly one student row; retry → no dup;
  enrollment failure path leaves user committed; backfill enrolls all eligible, resumes from
  cursor after simulated crash, skips already-enrolled and system/parent users.
- **End-to-end** — signup via web → intro course appears in `GET /api/v1/courses` (enrolled list).
- **Security** — role is always `student`; no cross-tenant enroll; backfill requires admin to
  trigger.
- **Performance / load** — backfill throughput on N=100k synthetic users within target window,
  DB p95 unaffected.
- **Manual exploratory** — toggle flag off mid-backfill (job halts cleanly, resumes on re-enable).

## 17. Documentation & Training

- Update `docs/runbooks/intro-course.md`: trigger/monitor backfill, interpret enroll metrics,
  add a new creation path (call `EnsureEnrollment`).
- Admin doc: "new users are auto-enrolled; existing users backfilled once."
- OpenAPI: backfill admin endpoints.

## 18. Open Questions

1. Users in non-`default` orgs (multi-tenant): enroll them into the shared `default`-org course,
   or require an org-scoped variant first (IC08)? (v1 leaning: enroll only same-org users;
   others deferred to IC08.)
2. Should parents/observers get a *read-only* intro (a parent-oriented variant) rather than being
   skipped? (Deferred; skip for v1.)
3. Backfill trigger: automatic on first flag-on deploy, or admin-initiated only? (Leaning:
   auto-start once, admin can re-trigger.)
4. Should enrollment be transactional with signup (atomic) or always enqueued (never blocks)?
   (Leaning: enqueue for external paths, best-effort inline for signup with job fallback.)

## 19. References

- Existing files: `server/internal/service/authservice/credentials.go` (Signup + provisioning),
  `server/internal/repos/user/user.go`, `.../clever.go`,
  `server/internal/httpserver/courses_routes.go` (`handleCourseEnrollmentsSelfStudent`),
  `server/internal/background/scheduled_jobs.go`, `server/migrations/007_course_enrollments.sql`.
- Related plans: [IC01](IC01-foundation-provisioning-flag.md),
  [IC05](IC05-progress-completion-credential.md), [IC08](IC08-admin-governance-localization.md).

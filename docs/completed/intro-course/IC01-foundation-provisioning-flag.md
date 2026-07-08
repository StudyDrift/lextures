# IC01 — Intro Course Foundation: Provisioning, System Instructor & Feature Flag

> Implementation plan. Source: product direction — *"We need a good introduction course into
> Lextures … the first course that all new users are automatically enrolled in as a student …
> a feature flag that can be disabled by global platform feature flags, on by default."*
> Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC01 |
| **Section** | Intro Course |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | Courses/enrollment/gradebook (shipped), feature-flag stack (shipped) |
| **Unblocks** | IC02, IC03, IC04, IC05, IC06, IC07, IC08 |

---

## 1. Problem Statement

There is no canonical, platform-owned course to introduce new users to Lextures, and no
substrate to build one on that is safe to redeploy. Before any content, enrollment, grading, or
UX work can happen, the platform needs a single idempotently-provisioned course row, a
system-owned instructor identity to own it, and a global feature flag — on by default,
admin-disableable — that gates the entire epic. This plan builds exactly that: a real but
initially content-light course ("Welcome to Lextures"), a "Lextures Guide" system user enrolled
as its instructor, and the `intro_course_enabled` flag wired end-to-end. It ships nothing
student-visible on its own; IC02 enrolls students and IC03 fills the curriculum.

## 2. Goals

- Provision **exactly one** canonical intro course (`short_code = "LEX-WELCOME"`), idempotently,
  so repeated startups/deploys never create duplicates.
- Create a dedicated **system instructor** ("Lextures Guide") that owns the course, so it is
  never attributed to a real user and never counts against seat licenses.
- Wire the `intro_course_enabled` feature flag through the full stack (DB column → `config` →
  `platformconfig` read/patch → `/platform-features` API → web/mobile config), **default true**.
- Expose a stable internal Go interface (`introcourse.Service`) that IC02–IC08 call, so no other
  package hard-codes the course's identity or existence checks.
- Define the disable semantics: flag off = no new auto-enroll + hidden from discovery, but
  **no destruction** of enrollments, grades, submissions, or content.

## 3. Non-Goals

- Enrolling students (IC02) or backfilling existing users (IC02).
- Authoring any module/page/quiz/assignment content (IC03).
- Grading behaviour (IC04), completion/credential (IC05).
- Any web/mobile UI beyond the admin flag toggle already rendered by the platform-features panel
  (IC06/IC07 own learner UI).
- Per-org or per-tenant *variants* of the course (IC08 discusses; v1 is one shared course).

## 4. Personas & User Stories

- **As a platform admin**, I want a single global switch to enable/disable the intro course, so
  I can turn it off for a deployment (e.g. a district that provides its own onboarding).
- **As a platform engineer**, I want one `introcourse.Service` that guarantees the course
  exists and returns its ID, so enrollment/content/UX code never re-implements the lookup or
  risks creating a second course.
- **As a security reviewer**, I want the course owned by a non-login system identity, so there
  is no shared human account and no seat consumed.
- **As an operator**, I want provisioning to be idempotent and self-healing, so a redeploy or a
  restored backup converges to exactly one correct course.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain exactly one course identified by the immutable
  `short_code = "LEX-WELCOME"`. Provisioning MUST be idempotent: if the row exists it is
  updated in place (title/description/settings reconciled), never duplicated.
- **FR-2.** Provisioning MUST run automatically at server startup (behind the flag) and MUST be
  invokable on demand via an admin endpoint (IC08) and a CLI/`make` target, all converging to
  the same state.
- **FR-3.** The system MUST create/ensure a **system instructor** user "Lextures Guide" with a
  fixed UUID (`a0000000-0000-4000-8000-000000000002`), no usable password, `account_type` marking
  it non-human, excluded from seat counting, people directories, and analytics "active user"
  metrics. It MUST be enrolled in the course with `role='teacher'`.
- **FR-4.** The course MUST be created `published = true`, `visible_from = now()`, in the default
  organization, with a grading scale and default assignment groups (so IC04 has somewhere to
  attach graded items). It MUST NOT have `starts_at`/`ends_at` that would ever hide it.
- **FR-5.** The system MUST expose a `intro_course_enabled` boolean on
  `settings.platform_app_settings`, defaulting to **true** when the column/row is unset (the
  *only* new flag in the codebase that defaults on besides the documented exceptions).
- **FR-6.** When `intro_course_enabled` is **false**, provisioning MUST NOT create the course if
  it does not yet exist, and existing IC02 auto-enroll and IC06 discovery surfaces MUST treat the
  course as unavailable — but the course row, enrollments, grades, and content MUST be retained.
- **FR-7.** The `introcourse.Service` MUST expose `EnsureProvisioned(ctx) (Course, error)`,
  `CourseID(ctx) (uuid.UUID, bool, error)`, and `Enabled(cfg) bool`, and MUST be the single
  source of truth for the course's existence/identity across the codebase.
- **FR-8.** Provisioning MUST be safe under concurrency (multiple app instances starting at once)
  via a Postgres advisory lock keyed on the provisioning operation, so two instances cannot race
  to create two courses.
- **FR-9.** The flag's value MUST be readable by clients via the existing
  `GET /api/v1/platform-features` response (`introCourseEnabled`) with no new endpoint.

## 6. Non-Functional Requirements

- **Performance** — `CourseID` MUST be O(1) from a cached lookup (short_code → id, memoized per
  process, invalidated on provisioning). Startup provisioning MUST add < 250 ms to boot and be a
  no-op fast path (single indexed `SELECT`) when the course already exists and content is current.
- **Security** — The system instructor has no login credential (null/blocked password hash), no
  API tokens, and cannot be impersonated. Provisioning and the admin re-sync endpoint require the
  platform-admin scope. No tenant other than `default` may own the canonical course in v1.
- **Privacy & Compliance** — No PII introduced. The system user is excluded from FERPA/GDPR
  export/erase subject lists (it is not a data subject). Flag changes are written to the admin
  audit log (`admin_audit_log_enabled`).
- **Accessibility** — No new UI beyond the existing platform-features toggle row (already WCAG
  2.1 AA); IC03/IC06 carry content accessibility.
- **Scalability** — One course row, one system user; O(1) regardless of user count. Nothing here
  scales with tenants/users.
- **Reliability** — Provisioning is idempotent and advisory-locked; a partial failure leaves the
  DB in a re-convergable state (next run completes it). No destructive operations.
- **Observability** — Emit `intro_course_provision_total{result}` and
  `intro_course_provision_duration_seconds`; a `intro_course_present` gauge (0/1); log each
  provisioning run at INFO with the course id and whether it created vs. reconciled. Metrics via
  `telemetry` with the `lextures_` prefix.
- **Maintainability** — All course-identity constants (`short_code`, system user UUID) live in
  one `introcourse/constants.go`; no other package may hard-code them.
- **Internationalization** — Title/description stored as i18n keys resolvable at render (IC08
  owns translation); default English seeded here.
- **Backward compatibility** — Purely additive: one new flag column, one new course row, one new
  system user. No existing table altered destructively.

## 7. Acceptance Criteria

- **AC-1.** *Given* a fresh database with the flag defaulting on, *when* the server boots, *then*
  exactly one `course.courses` row with `short_code='LEX-WELCOME'` exists, `published=true`, owned
  (teacher enrollment) by the `a0000000-…-002` system user.
- **AC-2.** *Given* the course already exists, *when* provisioning runs again (restart, redeploy,
  admin re-sync), *then* no second course is created and the row's title/description/settings are
  reconciled to the canonical values.
- **AC-3.** *Given* two app instances booting simultaneously, *when* both provision, *then* only
  one course and one system user exist (advisory lock holds).
- **AC-4.** *Given* `intro_course_enabled=false` on a DB where the course was never created,
  *when* the server boots, *then* no course and no system user are created.
- **AC-5.** *Given* the flag is toggled in the admin panel, *when* `GET /platform-features` is
  called, *then* the response contains `introCourseEnabled` reflecting the new value, and the
  change is recorded in the admin audit log.
- **AC-6.** *Given* the system instructor user, *when* seat licensing counts active users and
  when the people directory / active-user analytics run, *then* the system user is excluded.

## 8. Data Model

```sql
-- server/migrations/370_intro_course_core.sql   (renumber on merge; 358 is taken by LP)

-- Feature flag (defaults ON: unset column ⇒ true in features.go).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS intro_course_enabled BOOLEAN;
COMMENT ON COLUMN settings.platform_app_settings.intro_course_enabled IS
    'Enables the canonical "Welcome to Lextures" intro course and auto-enrollment (IC epic). Default true.';

-- System instructor identity. account_type 'system' marks it non-human (excluded from seats,
-- directories, active-user analytics). Password hash is a blocked sentinel (never matches).
INSERT INTO "user".users (id, email, password_hash, display_name, account_type, org_id)
VALUES (
    'a0000000-0000-4000-8000-000000000002',
    'guide@system.lextures.invalid',
    '!',                                   -- unusable/blocked hash
    'Lextures Guide',
    'system',
    (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1)
)
ON CONFLICT (id) DO NOTHING;
```

- Requires adding `'system'` to the `account_type` CHECK (or enum) alongside `student`,
  `teacher`, `parent` — extend the existing constraint. Confirm the exact column shape against
  `server/internal/repos/user/user.go` (see Open Question 1).
- The **course row itself is created by application code** (`EnsureProvisioned`), not by the
  migration, so the same idempotent path serves startup, admin re-sync, and content sync (IC03).
  Rationale: course creation touches multiple tables (course, enrollment, assignment_groups) and
  is safer as a single transactional service function than as raw SQL duplicated in a migration.
- **Backfill:** none. On flag enable, `EnsureProvisioned` creates the course lazily; on flag
  disable, nothing is deleted.

Provisioned course defaults: `grading_scale='letter_standard'`, three assignment groups
(`Participation` 20%, `Quizzes` 40%, `Assignments` 40%) — IC04 refines weights.

## 9. API Surface

No new *learner* endpoints (the course is consumed via existing `/api/v1/courses/{code}/…`).
Two admin/ops additions (full definition in IC08; stubbed here):

```
POST /api/v1/admin/intro-course/resync         Auth: platform-admin
  → runs EnsureProvisioned + content sync; 200 { courseId, created|reconciled }

GET  /api/v1/platform-features                  (existing) now includes:
  { ..., "introCourseEnabled": true }
```

Internal Go interface:

```go
// server/internal/service/introcourse/service.go
type Service interface {
    EnsureProvisioned(ctx context.Context) (Course, error) // idempotent; advisory-locked
    CourseID(ctx context.Context) (id uuid.UUID, present bool, err error)
    Enabled(cfg config.Config) bool
}
// constants.go
const (
    ShortCode      = "LEX-WELCOME"
    SystemUserID   = "a0000000-0000-4000-8000-000000000002"
)
```

Flag wiring mirrors `learner_profile_enabled` exactly:
`config.Config.IntroCourseEnabled`; `platformconfig` read (`platformconfig.go`), patch
(`patch.go` `addBool("intro_course_enabled", …)`), defaults (`features.go`
`mergeBool(db.IntroCourseEnabled, true)`); `httpserver/platform_features.go` field
`IntroCourseEnabled bool json:"introCourseEnabled"`; web
`clients/web/src/lib/platform-features.ts` + `platform-feature-definitions.ts` +
`platform-settings-types.ts`. OpenAPI updated in `server/internal/openapi/openapi.go`.

## 10. UI / UX

Only the admin platform-features panel gains one toggle row (reuses the existing
`platform-feature-definitions.ts` machinery):

```
{ key: 'introCourseEnabled', label: 'Intro course ("Welcome to Lextures")',
  description: 'Auto-enroll every new user as a student in the guided intro course. On by default.' }
```

No learner-facing UI in this plan. Empty/loading/error states for discovery are IC06.

## 11. AI / ML Considerations

None in this plan. (IC03 content may reference AI features; IC04 may optionally route the
capstone through the grader agent. No model calls here.)

## 12. Integration Points

- **Feature-flag stack:** `server/internal/config/config.go`,
  `server/internal/repos/platformconfig/{platformconfig,patch,features}.go`,
  `server/internal/httpserver/platform_features.go`, `server/internal/openapi/openapi.go`,
  `clients/web/src/lib/platform-features.ts`,
  `clients/web/src/components/settings/{platform-feature-definitions.ts,platform-settings-types.ts}`.
- **Course/enrollment repos:** `server/internal/repos/course/`,
  `server/internal/repos/coursegrants/`, `course.course_enrollments`, `course.assignment_groups`.
- **System user precedent:** `server/internal/repos/communication/welcome.go`
  (`PlatformInboxSenderID = a0000000-…-001`); the guide user is `…-002`.
- **Startup wiring:** `server/internal/app/app.go` (call `EnsureProvisioned` after migrations,
  behind the flag).
- **Telemetry:** `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **After:** courses/enrollment/gradebook and the feature-flag stack — all shipped.
- **Before:** IC02–IC08 (all require the course to exist and the flag to be wired).
- **Shared infra:** Postgres, config/flag pipeline, telemetry, app startup hook — all present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Duplicate courses from racing app instances | M | H | Advisory lock on provision op + unique `short_code` index (already exists) |
| System user leaks into seat count / directories / analytics | M | M | `account_type='system'` filter added at each surface; AC-6 test |
| Flag defaults on and surprises a self-hosting admin | M | M | Documented default-on; single admin toggle; release note; disable is instant & non-destructive |
| Course accidentally attributed to first human user (legacy 007 backfill pattern) | L | M | Provision creates the teacher enrollment explicitly as the system user before any human exists |
| `account_type='system'` breaks an existing CHECK/enum | M | M | Extend constraint in same migration; add nodb test asserting insert succeeds |

## 15. Rollout Plan

- **Flag:** `intro_course_enabled`, default **true**.
- **Sequencing:** migration (flag column + system user + account_type extension) → service
  `EnsureProvisioned` behind flag + startup hook → flag wired through API/web → internal org
  verifies one course + system instructor → ship. IC02+ layer on top.
- **Dogfood:** enable on the internal/staging org first; confirm single course, correct owner,
  metrics emitting.
- **GA criteria:** idempotency proven across restarts; flag toggles reflected in
  `/platform-features`; system user excluded from seats/directories.
- **Rollback:** set `intro_course_enabled=false` (course retained, inert) or, to fully remove,
  run the `370_*.down.sql` (drops flag column; leaves course/system user for manual cleanup to
  avoid cascading grade/enrollment deletion).

## 16. Test Plan

- **Unit** — `EnsureProvisioned` idempotency (create then reconcile); `Enabled(cfg)` default-true
  logic; constants single-source; cached `CourseID` invalidation.
- **Integration (DB)** — fresh DB → one course + system teacher enrollment; re-run → still one;
  flag off on empty DB → no course; concurrent provision (goroutines) → one course (advisory lock).
- **End-to-end** — boot server, `GET /platform-features` shows `introCourseEnabled:true`; admin
  re-sync endpoint returns `reconciled`.
- **Security** — system user cannot log in (blocked hash); re-sync endpoint rejects non-admin;
  system user absent from `GET /people` and seat count.
- **Performance** — provisioning fast-path no-op < 5 ms with warm cache; boot overhead < 250 ms.
- **Manual exploratory** — restore a DB snapshot lacking the course; confirm convergence on boot.

## 17. Documentation & Training

- Runbook `docs/runbooks/intro-course.md`: force re-sync, disable the flag, locate the system
  user, read provisioning metrics.
- Admin doc: what "Intro course" toggle does and its default-on behaviour.
- OpenAPI: `introCourseEnabled` field + the admin resync endpoint.
- Engineering note: "the intro course identity lives in `introcourse/constants.go` — never
  hard-code `LEX-WELCOME`."

## 18. Open Questions

1. Does `"user".users` gate `account_type` with a CHECK, an enum, or free text? Confirm and
   extend to include `'system'` (verify against `server/internal/repos/user/user.go`).
2. Should the course live in the `default` org only, or be replicated per top-level org for
   multi-tenant deployments? (v1: `default` only; IC08 revisits per-org variants.)
3. Is startup the right provisioning trigger, or should it be migration-run-once + admin re-sync
   only (to avoid boot-time DB writes on every deploy)? (Leaning: startup fast-path no-op + admin
   re-sync; measure boot cost.)
4. Blocked-password sentinel: reuse the existing "no password" convention from SSO-only users, or
   introduce an explicit `login_disabled` flag on the system user?

## 19. References

- Existing files: `server/internal/repos/communication/welcome.go` (system sender precedent),
  `server/migrations/007_course_enrollments.sql`, `.../016_enrollment_teacher_not_owner.sql`,
  `.../027_course_grading.sql`, `server/internal/repos/platformconfig/features.go`,
  `server/internal/httpserver/platform_features.go`, `server/internal/app/app.go`.
- Related plans: [IC02](IC02-automatic-enrollment.md), [IC03](IC03-curriculum-content.md),
  [IC08](IC08-admin-governance-localization.md),
  [LP01 foundation](../learner-profile/LP01-foundation-derivation-engine.md) (flag-wiring model).

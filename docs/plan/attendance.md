# Attendance — Course Attendance Collection

> Implementation plan. Minimum viable attendance app for instructors within a course.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | Attendance |
| **Section** | Course Tools |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Full-stack team |
| **Depends on** | 5.4 (sections — optional section filter), gradebook grid (existing) |
| **Unblocks** | 13.2 (state-reportable daily attendance — can consume session data), 13.1 (parent portal attendance summary) |

---

## 1. Problem Statement

Instructors need a lightweight way to record who attended class without leaving the LMS. Today Lextures has no course-level attendance tool: no roll-taking UI, no student self-check-in, and no way to tie participation to the gradebook. Districts and colleges expect attendance to be optional per course, off by default, and simple enough to run in under two minutes at the start of class. This plan delivers a course-feature-flagged Attendance app with a class roster, two collection modes (roll call and self report), and an optional gradebook column per session.

## 2. Goals

- Add a per-course **Attendance** tool, disabled by default, toggled in Course Settings → Features (same pattern as Discussions, Live Sessions, Office Hours).
- Show the enrolled class list and let instructors start an attendance session quickly.
- Support two collection activities: **roll call** (instructor marks each student) and **self report** (students mark themselves during an open window; instructor reviews and finalizes).
- Optionally store each session as a **gradebook item** so attendance can count toward the course grade.
- Keep v1 scope minimal: present / absent / tardy (or excused) statuses, one session at a time per course section, no state-export reporting.

## 3. Non-Goals

- K-12 state-reportable daily attendance, CALPADS export, or school-wide dashboards (see [13.2 — Daily Attendance](../completed/13-k12-specific/13.2-daily-attendance.md)).
- Automated attendance via video join logs, geolocation, or biometric check-in.
- Attendance policies engine (auto-fail after N absences, truancy letters).
- Parent/guardian notifications on absence (deferred to 13.1 / 13.2 integration).
- Recurring attendance rules or automatic session creation from the course calendar.
- Multiple simultaneous open self-report sessions for the same roster.

## 4. Personas & User Stories

- **As an instructor**, I want to enable Attendance for my course in settings so that only courses that need it show the tool in the course menu.
- **As an instructor**, I want to see my class roster and pick roll call or self report in one click so that I can start taking attendance without navigating away.
- **As an instructor**, I want to mark students present, absent, or tardy in a roll-call grid with a “mark all present” shortcut so that I can finish roll in under two minutes.
- **As an instructor**, I want to open a self-report window where students check in themselves so that I can verify participation on large courses without reading every name aloud.
- **As an instructor**, I want each attendance session to appear as a gradebook column when I choose so that participation counts toward the final grade.
- **As a student**, I want to self-report my attendance when my instructor opens a session so that my presence is recorded before the window closes.
- **As a student**, I want to see my attendance grade in the gradebook when the instructor posts it so that I know how participation affects my score.

## 5. Functional Requirements

### Course feature flag

- **FR-1.** The system MUST add `attendance_enabled boolean NOT NULL DEFAULT false` on `course.courses`, exposed on `CoursePublic` and patchable via `PATCH /api/v1/courses/{course_code}/features` (same contract as `discussionsEnabled`, `officeHoursEnabled`).
- **FR-2.** When `attendance_enabled` is `false`, the Attendance nav item, routes, and API handlers MUST return 404 or be hidden; students and staff MUST NOT access attendance data for that course.
- **FR-3.** Only users with `course:{code}:item:create` (or equivalent staff manage permission) MUST be able to toggle the feature in Course Settings.

### Class list & session shell

- **FR-4.** The Attendance home page MUST show the course class list: active student-equivalent enrollments from the course roster, ordered by display name (reuse `GET /api/v1/courses/{course_code}/enrollments` or `ListStudentUsersForCourseCode`).
- **FR-5.** When course sections are enabled (5.4), the UI MUST provide a section selector and filter the roster to the selected section; when sections are disabled, the UI MUST show all course students.
- **FR-6.** The Attendance home page MUST expose a prominent control to choose the collection activity: **Roll call** or **Self report**, before starting a session.
- **FR-7.** An instructor MUST be able to create one **attendance session** per class meeting with: `title` (default: date + optional label), `collection_method` (`roll_call` | `self_report`), `session_date`, optional `section_id`, and optional gradebook settings (see FR-14–FR-16).

### Roll call

- **FR-8.** In roll-call mode, the instructor MUST mark each student with a status from a fixed v1 set: `present`, `absent`, `tardy`, `excused`, or `not_recorded` (default until saved).
- **FR-9.** The roll-call UI MUST provide **Mark all present** and MUST allow batch save of the full roster in one request.
- **FR-10.** Roll-call sessions MUST be editable by the creating instructor (or course staff with attendance manage permission) until the session is **closed**; after close, records are read-only except for users with gradebook override permission.

### Self report

- **FR-11.** In self-report mode, the instructor MUST open the session with a configurable window (`opens_at`, `closes_at`; default: open now, close in 15 minutes).
- **FR-12.** During an open self-report window, each enrolled student MUST be able to submit exactly one status for themselves (`present` or `tardy` in v1; `absent` is implicit if they do not check in before close).
- **FR-13.** After the window closes (or when the instructor clicks **Finalize**), the instructor MUST see submitted vs missing students and MUST be able to override any row before closing the session.

### Gradebook integration

- **FR-14.** When creating or closing a session, the instructor MUST be able to opt in: **Add to gradebook**. When enabled, the system MUST create a gradable `course_structure_items` row with `kind = 'attendance'` linked to the session.
- **FR-15.** A gradebook-linked session MUST define `points_possible` (integer, default 1) and MUST write per-student scores into `course.course_grades` using the existing gradebook grid pipeline (`points_earned` derived from status: present = full points, tardy = configurable fraction default 0.5, absent / no check-in = 0, excused = excused flag if supported else full points).
- **FR-16.** Gradebook columns for attendance sessions MUST appear in `GET /api/v1/courses/{course_code}/gradebook/grid` with `kind: "attendance"`, title from the session, and `maxPoints` from `points_possible`.
- **FR-17.** Gradebook scores MUST update when the instructor changes a student's final status on a closed session (same as manual grade edit permissions).

### Permissions & audit

- **FR-18.** Only course staff with a new permission `course:{code}:attendance:manage` MUST create, edit, finalize, or close sessions; students MUST only self-report on open self-report sessions for courses they are enrolled in.
- **FR-19.** Each student status row MUST store `recorded_by`, `recorded_at`, and `source` (`instructor` | `self` | `override`) for audit.

## 6. Non-Functional Requirements

- **Performance** — Class roster + session detail for 200 students MUST load in < 500 ms p95. Batch save of 200 status rows MUST complete in < 400 ms p95.
- **Security** — Students MUST NOT read or modify other students' attendance rows. Session IDs MUST be UUIDs. Self-report MUST verify enrollment and an open window server-side (not UI-only).
- **Privacy & Compliance** — Attendance is a FERPA-protected education record. No geolocation or device fingerprinting in v1. Retain records per org retention policy.
- **Accessibility** — Roll-call grid MUST be keyboard-navigable (`role="grid"`); status changes MUST announce via `aria-live`. Collection-mode toggle MUST be a radiogroup with visible labels. WCAG 2.1 AA.
- **Scalability** — One course, up to 500 students per session; indexed by `(course_id, session_date)` and `(session_id, student_user_id)`.
- **Reliability** — Status saves MUST be idempotent (upsert on `(session_id, student_user_id)`). Self-report submit MUST be safe to retry.
- **Observability** — Metrics: `attendance_sessions_created_total`, `attendance_records_saved_total`, `self_report_submissions_total`. Log session create/close with `course_id`, `session_id`, `collection_method`.
- **Maintainability** — Backend module: `server/internal/repos/attendancesessions/` and `server/internal/httpserver/course_attendance.go`. Frontend page: `clients/web/src/pages/lms/course-attendance.tsx`.
- **Internationalization** — Status labels and collection-mode names externalized; dates/times in viewer timezone.
- **Backward compatibility** — Additive schema only. Existing 13.2 tables (`course.attendance_codes`, `course.attendance_records`) remain separate until a later integration pass.

## 7. Acceptance Criteria

- **AC-1.** *Given* Attendance is disabled for a course, *When* a student navigates to `/courses/{code}/attendance`, *Then* the route is not linked in the nav and direct access returns 404 or redirects to the course dashboard.
- **AC-2.** *Given* an instructor enables Attendance in Course Settings, *When* they save, *Then* `attendanceEnabled: true` is returned on the course and the Attendance link appears in the course side nav after refresh.
- **AC-3.** *Given* an instructor on the Attendance page with 30 enrolled students, *When* they select **Roll call** and start a session, *Then* all 30 students appear in a grid with default `not_recorded` status.
- **AC-4.** *Given* a roll-call session, *When* the instructor clicks **Mark all present** and saves, *Then* one batch API call persists 30 rows and the UI shows “Attendance saved.”
- **AC-5.** *Given* an instructor starts a **Self report** session open for 10 minutes, *When* a enrolled student submits check-in, *Then* only that student's row updates to `present` with `source: self` and other students cannot submit on their behalf.
- **AC-6.** *Given* a self-report session whose window has closed, *When* the instructor finalizes and overrides one missing student to `absent`, *Then* the final status is saved with `source: override`.
- **AC-7.** *Given* an instructor creates a session with **Add to gradebook** enabled and `points_possible: 10`, *When* the session is closed with 28 present and 2 absent, *Then* the gradebook grid shows a new `attendance` column and those students have `280` and `0` points respectively (or equivalent stored `points_earned`).
- **AC-8.** *Given* a keyboard-only user on the roll-call grid, *When* they tab through rows and activate status controls, *Then* they can complete the roster without a mouse and each change is announced.
- **AC-9.** *Given* sections are enabled and the instructor selects Section A, *When* they start roll call, *Then* only Section A students appear, not Section B.

## 8. Data Model

```sql
-- server/migrations/NNN_course_attendance_sessions.sql

-- Per-course feature flag (default off)
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS attendance_enabled boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN course.courses.attendance_enabled IS
    'When true, the Attendance tool is available in this course (nav + API).';

-- One class meeting / attendance event
CREATE TABLE course.attendance_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    section_id          UUID REFERENCES course.course_sections (id) ON DELETE SET NULL,
    structure_item_id   UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL,
    title               TEXT NOT NULL,
    collection_method   TEXT NOT NULL CHECK (collection_method IN ('roll_call', 'self_report')),
    session_date        DATE NOT NULL,
    opens_at            TIMESTAMPTZ,
    closes_at           TIMESTAMPTZ,
    status              TEXT NOT NULL DEFAULT 'open'
                            CHECK (status IN ('open', 'closed')),
    gradebook_enabled   BOOLEAN NOT NULL DEFAULT false,
    points_possible     INTEGER CHECK (points_possible IS NULL OR points_possible > 0),
    tardy_points_ratio  NUMERIC(3,2) NOT NULL DEFAULT 0.5
                            CHECK (tardy_points_ratio >= 0 AND tardy_points_ratio <= 1),
    created_by          UUID NOT NULL REFERENCES "user".users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at           TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attendance_sessions_course_date
    ON course.attendance_sessions (course_id, session_date DESC);

-- Per-student result for a session
CREATE TABLE course.attendance_session_records (
    session_id      UUID NOT NULL REFERENCES course.attendance_sessions (id) ON DELETE CASCADE,
    student_user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'not_recorded'
                        CHECK (status IN ('present', 'absent', 'tardy', 'excused', 'not_recorded')),
    source          TEXT NOT NULL DEFAULT 'instructor'
                        CHECK (source IN ('instructor', 'self', 'override')),
    recorded_by     UUID REFERENCES "user".users (id),
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, student_user_id)
);

CREATE INDEX idx_attendance_session_records_student
    ON course.attendance_session_records (student_user_id);

-- Gradebook: new structure item kind
ALTER TABLE course.course_structure_items DROP CONSTRAINT IF EXISTS course_structure_items_kind_check;
ALTER TABLE course.course_structure_items
    ADD CONSTRAINT course_structure_items_kind_check
    CHECK (kind IN (
        'module', 'heading', 'content_page', 'assignment', 'quiz',
        'external_link', 'survey', 'lti_link', 'h5p', 'vibe_activity', 'attendance'
    ));

-- Extend parent-child constraint similarly for 'attendance' as leaf kind.
```

**Backfill:** None. `attendance_enabled` defaults to `false` for all existing courses.

**Gradebook sync:** On session close (or on save when `gradebook_enabled`), upsert `course.course_grades` rows for each student from computed points; set `course_structure_items.points_worth` from `points_possible`.

## 9. API Surface

| Verb | Path | Auth scope | Description |
|---|---|---|---|
| GET | `/api/v1/courses/{course_code}/attendance/sessions` | staff: `attendance:manage`; student: enrolled + feature on | List sessions (paginated, filter by date/section) |
| POST | `/api/v1/courses/{course_code}/attendance/sessions` | `attendance:manage` | Create session (`collectionMethod`, `title`, `sessionDate`, `sectionId?`, `gradebookEnabled?`, `pointsPossible?`, `opensAt?`, `closesAt?`) |
| GET | `/api/v1/courses/{course_code}/attendance/sessions/{id}` | staff or enrolled student (limited fields for students) | Session detail + roster rows |
| PATCH | `/api/v1/courses/{course_code}/attendance/sessions/{id}` | `attendance:manage` | Update open session (window times, title) |
| PUT | `/api/v1/courses/{course_code}/attendance/sessions/{id}/records` | `attendance:manage` | Batch upsert roster statuses (roll call / finalize) |
| POST | `/api/v1/courses/{course_code}/attendance/sessions/{id}/self-report` | enrolled student | Student self check-in (`status`: `present` \| `tardy`) |
| POST | `/api/v1/courses/{course_code}/attendance/sessions/{id}/close` | `attendance:manage` | Close session; optional gradebook write |

**Request example (create roll-call session with gradebook):**

```json
{
  "collectionMethod": "roll_call",
  "title": "Week 3 — Lecture",
  "sessionDate": "2026-06-03",
  "sectionId": null,
  "gradebookEnabled": true,
  "pointsPossible": 5
}
```

**Response shape (session detail):**

```json
{
  "id": "…",
  "collectionMethod": "self_report",
  "status": "open",
  "opensAt": "2026-06-03T14:00:00Z",
  "closesAt": "2026-06-03T14:15:00Z",
  "gradebookEnabled": true,
  "structureItemId": "…",
  "records": [
    { "studentUserId": "…", "displayName": "Alex", "status": "present", "source": "self" }
  ]
}
```

**Feature flag patch** — add `attendanceEnabled` to `patchCourseFeaturesBody` and `CoursePublic`.

## 10. UI / UX

### New surfaces

- **Course Settings → Features:** toggle “Attendance” (description: “Take roll call or run self-report check-ins; optionally add sessions to the gradebook.”). Default off.
- **Course nav:** `Attendance` link (icon: `ClipboardList` or `UserCheck`) when `attendanceEnabled`.
- **Page:** `clients/web/src/pages/lms/course-attendance.tsx` — main instructor/student hub.

### Key flows

1. **Enable feature** — Settings → Features → Attendance on → nav link appears.
2. **Start roll call** — Attendance → section filter (if any) → choose **Roll call** → **Start session** → grid → Mark all present → adjust exceptions → Save → Close session.
3. **Start self report** — Attendance → choose **Self report** → set window (default 15 min) → **Open session** → students see banner/button to check in → instructor monitors live count → Finalize → Close (gradebook optional).
4. **Gradebook** — When “Add to gradebook” checked, closing the session creates/updates the column; instructor opens Gradebook to verify scores.

### Layout (instructor home)

```
┌─────────────────────────────────────────────────────────┐
│ Attendance          [ Section ▼ ]    [ + New session ]  │
├─────────────────────────────────────────────────────────┤
│ Collection activity:  ( • Roll call )  ( ○ Self report )│
├─────────────────────────────────────────────────────────┤
│ Past sessions table (date, method, present/absent, link)  │
└─────────────────────────────────────────────────────────┘
```

### Student view

- Active self-report session: prominent “Check in” card on Attendance page and optional course dashboard banner.
- No access to other students’ statuses.
- Closed sessions: read-only own history (optional v1.1; minimum v1 is check-in only).

### States

- **Empty:** “No attendance sessions yet. Start roll call or open self report.”
- **Loading:** skeleton roster rows.
- **Error:** toast + retry on save failure; optimistic UI rolled back.
- **Offline:** not supported in v1; show network error.

### Files to touch

- `clients/web/src/context/course-nav-features-context.tsx` — `attendanceEnabled`
- `clients/web/src/components/layout/side-nav-course-links.tsx` — nav link
- `clients/web/src/pages/lms/course-features-section.tsx` — toggle
- `clients/web/src/lib/courses-api.ts` — types + API helpers
- `server/internal/httpserver/course_features.go` — patch body
- `server/internal/repos/course/features.go` — column update
- `server/internal/httpserver/gradebook_grid.go` — include `attendance` columns

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- **Course roster** — `server/internal/repos/enrollment/list_roster.go`, `ListStudentUsersForCourseCode`.
- **Sections (5.4)** — `course.course_sections`, section filter on roster and sessions.
- **Gradebook** — `course.course_structure_items`, `course.course_grades`, `gradebook_grid.go`.
- **Permissions** — extend `courseroles` catalog with `attendance:manage` (staff roles: owner, teacher, instructor, ta).
- **13.2 (future)** — closed session records MAY map into `course.attendance_records` using org attendance codes when platform `ff_attendance` and K-12 mode are enabled; not required for v1.

## 13. Dependencies & Sequencing

- Must ship after: existing course feature-flag pipeline, gradebook grid, enrollment roster APIs.
- Should ship with: permission seed for `attendance:manage`.
- Optional enhancement after v1: 5.4 section-only filtering (works without sections using full-course roster).
- Must ship before: deep 13.2 / parent-portal attendance summaries that expect structured records.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Students share self-report links or check in for others | M | M | Require authenticated session; one submission per student; instructor review step before close |
| Gradebook column proliferation (daily sessions) | M | M | Default gradebook off; instructor opt-in per session; document best practice |
| Overlap with 13.2 K-12 attendance schema | M | L | Separate session tables in v1; document integration path; do not dual-write until 13.2 ships |
| Large roster batch save timeouts | L | M | Single transaction upsert; limit 500 rows; integration test at 200 students |

## 15. Rollout Plan

- **Course flag:** `attendance_enabled` on `course.courses`, default `false` (not a platform-wide kill switch).
- **Sequencing:** migration → API + permissions → gradebook column kind → instructor UI → student self-report → dogfood.
- **Pilot:** 2–3 courses (one HE lecture, one K-12 sectioned course).
- **GA criteria:** AC-1 through AC-9 pass; axe-core zero Critical/Serious on Attendance pages.
- **Rollback:** disable per course via settings; hide nav link; retain data.

## 16. Test Plan

- **Unit:** Points calculation from status + `tardy_points_ratio`; self-report window validation; permission checks.
- **Integration:** Create roll-call session → batch save → close with gradebook → verify `course_grades` and grid column.
- **Integration:** Self-report open → student submit → duplicate submit rejected → finalize → gradebook scores.
- **End-to-end:** Playwright — enable feature → roll call → verify gradebook cell.
- **Security:** Student cannot PUT another student's row; cannot self-report when window closed; feature off returns 404.
- **Accessibility:** axe on roll-call grid; keyboard-only roll completion (AC-8).
- **Performance:** Batch save 200 rows < 400 ms in CI benchmark test.

## 17. Documentation & Training

- Help center: “Enable Attendance for your course”, “Take roll call”, “Run a self-report check-in”, “Add attendance to the gradebook”.
- Instructor release note when feature ships.
- API reference for session endpoints (internal/OpenAPI).

## 18. Open Questions

1. Should excused absences award full points, zero points, or be excluded from the gradebook column (excused flag)?
2. For self report, should late check-ins after `closes_at` be blocked entirely or allowed as `tardy` for a grace period?
3. Should students see a history of their past session statuses in v1, or only the active check-in prompt?
4. When sections are enabled but the instructor has multiple sections, should one session span all their sections or always require an explicit section pick?
5. Should closing a session auto-post grades to students or follow the course grade posting policy?

## 19. References

- Course feature flags: `server/internal/httpserver/course_features.go`, `server/internal/repos/course/features.go`, `clients/web/src/pages/lms/course-features-section.tsx`
- Gradebook grid: `server/internal/httpserver/gradebook_grid.go`, `server/migrations/067_course_grades.sql`
- Roster: `server/internal/repos/enrollment/list_roster.go`
- K-12 daily attendance (separate, richer scope): [13.2 — Daily Attendance](../completed/13-k12-specific/13.2-daily-attendance.md), `server/migrations/217_attendance.sql`, `server/internal/repos/attendance/repo.go`
- Plan template: [_TEMPLATE.md](./_TEMPLATE.md)

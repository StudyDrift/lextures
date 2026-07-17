# MOB.4 — Course Enrollment Management (mobile)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: `POST /api/v1/courses/{courseCode}/enrollments` in
> [`clients/web/src/lib/courses-api.ts`](../../../clients/web/src/lib/courses-api.ts),
> `enrollment-invitation-api.ts`, `enrollment-state-api.ts`,
> `components/enrollment/*`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.4 |
| **Section** | Mobile parity |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING (add path) |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | — |
| **Unblocks** | — |

## 1. Problem Statement

On mobile, the course People screen (`CoursePeopleView` / `CoursePeopleScreen`,
backed by `CoursePeopleLogic`) can **list, filter by role, and remove**
enrollments, but it cannot **add** anyone — there is no invite/add-person flow,
no role selection, and no invitation state management. Web offers all of this.
As a result, an instructor cannot add a TA, enroll a student, or approve a
pending self-enrollment from their phone, forcing roster changes back to the web
app during the busiest part of term.

## 2. Goals

- Let instructors/admins add a person to a course on mobile by email/user with a
  chosen role and (optionally) section.
- Support the enrollment lifecycle web supports: pending/invited → active,
  approve/decline invitations, deactivate/remove.
- Keep permission and role rules identical to web/server.
- Reuse the existing People screen; add actions inline.

## 3. Non-Goals

- Enrollment groups / group sets (`enrollment-groups/*`) — separate scope.
- Bulk roster import (CSV/SIS) — covered by SIS/import plans.
- Section creation (owned by `CourseSectionsSettings`).
- Cross-org invitations / provisioning.

## 4. Personas & User Stories

- **As an instructor**, I want to add a TA by email with the TA role so they can
  help grade immediately.
- **As a K-12 teacher**, I want to enroll a transfer student mid-term from my
  phone.
- **As a course admin**, I want to approve a pending self-enrollment request on
  the go.
- **As an instructor**, I want to deactivate (not delete) a student who dropped.

## 5. Functional Requirements

- **FR-1.** The People screen MUST offer an "Add people" action when the viewer
  has the manage-enrollments permission.
- **FR-2.** Add MUST collect one or more identifiers (email or existing user),
  a role (student, TA, teacher/instructor, and any custom roles the org allows),
  and optionally a section.
- **FR-3.** Add MUST `POST /api/v1/courses/{courseCode}/enrollments` with the
  chosen role/section, matching web's request shape.
- **FR-4.** The screen MUST surface enrollment state (active, invited/pending,
  inactive) using the existing state badge semantics.
- **FR-5.** For pending invitations, the app MUST offer approve/decline via the
  invitation endpoints (`…/enrollments/{id}/invitation/approve|decline`).
- **FR-6.** The app MUST support changing enrollment state
  (`…/enrollments/{id}/state`) — e.g. deactivate/reactivate — and removal
  (existing).
- **FR-7.** The app SHOULD support messaging an enrollee
  (`…/enrollments/{id}/message`) if that action is in the People UI on web.
- **FR-8.** All actions MUST be permission-gated; unavailable actions are hidden.
- **FR-9.** Adding a user already enrolled MUST show a clear, localized conflict
  message (server 409 parity).

## 6. Non-Functional Requirements

- **Performance** — add returns and optimistically inserts the new row;
  list refresh < 1 s.
- **Security** — server enforces manage-enrollments; no client-only gating;
  email inputs validated; no enumeration leakage in errors.
- **Privacy & Compliance** — emails/roster are PII: avoid disk caching of the
  roster; FERPA-safe error messages.
- **Accessibility** — WCAG 2.1 AA; role picker and email entry fully labelled;
  state badges have text equivalents.
- **Scalability** — n/a (per-course writes); list paginates.
- **Reliability** — idempotent add (duplicate → 409 handled); optimistic UI
  rolls back on failure.
- **Observability** — `enrollment_{added,invited,approved,declined,state_changed,removed}`
  with role (no PII).
- **Maintainability** — extend `CoursePeopleLogic` + `LMSAPICoursePeople`
  wrappers on both platforms.
- **Internationalization** — `mobile.people.*` keys (extend existing).
- **Backward compatibility** — no API change.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor, *when* they add a person by email with role
  TA, *then* a TA enrollment is created and appears in the list.
- **AC-2.** *Given* a pending invitation, *when* the instructor approves it,
  *then* the enrollment becomes active.
- **AC-3.** *Given* an active student, *when* the instructor deactivates them,
  *then* their state shows inactive (not removed).
- **AC-4.** *Given* a user without manage-enrollments, *then* no add/approve/
  state actions are shown.
- **AC-5.** *Given* an already-enrolled email, *when* adding, *then* a localized
  "already enrolled" message appears (no duplicate).

## 8. Data Model

- **No new tables.** Uses existing `enrollments` and invitation/state fields.
  Client extends `CoursePeopleLogic` state with add/invite view models only.

## 9. API Surface

Existing endpoints (reused):

- `POST /api/v1/courses/{courseCode}/enrollments` — add/invite (role, section).
- `POST /api/v1/courses/{courseCode}/enrollments/{id}/invitation/approve`
- `POST /api/v1/courses/{courseCode}/enrollments/{id}/invitation/decline`
- `PATCH/PUT /api/v1/courses/{courseCode}/enrollments/{id}/state`
- `POST /api/v1/courses/{courseCode}/enrollments/{id}/message` (if surfaced)
- `DELETE`/remove (already used by mobile).

No new server routes.

## 10. UI / UX

- **Modified screen:** People (`CoursePeopleView`/`CoursePeopleScreen`) gains an
  "Add people" button and per-row actions (approve/decline, deactivate, remove,
  message).
- **New sheet:** Add People — identifier entry (email/user search), role picker,
  optional section picker, submit.
- **Flows:** People → Add → enter emails + role → confirm → rows appear.
  Pending row → approve/decline. Row → state change.
- **States:** loading, empty roster, submitting, duplicate/conflict, invalid
  email, offline (block writes), no-permission (actions hidden).
- **Mobile/responsive:** action sheet / bottom sheet for row actions; chips for
  multi-email entry.
- **Accessibility:** labelled inputs; confirm dialogs for destructive/state
  changes; state announced.
- **Copy & i18n:** `mobile.people.add.*`, `mobile.people.invite.*`,
  `mobile.people.state.*`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- iOS `Core/LMS/CoursePeopleLogic.swift` + a `LMSAPICoursePeople`/enrollment
  wrapper; `Features/Courses/CoursePeopleView.swift`.
- Android `core/lms/CoursePeopleLogic.kt` + enrollment API; `CoursePeopleScreen.kt`.
- Reuse role/section models already present from course settings.

## 13. Dependencies & Sequencing

- Must ship after: —.
- Must ship before: —.
- Shared infra: none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Role list mismatch vs org custom roles | M | M | Fetch allowed roles from server, not hard-code |
| Adding wrong person (typo email) | M | M | Confirm summary before submit; show resolved name where possible |
| Roster PII cached to disk | L | H | In-memory only; no persistence of emails |

## 15. Rollout Plan

- Flag: `ff_mobile_enrollment_add` (default off → on).
- Sequence: ship behind flag → dogfood with instructors → GA.
- GA criteria: AC-1..5 pass on both platforms.
- Rollback: flag off returns People to view/remove only.

## 16. Test Plan

- **Unit** — add request building, role/section validation, state transitions,
  409 handling.
- **Integration** — add → approve → deactivate → remove against a test course.
- **End-to-end** — instructor adds TA and student on device.
- **Security** — authz: non-managers cannot add/approve; server rejects.
- **Accessibility** — screen-reader run of add sheet + row actions.
- **Manual** — duplicate email, invalid email, offline.

## 17. Documentation & Training

- Help article: "Add people to your course on mobile" (roles explained).
- Note on invitation vs. direct enrollment behaviour.

## 18. Open Questions

1. Does the People UI on web expose "message enrollee" — do we bring it to
   mobile v1 or defer?
2. Should we support inviting a not-yet-registered email (pending invitation),
   or only existing org users, in v1?
3. Section assignment at add time — required for K-12, optional for HE?

## 19. References

- Web: `clients/web/src/lib/courses-api.ts` (enrollments),
  `enrollment-invitation-api.ts`, `enrollment-state-api.ts`,
  `components/enrollment/*`.
- iOS: `clients/ios/Lextures/Core/LMS/CoursePeopleLogic.swift`,
  `Features/Courses/CoursePeopleView.swift`.
- Android: `.../core/lms/CoursePeopleLogic.kt`,
  `.../features/courses/CoursePeopleScreen.kt`.

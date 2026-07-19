# MOB.1 — Course Creation Wizard (mobile parity)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: [`clients/web/src/pages/lms/course-create.tsx`](../../../clients/web/src/pages/lms/course-create.tsx),
> [`course-create-templates.ts`](../../../clients/web/src/pages/lms/course-create-templates.ts).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.1 |
| **Section** | Mobile parity |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | — (course APIs shipped) |
| **Unblocks** | MOB.2 (Canvas import is a create entry point) |


---

## Implementation notes (2026-07-18)

- **Flag**: `ffMobileCourseCreateV2` (DB-backed, default OFF) stages competency authoring, Canvas create entry, draft resume, and telemetry. Base entry still gated by `ffMobileCreateCourse` *or* v2.
- **Also wired**: `ffMobileCreateCourse` to platform settings (was client-decode-only) so the New course entry can be enabled from Settings → Global platform.
- **Logic**: `CourseCreateLogic` on iOS/Android — source chooser, competency/sub-outcome drafts, `validateCompetencies` parity with web, draft store, observability counters.
- **API**: `createModuleAssignment` / `createModuleQuiz`; Android `createCourseOutcomeSubOutcome`; `PatchCourseOutcomeBody.moduleStructureItemId`.
- **UI**: `CourseCreateView` / `CourseCreateScreen` — source step, competency editor (v2), Canvas coming-soon handoff to MOB.2; traditional path unchanged.
- **i18n**: `mobile.createCourse.*` extended; synced via `scripts/sync-mobile-locales.py`.
- **Tests**: unit coverage for permission/v2 gate, competency validation, template/update parity (XCUITest/Espresso not present in repo).


## 1. Problem Statement

Instructors on web create a course through a 3-step wizard
(`/courses/create`): Basics → Syllabus template → First module / Competencies,
supporting both traditional and competency-based courses. iOS
(`CourseCreateView` + `CourseCreateLogic`, 3 steps) and Android
(`CourseCreateScreen` + `CourseCreateLogic`) already implement the skeleton of
this wizard, but the mobile flow diverges from web in several fields and edge
paths (term/grade-level pickers, edit-on-back, competency sub-outcome authoring,
and the "start from Canvas" entry point). The result is that instructors who
start a course on a phone frequently have to finish it on the web, breaking the
"run my class from my pocket" promise for the K-12 and self-learner segments.

## 2. Goals

- Reach field-for-field parity with the web wizard for both course modes.
- Let an instructor create a usable, correctly-structured course end-to-end on
  iOS and Android without touching the web app.
- Make the wizard resumable and editable (re-entering Basics updates, not
  duplicates, the draft course).
- Provide the "import from Canvas" branch as a first-class create option
  (handoff to MOB.2).
- Keep the mobile wizard visually and behaviourally consistent with the shipped
  motion language (AN.\* springs) and accessibility standards.

## 3. Non-Goals

- Full course-settings authoring after creation (owned by the shipped
  `Courses/Settings/*` screens).
- Blueprint/cross-listing setup at creation time (post-create settings only).
- Canvas import mechanics themselves — specified in [MOB.2](MOB.2-canvas-course-import.md).
- AI lesson generation ("Sparkles") beyond linking to the existing generator.

## 4. Personas & User Stories

- **As an instructor (HE)**, I want to spin up a 15-week course from a template
  on my phone so that I can prep between meetings.
- **As a K-12 teacher**, I want to create a competency-based course with
  outcomes and sub-outcomes so my standards map is right from day one.
- **As a self-learner/creator**, I want to create a blank course and add the
  first module so I can start authoring immediately.
- **As an instructor migrating from Canvas**, I want "create from Canvas import"
  offered in the same place I create any course.

## 5. Functional Requirements

- **FR-1.** The wizard MUST expose the same Basics fields as web: title,
  description, course mode (traditional | competency_based), term (from
  `fetchOrgTerms`), and grade level.
- **FR-2.** Submitting Basics MUST call `POST` create-course with
  `{ title, description, courseType, termId?, gradeLevel? }`; returning to
  Basics on an existing draft MUST `PUT` the course, not create a second one.
- **FR-3.** Step 2 MUST offer the blank template plus the shared starter
  templates and, on continue, apply the chosen template's syllabus sections via
  the syllabus patch endpoint.
- **FR-4.** For traditional courses, Step 3 MUST let the user name (or skip) the
  first module and create it.
- **FR-5.** For competency-based courses, Step 3 MUST let the user author
  competencies, each with ≥1 sub-outcome and an assessment title, mirroring
  web's validation rules.
- **FR-6.** The wizard MUST validate identically to web (non-empty title;
  competency completeness) and surface inline, localized errors.
- **FR-7.** On finish, the app MUST refresh course lists and deep-link into the
  new course workspace.
- **FR-8.** The create entry point MUST offer a secondary path "Import from
  Canvas" that routes to the MOB.2 flow.
- **FR-9.** The create action MUST be gated by the `course:create` permission
  (`CourseCreateLogic.courseCreatePermission`); users lacking it MUST NOT see
  the entry point.
- **FR-10.** Draft state SHOULD survive app backgrounding within a session.

## 6. Non-Functional Requirements

- **Performance** — step transitions < 100 ms; create/patch round-trips show
  optimistic progress; term fetch cached per org for the session.
- **Security** — all calls carry the auth token; server enforces `course:create`
  and org scoping. No client-side trust of permission.
- **Privacy & Compliance** — no new PII; course metadata only.
- **Accessibility** — WCAG 2.1 AA: step indicator exposes "step N of 3" to
  VoiceOver/TalkBack; every field labelled; error text associated with fields;
  min 44×44 pt targets.
- **Scalability** — n/a (single-record writes).
- **Reliability** — idempotent resume: re-entering Basics updates the same draft
  by `courseCode`; network failure leaves a recoverable draft.
- **Observability** — emit `course_create_started/step_completed/finished` with
  mode + template id via the existing mobile telemetry channel.
- **Maintainability** — share `CourseCreateLogic` state machine across
  platforms; keep template data in sync with `course-create-templates.ts`.
- **Internationalization** — all strings via `mobile.createCourse.*` keys
  (already partially present); templates localized.
- **Backward compatibility** — no API changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor with `course:create`, *when* they complete all
  3 steps for a traditional course, *then* a course exists with the chosen
  template's syllabus and a first module, and they land on its workspace.
- **AC-2.** *Given* a competency course, *when* they submit Step 3 with a
  competency missing a sub-outcome, *then* a localized validation error blocks
  submission (parity with web `validateCompetencies`).
- **AC-3.** *Given* a draft created in Step 1, *when* they go back and change the
  title, *then* the same course is updated (no duplicate course).
- **AC-4.** *Given* a user without `course:create`, *when* they open the courses
  tab, *then* no create entry point is shown.
- **AC-5.** *Given* the create screen, *when* the user taps "Import from Canvas",
  *then* the MOB.2 flow opens.

## 8. Data Model

- **No new tables.** Reuses the courses, syllabus, modules, terms, and outcomes
  tables already backing web create.
- Client-side draft model only (`CourseCreateLogic` state): mode, template id,
  competency drafts, first-module title. No migration.

## 9. API Surface

Existing endpoints (already consumed by web); mobile wires the same calls:

- `POST /api/v1/courses` — create (`{title, description, courseType, termId?, gradeLevel?}`).
- `PUT /api/v1/courses/{courseCode}` — update draft on back-navigation.
- `PATCH` course syllabus (template application) — as used by
  `patchCourseSyllabus`.
- `POST /api/v1/courses/{courseCode}/structure/modules` — first module.
- Competency/outcome writes as used by the web competency finish path.
- `GET` org terms (`fetchOrgTerms`) and grade-level enum.

No OpenAPI changes; document that mobile now consumes these.

## 10. UI / UX

- **New/expanded screens:** iOS `CourseCreateView`, Android
  `CourseCreateScreen` — bring to parity. Add a create-source chooser
  (Scratch/Template vs Import from Canvas).
- **Flows:** (1) Choose source → (2) Basics → (3) Syllabus template →
  (4a) First module *or* (4b) Competencies → finish → workspace.
- **States:** loading (term fetch), submitting (per step), inline validation
  errors, offline (disable submit, keep draft), empty template list fallback to
  blank.
- **Mobile/responsive:** single-column step layout; sticky footer nav
  (Back/Continue); competency editor uses expandable cards.
- **Accessibility:** focus order top→bottom; step announcement; error focus.
- **Copy & i18n:** `mobile.createCourse.*` (extend existing keys for
  competencies, terms, grade level, import-source).

## 11. AI / ML Considerations

Not AI-touching, except an optional link to the existing lesson generator from
the finished-course screen (no new model usage).

## 12. Integration Points

- iOS: `Core/LMS/CourseCreateLogic.swift`, `LMSAPICourseCreate.swift`,
  `Features/Courses/Create/CourseCreateView.swift`.
- Android: `core/lms/CourseCreateLogic.kt`,
  `core/lms/LmsFeatureModelsCourseCreate.kt`,
  `features/courses/create/CourseCreateScreen.kt`.
- Shared template source parity with `course-create-templates.ts`.
- Hands off to MOB.2 (Canvas) and the shipped course workspace nav.

## 13. Dependencies & Sequencing

- Must ship after: nothing (backend ready).
- Must ship before: MOB.2 depends on this screen for its entry point.
- Shared infra: none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Template drift between web and mobile | M | M | Snapshot test comparing template ids/sections to `course-create-templates.ts` |
| Competency editor complexity on small screens | M | M | Card-based progressive disclosure; usability test |
| Duplicate courses on back-nav | M | H | Keep `createdCourse.courseCode`; switch to PUT (AC-3 test) |

## 15. Rollout Plan

- Flag: `ff_mobile_course_create_v2` (default off → staged on).
- Sequence: ship parity behind flag → dogfood with instructor cohort → GA.
- GA criteria: AC-1..5 pass on both platforms; crash-free ≥ 99.5%.
- Rollback: flag off falls back to current basic wizard.

## 16. Test Plan

- **Unit** — `CourseCreateLogic` state machine (step gating, validation, draft
  resume) on both platforms.
- **Integration** — create → syllabus patch → module create against a test org.
- **End-to-end** — XCUITest / Espresso happy paths for both modes + back-edit.
- **Security** — permission-gated entry; server rejects without `course:create`.
- **Accessibility** — VoiceOver/TalkBack script through all steps; axe-equivalent
  for any web-view fallback.
- **Performance** — step transition timing; term-fetch cache hit.
- **Manual** — offline draft recovery checklist.

## 17. Documentation & Training

- Update mobile help center "Create a course" with iOS/Android screenshots.
- Instructor quick-start note in release comms.
- API reference: mark create/syllabus/module endpoints as mobile-consumed.

## 18. Open Questions

1. Does mobile need grade-level parity for HE (web hides it for HE orgs)?
2. Should "Import from Canvas" appear for orgs without Canvas configured, or be
   hidden until integration is present?
3. Do we surface the AI lesson generator at finish, or defer to a later plan?

## 19. References

- Web: `clients/web/src/pages/lms/course-create.tsx`,
  `course-create-templates.ts`, `clients/web/src/lib/courses-api.ts`.
- iOS: `clients/ios/Lextures/Core/LMS/CourseCreateLogic.swift`,
  `Features/Courses/Create/CourseCreateView.swift`.
- Android: `.../core/lms/CourseCreateLogic.kt`,
  `.../features/courses/create/CourseCreateScreen.kt`.
- Related: [MOB.2](MOB.2-canvas-course-import.md), animations plans
  (`../animations/`).

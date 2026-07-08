# Intro Course — "Welcome to Lextures"

> Goal: give every new user a **guided, graded, first-run course** that teaches Lextures
> itself — its features, how learning works on the platform, the autonomous learner profile,
> the mobile app, and how to bring existing courses in from Canvas. Every new account is
> **automatically enrolled as a student** in this one canonical course, gated by a global
> platform feature flag that is **on by default** and can be turned off by a platform admin.

## Why this folder exists

New users land in an empty product. There is no shared, opinionated introduction to what
Lextures does or how to get value from it, so activation depends on users discovering features
on their own. This epic ships a real course — modules, content pages, quizzes, and graded
assignments — that doubles as the canonical demonstration of the product. Because it is a
*real* course (not a marketing tour), it also seeds every account with a populated dashboard,
a gradebook with entries, and enough behavioural signal to bootstrap the
[learner profile](../learner-profile/README.md) and adaptive engines.

### Hard constraints (from product direction)

1. **One canonical course, shared.** A single platform-owned course ("Welcome to Lextures"),
   provisioned idempotently, owned by a system instructor. Not a per-user copy (that would
   explode course/grade rows); every learner is a *student* enrolled in the same course.
2. **Automatic enrollment.** Every new user, however they are created (password signup, SSO,
   Clever/ClassLink, Canvas-import provisioning, admin bulk create), is enrolled as a student
   the moment the account exists. Existing users are backfilled once when the flag turns on.
3. **On by default, admin-disableable.** A global platform feature flag `intro_course_enabled`
   defaults **true**. A platform admin can turn it off; doing so stops new auto-enrollments and
   hides discovery surfaces but never deletes enrollments, grades, or content.
4. **Graded, but human-instructor-free.** Students are given assignments and grades, but no
   human grades at platform scale: quizzes auto-score, assignments auto-complete to full
   points (optionally routed through the grader agent when enabled). The gradebook is real.
5. **Content lives in code.** The curriculum is authored as versioned markdown fixtures in the
   repo and synced idempotently into the course, so content is code-reviewed, localizable, and
   re-deployable — never hand-edited in production and lost on the next sync.

## Conventions

- **File naming:** `IC{NN}-{kebab-slug}.md` (`IC` = *Intro Course*, mirroring `LP##`/`M##`).
- Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md). Because `docs/MISSING_FEATURES.md`
  was retired, the template's "Source" line points at this product direction.
- A plan is **ready** when every template section is filled (no `…` placeholders).
- **Migrations:** the highest committed migration is `357_*` and the learner-profile epic
  reserves `358_*`. This epic reserves `370_*` onward (each plan states its number); renumber
  on merge if the sequence has advanced.
- **Code layout:** provisioning + sync + enrollment service in
  `server/internal/service/introcourse/`, repo in `server/internal/repos/introcourse/`,
  admin/status handlers in `server/internal/httpserver/intro_course_http.go`. The course
  itself is a normal `course.courses` row consumed by all existing course/gradebook/mobile
  surfaces. Content fixtures live under `server/internal/service/introcourse/content/`.
- **Course identity (stable keys, used for idempotent provisioning):**
  - `short_code = "LEX-WELCOME"` — the immutable idempotency key.
  - `title = "Welcome to Lextures"`, `course_code = "Getting Started with Lextures"`.
  - Owner/instructor: a dedicated system user "Lextures Guide"
    (`a0000000-0000-4000-8000-000000000002`, sibling of the existing platform inbox sender
    `a0000000-0000-4000-8000-000000000001`).

## Severity legend

- **BLOCKER** — the "every new user gets a guided first course" promise is not real without it.
- **MAJOR** — a surface a user or admin would expect and activation suffers without.
- **MINOR** — parity, polish, or an additional surface.

## Story index

### Platform (build once — everything else plugs into it)

| ID | Plan | Severity | Depends on |
|---|---|---|---|
| IC01 | [Foundation — course provisioning, system instructor & feature flag](IC01-foundation-provisioning-flag.md) | BLOCKER | — |
| IC02 | [Automatic student enrollment on account creation + backfill](IC02-automatic-enrollment.md) | BLOCKER | IC01 |

### Curriculum & assessment

| ID | Plan | Severity | Depends on |
|---|---|---|---|
| IC03 | [Curriculum & content — features, learning patterns, learner profile, mobile, Canvas import](IC03-curriculum-content.md) | BLOCKER | IC01 |
| IC04 | [Graded assessments & automated grading (quizzes, assignments, gradebook)](IC04-graded-assessments-autograding.md) | MAJOR | IC01, IC03 |
| IC05 | [Progress, completion & completion credential](IC05-progress-completion-credential.md) | MAJOR | IC02, IC03, IC04 |

### Surfaces & governance

| ID | Plan | Severity | Depends on |
|---|---|---|---|
| IC06 | [Web onboarding surfaces & discoverability](../completed/intro-course/IC06-web-onboarding-surfaces.md) ✓ | MAJOR | IC02, IC03 |
| IC07 | [Mobile intro course experience](../completed/intro-course/IC07-mobile-intro-course.md) ✓ | MINOR | IC03, IC06 |
| IC08 | [Admin governance, localization & content versioning](../completed/intro-course/IC08-admin-governance-localization.md) ✓ | MAJOR | IC01, IC03 |

## Sequencing at a glance

```
IC01 Foundation (course + system instructor + flag) ─┬─► IC02 Auto-enrollment ─┐
                                                      ├─► IC03 Curriculum ──────┼─► IC05 Progress/completion
                                                      │        │                │
                                                      │        └─► IC04 Grading ┘
                                                      ├─► IC06 Web surfaces ─► IC07 Mobile
                                                      └─► IC08 Admin / i18n / versioning
```

IC01 ships first and alone (a real, empty-of-students course behind a flag). IC02 makes every
account a student. IC03 fills the course with content; IC04 makes its items graded; IC05 closes
the loop with completion. IC06/IC07 are the discovery/UX surfaces; IC08 is the admin + localization
+ content-versioning governance layer and is a **launch gate for GA** (an admin must be able to
turn it off and localize it before broad rollout).

## Relationship to existing plans & code

This epic **reuses**, and does not duplicate, shipped machinery:

- **Courses / modules / gradebook** — `course.courses`, `course.course_structure_items`,
  `course.module_content_pages`, `course.module_quizzes`, `course.module_assignments`,
  `course.course_grades`, `course.assignment_groups`. The intro course is an ordinary course.
- **Enrollment** — `course.course_enrollments` (`role='student'`); the auto-enroll path reuses
  the existing enrollment repo and grant model.
- **Feature-flag stack** — `settings.platform_app_settings` + `config.Config` +
  `repos/platformconfig` + `httpserver/platform_features.go` +
  `clients/web/src/lib/platform-features.ts` (modelled exactly on the just-added
  `learner_profile_enabled` flag).
- **Learner profile ([LP epic](../learner-profile/README.md))** — the intro course is the first
  source of behavioural signal for a brand-new account; IC03/IC04 deliberately generate quiz
  attempts, submissions, and page views so LP facets have something to derive from on day one.
- **Grader agent** (`grader_agent_enabled`) and **completion credentials**
  (`ff_completion_credentials`, `credentials_http.go`) — optional consumers for IC04/IC05.
- **Canvas import** (`server/internal/httpserver/canvas_*`) — the *subject* of IC03's import
  module and the reason the course teaches it; not modified by this epic.

# IC06 — Web Onboarding Surfaces & Discoverability

> Implementation plan. Source: product direction — the intro course must be the obvious first
> thing a new user does. Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC06 |
| **Section** | Intro Course |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web team |
| **Depends on** | IC02 (enrolled students), IC03 (content), IC05 (progress) |
| **Unblocks** | IC07 (mirrors these surfaces on mobile) |

---

## 1. Problem Statement

Auto-enrollment (IC02) puts the intro course in a new user's course list, but a course buried in a
list is not an onboarding experience. New users need the intro course **surfaced prominently** on
first login — a clear call-to-action, a progress indicator as they work through it, and a
celebration on completion — so activation actually happens. This plan adds the web discovery and
progress surfaces that turn the enrolled course into a guided first-run flow.

## 2. Goals

- Surface the intro course **prominently on the dashboard** for enrolled users who haven't
  completed it (a "Start here" / "Continue onboarding" card with progress from IC05).
- Provide a **first-login entry point** (welcome banner / CTA) that deep-links straight into the
  next incomplete item.
- Show **progress** (percent, modules done, next up) inside the course and on the dashboard card,
  reading IC05's `GET /me/intro-course`.
- **Celebrate completion** (badge/certificate + dismissible congrats) and then gracefully demote
  the course from prominence.
- Degrade gracefully when the flag is off / user not enrolled / already completed / errors.

## 3. Non-Goals

- The course content (IC03), grading (IC04), or completion/credential logic (IC05) — this plan
  only *renders* their data.
- Mobile surfaces (IC07).
- A general onboarding-checklist framework (there is an `ff_onboarding_flow` flag; this plan
  integrates with it if present but does not build it).
- Admin configuration UI (IC08).

## 4. Personas & User Stories

- **As a first-time user**, I want an unmistakable "start here" on my dashboard, so I know what to
  do first.
- **As a returning learner mid-course**, I want a "continue where you left off" that jumps to my
  next item, so I don't hunt.
- **As someone who finished**, I want a celebration and my certificate, then the onboarding card
  to step aside, so my dashboard reflects real coursework.
- **As a user in a deployment with the flag off**, I want no dangling onboarding UI.

## 5. Functional Requirements

- **FR-1.** The dashboard MUST show a prominent **intro-course card** when the caller is enrolled
  and not completed, showing title, progress (from `GET /me/intro-course`), and a CTA deep-linking
  to `nextItem.route`.
- **FR-2.** On **first login after enrollment**, the app SHOULD show a one-time welcome
  banner/toast ("Welcome — start with the guided intro course") that routes into the course; it
  MUST be dismissible and not reappear once dismissed or once the course is started.
- **FR-3.** Inside the intro course, a **progress indicator** (e.g. "Module 3 of 7 · 42%") MUST be
  shown, reading IC05 data; each module reflects done/current/upcoming state.
- **FR-4.** On **completion**, the app MUST show a celebration (confetti/badge + link to the
  certificate when `credentialId` present) once, then demote the dashboard card to a small
  "Onboarding complete ✓ (revisit)" state.
- **FR-5.** All surfaces MUST be gated on `introCourseEnabled` (platform-features) **and** actual
  enrollment; when off/not-enrolled, nothing renders (no empty shells).
- **FR-6.** Surfaces MUST handle loading (skeleton), error (fall back to a plain link to the
  course), and the already-completed state (no CTA nagging).
- **FR-7.** Deep links MUST route to real in-app locations (course item, gradebook, mobile
  download, learner profile, Canvas import) so the tour is interactive.
- **FR-8.** The welcome-banner "seen/dismissed" state MUST persist per user (server-side pref or
  existing onboarding-event mechanism), not just local storage, so it's stable across devices.

## 6. Non-Functional Requirements

- **Performance** — Dashboard card adds one lightweight `GET /me/intro-course` (p95 ≤ 80 ms,
  IC05); no layout shift (reserve space / skeleton). Celebration assets lazy-loaded.
- **Security** — All data is the caller's own; no cross-user reads. Deep links same-origin.
- **Privacy & Compliance** — No new PII. Certificate link respects IC05 share/consent settings.
- **Accessibility** — WCAG 2.1 AA: card and banner keyboard-focusable, progress announced via
  `aria`, celebration not conveyed by color/animation alone and honoring reduced-motion
  (`ff_high_contrast_reduced_motion`), focus management on banner dismiss.
- **Scalability** — Pure client rendering over one cached endpoint; no scaling concerns.
- **Reliability** — Graceful degradation to a static "Open the intro course" link if the progress
  endpoint fails.
- **Observability** — Client events: `intro_course_card_view`, `intro_course_cta_click`,
  `intro_course_banner_dismiss`, `intro_course_completed_celebration_view` (existing analytics
  pipeline) to measure the funnel.
- **Maintainability** — One `IntroCourseCard` + `useIntroCourseProgress` hook reused across
  dashboard/course/completion; no duplicated fetch logic.
- **Internationalization** — All copy via existing i18n; RTL-safe (`rtl_enabled`).
- **Backward compatibility** — Additive components; hidden entirely when flag off.

## 7. Acceptance Criteria

- **AC-1.** *Given* an enrolled, not-completed student on the dashboard, *when* it loads, *then* a
  prominent intro-course card shows correct progress and a CTA to the next item.
- **AC-2.** *Given* a first login, *when* the welcome banner appears and the user dismisses it,
  *then* it does not reappear on the next login (server-persisted).
- **AC-3.** *Given* a mid-course student, *when* they click "Continue", *then* they land on their
  next incomplete item.
- **AC-4.** *Given* a student completes the course, *when* the dashboard next loads, *then* a
  one-time celebration shows (with certificate link if present) and the card demotes to a small
  completed state.
- **AC-5.** *Given* `introCourseEnabled=false` or a non-enrolled user, *when* the dashboard loads,
  *then* no intro-course surfaces render.
- **AC-6.** *Given* the progress endpoint errors, *when* the dashboard loads, *then* a static
  "Open the intro course" link renders (no crash, no infinite spinner).
- **AC-7.** *Given* reduced-motion is set, *when* completion is celebrated, *then* no motion-heavy
  animation plays.

## 8. Data Model

No new tables. Reads IC05 `GET /me/intro-course`. Welcome-banner dismissal persisted via an
existing user-preference/onboarding-event store (reuse `repos/onboardingevent` or user prefs;
confirm the right store) rather than a new table.

## 9. API Surface

No new server endpoints — consumes IC05's `GET /api/v1/me/intro-course` and the existing
`GET /api/v1/platform-features` (`introCourseEnabled`). Banner-dismissal uses an existing prefs
endpoint (or a tiny `PUT /api/v1/me/intro-course/banner-dismissed` if none fits — prefer reuse).

## 10. UI / UX

New/changed components (web):

1. **`IntroCourseCard`** (dashboard) — states: not-started ("Start here"), in-progress
   (progress + "Continue"), completed (small "✓ Onboarding complete · revisit"). Hidden when flag
   off / not enrolled.
2. **`IntroWelcomeBanner`** — one-time, dismissible first-login CTA.
3. **In-course progress rail** — module list with done/current/upcoming, percent, "next up".
4. **`IntroCompletionCelebration`** — modal/toast on first completion; certificate link; reduced
   motion aware.

Key flows: (1) first login → banner → next item; (2) return → dashboard card → continue; (3)
finish last item → celebration → certificate → demoted card. Empty/loading/error/offline states
specified in FR-6. Copy via i18n; focus order and ARIA annotated in the component specs.

## 11. AI / ML Considerations

None.

## 12. Integration Points

- **Data:** IC05 `/me/intro-course`, platform-features (`introCourseEnabled`).
- **Web app:** dashboard (`clients/web/src/…` dashboard/home), course view, i18n, analytics client,
  reduced-motion/RTL flags.
- **Prefs:** existing onboarding-event / user-preference store for banner dismissal.
- **Deep-link targets:** course items, gradebook, mobile download page, learner-profile settings,
  Canvas import.

## 13. Dependencies & Sequencing

- **After:** IC02 (enrollment), IC03 (content to link into), IC05 (progress/completion data).
- **Before:** IC07 (mobile mirrors these surfaces).
- **Shared infra:** web app, i18n, analytics — present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Onboarding card nags completed users | M | M | Demote to small completed state after completion (FR-4) |
| Banner re-shows across devices | M | L | Server-persisted dismissal (FR-8) |
| Layout shift / spinner jank on dashboard | M | M | Skeleton + reserved space; cached endpoint |
| Dangling UI when flag off | L | M | Gate every surface on flag + enrollment (FR-5, AC-5) |
| Deep link to unconfigured feature (e.g. Canvas not set up) | M | M | Links go to the feature's own empty/setup state; IC03 conditions modules on flags |

## 15. Rollout Plan

- **Flag:** `intro_course_enabled` (via `introCourseEnabled`).
- **Sequencing:** build components behind the flag → verify against IC05 data on staging → a11y
  audit → enable. Optionally A/B the banner copy.
- **Dogfood:** internal new-account walkthroughs on web across the three states.
- **GA criteria:** funnel events firing; a11y pass; graceful states verified; no nag after
  completion.
- **Rollback:** disable flag → surfaces vanish (course still accessible via course list).

## 16. Test Plan

- **Unit** — card state machine (not-started/in-progress/completed); banner dismissal persistence;
  flag/enrollment gating.
- **Integration** — hook fetches `/me/intro-course`; error → static link fallback.
- **End-to-end (Playwright)** — new user: banner → next item; mid-course continue; completion
  celebration + certificate; flag-off renders nothing.
- **Accessibility** — axe on dashboard with card + banner + celebration; keyboard/focus/reduced-
  motion/RTL checks.
- **Performance** — dashboard TTI unaffected; no CLS from the card.
- **Manual exploratory** — dismiss banner on device A, confirm hidden on device B; complete then
  reload.

## 17. Documentation & Training

- Help-center: "Your Welcome to Lextures course & where to find it."
- Component docs for `IntroCourseCard` / hook reuse.
- Release note: the new first-run experience.

## 18. Open Questions

1. Should the intro course be *pinned* above all other courses, or shown as a distinct
   "Onboarding" panel separate from the course grid? (Leaning distinct panel.)
2. Integrate with `ff_onboarding_flow` (if it's a checklist) as one step, or stand alone? (Depends
   on what that flag renders — confirm.)
3. How aggressive should re-prompting be for users who ignore the course (email nudge via IC05
   event)? (Coordinate with IC05/notifications; avoid nagging.)
4. Certificate display: inline badge, dedicated page, or link to the credentials surface?

## 19. References

- Existing files: `clients/web/src/lib/platform-features.ts`, web dashboard/home components,
  `server/internal/repos/onboardingevent/onboardingevent.go`, reduced-motion/RTL flags.
- Related plans: [IC02](IC02-automatic-enrollment.md), [IC05](IC05-progress-completion-credential.md),
  [IC07](IC07-mobile-intro-course.md).

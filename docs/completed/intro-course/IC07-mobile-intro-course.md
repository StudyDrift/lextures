# IC07 — Mobile Intro Course Experience

> Implementation plan. Source: the intro course teaches the mobile app *and* must be completable
> on it. Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC07 |
| **Section** | Intro Course |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Mobile team |
| **Depends on** | IC03 (content), IC06 (web surfaces to mirror) |
| **Unblocks** | — |

---

## 1. Problem Statement

The intro course includes a module about the mobile app, and many new users will open Lextures
first on their phone. The course must therefore be **fully discoverable and completable on
mobile** — reading pages, taking quizzes, submitting the capstone, seeing progress, and getting a
completion celebration — mirroring the web surfaces (IC06). Without this, a mobile-first user
either can't finish onboarding or gets an inconsistent experience from the one the course itself
is describing.

## 2. Goals

- Surface the intro course prominently in the **mobile home** (a "Start here" card with progress
  from `GET /me/intro-course`).
- Ensure intro-course **content pages, quizzes, and the capstone assignment render and submit**
  correctly on mobile using existing mobile course components.
- Mirror **progress + completion celebration** on mobile.
- Make the mobile-module's **"try it" deep links** work natively (open the app's mobile download
  page / notification settings / relevant screens).
- Degrade gracefully when the flag is off / offline / not enrolled.

## 3. Non-Goals

- Content authoring (IC03), grading (IC04), completion logic (IC05) — mobile only renders their
  data via existing APIs.
- New backend endpoints (reuses IC05's `/me/intro-course` + existing course/quiz/assignment APIs).
- Full offline authoring of submissions (reuse whatever offline support the mobile app already
  has; no new offline framework here).

## 4. Personas & User Stories

- **As a mobile-first new user**, I want the intro course front-and-center on my phone, so I can
  onboard without a laptop.
- **As a student on the go**, I want to read a page, take the quiz, and submit the capstone from
  mobile, so I can finish in spare moments.
- **As a user learning about the app**, I want the "install/notifications" try-it steps to open
  the right native screens, so the lesson is real.

## 5. Functional Requirements

- **FR-1.** The mobile home MUST show an **intro-course card** (title + progress + CTA to next
  item) when enrolled and not completed, gated on `introCourseEnabled` and enrollment.
- **FR-2.** Intro-course **content pages** MUST render on mobile (markdown, images with alt text,
  in-app deep links) via existing mobile content-page rendering.
- **FR-3.** Module **quizzes** MUST be takeable on mobile and auto-score (IC04) via existing
  mobile quiz components; the **capstone assignment** MUST be submittable (text at minimum).
- **FR-4.** **Progress** and a **completion celebration** (with certificate link when present)
  MUST appear on mobile, mirroring IC06.
- **FR-5.** The mobile module's **deep links** MUST resolve to native screens where applicable
  (e.g. push-notification settings, account/profile, the learner-profile screen from
  [LP10](../learner-profile/LP10-mobile-learner-profile.md)); non-native targets open in-app web.
- **FR-6.** All surfaces MUST handle **offline/enrolled/completed/error** states gracefully
  (cached progress if available; static entry if not).

## 6. Non-Functional Requirements

- **Performance** — Home card reads the cached `/me/intro-course`; content pages/quizzes reuse
  existing mobile fetch/caching. No new heavy assets; celebration lazy.
- **Security** — Caller's own data only; deep links same-origin; submissions authenticated.
- **Privacy & Compliance** — No new PII; certificate/share respects IC05 consent.
- **Accessibility** — Mobile a11y parity: dynamic type, screen-reader labels, sufficient contrast,
  reduced-motion honored on the celebration, touch targets ≥ 44px.
- **Scalability** — Client-side over existing endpoints; none.
- **Reliability** — Graceful offline: show cached progress + last-synced content; queue the
  capstone submission if the app already supports offline submission, else require connectivity
  with a clear message.
- **Observability** — Mobile analytics: card view, CTA tap, completion celebration view (existing
  mobile analytics pipeline).
- **Maintainability** — Reuse existing mobile course/quiz/assignment components; add only the
  home card + progress/celebration glue.
- **Internationalization** — Existing mobile i18n + RTL; strings from IC03/IC06 keys.
- **Backward compatibility** — Additive; hidden when flag off.

## 7. Acceptance Criteria

- **AC-1.** *Given* an enrolled, not-completed student on mobile home, *then* the intro-course card
  shows correct progress and routes to the next item.
- **AC-2.** *Given* a mobile student, *when* they read a page, take a module quiz, and submit the
  capstone, *then* all render/submit correctly and grades appear (IC04).
- **AC-3.** *Given* completion, *when* mobile home reloads, *then* a celebration shows (certificate
  link if present) and the card demotes.
- **AC-4.** *Given* the mobile module's "notification settings" try-it link, *when* tapped, *then*
  it opens the native notifications screen.
- **AC-5.** *Given* the flag is off or the user isn't enrolled, *then* no intro-course surfaces
  appear on mobile.
- **AC-6.** *Given* the device is offline, *when* the home loads, *then* cached progress or a
  static entry shows without crashing.

## 8. Data Model

No new tables. Reuses IC05 `/me/intro-course` and existing course/quiz/assignment APIs.

## 9. API Surface

No new endpoints — mobile consumes existing IC05 + course APIs. Deep-link scheme reuses the app's
existing route/URL handling.

## 10. UI / UX

- **Mobile intro-course card** on home (states mirror IC06: not-started / in-progress / completed).
- **In-course progress** (module list + percent) using existing mobile course navigation.
- **Completion celebration** sheet with certificate link; reduced-motion aware.
- Content pages, quizzes, capstone submission use existing mobile components (verify layout at
  small widths and with dynamic type). Empty/loading/offline/error states per FR-6.

## 11. AI / ML Considerations

None (grader-agent capstone path, if enabled, is server-side per IC04; mobile just displays the
result/feedback).

## 12. Integration Points

- **Data/APIs:** IC05 `/me/intro-course`; existing mobile course, content-page, quiz, and
  assignment-submission flows; platform-features (`introCourseEnabled`).
- **Native screens:** notification settings, account/profile, learner-profile
  ([LP10](../learner-profile/LP10-mobile-learner-profile.md)).
- **Mobile app:** `clients/mobile` (+ `clients/ios`, `clients/android`), mobile i18n
  (`clients/mobile/locales`), mobile analytics.
- **Related mobile plans:** `docs/plan/mobile/` (course rendering, settings) and
  `docs/MOBILE_PLAN.md`.

## 13. Dependencies & Sequencing

- **After:** IC03 (content), IC06 (web surfaces to mirror), IC05 (progress data).
- **Before:** —
- **Shared infra:** existing mobile course/quiz/assignment components — present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Content pages/quizzes render poorly on small screens | M | M | Verify at min widths + dynamic type; a11y pass; reuse tested mobile components |
| Capstone not submittable offline | M | M | Reuse existing offline submission if present; else clear "connect to submit" message |
| Deep links don't resolve to native screens | M | M | Map known targets to native routes; fallback to in-app web |
| Divergence from web progress/celebration | L | M | Same `/me/intro-course` source; shared copy keys |

## 15. Rollout Plan

- **Flag:** `intro_course_enabled` (`introCourseEnabled`).
- **Sequencing:** verify content/quiz/capstone rendering on mobile → add home card + progress +
  celebration → a11y + device matrix pass → ship with the mobile release train.
- **Dogfood:** internal testers complete the course entirely on iOS and Android.
- **GA criteria:** full completion possible on mobile; a11y pass; graceful offline/flag-off.
- **Rollback:** disable flag → mobile surfaces hidden (course still reachable in course list).

## 16. Test Plan

- **Unit** — home card state machine; flag/enrollment gating.
- **Integration** — `/me/intro-course` fetch + cache; deep-link resolution to native screens.
- **End-to-end** — complete the whole course on device (read → quiz → capstone → completion);
  offline home; flag-off hides surfaces.
- **Accessibility** — screen reader (VoiceOver/TalkBack) through one full module; dynamic type;
  reduced motion; contrast.
- **Device matrix** — small/large phones, tablets; iOS + Android.
- **Manual exploratory** — background/foreground mid-quiz; low connectivity submission.

## 17. Documentation & Training

- Help-center: "Doing the Welcome to Lextures course on mobile."
- Mobile release note.
- QA device-matrix checklist for the intro course.

## 18. Open Questions

1. Is offline submission of the capstone in scope for v1, or require connectivity? (Depends on
   existing mobile offline support.)
2. Should the mobile app *specifically* prompt onboarding on first launch (push/interstitial), or
   just surface the home card? (Coordinate with mobile onboarding + IC06.)
3. Which "try it" targets have native screens vs. in-app web fallback? (Enumerate during build.)

## 19. References

- Existing: `docs/MOBILE_PLAN.md`, `docs/plan/mobile/`, `clients/mobile`, mobile course/quiz
  components, [LP10 mobile learner profile](../learner-profile/LP10-mobile-learner-profile.md).
- Related plans: [IC03](IC03-curriculum-content.md), [IC06](IC06-web-onboarding-surfaces.md),
  [IC05](IC05-progress-completion-credential.md).

## 20. Implementation (2026-07)

**iOS (`clients/ios`):**

- `IntroCourseLogic`, models, and `LMSAPI.fetchIntroCourseProgress` / `markIntroCelebrationSeen`.
- Dashboard **intro-course card** (`IntroCourseEntryCard`) gated on `introCourseEnabled` with
  offline-cached progress, fallback link, and completed revisit state.
- **In-course progress rail** (`IntroCourseProgressRail`) on `C-WLCOME` modules tab.
- **Completion celebration** sheet (`IntroCompletionCelebrationSheet`) with credentials deep link.
- **Deep links:** `/settings/account`, `/settings/notifications`, `/settings/learner-profile`,
  `/courses`, and `/courses/.../modules/{content|quiz}/<id>` routes; content pages use
  `ContentLinkRouter` for in-app markdown links.
- **i18n:** `mobile.introCourse.*` keys in `clients/mobile/locales/*.json` (synced to
  `Localizable.xcstrings`).
- **Tests:** `IntroCourseLogicTests` (card state, celebration gating, deep-link resolution).

**Android (`clients/android`):** mirror of the above in `features/introcourse/` plus
`IntroCourseLogicTest` / extended `DeepLinkRouterTest`.

**Platform flag:** `introCourseEnabled` on `PlatformFeatures` / `MobilePlatformFeatures` (default
on when unset, matching web).

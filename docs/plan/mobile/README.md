# Lextures Mobile — Web-Parity Plan

> Goal: make the **native iOS (SwiftUI) and Android (Jetpack Compose) apps as useful
> as the web application** for users who do not have a computer. The student journey
> must be fully completable on a phone; instructors, parents, and self-learners get
> the high-value subset of their workflows. The UI/UX must be clean, intuitive, and
> obvious to any student with no training.

This folder holds one implementation story per mobile feature, following the
mobile-tuned story format established by [`../speed-grader-mobile.md`](../speed-grader-mobile.md)
(leaner and more actionable than the full web [`../_TEMPLATE.md`](../_TEMPLATE.md):
it adds **What already exists (reuse)**, **Files to touch**, and **Phasing**, which
matter for native work). The umbrella program docs live in
[`../21-mobile-offline-cross-platform/`](../21-mobile-offline-cross-platform/).

---

## 1. Why this matters

Not every Lextures user owns a laptop. K-12 students are phone-first, self-learners
study on commutes, and many households share a single device. If a feature exists
only on the web, those users effectively cannot use it. For launch, the bar is:
**a student can do their entire academic life — read content, take quizzes, submit
work, see grades and feedback, communicate, and study — without ever opening a
browser.** This plan sequences the work to hit that bar, then widens to parents,
self-learners, and on-the-go instructors.

## 2. Platform strategy (decided)

- **Native, not hybrid.** The apps are already native: iOS in SwiftUI
  (`clients/ios/Lextures`), Android in Jetpack Compose
  (`clients/android/app/src/main/kotlin/com/lextures/android`). We continue native —
  the 21.1 plan's "hybrid/Capacitor" open question is **closed in favor of native**,
  because the foundation (auth, networking, design system, navigation shell,
  notebooks, grading) already ships natively and performs well.
- **Thin client over the existing REST API.** No bespoke mobile backend. Each story
  reuses existing endpoints under `server/internal/httpserver/`; new server work is
  called out explicitly per story and kept minimal (push tokens, a couple of
  read aggregations). Every mobile request sends `X-Platform: ios|android` and
  `X-App-Version`.
- **Shared building blocks per platform.** New screens reuse the existing
  `Core/Networking` API client, `Core/Design` theme (`LexturesTheme`/`LexturesType`),
  `Core/Auth` session + secure token store, and the `LMSAPI*` layer. Stories add
  endpoints/models to those layers rather than spinning up parallel stacks.
- **Parity of capability, not pixels.** Mobile re-imagines each flow for a phone
  (one-question-at-a-time quizzes, card lists instead of grids, bottom sheets
  instead of modals). It is not a literal port of the web DOM.

## 3. Current state (what already ships natively)

Verified in the iOS/Android trees as of this plan:

| Area | iOS | Android | Notes |
|---|---|---|---|
| Auth (email/password, signup, secure token store, session, biometric lock, session mgmt, SSO/MFA/magic link) | ✅ | ✅ | Keychain / encrypted store; SSO/MFA/magic link ([M1.1](../completed/mobile/M1.1-sso-mfa-magic-link.md)) |
| Navigation shell (Home, Courses, Notebooks, Inbox, Profile tabs) | ✅ | ✅ | 5-tab bar; no deep-link routing yet |
| Dashboard / Home + announcements | ✅ | ✅ | Read-only |
| Courses: list, detail, syllabus, **grades (feedback, what-if)**, attendance (read), item detail | ✅ | ✅ | Student grades with rubric/annotation/a-v feedback |
| Grading: backlog, submissions list, **Speed Grader** | ✅ | ✅ | Instructor; see `../speed-grader-mobile.md` |
| Inbox: list, thread, compose | ✅ | ✅ | 1:1 messaging |
| Notebooks: editor, drawing, pages, slash commands, markdown, sync | ✅ | ✅ | Rich; near-parity with web notebooks |
| Profile + notifications list | ✅ | ✅ | Read; no settings depth |

**API already wired (mobile side):** courses, course structure, item detail, my
grades, my submission, grading backlog/submissions, submission grade put, mailbox
messages, send message, unread count, notifications, broadcasts, attendance
sessions, quiz **attempts (read)**, syllabus, notebooks, me.

**The big student-facing gaps** (drive P0): taking a quiz, **submitting** an
assignment, rendering module content types (pages/files/links/LTI/H5P/SCORM),
discussions, calendar/to-dos, viewing rich feedback (annotations, audio/video,
rubrics), what-if grades, the AI tutor, adaptive review/paths, native push +
deep links, and offline.

## 4. Epics & numbering

Stories are named `M{epic}.{n}-{slug}.md`. Each story targets **both** iOS and
Android unless noted. Epics:

| Epic | Theme |
|---|---|
| **M0** | Foundation & Platform — offline cache/sync, native push (APNs/FCM), deep links, biometric lock, accessibility, i18n/RTL, app-version/observability |
| **M1** | Auth & Onboarding — SSO, MFA, magic link, biometric unlock, onboarding/diagnostic |
| **M2** | Home, Dashboard & Notifications — dashboard widgets, to-dos, notification center, global feed |
| **M3** | Courses, Modules & Content — module list, content pages, files, external/LTI/H5P/SCORM/textbook items, conditional release |
| **M4** | Quizzes & Assessment — quiz taker, all question types, math/code input, timers/auto-submit, multiple attempts, lockdown |
| **M5** | Assignments & Submissions — file/text/media upload, resumable uploads, resubmission, peer review, originality |
| **M6** | Grades & Feedback — grades detail, what-if, rubric/annotation/audio-video feedback, standards-based view |
| **M7** | Communication — discussions, announcements, office hours, group spaces, collab docs, AI tutor, in-context help |
| **M8** | Adaptive & Study Tools — review (spaced repetition), paths, recommendations, hints, mastery, study insights, reading log |
| **M9** | Self-Learner & Commerce — catalog, enroll, checkout/billing, certificates/badges, gamification, reviews |
| **M10** | K-12 & Parent — parent portal, attendance-taking, behavior/PBIS, report cards, conference booking, hall pass, age-appropriate UI |
| **M11** | Instructor on Mobile — grading depth, take attendance, post announcement, quick authoring caps |
| **M12** | Portfolios & Credentials — e-portfolio, co-curricular transcript, CCR, certificates wallet |

## 5. Full backlog (every web feature → mobile disposition)

Disposition: **P0** = launch-blocking student core · **P1** = high-value breadth ·
**P2** = nice-to-have / later · **Web-only** = stays on web (authoring/admin-heavy or
desktop-bound), reachable via an in-app web view when a student truly needs it.
"Built" = already shipping natively (extend only).

### Student-critical (P0)

| Story | Source feature(s) | Status today | Disposition |
|---|---|---|---|
| [M4.1 Quiz taker & question types](../completed/mobile/M4.1-quiz-taker.md) | 2.2 question types, 2.3 math, 2.7 timers/auto-submit, 2.8 shuffling, 2.9 attempts, quiz attempt page | **DONE** | **P0** |
| [M5.1 Assignment submission](M5.1-assignment-submission.md) | 3.13 resubmission, 8.2 resumable upload, module assignment page | None (grade-only) | **P0** |
| [M3.1 Module content viewer](../completed/mobile/M3.1-module-content-viewer.md) ✅ | module content/external-link/textbook pages, 8.7 image/pdf preview, 1.11 conditional release | Done | **P0** |
| [M3.2 Course files browser](../completed/mobile/M3.2-course-files.md) ✅ | course-files-page, 8.1 storage, 8.7 preview | Done | **P0** |
| [M2.1 Calendar & to-dos](M2.1-calendar-todos.md) | calendar, todos-page, 16.5 feeds | None | **P0** |
| [M6.1 Grades, feedback & what-if](../completed/mobile/M6.1-grades-feedback.md) ✅ | my-grades, 3.1 annotation, 3.2 a/v feedback, 3.16 what-if, rubrics | **DONE** | **P0** |
| [M7.1 Course discussions](M7.1-discussions.md) | 6.1 threaded forums, course-discussions-page | None | **P0** |
| [M7.2 AI tutor & Ask-AI](M7.2-ai-tutor.md) | 6.9 AI tutor, ask-ai-page, 15.12 study buddy | None | **P0** |
| [M8.1 Review & spaced repetition](M8.1-review-spaced-repetition.md) | 1.5 spaced repetition, review-session-page | None | **P0** |
| [M0.1 Native push & deep links](M0.1-push-deep-links.md) | 6.3 push, 21.5 APNs/FCM | None | **P0** |
| [M0.2 Offline mode & sync](../completed/mobile/M0.2-offline-sync.md) ✅ | 7.3 offline PWA (web), 21.x offline | None | **P0** |

### High-value breadth (P1)

| Story | Source feature(s) | Disposition |
|---|---|---|
| [M1.1 SSO, MFA & magic link](../completed/mobile/M1.1-sso-mfa-magic-link.md) ✅ | 4.1/4.2 SSO, 4.6 MFA, 4.7 magic link | **Done** |
| [M1.2 Biometric unlock & sessions](../completed/mobile/M1.2-biometric-sessions.md) ✅ | 4.8/4.9 sessions, biometric | **Done** |
| [M1.3 Onboarding & placement diagnostic](../completed/mobile/M1.3-onboarding-diagnostic.md) | onboarding, 1.7/15.11 diagnostic | **Done** |
| [M2.2 Notification center & preferences](M2.2-notification-center.md) | 6.2 notifications, notification prefs | **P1** |
| [M3.3 Interactive content: H5P/SCORM/LTI](../completed/mobile/M3.3-interactive-content.md) ✅ | 8.12 H5P, 2.14 SCORM/xAPI, 2.12 LTI 1.3 | **Done** |
| [M5.2 Peer review](M5.2-peer-review.md) | 3.15 peer review | **P1** |
| [M6.2 Standards-based grades & mastery](M6.2-standards-mastery.md) | 3.7 SBG, 9.3 mastery heatmap, 13.4 report cards (student) | **P1** |
| [M7.3 Office hours booking](../completed/mobile/M7.3-office-hours.md) ✅ | 6.7 office hours | **Done** |
| [M7.4 Group spaces & collab docs](M7.4-groups-collab.md) | 6.6 groups, 6.5 collab docs | **P1** |
| [M8.2 Adaptive paths & recommendations](M8.2-paths-recommendations.md) | 1.4 paths, 1.8 recommendations, my-paths | **P1** |
| [M8.3 Study insights & self-reflection](M8.3-study-insights.md) | 9.1 progress, 9.9 reflection, study-insights | **P1** |
| [M8.4 Reading log & book club](M8.4-reading-log.md) | 13.8 leveled reader, reading-log/dashboard | **P1** |
| [M9.1 Catalog browse & enroll](M9.1-catalog-enroll.md) | 15.1 catalog, 15.2 self-paced enroll, 14.2 registration | **P1** |
| [M9.2 Checkout & billing](M9.2-checkout-billing.md) | 15.3 Stripe billing, checkout, 16.8 payment abstraction | **P1** |
| [M9.3 Certificates, badges & gamification](M9.3-certificates-gamification.md) | 15.5 certs/badges, 15.9 gamification, leaderboard | **P1** |
| [M10.1 Parent portal](M10.1-parent-portal.md) | 13.1 parent portal, parent-dashboard | **P1** |
| [M10.2 Conference booking (parent)](M10.2-conference-booking.md) | 13.12 conference scheduling | **P1** |
| [M11.1 Take attendance (instructor)](M11.1-take-attendance.md) | 13.2 daily attendance | **P1** |
| [M0.3 Accessibility: VoiceOver/TalkBack & dynamic type](../completed/mobile/M0.3-accessibility.md) ✅ | 12.1/12.2/12.8 a11y | **P1** |
| [M0.4 i18n, locale & RTL](../completed/mobile/M0.4-i18n-rtl.md) | 11.1 i18n, 11.2 RTL, 11.3/11.4 locale/tz | **Done** |
| [M1.4 Profile, settings & accommodations](M1.4-settings-accommodations.md) | settings, my-accommodations, 12.10 engine | **P1** |

### Later / situational (P2)

| Story | Source feature(s) | Disposition |
|---|---|---|
| [M3.4 Conditional release & prerequisites UX](../completed/mobile/M3.4-conditional-release.md) ✅ | 1.11 conditional release polish | **Done** |
| [M4.2 Lockdown / kiosk mode](M4.2-lockdown-kiosk.md) | 2.10 lockdown | **P2** |
| [M5.3 Code-execution questions](M5.3-code-execution.md) | 2.4 code exec | **P2** |
| [M10.3 Behavior/PBIS & hall pass](M10.3-behavior-hallpass.md) | 13.3 behavior, 13.9 hall pass | **P2** |
| [M10.4 Age-appropriate UI mode](M10.4-age-appropriate-ui.md) | 13.11 age-appropriate UI | **P2** |
| [M11.2 Post announcement / broadcast](M11.2-instructor-announce.md) | 13.10 broadcast, announcements compose | **P2** |
| [M12.1 e-Portfolio & artifacts](M12.1-eportfolio.md) | 14.12 e-portfolio, portfolios pages | **P2** |
| [M12.2 Credentials wallet & transcripts](M12.2-credentials-transcripts.md) | 14.13 CCR, transcripts, 15.6 LinkedIn share | **P2** |

### Newly identified gap stories (2026-06-30 parity scan)

A scan of the web LMS pages and server routes against the stories above surfaced
**distinct, student/instructor-facing surfaces with live endpoints that no story
covered**, plus the navigation redesign needed to hold them. Added as stories:

| Story | Source feature(s) | Server endpoints (exist) | Disposition |
|---|---|---|---|
| [M0.5 Redesign: role-aware IA & navigation](M0.5-redesign-information-architecture.md) | shell redesign; course workspace; role adaptation | — (client) | **P0 (foundation)** |
| [M0.6 Universal search & command palette](M0.6-universal-search.md) | global search, `command-palette-go-to` | `/search`, `/search/query`, `/library/search`, `/oer/search`, `/standards/search` | **P1** |
| [M3.5 Vibe activities](M3.5-vibe-activities.md) | `course-module-vibe-activity-page` (AI interactive content type) | `/courses/{c}/vibe-activities/{item}` | **P1** |
| [M3.6 Library, e-reserves & OER](M3.6-library-ereserves-oer.md) | 14.10 e-reserves, OER, `library-catalog-page` | `/library/search`, `/oer/search`, `/courses/{c}/library-resources/*` | **P1** |
| [M6.3 Immersive reader: read-aloud, captions & translation](M6.3-immersive-reader.md) | 12.x read-aloud/captions, 11.x translation, reading prefs | `/files/{o}/captions/*`, course-translation, reading-preferences | **P1** |
| [M7.5 Live classes & virtual meetings](M7.5-live-classes-virtual-meetings.md) | `course-live-page`, virtual meetings, whiteboards | `/meetings/*`, `/courses/{c}/meetings`, `/courses/{c}/whiteboards/*` | **P1** |
| [M7.6 Course feed & channels (real-time)](M7.6-course-feed-channels.md) | `course-feed-page` (distinct from M7.1 discussions) | `/courses/{c}/feed/*`, `/feed/ws` | **P1** |
| [M7.7 Course evaluations & surveys](M7.7-course-evaluations-surveys.md) | 14.7 course evaluations, surveys | `/courses/{c}/evaluations/status\|submit\|results` | **P1** |
| [M7.8 Academic advising (student)](M7.8-academic-advising.md) | advising notes + appointments (HE) | `/me/advising-notes`, `/me/advising/config`, scheduler | **P2** |
| [M11.3 Instructor insights & at-risk](M11.3-instructor-insights-at-risk.md) | `course-at-risk`, `course-whats-working`, student-progress | at-risk, instructor-insights, student-progress | **P1** |
| [M1.5 Profile depth: demographics, custom fields & research consent](../completed/mobile/M1.5-profile-depth-demographics-consent.md) | demographics, custom fields, research studies | demographics, custom-fields, research-consent | **DONE** |

> **The redesign ([M0.5](M0.5-redesign-information-architecture.md)) is the keystone.**
> The current flat 5-tab shell + four-chip course detail cannot surface the breadth above.
> M0.5 introduces a role-aware shell, a scalable course **workspace** sub-nav, a **More**
> hub, and a header **search** entry — and defines the destination-registry contract every
> story above plugs into. Sequence it first within its wave.

### Stays web-only (reachable via in-app web view)

Authoring and admin surfaces that are desktop-bound and out of scope for native:
question-bank/blueprint/outcomes authoring (2.1, 5.6, 9.5 authoring), gradebook
grid editing & curving/CSV (3.11, 3.17 instructor), org/admin consoles (5.x admin,
10.x compliance, 17.x ops, 18 admin experience), SIS/SCIM/OneRoster config (4.3–4.5,
13.7, 14.1), integrations/webhooks/marketplace (16.x), CLI (21-cli), course
authoring/import (QTI/Common Cartridge 2.13, Canvas import), proctoring config
(14.9), evaluation/template authoring (14.7). Mobile links out to these with a
"best on a larger screen" affordance rather than reimplementing them.

## 6. Sequencing / roadmap

0. **Wave 0 — Redesign foundation (P0).** [M0.5](M0.5-redesign-information-architecture.md)
   role-aware IA / course workspace / More hub / search entry lands first (behind a flag,
   with a clean fallback to the current shell) so every wave below has discoverable homes
   and a destination-registry contract to plug into. [M0.6](M0.6-universal-search.md)
   universal search follows immediately to fill the search entry.
1. **Wave 1 — Student core (P0).** M0.1 push + M0.2 offline scaffolding land first
   (cross-cutting), then M4.1 quiz taker, M5.1 submission, M3.1/M3.2 content+files,
   M2.1 calendar, M6.1 grades+feedback, M7.1 discussions, M7.2 AI tutor, M8.1 review.
   Exit criteria: a student completes a full week of coursework with no browser.
2. **Wave 2 — Breadth (P1).** Auth depth (M1.x), interactive content (M3.3 ✅) +
   M3.5 vibe activities + M3.6 library/e-reserves, standards/mastery (M6.2),
   M6.3 immersive reader, groups/office-hours (M7.3/M7.4) + M7.5 live classes +
   M7.6 course feed + M7.7 evaluations, adaptive (M8.2–M8.4), self-learner commerce
   (M9.x), parent (M10.1/M10.2), accessibility & i18n (M0.3/M0.4), settings (M1.4),
   instructor attendance (M11.1) + M11.3 instructor insights/at-risk.
3. **Wave 3 — Situational (P2).** Lockdown, code exec, behavior/hall-pass,
   age-appropriate UI, e-portfolio/credentials, instructor broadcast, M7.8 advising,
   M1.5 profile depth (demographics/custom fields/research consent).

Each wave ships behind staged store releases (TestFlight / Play internal track →
pilot cohort → GA). Per-feature server flags gate anything risky.

## 7. Cross-cutting requirements (every story inherits)

- **Accessibility:** VoiceOver/TalkBack labels, Dynamic Type / font scaling, ≥44×44pt
  targets, reduced-motion respect. Reuse `LexturesTheme`/`LexturesType`.
- **Offline-aware:** read paths show last-cached data with a staleness indicator;
  write paths queue and sync (see [M0.2](M0.2-offline-sync.md)).
- **Secure:** auth via the existing secure token store; no PII in logs; certificate
  pinning configurable.
- **Empty / loading / error / offline** states are mandatory for every screen.
- **Dark mode** follows system; both platforms already theme-aware.
- **No new backend unless the story says so**, and then it's minimal and listed.

## 8. Conventions

- File naming: `M{epic}.{n}-{kebab-slug}.md`.
- A story is "ready" when every section is filled (no `…` placeholders), names the
  exact iOS/Android files to touch, and lists acceptance criteria testable on device.
- Cross-link stories with relative markdown links.
- Reference existing native code by path (e.g.
  `clients/ios/Lextures/Features/Courses/CourseDetailView.swift`).
</content>
</invoke>

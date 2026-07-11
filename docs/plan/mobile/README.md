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
| Navigation shell (role-aware two-level left drawer: global + course) | ✅ | ✅ | Web-parity drawer replaced the bottom tab bar (#419); deep links wired ([M0.1](../completed/mobile/M0.1-push-deep-links.md)) |
| Dashboard / Home + announcements | ✅ | ✅ | Read + staff/admin compose |
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
assignment ✅, rendering module content types (pages/files/links/LTI/H5P/SCORM),
**discussions** ✅, calendar/to-dos, viewing rich feedback (annotations, audio/video,
rubrics), what-if grades, **the AI tutor** ✅, adaptive review/paths, native push +
deep links, and offline. **Peer review** ✅ (M5.2). **Reading log & leveled library** ✅ (M8.4).
**Immersive reader** (read-aloud, captions, translation, reading prefs) ✅ (M6.3).

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
| **M13** | Course Settings & Configuration — the full `/courses/{code}/settings/*` area: general, features, sections, grading, outcomes, grading agents, plagiarism, accessibility, translations, import/export, blueprint, archived |
| **M14** | Platform, Org & Account Administration — the global `/settings/*` area (permission-gated): integrations/API access, roles, people, orgs/units/terms, branding & AI governance, platform config, AI models/prompts/reports, integrations/provisioning, transcripts/advising, archived courses |

## 5. Full backlog (every web feature → mobile disposition)

Disposition: **P0** = launch-blocking student core · **P1** = high-value breadth ·
**P2** = nice-to-have / later · **Web-only** = stays on web (authoring/admin-heavy or
desktop-bound), reachable via an in-app web view when a student truly needs it.
"Built" = already shipping natively (extend only).

### Student-critical (P0)

| Story | Source feature(s) | Status today | Disposition |
|---|---|---|---|
| [M4.1 Quiz taker & question types](../completed/mobile/M4.1-quiz-taker.md) | 2.2 question types, 2.3 math, 2.7 timers/auto-submit, 2.8 shuffling, 2.9 attempts, quiz attempt page | **DONE** | **P0** |
| [M5.1 Assignment submission](../completed/mobile/M5.1-assignment-submission.md) ✅ | 3.13 resubmission, 8.2 resumable upload, module assignment page | **DONE** | **P0** |
| [M3.1 Module content viewer](../completed/mobile/M3.1-module-content-viewer.md) ✅ | module content/external-link/textbook pages, 8.7 image/pdf preview, 1.11 conditional release | Done | **P0** |
| [M3.2 Course files browser](../completed/mobile/M3.2-course-files.md) ✅ | course-files-page, 8.1 storage, 8.7 preview | Done | **P0** |
| [M2.1 Calendar & to-dos](../completed/mobile/M2.1-calendar-todos.md) ✅ | calendar, todos-page, 16.5 feeds | **DONE** | **P0** |
| [M6.1 Grades, feedback & what-if](../completed/mobile/M6.1-grades-feedback.md) ✅ | my-grades, 3.1 annotation, 3.2 a/v feedback, 3.16 what-if, rubrics | **DONE** | **P0** |
| [M7.1 Course discussions](../completed/mobile/M7.1-discussions.md) ✅ | 6.1 threaded forums, course-discussions-page | **DONE** | **P0** |
| [M7.2 AI tutor & Ask-AI](../completed/mobile/M7.2-ai-tutor.md) ✅ | 6.9 AI tutor, ask-ai-page, 15.12 study buddy | **DONE** | **P0** |
| [M8.1 Review & spaced repetition](../completed/mobile/M8.1-review-spaced-repetition.md) ✅ | 1.5 spaced repetition, review-session-page | **DONE** | **P0** |
| [M0.1 Native push & deep links](../completed/mobile/M0.1-push-deep-links.md) ✅ | 6.3 push, 21.5 APNs/FCM | **DONE** | **P0** |
| [M0.2 Offline mode & sync](../completed/mobile/M0.2-offline-sync.md) ✅ | 7.3 offline PWA (web), 21.x offline | None | **P0** |

### High-value breadth (P1)

| Story | Source feature(s) | Disposition |
|---|---|---|
| [M1.1 SSO, MFA & magic link](../completed/mobile/M1.1-sso-mfa-magic-link.md) ✅ | 4.1/4.2 SSO, 4.6 MFA, 4.7 magic link | **Done** |
| [M1.2 Biometric unlock & sessions](../completed/mobile/M1.2-biometric-sessions.md) ✅ | 4.8/4.9 sessions, biometric | **Done** |
| [M1.3 Onboarding & placement diagnostic](../completed/mobile/M1.3-onboarding-diagnostic.md) | onboarding, 1.7/15.11 diagnostic | **Done** |
| [M2.2 Notification center & preferences](../completed/mobile/M2.2-notification-center.md) ✅ | 6.2 notifications, notification prefs | **Done** |
| [M3.3 Interactive content: H5P/SCORM/LTI](../completed/mobile/M3.3-interactive-content.md) ✅ | 8.12 H5P, 2.14 SCORM/xAPI, 2.12 LTI 1.3 | **Done** |
| [M5.2 Peer review](../completed/mobile/M5.2-peer-review.md) ✅ | 3.15 peer review | **DONE** | **P1** |
| [M6.2 Standards-based grades & mastery](../completed/mobile/M6.2-standards-mastery.md) ✅ | 3.7 SBG, 9.3 mastery heatmap, 13.4 report cards (student) | **Done** |
| [M7.3 Office hours booking](../completed/mobile/M7.3-office-hours.md) ✅ | 6.7 office hours | **Done** |
| [M7.4 Group spaces & collab docs](../completed/mobile/M7.4-groups-collab.md) ✅ | 6.6 groups, 6.5 collab docs | **Done** |
| [M8.2 Adaptive paths & recommendations](../completed/mobile/M8.2-paths-recommendations.md) ✅ | 1.4 paths, 1.8 recommendations, my-paths | **Done** |
| [M8.3 Study insights & self-reflection](../completed/mobile/M8.3-study-insights.md) ✅ | 9.1 progress, 9.9 reflection, study-insights | **Done** |
| [M8.4 Reading log & book club](../completed/mobile/M8.4-reading-log.md) ✅ | 13.8 leveled reader, reading-log/dashboard | **Done** |
| [M9.1 Catalog browse & enroll](../completed/mobile/M9.1-catalog-enroll.md) ✅ | 15.1 catalog, 15.2 self-paced enroll, 14.2 registration | **Done** |
| [M9.2 Checkout & billing](../completed/mobile/M9.2-checkout-billing.md) ✅ | 15.3 Stripe billing, checkout, 16.8 payment abstraction | **Done** |
| [M9.3 Certificates, badges & gamification](../completed/mobile/M9.3-certificates-gamification.md) ✅ | 15.5 certs/badges, 15.9 gamification, leaderboard | **Done** |
| [M10.1 Parent portal](../completed/mobile/M10.1-parent-portal.md) | 13.1 parent portal, parent-dashboard | **P1** ✅ |
| [M10.2 Conference booking (parent)](../completed/mobile/M10.2-conference-booking.md) ✅ | 13.12 conference scheduling | **Done** |
| [M11.1 Take attendance (instructor)](../completed/mobile/M11.1-take-attendance.md) ✅ | 13.2 daily attendance | **Done** |
| [M0.3 Accessibility: VoiceOver/TalkBack & dynamic type](../completed/mobile/M0.3-accessibility.md) ✅ | 12.1/12.2/12.8 a11y | **Done** |
| [M0.4 i18n, locale & RTL](../completed/mobile/M0.4-i18n-rtl.md) | 11.1 i18n, 11.2 RTL, 11.3/11.4 locale/tz | **Done** |
| [M1.4 Profile, settings & accommodations](../completed/mobile/M1.4-settings-accommodations.md) ✅ | settings, my-accommodations, 12.10 engine | **Done** |

### Later / situational (P2)

| Story | Source feature(s) | Disposition |
|---|---|---|
| [M3.4 Conditional release & prerequisites UX](../completed/mobile/M3.4-conditional-release.md) ✅ | 1.11 conditional release polish | **Done** |
| [M4.2 Lockdown / kiosk mode](../completed/mobile/M4.2-lockdown-kiosk.md) ✅ | 2.10 lockdown | **DONE** |
| [M5.3 Code-execution questions](../completed/mobile/M5.3-code-execution.md) ✅ | 2.4 code exec | **Done** |
| [M10.3 Behavior/PBIS & hall pass](M10.3-behavior-hallpass.md) | 13.3 behavior, 13.9 hall pass | **P2** |
| [M10.4 Age-appropriate UI mode](../completed/mobile/M10.4-age-appropriate-ui.md) | 13.11 age-appropriate UI | **DONE** |
| [M11.2 Post announcement / broadcast](../completed/mobile/M11.2-instructor-announce.md) | 13.10 broadcast, announcements compose | **Done** |
| [M12.1 e-Portfolio & artifacts](M12.1-eportfolio.md) | 14.12 e-portfolio, portfolios pages | **P2** |
| [M12.2 Credentials wallet & transcripts](../completed/mobile/M12.2-credentials-transcripts.md) | 14.13 CCR, transcripts, 15.6 LinkedIn share | **Done** |

### Newly identified gap stories (2026-06-30 parity scan)

A scan of the web LMS pages and server routes against the stories above surfaced
**distinct, student/instructor-facing surfaces with live endpoints that no story
covered**, plus the navigation redesign needed to hold them. Added as stories:

| Story | Source feature(s) | Server endpoints (exist) | Disposition |
|---|---|---|---|
| [M0.5 Redesign: role-aware IA & navigation](../completed/mobile/M0.5-redesign-information-architecture.md) | shell redesign; course workspace; role adaptation | — (client) | **Done** |
| [M0.6 Universal search & command palette](../completed/mobile/M0.6-universal-search.md) | global search, `command-palette-go-to` | `/search`, `/search/query`, `/library/search`, `/oer/search`, `/standards/search` | **Done** |
| [M3.5 Vibe activities](../completed/mobile/M3.5-vibe-activities.md) | `course-module-vibe-activity-page` (AI interactive content type) | `/courses/{c}/vibe-activities/{item}` | **Done** |
| [M3.6 Library, e-reserves & OER](../completed/mobile/M3.6-library-ereserves-oer.md) | 14.10 e-reserves, OER, `library-catalog-page` | `/library/search`, `/oer/search`, `/courses/{c}/library-resources/*` | **Done** |
| [M6.3 Immersive reader: read-aloud, captions & translation](../completed/mobile/M6.3-immersive-reader.md) ✅ | 12.x read-aloud/captions, 11.x translation, reading prefs | `/files/{o}/captions/*`, course-translation, reading-preferences | **Done** |
| [M7.5 Live classes & virtual meetings](../completed/mobile/M7.5-live-classes-virtual-meetings.md) ✅ | `course-live-page`, virtual meetings, whiteboards | `/meetings/*`, `/courses/{c}/meetings`, `/courses/{c}/whiteboards/*` | **Done** |
| [M7.6 Course feed & channels (real-time)](../completed/mobile/M7.6-course-feed-channels.md) | `course-feed-page` (distinct from M7.1 discussions) | `/courses/{c}/feed/*`, `/feed/ws` | **Done** |
| [M7.7 Course evaluations & surveys](M7.7-course-evaluations-surveys.md) | 14.7 course evaluations, surveys | `/courses/{c}/evaluations/status\|submit\|results` | **P1** |
| [M7.8 Academic advising (student)](../completed/mobile/M7.8-academic-advising.md) | advising notes + appointments (HE) | `/me/advising-notes`, `/me/advising/config` | **DONE** |
| [M11.3 Instructor insights & at-risk](M11.3-instructor-insights-at-risk.md) | `course-at-risk`, `course-whats-working`, student-progress | at-risk, instructor-insights, student-progress | **P1** |
| [M11.4 Course People (roster) for teachers](M11.4-course-people-roster.md) | `course-enrollments-page` ("People" tab); `CourseWorkspaceSection.people` (registered, placeholder) | `/courses/{c}/enrollments`, `/courses/{c}/enrollments/{id}` (DELETE), `/courses/{c}/enrollments/{id}/message` | **P1** |
| [M1.5 Profile depth: demographics, custom fields & research consent](../completed/mobile/M1.5-profile-depth-demographics-consent.md) | demographics, custom fields, research studies | demographics, custom-fields, research-consent | **DONE** |

> **The redesign ([M0.5](../completed/mobile/M0.5-redesign-information-architecture.md)) is the keystone.**
> The former flat 5-tab shell + chip-based course detail could not surface the breadth above.
> M0.5 landed the role-aware destination registry, and the web-parity **two-level left drawer**
> (a global drawer + a course-scoped drawer, replacing the bottom tab bar and workspace chips; #419)
> now hosts it — plus a **More** hub and a header **search** entry. Every story above plugs into
> that destination-registry contract.

### Newly identified gap stories (2026-07-05 settings parity scan)

A scan of the web **settings** surfaces — the course settings area
(`pages/lms/course-settings.tsx` + `side-nav-course-settings-links.tsx`) and the global
settings area (`pages/lms/settings.tsx`) — against the mobile app found that **none of the
course-settings sections and none of the admin/global-settings views exist on mobile**. Only
account/profile ([M1.4](../completed/mobile/M1.4-settings-accommodations.md)) and notification
prefs ([M2.2](../completed/mobile/M2.2-notification-center.md)) were covered. This adds two
epics. Every course-settings server endpoint already exists (no backend work); global-settings
endpoints exist and are permission-gated (`rbac:manage` / `tenant:org-units:admin`) plus feature
flags. Admin-heavy consoles are scoped as **status/review + guarded light actions natively, with
link-out to web for credential entry and deep authoring**, consistent with the doctrine below.

**M13 — Course Settings & Configuration** (instructor/course-admin; gated by
`courseItemCreatePermission`; all endpoints exist):

| Story | Web source (settings tab) | Disposition |
|---|---|---|
| [M13.1 Settings shell + General](../completed/mobile/M13.1-course-settings-general.md) | `general` (basics, home, schedule, visibility, hero, theme, tz, publish) | **P1** (keystone) ✅ |
| [M13.2 Features, tools & caption policy](./M13.2-course-features-tools.md) | `features` (+ caption policy, consortium) | **P1** |
| [M13.3 Sections & cross-listing](./M13.3-course-sections-cross-listing.md) | `sections` (flag) | **P1** |
| [M13.4 Grading settings](./M13.4-course-grading-settings.md) | `grading` (scale, weighted groups) | **P1** |
| [M13.5 Outcomes settings](./M13.5-course-outcomes-settings.md) | `outcomes` | **P1** |
| [M13.6 Grading agents](./M13.6-course-grading-agents.md) | `grading-agents` (flag) | **P2** |
| [M13.7 Plagiarism & AI-authorship](./M13.7-course-plagiarism-settings.md) | `plagiarism` (flag) | **P2** |
| [M13.8 Accessibility (alt-text) review](./M13.8-course-accessibility-review.md) | `accessibility` (flag) | **P2** |
| [M13.9 Translations & localization](../completed/mobile/M13.9-course-translations-localization.md) | `translations` (flag) | **P2** ✅ |
| [M13.10 Import / export & backup](../completed/mobile/M13.10-course-import-export.md) | `import-export` | **P2** ✅ |
| [M13.11 Blueprint (curriculum sync)](../completed/mobile/M13.11-course-blueprint-sync.md) | `blueprint` | **P2** ✅ |
| [M13.12 Archived content](../completed/mobile/M13.12-course-archived-content.md) | `archive` | **P2** ✅ |

**M14 — Platform, Org & Account Administration** (global `/settings/*`; permission-gated):

| Story | Web source (settings view) | Permission | Disposition |
|---|---|---|---|
| [M14.1 Account integrations & API access](../completed/mobile/M14.1-account-integrations-api-access.md) | `integrations` (keys, calendar subs, MCP, service tokens) | user (service tokens: `rbac:manage`) | **P1** ✅ |
| [M14.2 Roles & permissions](../completed/mobile/M14.2-roles-permissions-admin.md) | `roles` | `rbac:manage` | **Done** |
| [M14.3 People / user management](./M14.3-people-user-management.md) | `people` | `rbac:manage` | **P2** |
| [M14.4 Organizations, units & terms](./M14.4-organizations-units-terms.md) | `organizations`/`org-units`/`terms` | `rbac:manage` / `tenant:org-units:admin` | **P2** |
| [M14.5 Branding, AI governance & provider](./M14.5-org-branding-ai-governance.md) | `org-branding` | `rbac:manage` / `tenant:org-units:admin` | **P2** |
| [M14.6 Global platform config](./M14.6-global-platform-config.md) | `platform` | `rbac:manage` | **P2** (read + guarded flag toggle) |
| [M14.7 AI models, prompts & reports](./M14.7-ai-models-prompts-reports.md) | `ai-models`/`ai-prompts`/`ai-reports` | `rbac:manage` | **P2** |
| [M14.8 Integrations & provisioning admin](./M14.8-integrations-provisioning-admin.md) | `lti-tools`/`scim-provisioning`/`cloud-providers`/`lrs-integrations`/`oer-providers` (flags) | `rbac:manage` | **P2** (status + link-out) |
| [M14.9 Transcripts & advising config](./M14.9-transcripts-advising-config.md) | `transcripts`/`advising` (flags) | `rbac:manage` | **P2** |
| [M14.10 Global archived courses](../completed/mobile/M14.10-global-archived-courses.md) | `archive` | `rbac:manage` | **P2** ✅ |

> **Sequencing note.** [M13.1](../completed/mobile/M13.1-course-settings-general.md) is the keystone for M13 — it
> lands the course-settings shell (drawer entry, permission gate, save/unsaved-changes scaffold,
> flag-aware section list) that M13.2–M13.12 plug into. For M14,
> [M14.1](../completed/mobile/M14.1-account-integrations-api-access.md) (user-facing) ships first; the admin views
> (M14.2–M14.10) share a `mobile_admin_settings` flag and lead with read/status before edit.

### Stays web-only (reachable via in-app web view)

Authoring and admin surfaces that are desktop-bound and out of scope for native:
question-bank/blueprint/outcomes authoring (2.1, 5.6, 9.5 authoring), gradebook
grid editing & curving/CSV (3.11, 3.17 instructor), org/admin consoles (5.x admin,
10.x compliance, 17.x ops, 18 admin experience), SIS/SCIM/OneRoster config (4.3–4.5,
13.7, 14.1), integrations/webhooks/marketplace (16.x), CLI (21-cli), course
authoring/import (QTI/Common Cartridge 2.13, Canvas import), proctoring config
(14.9), evaluation/template authoring (14.7). Mobile links out to these with a
"best on a larger screen" affordance rather than reimplementing them.

> **Updated by the 2026-07-05 settings scan.** The *settings/config* surfaces themselves are no
> longer treated as pure web-only: **M13** brings the course-settings area to mobile natively, and
> **M14** brings the permission-gated global settings to mobile as review/status + guarded light
> actions. What stays web-only is the **deep authoring and credential entry** behind those
> settings — SIS/SCIM/cloud/LTI secret entry, custom-domain DNS, permission-matrix authoring,
> QTI/Common Cartridge import, and template authoring — which M14 links out to.

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
2. **Wave 2 — Breadth (P1).** Auth depth (M1.x ✅), interactive content (M3.3 ✅) +
   M3.5 vibe activities ✅ + M3.6 library/e-reserves ✅, standards/mastery (M6.2 ✅),
   M6.3 immersive reader ✅, groups/office-hours (M7.3 ✅/M7.4 ✅) + M7.5 live classes ✅ +
   M7.6 course feed ✅ + M7.7 evaluations, adaptive (M8.2 ✅/M8.3 ✅/M8.4 ✅), self-learner commerce
   (M9.1 ✅/M9.2 ✅/M9.3 ✅), parent (M10.1 ✅/M10.2), accessibility & i18n (M0.3 ✅/M0.4 ✅), settings (M1.4 ✅),
   instructor attendance (M11.1 ✅) + M11.3 instructor insights/at-risk + M11.4 course
   people/roster. **Course settings (M13):** the settings shell + General (M13.1) lands as the
   keystone, then the high-value instructor sections — Features (M13.2), Sections (M13.3),
   Grading (M13.4), Outcomes (M13.5). **Account integrations (M14.1)** ships here too (user-facing).
3. **Wave 3 — Situational (P2).** Lockdown, code exec, behavior/hall-pass,
   age-appropriate UI, e-portfolio/credentials, instructor broadcast, M7.8 advising,
   M1.5 profile depth (demographics/custom fields/research consent). **Remaining course settings
   (M13.6–M13.12):** grading agents, plagiarism, accessibility, translations, import/export,
   blueprint, archived. **Admin/global settings (M14.2–M14.10):** roles, people, orgs/units/terms,
   branding & AI governance, platform config, AI models/prompts/reports, integrations/provisioning,
   transcripts/advising, archived courses — all behind `mobile_admin_settings`, read/status first.

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

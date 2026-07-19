# Mobile Parity Plans

Implementation plans for web-client features that the iOS and Android apps do
**not** yet reach parity on. Every plan follows the structure in
[`../_TEMPLATE.md`](../_TEMPLATE.md).

- **Section prefix:** `MOB` — one plan per gap surfaced in the 2026-07-17
  mobile ↔ web parity scan.
- **File naming:** `MOB.{number}-{kebab-slug}.md`.
- **Scope note:** these are *client-side* plans. The backend APIs already exist
  (the web client consumes them); the work is to build the iOS (SwiftUI) and
  Android (Jetpack Compose) surfaces that call them. Where a plan does require
  new server work it is called out explicitly in §8/§9.
- A plan is "ready" when every template section is filled (no `…` placeholders).

## Severity legend

- **BLOCKER** — cannot sell the mobile app into the listed market without it.
- **MAJOR** — RFP / parity gap that loses deals or drives users back to web.
- **MINOR** — polish / long-tail parity.

## Parity matrix

| ID | Plan | Web reference | Mobile today | Status | Severity | Effort |
|---|---|---|---|---|---|---|
| [MOB.1](../../completed/mobile/MOB.1-course-creation-wizard.md) | Course creation wizard | `pages/lms/course-create.tsx` | Full parity behind `ffMobileCourseCreateV2` | **DONE** | MAJOR | M |
| [MOB.2](../../completed/mobile/MOB.2-canvas-course-import.md) | Canvas course import | `pages/lms/canvas-import-courses-panel.tsx` | Full wizard behind `ffMobileCanvasImport` | **DONE** | MAJOR | M |
| [MOB.3](../../completed/mobile/MOB.3-system-settings-parity.md) | System settings parity | `pages/admin/*`, `components/settings/*`, `side-nav-admin-links.tsx` | Phase 1 hub + audit log behind `ffMobileAdminConsole` | **DONE (Phase 1)** | MAJOR | L |
| [MOB.4](../../completed/mobile/MOB.4-course-enrollment-management.md) | Course enrollment management | `components/enrollment/*`, `courses-api.ts` enrollments | Add/state/message/remove on People behind `ffMobileEnrollmentAdd` | **DONE** | MAJOR | S |
| [MOB.5](MOB.5-interactive-quizzes.md) | Interactive quizzes | `pages/live-quiz-play-page.tsx`, `components/live-quiz/*` | none | MISSING | BLOCKER (K12) | L |
| [MOB.6](../../completed/mobile/MOB.6-whiteboards.md) | Whiteboards (authoring) | `components/whiteboard/*` | **DONE** (save-based authoring) | DONE | MAJOR | M |
| [MOB.7](../../completed/mobile/MOB.7-marketplace-purchases.md) | Marketplace purchases & library | `pages/marketplace/*`, `pages/checkout/*`, `/me/purchases` | **DONE** (Stripe handoff + Purchased courses) | DONE | MAJOR | M |
| [MOB.8](MOB.8-collaboration-boards-completion.md) | Collaboration boards completion | `components/boards/*` (VC.8–VC.10) | VC.M1–M7 shipped; M8–M10 absent | PARTIAL | MINOR | M |

## Sequencing at a glance

```
MOB.1 ──▶ MOB.2            (Canvas import is a create-course entry point)
MOB.3 (independent, phaseable)
MOB.4 (independent)
MOB.5 (independent; backend IQ.1–IQ.11 shipped)
MOB.6 (done — save-based; realtime follow-up if needed)
MOB.7 (done — external Stripe checkout; IAP deferred)
MOB.8 ◀── depends on shipped VC.M1–M7
```

## Cross-platform note

Each plan targets **both** iOS and Android. The two clients mirror each other:
iOS logic lives in `clients/ios/Lextures/Core/LMS/*Logic.swift` +
`LMSAPI*.swift`; Android logic in
`clients/android/app/src/main/kotlin/com/lextures/android/core/lms/*Logic.kt` +
`*Api.kt`. Views live under each client's `Features/` (iOS) or `features/`
(Android) tree. Realtime features reuse `Core/Realtime/WebSocketClient.swift`
and `core/realtime` on Android.

# FB3 — Mobile "Share Feedback" (iOS + Android)

> Implementation plan. Source: Product request — in-app "Share Feedback" mechanism (2026-07-10). Follows [../_TEMPLATE.md](../_TEMPLATE.md), mobile-tuned. Consumes [FB0](./FB0-feedback-foundation-api.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | FB3 |
| **Section** | Feedback — In-App Feedback & Admin Review |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1w) — one form + entry point across two native apps |
| **Platforms** | iOS (SwiftUI), Android (Compose) |
| **Backend** | None new — consumes FB0 `POST /api/v1/feedback` |
| **Owner (proposed)** | Mobile |
| **Depends on** | [FB0](./FB0-feedback-foundation-api.md) |
| **Unblocks** | Feedback loop on mobile |
| **Permission** | Any authenticated user |

---

## 1. Problem Statement

Mobile users can't send feedback from inside the iOS or Android apps. Given how much usage is on phones, that's a large blind spot. This story adds a **Share Feedback** entry point to each app — placed where it fits mobile conventions (the Profile/More area) — that opens a short native form and posts to the same FB0 endpoint the web uses.

## 2. Goals

- A discoverable **Share Feedback** action in each app, placed per-platform convention.
- A native form: required message, optional category, submit — mirroring web (FB1).
- Auto-attach source (`ios`/`android`), app version, and current context.
- Accessible (VoiceOver/TalkBack, Dynamic Type), localized, and RTL-safe on both apps.
- Graceful, non-blocking failure with retry.

## 3. Non-Goals

- No backend/schema/endpoint work (FB0).
- No admin/list/detail on mobile (web-only, FB2).
- No screenshot/log attachment (future — §18).
- No offline queue for feedback in MVP (requires connectivity; show a clear message when offline).

## 4. Personas & User Stories

- **As a mobile user**, I open Profile/More, tap **Share Feedback**, type a note, and send it.
- **As a user who hit an issue in the app**, I want source + app version attached automatically for triage.
- **As a VoiceOver/TalkBack user**, I can reach, complete, and submit the form without sighted assistance.

## 5. Functional Requirements

- **FR-1.** Each app MUST expose a **Share Feedback** entry point in a sensible location:
  - **iOS:** a row in `ProfileView` (via `ProfileSettingsCards`/`ProfileViewSections`), and/or a nav-bar action where the Profile tab is shown (`MainTabView`).
  - **Android:** a row/item in `ProfileTab` settings list.
  It MUST be gated by the `ff_feedback` platform flag.
- **FR-2.** The entry point SHOULD stand out modestly (accent tint / icon) without breaking the settings-list aesthetic.
- **FR-3.** Tapping it MUST present a form (sheet/modal) with a required **message** field, an optional **category** picker (`bug`/`idea`/`question`/`praise`/`other`), and **Send** / **Cancel**.
- **FR-4.** **Send** MUST be disabled until the message is non-empty; a counter approaching the 5,000 cap SHOULD show.
- **FR-5.** On Send, the app MUST `POST /api/v1/feedback` via the shared API client with `source` = `ios`|`android`, `app_version` = build/version, and `context` (current screen/route + locale).
- **FR-6.** On success the sheet MUST dismiss and a confirmation (toast/snackbar/banner) MUST show.
- **FR-7.** On error the form MUST stay open, preserve input, and show the error; `429` MUST show a friendly rate-limit message; offline MUST show a clear "no connection" message.
- **FR-8.** The form MUST support Dynamic Type / font scaling and VoiceOver/TalkBack labels, with ≥44pt/48dp targets.

## 6. Non-Functional Requirements

- **Performance** — form presents instantly; submit non-blocking; no main-thread network.
- **Security** — send via the authenticated shared client; identity never in the body (server derives it); no HTML rendering of any response.
- **Privacy** — copy notes staff read feedback; capture only screen route as context, not screen contents.
- **Accessibility** — VoiceOver (iOS) / TalkBack (Android) labels + traits; Dynamic Type / font scale; sufficient contrast on the accent entry point; focus order into/out of the sheet.
- **Reliability** — failed submit is retryable; never crashes; requires connectivity (no silent loss).
- **Internationalization** — strings from `clients/mobile/locales/*.json`; RTL layouts correct on both apps.
- **Maintainability** — logic (validation, payload building) in a testable, shared-style layer (`Core/LMS` / `core/lms`) mirroring existing feature-logic files.

## 7. Acceptance Criteria

- **AC-1.** *Given* `ff_feedback` on, *Then* a **Share Feedback** entry appears in Profile/More on both apps.
- **AC-2.** *Given* it's tapped, *Then* a native form with message + category + Send/Cancel presents.
- **AC-3.** *Given* an empty message, *Then* Send is disabled.
- **AC-4.** *Given* a valid message, *When* Send is tapped, *Then* a `POST /api/v1/feedback` fires with the correct `source` and `app_version`, the sheet dismisses, and a confirmation shows.
- **AC-5.** *Given* the endpoint returns `429`, *Then* a friendly rate-limit message shows and input is preserved.
- **AC-6.** *Given* no connectivity, *Then* an offline message shows and nothing is lost.
- **AC-7.** *Given* VoiceOver/TalkBack, *Then* all controls are labeled and operable.
- **AC-8.** Both apps build and pass their unit suites.

## 8. Data Model

- None (client only). Consumes FB0.

## 9. API Surface

- Consumes `POST /api/v1/feedback` (FB0 §9). Confirm the shared client sends the platform `source` and app version. No new endpoints.

## 10. UI / UX

- **Entry point (iOS):** a labeled row (icon + "Share Feedback") in the Profile settings cards; consider a nav-bar `Menu`/button on the Profile/Home tab. **Android:** a settings-list item in `ProfileTab` with a leading icon.
- **Form:** presented as a sheet (iOS `.sheet` / Android `ModalBottomSheet` or full-screen dialog) — message `TextEditor`/multiline field, category `Picker`/dropdown, Send (primary) + Cancel.
- **States:** default, submitting (disabled Send + progress), success (dismiss + confirmation), error (inline, retryable), rate-limited, offline. Empty → Send disabled.
- **Copy/i18n keys:** `feedback.entry`, `feedback.title`, `feedback.message.label`, `feedback.message.placeholder`, `feedback.category.label` + options, `feedback.send`, `feedback.cancel`, `feedback.success`, `feedback.error`, `feedback.rateLimited`, `feedback.offline` (shared keys with web where possible).
- **Accessibility:** labeled fields, Send state announced, focus into message on present and back to entry point on dismiss; Dynamic Type / font scaling verified.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- **iOS:** entry in `clients/ios/Lextures/Features/Profile/ProfileSettingsCards.swift` / `ProfileViewSections.swift` (and optionally `Features/Home/MainTabView.swift`); new `Features/Feedback/ShareFeedbackView.swift`; API + logic in `Core/LMS/LMSAPIFeedback.swift` + `Core/LMS/FeedbackLogic.swift` (mirroring existing `LMSAPI*` / `*Logic` pairs); feature-flag gating via the existing platform-features analog.
- **Android:** entry in `clients/android/.../features/profile/ProfileTab.kt`; new `features/feedback/ShareFeedbackScreen.kt`; API in `core/lms/LmsApi.kt` + `core/lms/FeedbackLogic.kt`.
- Shared locales: `clients/mobile/locales/*.json`.

## 13. Dependencies & Sequencing

- After FB0 (submit endpoint + `ff_feedback` flag). Independent of FB1/FB2. Recommend building both platforms from a shared logic/validation contract to keep parity.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Entry point undiscoverable | M | M | Place in Profile/More with icon + label; consider a nav action; dogfood |
| Two-app drift (iOS vs Android behavior) | M | M | Shared logic contract + shared i18n keys + parity test checklist |
| Feedback lost when offline | M | M | Require connectivity, clear offline message, preserve input (no silent drop) |
| Users expect a reply | L | L | Copy sets expectations (as FB1) |

## 15. Rollout Plan

- Gated by `ff_feedback` (FB0). Internal build → staged → GA; ship iOS + Android together for parity.
- Watch `feedback_submitted_total{source="ios"}` / `{source="android"}` and client crash/error telemetry.
- Rollback: flag off hides the entry point on both apps.

## 16. Test Plan

- **Unit** — validation + payload building (source/app_version/context) in the shared logic layer, both apps; Send-disabled logic; error/429/offline mapping.
- **Integration** — mocked submit success + failure via the shared API client.
- **UI (XCUITest / Espresso)** — open from Profile → type → send → confirmation + dismiss; flag-off hides entry.
- **Accessibility** — VoiceOver/TalkBack pass; Dynamic Type / large font; contrast on entry point.
- **Manual** — RTL locale on both apps; offline; small-device layout.

## 17. Documentation & Training

- Help-center mobile note: "Share feedback from the app" (both platforms).
- Release notes for iOS + Android.

## 18. Open Questions

1. **Best entry placement** — Profile row only, or also a nav-bar action / long-press shortcut? Settle in mobile UX review.
2. **Attachments** (screenshot/device logs) — high value for bug reports on mobile; future, needs storage + scan.
3. **Offline queue** — worth queuing feedback for later send? MVP requires connectivity.
4. **Shake-to-feedback** gesture — nice-to-have, later.

## 19. References

- Existing files this work touches: iOS `Features/Profile/ProfileView.swift` / `ProfileSettingsCards.swift` / `ProfileViewSections.swift`, `Features/Home/MainTabView.swift`, `Core/LMS/LMSAPI*.swift`; Android `features/profile/ProfileTab.kt`, `core/lms/LmsApi.kt`.
- Mobile plan conventions: [../mobile/M14.6-global-platform-config.md](../mobile/M14.6-global-platform-config.md) (two-platform, mobile-tuned template).
- Related plans: [FB0](./FB0-feedback-foundation-api.md), [FB1](./FB1-web-share-feedback-button.md), [FB2](./FB2-web-feedback-admin.md).

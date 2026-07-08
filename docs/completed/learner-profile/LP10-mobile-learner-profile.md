# LP10 — Mobile Learner Profile (Read-Only + Controls)

> Implementation plan. The mobile surface for the learner profile (LP07 web parity, mobile-tuned).
> Follows [../_TEMPLATE.md](../_TEMPLATE.md), mobile-tuned like the `docs/plan/mobile/` track.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP10 |
| **Section** | Learner Profile (Mobile) |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | **DONE** — read-only profile + LP08 controls on iOS and Android |
| **Estimated effort** | S–M (1–3w) |
| **Platforms** | iOS (SwiftUI), Android (Compose) |
| **Backend** | None new — reuses LP01 read API + LP08 control endpoints |
| **Owner (proposed)** | Mobile Platform |
| **Depends on** | LP07 (web parity + API), LP08 (control endpoints) |
| **Unblocks** | On-the-go transparency + control of the profile |

---

## Implementation notes (2026-07-07)

- **Gating**: `learnerProfileEnabled` (server `learner_profile_enabled`) **and** client `ffMobileLearnerProfile` (default true) on both platforms.
- **Logic**: `LearnerProfileLogic` (iOS/Android) — flag gating, facet sort order, insight value formatting, evidence aggregation, rhythm/modality chart text alternatives, offline cache keys.
- **API**: `LMSAPILearnerProfile.swift` + `LmsApi` methods for `GET /me/learner-profile`, facet detail/evidence, and LP08 `pause`/`resume`/`reset`/`export`.
- **Models**: `LMSFeatureModelsLearnerProfile` / `LmsFeatureModelsLearnerProfile.kt`.
- **UI**: `LearnerProfileView` / `LearnerProfileScreen` — intro card, facet cards with lazy evidence disclosure, paused/empty/offline/error states, manage section (export share sheet, pause/resume confirmations, reset phrase confirmation). Entry via Profile tab (`LearnerProfileEntryCard` / `ProfileTab`).
- **i18n**: `mobile.learnerProfile.*` keys in all five `clients/mobile/locales/` files; synced to iOS `Localizable.xcstrings` and Android `strings.xml`.
- **Tests**: `LearnerProfileLogicTests` (iOS), `LearnerProfileLogicTest` (Android). iOS + Android unit tests pass locally.
- **Xcode**: `scripts/patch_xcodeproj_learner_profile.py` registers Swift sources in `project.pbxproj`.

## 1. Problem Statement

Most learners — especially K-12 and self-learners — live in the mobile app. The learner profile (LP07)
is web-only, so on a phone a learner can neither see what the platform has learned about them nor
exercise the transparency/control the epic promises. This plan brings a **read-only, provenance-first
Learner Profile** and its **privacy controls** to iOS and Android, mirroring LP07/LP08 with mobile
interaction patterns.

## 2. Goals

- Add a **Learner Profile** entry in the mobile app's user/settings area, gated by
  `learner_profile_enabled`.
- Render each facet with plain-language insights and an expandable **"Derived from …"** evidence view.
- Surface LP08 controls: **Download/export**, **Pause/Resume**, **Reset** (with confirmation).
- Handle empty / still-building / insufficient / paused / offline states; respect reduced-data.

## 3. Non-Goals

- No profile editing (autonomous, read-only) and no new backend.
- No mobile-side derivation — the server owns all computation.
- Instructor/guardian mobile views (guardian control can be a follow-on aligned with parent-portal).

## 4. Personas & User Stories

- **As a student on my phone**, I want to open my Learner Profile and see, in plain language, how I
  learn and where each insight came from.
- **As a privacy-conscious learner**, I want to pause or reset my profile from my phone.
- **As a commuter self-learner**, I want the profile to load fast and read cleanly on a small screen.

## 5. Functional Requirements

- **FR-1.** The section MUST appear only when `learner_profile_enabled`; otherwise absent.
- **FR-2.** MUST fetch `GET /api/v1/me/learner-profile` (LP01) and render one collapsible card per
  facet with a plain-language summary and top insights.
- **FR-3.** Each insight MUST offer a **"Derived from …"** disclosure that loads
  `.../facets/{key}/evidence` on demand (sources, counts, courses, window).
- **FR-4.** MUST show facet **last-computed** time and a **confidence** indicator (text + icon).
- **FR-5.** MUST surface LP08 controls: Download (export), Pause/Resume, Reset — Reset behind an
  explicit irreversible confirmation.
- **FR-6.** MUST handle: whole-profile "still building", per-facet insufficient, paused, loading,
  error, and **offline** (show last-cached profile read-only; queue no writes except via LP08 endpoints
  when back online).
- **FR-7.** MUST be self-only (the signed-in user's profile).

## 6. Non-Functional Requirements

- **Performance** — Profile list render < 700 ms p95 on a warm cache; evidence lazy-loaded.
- **Security** — Server enforces self-scope; no profile data in logs/analytics; secure local cache.
- **Privacy & Compliance** — Same FERPA/GDPR posture as LP07/LP08; show disclosure + link to privacy.
- **Accessibility** — VoiceOver/TalkBack labels; ≥ 44 pt targets; Dynamic Type; charts have text
  alternatives; confidence not by color alone.
- **Internationalization** — Localised chrome; RTL; localised dates/numbers.
- **Reliability** — Offline shows cached read-only profile; control actions require connectivity.

## 7. Acceptance Criteria

- **AC-1.** *Given* the flag off, *then* the section is absent.
- **AC-2.** *Given* a populated profile, *then* each facet renders with plain-language insights; an
  insight's "Derived from …" expands to its evidence.
- **AC-3.** *Given* a new learner, *then* the "still building" empty state shows (no fabricated insight).
- **AC-4.** *Given* the learner taps Pause, *then* the profile shows paused and Resume is offered.
- **AC-5.** *Given* Reset with confirmation, *then* the profile clears and shows the empty state.
- **AC-6.** *Given* offline, *then* the last-cached profile shows read-only and controls are disabled.
- **AC-7.** Both apps build; VoiceOver/TalkBack pass on the profile and evidence disclosures.

## 8. Data Model

- No schema changes. Cache the profile + expanded evidence locally per user; invalidate on refresh.

## 9. API Surface

- Reuse LP01 read endpoints and LP08 control endpoints (`pause`/`resume`/`reset`/`export`). Verify
  shapes against the web client when implementing.

## 10. UI / UX

- Entry in the mobile settings/account area → Learner Profile screen: intro "How this works" → facet
  cards (Study rhythm, How you like to learn, Strengths & growth, What you're drawn to, How you
  approach challenges) → "Manage your profile" (LP08 controls).
- Each facet card: summary, insight rows, tap-to-expand "Derived from …", footer last-computed +
  confidence. Charts (rhythm/modality) render as compact mobile visuals with text alternatives.
- States: loading skeleton, still-building, insufficient, paused banner, error, offline.

## 11. AI / ML Considerations

- Display only; no on-device inference. If LP09 adds an AI summary, label it AI-generated and keep it
  above the evidence-backed facets (parity with LP07).

## 12. Integration Points

- **iOS:** `Features/LearnerProfile/LearnerProfileView.swift`; API in
  `Core/LMS/LMSAPILearnerProfile.swift` (mirror existing `LMSAPI*` patterns).
- **Android:** `features/settings/learnerprofile/LearnerProfileScreen.kt`; `core/lms/LmsApi.kt`.
- Feature-flag plumbing mirrors existing mobile flag handling.

## 13. Dependencies & Sequencing

- After LP07 (web parity + stable API) + LP08 (controls). Independent of LP09. Aligns with the
  `docs/plan/mobile/` track conventions.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Provenance detail cramped on small screens | M | M | Progressive disclosure; summaries first, evidence on tap |
| Charts fail mobile a11y | M | M | Text alternatives + Dynamic Type; VoiceOver/TalkBack tests |
| Destructive Reset mis-tapped | M | M | Explicit irreversible confirmation sheet distinct from Pause |
| Offline staleness confusing | M | M | Clear "last updated" + offline banner |

## 15. Rollout Plan

Behind `learner_profile_enabled` + a mobile sub-flag. Internal → pilot → GA. Rollback: flag hides the
section.

## 16. Test Plan

- **Unit** — facet/evidence rendering; state machine; flag gating; offline cache.
- **Integration** — list/evidence fetch; pause/resume/reset/export against LP08.
- **E2E** — open profile with seeded data; expand evidence; pause; reset; offline read-only.
- **Accessibility** — VoiceOver/TalkBack on facets + disclosures; Dynamic Type; contrast.

## 17. Documentation & Training

- Student help (mobile): "View and manage your Learner Profile on mobile."

## 18. Open Questions

1. Where in the mobile IA does this live — under Account/Settings or a top-level "You" area?
   **Resolved:** Profile tab entry card (settings/account area).
2. Is guardian access needed on mobile v1, or web-only initially (align with parent portal)?
   **Deferred:** web/parent-portal only for v1.
3. Which facet visualisations are worth native charts vs. text-first on small screens?
   **Resolved:** text-first captions for rhythm/modality; no native chart widgets in v1.

## 19. References

- [LP07](./LP07-settings-page-transparency-ui.md), [LP08](./LP08-privacy-consent-controls.md),
  [LP01](./LP01-foundation-derivation-engine.md).
- Mobile conventions: [`../mobile/M13.5-course-outcomes-settings.md`](../mobile/M13.5-course-outcomes-settings.md).
- External: WCAG 2.1 AA (mobile); Apple HIG / Material accessibility.
# CC.9 — Mobile: Course Checklist on iOS & Android

> Implementation plan. Source: Course Checklist product request — "Make sure the mobile apps also have this
> item." Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.9 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team (iOS + Android) |
| **Depends on** | CC.2 (API), CC.7 (IA + copy), CC.8 (target table) |
| **Unblocks** | CC.10 |

---

## 1. Problem Statement

Teachers do course triage on their phones — between classes, on the bus, the night before term. Both apps
already have a registry-driven course workspace (`CourseWorkspaceSection` in
`clients/ios/Lextures/Core/Routing/MobileDestinations.swift` and
`clients/android/.../core/navigation/MobileDestinations.kt`) with an Overview section, a drawer and a
teaching role context, but no notion of course readiness. If the checklist only exists on the web, the badge
that tells a teacher something needs attention is invisible exactly when they have a spare five minutes.
CC.9 brings the checklist to both apps at parity with CC.7: a workspace section, a badge, crossed-off items,
expandable evidence, dismiss/restore, and native deep-linking to the fix.

## 2. Goals

- Add a **Checklist** section to the course workspace on both apps, positioned immediately after Overview,
  visible only to staff, with a count badge on the chip/drawer entry.
- Reach **functional parity** with CC.7: categories, crossed-off done items, progress, evidence tables,
  dismiss with reason, dismissed pile, restore, re-check.
- Route checklist targets to **native destinations** where one exists, and to the in-app browser (MB.1
  `LinkOpener`) where it does not — with a clear, non-broken experience either way.
- Respect platform accessibility (VoiceOver / TalkBack, Dynamic Type / font scale, Reduce Motion) and the
  apps' offline expectations.

## 3. Non-Goals

- No server or web work; CC.9 consumes CC.2 and CC.8's target table as-is.
- No mobile authoring surfaces. Where a fix has no native editor, CC.9 opens the web equivalent in the
  in-app browser rather than building an editor.
- No offline mutation queue — dismiss/restore require connectivity (see §5 FR-19).
- No push notifications about outstanding items (candidate for CC.10, not here).
- No tablet-specific layout beyond what the existing adaptive layout gives.

## 4. Personas & User Stories

- **As a teacher on my phone**, I want to see that my course has 6 outstanding items, so that I know to make
  time for it.
- **As a teacher**, I want to tick through the quick fixes (publish the course, post a welcome) from my
  phone, so that the list gets shorter without a laptop.
- **As a teacher**, I want tapping an item to open the right screen in the app when one exists, so that I am
  not bounced to a browser for everything.
- **As a teacher**, I want to dismiss an irrelevant item on mobile and have it stay dismissed on the web, so
  that the two surfaces agree.
- **As a VoiceOver user**, I want done items announced as completed rather than relying on a strikethrough,
  so that I can follow my progress.
- **As a student on mobile**, I want to never see this section, so that instructor to-dos stay hidden.

## 5. Functional Requirements

### Navigation & visibility

- **FR-1.** iOS MUST add `case checklist` to `CourseWorkspaceSection` with label key
  `mobile.ia.course.checklist`, deep-link segment `checklist`, and an SF Symbol
  (`checklist` / `checkmark.circle`). Android MUST add the mirrored
  `Checklist("mobile_ia_course_checklist", "checklist")` entry and a `DrawerMappings` icon
  (`Icons.Filled.Checklist`).
- **FR-2.** The section MUST appear **immediately after `Overview`** in the workspace ordering on both
  platforms, in the chip row and in the course drawer.
- **FR-3.** The section MUST be included in the workspace split only when the viewer is course staff with
  authoring capability — the same predicate the apps already use to show teaching-only sections — and MUST
  be absent (not disabled) otherwise.
- **FR-4.** The section MUST be absent when the app is in the Learning role context even for a
  dual-enrolled user, matching the web's "View as: Student" behaviour.
- **FR-5.** The chip and drawer entry MUST render a badge with `summary.outstandingEssential`, capped at
  `99+`, hidden at 0, with an accessibility label "N checklist items need attention".
- **FR-6.** `CourseDeepLinkSection` on both platforms MUST accept `checklist`, so a push or universal link to
  `/courses/{code}/checklist` opens the section. `?focus=` params on such a link MUST be forwarded per FR-13.

### Data & sync

- **FR-7.** Both apps MUST add a checklist API client mirroring CC.2: fetch, summary, refresh, dismiss,
  restore, recheck. Requests MUST use the existing authenticated client and error mapping.
- **FR-8.** The summary MUST be fetched when a course workspace opens and memoised for 60 s, and MUST be
  refetched after any dismiss/restore/recheck and on app foreground after > 60 s.
- **FR-9.** A `403` MUST be treated as "not staff": hide the section and clear any cached badge, without an
  error alert.
- **FR-10.** The full checklist MUST be fetched lazily when the section is opened, not with the course
  payload.
- **FR-11.** Checklist responses MUST **not** be written to the offline store (evidence can name students);
  the section MUST show an offline state ("Connect to see your checklist") rather than stale data. The
  **summary counts only** MAY be cached in memory for the session, never on disk.
- **FR-12.** `catalogVersion` mismatch MUST invalidate any in-memory cache.

### Rendering & interaction

- **FR-13.** The section MUST render: a header with progress (`18 of 26 done`) and a re-check control;
  categories as collapsible groups with outstanding counts; items with status, title, why, detail, progress,
  tier and source chips; a dismissed group at the bottom.
- **FR-14.** `done` items MUST render with a strikethrough title **and** an accessibility value of
  "Completed" (never decoration alone).
- **FR-15.** Items with evidence MUST expand in place to a native list (not a table) — one row per entity
  with its label, sublabel and a chevron — with "Showing first 200 of N" when truncated.
- **FR-16.** Tapping an item or an evidence row MUST resolve its target through the CC.8 **native target
  table**:
  - a mapped native destination → navigate in-app and apply the native highlight (FR-17);
  - `web-only` → open the equivalent web URL (with `?focus=`) via the MB.1 `LinkOpener` in the **in-app
    browser**, never Safari/Chrome;
  - unmapped/unknown → open the course section that most closely contains it, with no highlight.
- **FR-17.** Native highlight MUST scroll the target row/control into view, move accessibility focus to it,
  and apply a persistent (non-pulsing) outline for 4 s. Under Reduce Motion / "Remove animations", the scroll
  MUST be instant and the outline MUST appear without transition.
- **FR-18.** Dismiss MUST present a native sheet with the reason picker and an optional ≤ 500-char note, then
  optimistically move the item to the dismissed group; failure MUST roll back with an inline message.
- **FR-19.** Dismiss / restore / recheck MUST require connectivity; offline, the controls are disabled with
  an explanatory label. No mutation queue.
- **FR-20.** Pull-to-refresh MUST call `POST /checklist/refresh` and MUST respect the server rate limit
  (a `429` shows "Just checked — try again in a moment", not an error).
- **FR-21.** Empty states: all done → a plain completion panel with a "Show completed" toggle; no items →
  "Nothing to check right now"; error → inline retry; offline → offline panel.

### Parity

- **FR-22.** iOS and Android MUST use the **same copy keys** as web (server-provided `titleKey`/`whyKey`
  with English defaults) — no re-authored mobile copy.
- **FR-23.** A shared logic layer MUST hold the pure parts (status → presentation mapping, target
  resolution, progress math) in `CourseChecklistLogic.swift` / `CourseChecklistLogic.kt`, mirroring the
  existing `CourseCreateLogic` pattern, and MUST be unit-tested on both platforms with the **same fixture
  JSON**.
- **FR-24.** Both apps MUST add the new string resources to every shipped locale file
  (`Localizable.xcstrings`, `values*/strings.xml`, `clients/mobile/locales/*.json`), including the
  pseudo-locale `en-XA` used for layout testing.

## 6. Non-Functional Requirements

- **Performance** — Section opens to first content < 500 ms on a warm API; badge fetch adds one request per
  course open (memoised 60 s). Lists are lazily rendered (`LazyVStack` / `LazyColumn`); an evidence list of
  200 rows scrolls at 60 fps on a mid-tier device (iPhone 12 / Pixel 6a class).
- **Security** — All authorisation is server-side (CC.2 FR-1); the client predicate is UX only. Web fallbacks
  MUST go through `LinkOpener` so the bearer-token and in-app-browser policy from MB.1 applies, and MUST NOT
  leak tokens into an external browser.
- **Privacy & Compliance** — FR-11 (no disk persistence) is a hard requirement because evidence may name
  students. Dismissal notes are staff free text; they are not stored locally beyond the compose buffer.
- **Accessibility** — VoiceOver and TalkBack: categories are headers, items are elements with label
  (title) + value (status) + hint (action), evidence rows are individually navigable, the progress control
  exposes a percentage, and the dismiss sheet is properly labelled with focus returning to the item. Dynamic
  Type / font scale up to the largest accessibility size without truncation or clipping. Reduce Motion
  honoured (FR-17). Minimum touch target 44×44 pt / 48×48 dp. Colour is never the sole status carrier.
- **Scalability** — 120 items across 10 categories renders without jank; category collapse state kept in
  memory per session.
- **Reliability** — Optimistic mutations roll back on failure; a failed summary fetch never blocks the
  workspace; unknown statuses render as `unknown`; unknown categories render with their server-provided
  title.
- **Observability** — Mirror the CC.10 event dictionary: section view, item expand, evidence tap, target
  resolution (`native` / `web` / `unresolved`), dismiss (with reason), restore, recheck, refresh. No PII.
- **Maintainability** — Feature folders `Features/Checklist/` (iOS) and `features/checklist/` (Android);
  pure logic separated per FR-23; API clients alongside the existing `LMSAPI*` / repository patterns.
- **Internationalization** — FR-24 across all locales; RTL verified (Arabic ships): strikethrough, badge
  position, chevrons and the highlight outline offset must mirror.
- **Backward compatibility** — An older app receiving a newer catalog ignores unknown fields and renders
  unknown statuses as `unknown`; an unmapped target falls back to the in-app browser, so new server rules
  never break an old client.

## 7. Acceptance Criteria

- **AC-1.** *Given* a staff user opens a course on iOS, *Then* a Checklist chip appears immediately after
  Overview with a badge equal to `outstandingEssential`.
- **AC-2.** *Given* the same on Android, *Then* the drawer and chip row show the same entry, label and badge.
- **AC-3.** *Given* a student, *Then* the section is absent on both platforms and a deep link to
  `/courses/{code}/checklist` lands on the course overview with no error dialog.
- **AC-4.** *Given* the Learning role context for a dual-enrolled user, *Then* the section is absent.
- **AC-5.** *Given* `outstandingEssential = 0`, *Then* no badge renders; *Given* 137, *Then* `99+` with the
  correct accessibility label.
- **AC-6.** *Given* a done item, *Then* VoiceOver/TalkBack announces its title followed by "Completed".
- **AC-7.** *Given* `outcomes.assessment-mapping` with 11 evidence rows, *When* expanded, *Then* 11 rows
  render, each individually focusable by the screen reader.
- **AC-8.** *Given* an evidence row whose target maps to a native destination, *When* tapped, *Then* the app
  navigates natively and the target row is scrolled to, outlined and given accessibility focus.
- **AC-9.** *Given* a target with no native destination, *When* tapped, *Then* the in-app browser opens the
  web URL including `?focus=`, and the system browser is **not** launched.
- **AC-10.** *Given* Reduce Motion is enabled, *Then* the highlight appears with no animation and scrolling
  is instant.
- **AC-11.** *Given* the user dismisses an item with a reason, *Then* it moves to the dismissed group, the
  badge decrements, and the web reflects the same dismissal on next load.
- **AC-12.** *Given* the device is offline, *Then* the section shows the offline panel, dismiss/restore are
  disabled with an explanation, and no stale checklist data is shown.
- **AC-13.** *Given* the app was killed and relaunched offline, *Then* no checklist data is read from disk
  (asserted by a storage test).
- **AC-14.** *Given* pull-to-refresh twice in quick succession, *Then* the second returns `429` and shows the
  rate-limit message rather than an error.
- **AC-15.** *Given* the largest Dynamic Type / font-scale setting, *Then* item titles, details and evidence
  rows wrap without clipping on a 320 dp-wide device.
- **AC-16.** *Given* the shared fixture JSON, *Then* the iOS and Android logic unit tests produce identical
  presentation output (parity test).

## 8. Data Model

No server changes. Client state:

- In-memory view-model state per course: `summary`, `checklist`, `expandedCategories`, `expandedItems`.
- **No persistence** of checklist responses to Core Data / Room / DataStore (FR-11). A storage test asserts
  this on both platforms.
- The CC.8 native target table ships as a bundled JSON asset in both apps, generated from the shared source
  so the three clients cannot drift.

## 9. API Surface

Consumes CC.2 only. Client surfaces:

```swift
// iOS — Core/LMS/LMSAPIChecklist.swift
func fetchCourseChecklist(courseCode: String) async throws -> CourseChecklist
func fetchCourseChecklistSummary(courseCode: String) async throws -> CourseChecklistSummary
func refreshCourseChecklist(courseCode: String) async throws -> CourseChecklist
func dismissChecklistItem(courseCode: String, itemID: String, reason: DismissReason, note: String?) async throws -> CourseChecklistItem
func restoreChecklistItem(courseCode: String, itemID: String) async throws -> CourseChecklistItem
func recheckChecklistItem(courseCode: String, itemID: String) async throws -> CourseChecklistItem
```

```kotlin
// Android — core/lms/ChecklistApi.kt
suspend fun fetchChecklist(courseCode: String): CourseChecklist
suspend fun fetchChecklistSummary(courseCode: String): CourseChecklistSummary
suspend fun refreshChecklist(courseCode: String): CourseChecklist
suspend fun dismissItem(courseCode: String, itemId: String, reason: DismissReason, note: String?): CourseChecklistItem
suspend fun restoreItem(courseCode: String, itemId: String): CourseChecklistItem
suspend fun recheckItem(courseCode: String, itemId: String): CourseChecklistItem
```

## 10. UI / UX

**Placement**

```
Course workspace chips:   [ Overview ] [ Checklist ⑥ ] [ Modules ] [ Grades ] … [ More ]
Course drawer:            Overview · Checklist ⑥ · Modules · …
```

**Section layout (both platforms)**

```
Course checklist                                     ⟳
18 of 26 done · 5 need attention · checked 4 min ago
▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░

▾ Foundations & orientation                 2 outstanding
  ✓  ̶S̶e̶t̶ ̶c̶o̶u̶r̶s̶e̶ ̶d̶a̶t̶e̶s̶
  ○  Publish your course                              ⋯
     Students can't see the course. Starts in 3 days.
  ○  Add students to the course                       ⋯
     3 invitations are pending.               Show 3 ›

▾ Outcomes & alignment                      1 outstanding
  ◐  Map every assessment to an outcome      13 / 24    ⋯
     11 of 24 assessments aren't mapped.      Show 11 ›

▸ Dismissed (3)
```

**Flows**

1. Course opens → badge appears on the Checklist chip.
2. Tap Checklist → categories load → tap an item → native screen with the control outlined, or the in-app
   browser on the web equivalent.
3. Tap "Show 11" → inline list of the eleven items → tap one → that item's editor (native or in-app web).
4. Overflow → Dismiss → reason sheet → item moves to Dismissed, badge decrements.
5. Pull to refresh → re-check.

**States** — loading skeleton; offline panel; error retry; rate-limited toast; all-done panel with "Show
completed"; unknown items with a re-check control.

## 11. AI / ML Considerations

None. Like CC.7, the item row reserves an action slot that CC.10 may populate with an AI-assisted fix; CC.9
renders nothing there in this plan.

## 12. Integration Points

- **iOS** — modified: `Core/Routing/MobileDestinations.swift` (`CourseWorkspaceSection`,
  `CourseDeepLinkSection`), `Features/Courses/CourseDetailView.swift`, `Features/Courses/CourseWorkspaceNav.swift`,
  `Features/Navigation/CourseDrawer.swift`, `Resources/Localizable.xcstrings`. New:
  `Features/Checklist/` (view, view-model, evidence list, dismiss sheet), `Core/LMS/LMSAPIChecklist.swift`,
  `Core/LMS/CourseChecklistLogic.swift`. Reuses the MB.1 `LinkOpener` and in-app browser.
- **Android** — modified: `core/navigation/MobileDestinations.kt`, `features/navigation/DrawerMappings.kt`,
  `features/courses/CourseDetailScreen.kt`, `res/values*/strings.xml`. New: `features/checklist/`,
  `core/lms/ChecklistApi.kt`, `core/lms/CourseChecklistLogic.kt`. Reuses the MB.1 link handling.
- **Shared** — the CC.8 native target table asset; `clients/mobile/locales/*.json` string additions.

## 13. Dependencies & Sequencing

- Must ship after: CC.2 (API), CC.7 (IA, copy conventions, interaction model), CC.8 (target table — without
  it every target falls back to the in-app browser, which is a degraded but shippable v1).
- Must ship before: CC.10's cross-surface analytics rollup.
- Requires MB.1's `LinkOpener` (already shipped) for the web-fallback path.
- iOS and Android should land in the same app-release train so parity is observable.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Most targets have no native destination, so the section becomes a browser launcher | **H** | M | MB.1 in-app browser keeps users in-app; target table prioritises native mapping for the highest-frequency items (dates, publish, enrollments, modules, syllabus); measure `target_resolution=web` rate and expand native coverage |
| iOS and Android drift in behaviour | **H** | M | FR-23 shared pure logic + FR-22 shared copy keys + AC-16 shared-fixture parity test |
| Checklist data cached to disk leaks student names | L | **H** | FR-11 hard rule; AC-13 storage test on both platforms |
| Badge fetch on every course open costs battery/data | M | M | 60 s memo, summary endpoint is snapshot-served and tiny; single request per course open |
| Long item copy breaks at large font scales | M | M | AC-15 at the largest accessibility size on a 320 dp device; pseudo-locale `en-XA` layout pass |
| Dismiss without connectivity confuses users | M | L | FR-19 disabled controls with an explicit reason, no silent queue |
| Workspace chip row becomes crowded | M | L | The section participates in the existing visible/overflow split; it is placed second so it rarely overflows for staff |

## 15. Rollout Plan

**No feature flag.** The section is gated by role, not by a toggle.

1. Land the shared logic + API clients + target table asset first (no UI), with unit tests green on both
   platforms.
2. Land the section behind normal staff-role gating in one app-release train for both platforms.
3. Because app releases are slow to propagate, ship the native section only **after** the web checklist has
   been live for at least one release cycle, so the copy and item catalog have stabilised — a client update
   is the expensive way to fix wording.
4. GA criteria: parity test green; VoiceOver and TalkBack scripts signed off; `target_resolution=unresolved`
   rate < 2%; crash-free rate unchanged.
5. Rollback: staged rollout on Play, phased release on App Store; the section can be emptied server-side by
   retiring rules (server-only), which is the fastest lever if something is badly wrong.

## 16. Test Plan

- **Unit** — Shared-fixture parity (AC-16). Status → presentation mapping. Target resolution (native / web /
  unresolved). Progress math. Badge cap and label. Role gating. Rate-limit handling.
- **Integration** — API clients against a mocked server (success, 403, 429, malformed payload). Storage test
  asserting nothing is persisted (AC-13). Deep-link routing for `checklist` including `?focus=` forwarding.
- **End-to-end** — XCUITest and Espresso: open course → badge visible → open section → expand evidence →
  tap a row → land natively with a highlight → back → dismiss with reason → badge decrements → restore.
  Student build: section absent.
- **Security** — Assert the web fallback goes through `LinkOpener` (never `openURL` / `ACTION_VIEW` direct)
  and that no bearer token is placed in an externally-opened URL.
- **Accessibility** — VoiceOver and TalkBack walkthroughs covering headers, item label/value/hint, evidence
  rows, progress, dismiss sheet focus return; Dynamic Type / font scale at maximum (AC-15); Reduce Motion
  (AC-10); contrast of the highlight outline in light and dark; RTL pass in Arabic.
- **Performance / load** — Frame-rate measurement scrolling 200 evidence rows on mid-tier devices; cold
  section-open timing; battery/network impact of the badge fetch over a simulated day.
- **Manual exploratory** — Small-device layout (iPhone SE / 320 dp Android), tablet, split-screen, offline
  and flaky-network behaviour, backgrounding mid-dismiss, pseudo-locale layout pass.

## 17. Documentation & Training

- Mobile README notes for the new feature folders and the shared logic contract.
- Help-centre article updated with mobile screenshots and a note that dismissals sync across surfaces.
- `docs/MOBILE_PLAN.md` updated with the new workspace section.
- Contributor note: the native target table is generated — do not hand-edit the bundled asset.

## 18. Open Questions

1. Should the badge also appear on the **course card** in the courses list, so a teacher sees which of six
   courses needs work without opening each? Proposed: yes, in a follow-up, once the summary endpoint's load
   profile is understood.
2. Should mobile support the dismiss **note**, or reason-only to keep the sheet light? Proposed: support it,
   optional, collapsed behind "Add a note".
3. Is a push notification ("your course starts in 3 days and 4 items need attention") wanted? Deferred to
   CC.10; would need notification-preference plumbing.
4. Which targets justify native destinations in v1? Proposed priority: course dates/publish (settings),
   enrollments, modules, syllabus, announcements — everything else via in-app browser initially.
5. Should the section be reachable from the Teach hub as well as the course workspace? Proposed: yes if
   cheap, since that is where teaching-mode users start.

## 19. References

- Existing files this work touches: `clients/ios/Lextures/Core/Routing/MobileDestinations.swift`,
  `.../Features/Courses/CourseWorkspaceNav.swift`, `.../Features/Courses/CourseDetailView.swift`,
  `.../Features/Navigation/CourseDrawer.swift`, `.../Resources/Localizable.xcstrings`;
  `clients/android/app/src/main/kotlin/com/lextures/android/core/navigation/MobileDestinations.kt`,
  `.../features/navigation/DrawerMappings.kt`, `.../features/courses/CourseDetailScreen.kt`,
  `clients/android/app/src/main/res/values*/strings.xml`, `clients/mobile/locales/*.json`.
- Precedent: [MB.1 in-app browser & `LinkOpener`](../../completed/mobile/MB.1-in-app-browser.md) (link
  routing policy), the CT.M mobile parity plans in [`docs/completed/content_tools/`](../../completed/content_tools/),
  and the motion tokens in [`docs/completed/animations/`](../../completed/animations/).
- Related plans: [CC.2](CC.2-checklist-state-api-and-dismissals.md),
  [CC.7](CC.7-web-checklist-page-and-nav-badge.md), [CC.8](CC.8-deep-link-and-highlight-targeting.md),
  [CC.10](CC.10-analytics-guidance-and-rollout.md).

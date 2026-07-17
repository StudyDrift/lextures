# VC.M1 — Mobile Boards: Foundation, Feature Flag, List & Board Shell

> Implementation plan. Source: bring the shipped web **Visual Collaboration Boards** (VC.1–VC.10, see [../../completed/visual-collaboration/](../../completed/visual-collaboration/)) to the native mobile apps. Landscape: [visual-collaboration/README](README.md). Mirrors the web foundation [VC.1](../../completed/visual-collaboration/VC.1-foundation-and-feature-flag.md) on iOS (SwiftUI) and Android (Jetpack Compose), reusing the existing course-features flag plumbing and course-detail navigation.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M1 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad + Collaboration squad |
| **Depends on** | Web VC.1 (shipped) |
| **Unblocks** | VC.M2, VC.M3, VC.M4, VC.M5, VC.M6, VC.M7 |

---

## 1. Problem Statement

The web app ships a full multi-format, real-time collaboration board (VC.1–VC.10), but the native iOS and
Android apps have **no boards surface at all** — a learner on their phone cannot even see that a course has
boards, let alone open one. Because a board's core moment (drop a card, watch the wall fill in) is inherently
mobile-friendly, this is the highest-value parity gap. VC.M1 lays the mobile foundation: surface the
`visualBoardsEnabled` flag, add a Boards entry to course detail, list a course's boards, and open a board
**shell** that the later mobile stories fill in.

## 2. Goals

- Surface the existing per-course `visualBoardsEnabled` flag in the mobile `CourseFeatures` model so mobile
  gates Boards exactly as web does (and as mobile already gates whiteboard/collab-docs/feed).
- Add a **Boards** entry point in the course-detail segmented chip row (parity with the web nav link), visible
  only when the flag is on.
- Ship a mobile **boards list** screen (read boards for a course, ordered by `updatedAt`, archived excluded)
  and a **board detail shell** (header + placeholder surface) on both platforms.
- Establish a shared mobile `BoardsApi` client + models (`Board`) that VC.M2–VC.M7 extend, mirroring the
  repo's `LMSAPI*` / `LmsApi` conventions.
- Support board **create/rename/archive** for members with the create permission (staff), matching web CRUD.

## 3. Non-Goals

- Posts / cards and their content types (VC.M2).
- Layouts and arrangement (VC.M3).
- Real-time sync over the board WebSocket (VC.M4) — VC.M1 uses plain REST fetch + pull-to-refresh.
- Reactions/comments/grades (VC.M5), sharing/access (VC.M6), moderation (VC.M7).
- Any server change — the REST/flag surface already exists from web VC.1; this is a pure client build.

## 4. Personas & User Stories

- **As a student on my phone**, I want a "Boards" tab in my course so I can find the shared wall my
  instructor set up.
- **As a student**, I want the Boards entry to appear only when my instructor turned boards on for the course.
- **As an instructor on mobile**, I want to create a board and rename/archive it from my phone.
- **As a self-learner**, I want to open my personal study-course board on the go.

## 5. Functional Requirements

- **FR-1.** The mobile `CourseFeatures` model MUST decode `visualBoardsEnabled` from `GET /api/v1/courses/{code}`
  (iOS `LMSModels.swift`, Android `LmsModels.kt`), alongside the existing `whiteboardEnabled` / `collabDocsEnabled`.
- **FR-2.** The course-detail screen MUST show a **Boards** segmented chip (or list entry) only when
  `visualBoardsEnabled == true`, mirroring how the existing chips gate on their flags.
- **FR-3.** The app MUST call `GET /api/v1/courses/{code}/boards` and render a single-column list of board
  cards (title, description, relative `updatedAt`); archived boards excluded unless an explicit "show archived"
  toggle passes `?includeArchived=true`.
- **FR-4.** Tapping a board MUST call `GET /api/v1/courses/{code}/boards/{board_id}` and open the board detail
  shell (header with title + overflow menu; a placeholder surface region filled by VC.M2/VC.M3).
- **FR-5.** Members with the create permission MUST be able to create a board (`POST …/boards`, body
  `{title, description?}`), rename/archive (`PATCH …/boards/{id}`), from an overflow/FAB action; the create
  affordance MUST be hidden for members without permission.
- **FR-6.** When `visualBoardsEnabled` is false (or the master flag is off, surfaced as a `404` from the list
  endpoint), the Boards entry MUST NOT appear and any deep link MUST resolve to a graceful "not available"
  state, not a crash.
- **FR-7.** The boards list and board shell MUST provide loading (skeleton cards), empty ("No boards yet"),
  and error (retry) states consistent with the shared mobile components (`LMSEmptyState` / skeletons).
- **FR-8.** All new copy MUST be added to the mobile locale catalogs (`clients/mobile/locales/*.json`) — no
  hardcoded English strings.

## 6. Non-Functional Requirements

- **Performance** — board list renders within one frame of the response; list fetch reuses the shared
  `APIClient` / `ApiClient` with the standard auth + retry behaviour.
- **Security** — every request goes through the authenticated client; board ids are UUIDs; no course-scope
  leakage (the server enforces course access — the client never trusts a board id from another course).
- **Privacy & Compliance** — boards and their future posts are education records; the mobile client displays
  only what the server returns and never caches board content to unprotected device storage.
- **Accessibility** — board cards are buttons/links with visible focus and VoiceOver/TalkBack labels; the
  create action has an accessible label; meets the mobile audit checklist ([../../accessibility/mobile-audit-checklist.md](../../accessibility/mobile-audit-checklist.md)).
- **Scalability** — list is a simple paged fetch; card rendering is virtualized by the platform list.
- **Reliability** — pull-to-refresh re-fetches; archive is a reversible soft action.
- **Observability** — reuse the app's existing screen-view / error logging; no new telemetry pipeline.
- **Internationalization** — strings externalised to locale JSON; timestamps use the device locale/timezone;
  RTL respected (Arabic catalog exists).
- **Backward compatibility** — additive; older app versions simply never show Boards (flag-gated), no forced upgrade.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course with `visualBoardsEnabled = true`, *when* a member opens course detail, *then* a
  **Boards** entry is visible and lists the course's non-archived boards.
- **AC-2.** *Given* `visualBoardsEnabled = false`, *when* course detail loads, *then* no Boards entry appears.
- **AC-3.** *Given* a member with create permission, *when* they create a board with a title, *then* it appears
  at the top of the list; *when* a member without permission views the list, *then* no create action is shown.
- **AC-4.** *Given* a board, *when* it is renamed/archived, *then* the list reflects the change (archived board
  disappears from the default list).
- **AC-5.** *Given* the boards list is empty, *when* it renders, *then* an empty state is shown, not a spinner.
- **AC-6.** *Given* a deep link to a board in a flag-off course, *when* opened, *then* a graceful "not
  available" screen is shown.
- **AC-7.** *Given* both platforms, *when* CI builds run, *then* iOS `xcodebuild build` and Android
  `./gradlew :app:compileDebugKotlin` are green.

## 8. Data Model

No server schema changes — VC.1's `board.boards` table and `course.courses.visual_boards_enabled` column
already exist. Client-side models only:

```swift
// iOS — Core/LMS (new file, e.g. LMSBoardModels.swift)
struct Board: Decodable, Identifiable {
    let id: String
    let courseId: String
    var title: String
    var description: String
    let slug: String
    var archived: Bool
    let createdBy: String?
    let createdAt: String   // RFC3339
    var updatedAt: String
}
```
```kotlin
// Android — core/lms (new file, e.g. BoardModels.kt)
@Serializable data class Board(
  val id: String, val courseId: String, val title: String, val description: String = "",
  val slug: String, val archived: Boolean = false, val createdBy: String? = null,
  val createdAt: String, val updatedAt: String,
)
```

## 9. API Surface

No new endpoints. Mobile consumes web VC.1's routes:

| Verb | Path | Auth |
|---|---|---|
| GET | `/api/v1/courses/{code}/boards` (`?includeArchived`) | course access |
| POST | `/api/v1/courses/{code}/boards` | `course:{code}:item:create` |
| GET | `/api/v1/courses/{code}/boards/{board_id}` | course access |
| PATCH | `/api/v1/courses/{code}/boards/{board_id}` | `item:create` |
| DELETE | `/api/v1/courses/{code}/boards/{board_id}` (archive; `?hard` needs manage) | `item:create` |

Response shape is VC.1's `Board`. No OpenAPI change (already documented).

## 10. UI / UX

- **Boards entry** — a **Boards** chip in the course-detail segmented row (iOS `LMSSegmentedChips`, Android
  `LmsSegmentedChips`), gated on the flag; selecting it shows the boards list within the course detail stack.
- **Boards list** — single-column card stack: title (serif), description (2-line clamp), relative updated-at;
  a "New board" action (FAB / toolbar) for permitted users; "Show archived" toggle.
- **Board detail shell** — screen header (title + overflow: rename, archive) over a placeholder surface region
  ("This board is empty" until VC.M2/VC.M3), pull-to-refresh.
- **States** — skeleton cards while loading; `LMSEmptyState` when empty; inline error with retry.
- **Accessibility** — cards are tappable rows with combined labels; overflow actions labelled; dark-mode via
  scheme-aware theme helpers only.
- **Copy & i18n** — new `boards.*` keys in `clients/mobile/locales/*.json`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse (iOS)**: `Core/Networking/APIClient.swift`, `Core/LMS/CourseFeaturesLogic.swift` +
  `LMSModels.swift` (add the flag), course-detail segmented screen (`Features/Courses/CourseSections.*`).
- **Reuse (Android)**: `core/network/ApiClient.kt`, `core/lms/CourseFeaturesLogic.kt` + `LmsModels.kt`,
  `features/courses/CourseSections.kt`.
- **New (iOS)**: `Core/LMS/LMSBoardModels.swift`, `Core/LMS/LMSAPIBoards.swift`,
  `Features/Boards/{BoardsListView,BoardDetailView}.swift` → regenerate the Xcode project
  (`python3 clients/ios/scripts/generate_xcodeproj.py`).
- **New (Android)**: `core/lms/BoardModels.kt`, `core/lms/BoardsApi.kt`,
  `features/boards/{BoardsListScreen,BoardDetailScreen}.kt`.

## 13. Dependencies & Sequencing

- Must ship after: web VC.1 (shipped).
- Must ship before: every other VC.M story.
- Shared infra: none new — existing mobile networking + course-detail navigation.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Flag not surfaced → Boards never appears | M | H | Add `visualBoardsEnabled` to both feature models + a decode test |
| Course-detail chip row already crowded | M | L | Boards chip is flag-gated; overflow if the row exceeds width |
| iOS project drift after new files | M | L | Run `generate_xcodeproj.py`; CI catches an unbuilt project |
| Deep-link to flag-off board | L | L | Graceful not-available screen (FR-6) |

## 15. Rollout Plan

- **Flag**: gated by the same `visualBoardsEnabled` per-course flag + platform master `VisualBoardsEnabled`;
  no new mobile flag.
- **Sequencing**: land models + flag → list → shell; ship behind the existing master flag (off in prod until
  the mobile MVP VC.M2–VC.M4 lands).
- **Rollback**: master flag off → Boards entry disappears on next course fetch; no data touched.

## 16. Test Plan

- **Unit** — `CourseFeatures` decodes `visualBoardsEnabled`; `Board` decode; list sorts/filters archived.
- **Integration** — list endpoint 404 (flag off) → not-available path; create/rename/archive round-trip;
  permission-gated create affordance.
- **End-to-end (device/sim)** — flag on → Boards chip → list → open shell → create → rename → archive.
- **Accessibility** — VoiceOver/TalkBack sweep of list + shell; dark-mode pass.
- **Manual** — flag on/off transitions; empty course; pull-to-refresh.
- **Build** — iOS `xcodebuild build`, Android `./gradlew :app:compileDebugKotlin` green.

## 17. Documentation & Training

- Update `clients/ios/README.md` and `clients/android/README.md` "Structure" tables with the Boards feature.
- Note the Boards surface in the mobile brand-system memory doc.
- End-user: "Boards on mobile" help-center note (screenshot of the course Boards chip).

## 18. Open Questions

1. Is Boards a **course-detail chip** or a top-level entry per course? (Recommendation: chip, matching web's
   in-course nav and mobile's existing IA.)
2. Do we allow board **create** on mobile v1, or read-only until later? (Recommendation: allow create — it is
   a small CRUD and instructors value phone-side setup.)
3. Should archived boards be reachable at all on mobile? (Recommendation: behind a "Show archived" toggle,
   read-only.)

## 19. References

- Web plan: [VC.1](../../completed/visual-collaboration/VC.1-foundation-and-feature-flag.md).
- Existing mobile files: `clients/ios/Lextures/Core/LMS/{LMSModels,CourseFeaturesLogic}.swift`,
  `clients/android/app/src/main/kotlin/com/lextures/android/core/lms/{LmsModels,CourseFeaturesLogic}.kt`,
  `clients/ios/Lextures/Features/Courses/CourseSections.swift`, `docs/MOBILE_PLAN.md`.
- Related mobile plans: [VC.M2](VC.M2-mobile-posts-and-content.md), [VC.M4](VC.M4-mobile-realtime-and-presence.md).

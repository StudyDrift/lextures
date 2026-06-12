# Lextures Mobile App — Redesign & Feature-Complete Plan

> **Status doc.** Check items off as they land. If work is interrupted, resume from the first
> unchecked item in the lowest-numbered phase. Every endpoint listed below was verified against
> `server/internal/httpserver` (route registration) and/or `clients/web/src/lib/*-api.ts`
> (response shapes) on 2026-06-11.

## 0. Where we are today

Both native clients (SwiftUI iOS 17+, Jetpack Compose Android) share a brand system
(cream paper / deep teal / coral / amber, serif display type, floating cards, deterministic
course-cover gradients) and currently ship **four tabs**: Dashboard, Courses, Notebooks, Inbox.

What exists:
- Auth (login/signup/keychain-keystore session), splash.
- Dashboard: hero greeting, stats row (courses / due this week / unread), due-this-week list,
  first-5 course list. Sign-out hidden in a toolbar menu.
- Courses list → course detail (modules + items) → item detail (read-only content + settings
  "preview box" for assignment/quiz/page/external link).
- Notebooks: rich editor (markdown, drawings, tasks, slash commands) with server sync.
- Inbox: folders, search, read/star/trash, compose.

What's missing for a "feature complete" LMS app: grades, submission status, syllabus,
announcements, notifications, attendance, profile/settings, and **every teacher/TA surface**
(grading backlog, grading, roster). The visual shell is also a stock tab bar; the inspiration
direction (TripGlide / Neobank / heart-rate app mocks) calls for a floating pill nav, avatar
header, segmented chip sections, and stronger card hierarchy.

### Key code locations

| Concern | iOS | Android |
|---|---|---|
| Theme | `clients/ios/Lextures/Core/Design/LexturesTheme.swift` | `clients/android/.../core/design/LexturesTheme.kt` |
| Shared LMS components | `clients/ios/Lextures/Features/Home/MainTabView.swift` | `clients/android/.../features/home/LmsComponents.kt` |
| Tab shell | `MainTabView.swift` | `features/home/HomeScreen.kt` |
| API layer | `Core/LMS/LMSAPI.swift`, `LMSModels.swift` | `core/lms/LmsApi.kt`, `LmsModels.kt` |
| Networking | `Core/Networking/APIClient.swift` | `core/network/ApiClient.kt` |

**iOS project file**: `Lextures.xcodeproj` is *generated*. After adding/removing Swift files run
`python3 clients/ios/scripts/generate_xcodeproj.py` (it globs all `*.swift` under `Lextures/`).

**Build verification**:
- iOS: `cd clients/ios && xcodebuild -project Lextures.xcodeproj -scheme Lextures -destination 'generic/platform=iOS Simulator' build` (or any available sim).
- Android: `cd clients/android && ./gradlew :app:compileDebugKotlin`.

---

## 1. Design language (applies to every screen below)

Inspired by the reference shots (travel app, neobank, health app): generous white cards on warm
paper, oversized serif display headings, pill chips, a floating rounded bottom nav, avatar in the
top-right, and segmented chip rows instead of nested nav where a screen has multiple facets.
Stay 100% inside the existing brand palette — coral strictly for urgency, amber for highlights.

Rules:
1. **Floating pill tab bar** (custom, not the stock bar): deep-teal capsule floating ~12pt above
   the bottom safe area, white/cream icons, selected item gets a filled circular "puck"
   (cream bg, deep-teal icon) + label. 5 tabs: **Home, Courses, Notebooks, Inbox, Profile**.
   Inbox badge = coral dot with count.
2. **Avatar header**: every root tab gets a serif greeting/title row with a tappable initials
   avatar (teal circle) on the right that jumps to Profile. Replaces the old toolbar menu.
3. **Segmented chip row** (`LMSSegmentedChips` / `LmsSegmentedChips`): horizontally scrolling
   pills — selected pill is solid deep teal w/ white text, unselected are white cards with
   hairline border. Used in Course Detail (Overview/Modules/Grades/…), Inbox folders (existing
   menu→chips), Notifications filter.
4. **Hero cards**: course detail keeps the gradient banner; dashboard hero panel stays.
5. **Progress ring**: small circular progress indicator (course % complete where computable from
   structure counts) on course cards — echoes the health-app ring.
6. **Empty/loading states**: keep `LMSEmptyState`; add card-shaped redacted skeletons
   (`.redacted(reason:)` iOS / shimmer-free placeholder cards Android) instead of bare spinners.

No new colors, no new fonts. Dark mode: all new components must read from the existing
scheme-aware helpers (never hardcoded grays).

---

## 2. API inventory used by this plan (verified)

Already used today: `/api/v1/courses`, `/courses/{code}`, `/courses/{code}/structure`,
`/courses/{code}/{content-pages|assignments|quizzes|external-links}/{id}`,
`/communication/messages[...]`, `/communication/unread-count`, `/me/notebooks[...]`,
`/me/notebook-tasks`.

New in this plan:

| Feature | Endpoint | Shape (source of truth) |
|---|---|---|
| Profile | `GET /api/v1/me` | `{id, email, displayName}` (`me.go handleGetMe`) |
| Notifications | `GET /api/v1/me/notifications` | `{notifications:[{id,userId,eventType,title,body,actionUrl?,isRead,createdAt}], unreadCount}` (`push_http.go`, `repos/notificationsinbox`) |
| Mark read | `POST /me/notifications/{id}/read`, `POST /me/notifications/read-all` | 204 |
| Announcements | `GET /api/v1/me/broadcasts` | `{broadcasts:[{id,type:'announcement'\|'emergency',subject,body,sentAt,createdAt,…}]}` (`broadcasts-api.ts`) |
| Acknowledge | `POST /api/v1/broadcasts/{id}/acknowledge` | 2xx |
| My grades | `GET /courses/{code}/my-grades` | `{columns:[{id,kind,title,maxPoints,dueAt,…}], grades:{itemId:score}, displayGrades:{...}, assignmentGroups, gradingScheme?, heldGradeItemIds?, droppedGrades?, gradeStatuses?}` (`courses-api.ts fetchCourseMyGrades`) |
| Syllabus | `GET /courses/{code}/syllabus` | `{sections:[…markdown sections…], updatedAt, requireSyllabusAcceptance, syllabusAcceptancePending?}` |
| My submission | `GET /courses/{code}/assignments/{item}/submissions/mine` | `{submission: {id, submittedAt, attachmentFilename?, versionNumber?, resubmissionRequested?, …} \| null}` |
| Submission grade | `GET .../submissions/{sid}/grade` | `{submissionId, pointsEarned?, maxPoints?, instructorComment?, posted?, excused?}` |
| Grade (staff) | `PUT .../submissions/{sid}/grade` | body `{pointsEarned?, instructorComment?, clearGrade?}` |
| Submissions list (staff) | `GET .../assignments/{item}/submissions?graded=ungraded\|graded\|all` | `{submissions:[ModuleAssignmentSubmissionApi]}` |
| Grading backlog (staff) | `GET /courses/{code}/grading-backlog` | `{items:[{assignmentId,assignmentTitle,ungradedCount}]}` |
| Attendance sessions | `GET /courses/{code}/attendance/sessions` | `{sessions:[{id,title,collectionMethod,sessionDate,status,…}]}` (`course-attendance-api.ts`) |
| Session detail | `GET .../attendance/sessions/{sid}` | `AttendanceSession & {records?, myRecord?, canSelfReport?}` |
| Self report | `POST .../attendance/sessions/{sid}/self-report` | body `{status}` |
| Roster | `GET /courses/{code}/enrollments` | verify exact row shape in `courses-api.ts` when implementing |

**Note — no student submission-create endpoint exists server-side yet** (web references
`/submissions/upload` but the server only registers GET list/mine + grade GET/PUT). Mobile
therefore shows submission *status* + received grade/feedback, and must not promise an upload
button. Revisit when the server lands upload.

---

## 3. Phases & checklists

Each item is done when it compiles on **both** platforms (or is explicitly platform-tagged),
respects dark mode, and is reachable in the UI.

### Phase A — New shell & design system primitives

- [x] **A1. iOS: floating pill tab bar.** Replace `TabView` in `MainTabView.swift` with a custom
  `ZStack` shell: content view + overlay `LexturesTabBar` (deep-teal capsule, 5 items, selected
  puck, coral inbox badge). Tabs: Home, Courses, Notebooks, Inbox, Profile. Keep per-tab
  `NavigationStack`s alive (use `@State selectedTab` + `ZStack` with opacity/zIndex so state
  survives tab switches).
- [x] **A2. Android: floating pill tab bar.** Same design in `HomeScreen.kt`: `Box` with content
  + `Box(Modifier.align(BottomCenter))` capsule row, replacing `Scaffold.bottomBar`'s stock
  `NavigationBar`. Preserve tab state via `rememberSaveable` + keep composables in a
  `when` (existing pattern fine).
- [x] **A3. Segmented chip row component.** `LMSSegmentedChips(options, selection)` (iOS, in
  `MainTabView.swift` alongside the other shared components) and `LmsSegmentedChips`
  (Android, `LmsComponents.kt`).
- [x] **A4. Avatar header component.** Serif title + initials avatar (from displayName/email)
  linking to Profile tab. iOS `LMSScreenHeader`; Android `LmsScreenHeader`.
- [x] **A5. Skeleton loading cards.** iOS: `LMSSkeletonCard` using `.redacted`; Android:
  placeholder cards with low-alpha surfaces. Swap into Dashboard/Courses/Grades screens.
- [x] **A6. Regenerate Xcode project** after new files (`generate_xcodeproj.py`) + both builds green.

### Phase B — Profile tab (new)

- [x] **B1. API: `GET /me` + `GET /me/notifications` (unreadCount only here).** iOS: add
  `fetchMe`, `MeProfile` model. Android: same in `LmsApi.kt`.
- [x] **B2. Profile screen.** Hero card with big initials avatar, display name (serif), email.
  Sections (cards): **Account** (email, display name), **Notifications** (link to
  Notifications screen B3), **About** (app version, server URL from config), **Sign out**
  (coral, confirmation dialog). Replaces the old toolbar sign-out (keep menu removal in A1/A4).
- [x] **B3. Notifications screen** (pushed from Profile bell + Home bell): list of
  notification rows (title, body, relative time, unread dot), "Mark all read" toolbar button,
  tap = mark read. Filter chips: All / Unread.
- [x] **B4. Announcements card on Home** (`GET /me/broadcasts`): top-of-feed card for the most
  recent unexpired announcement, coral left-accent when `type == 'emergency'`; "Acknowledge"
  button posts acknowledge and dismisses. Full list lives behind "See all" → Announcements
  screen (simple list).

### Phase C — Course detail redesign (segmented) + student academics

- [x] **C1. Segmented course detail.** Keep gradient hero banner; under it a chip row:
  **Overview · Modules · Grades · Attendance** (Attendance only when sessions exist or role
  is staff; Grades only when `viewerIsStudent`). Modules = existing module cards.
- [x] **C2. Overview tab = syllabus.** Render `GET /courses/{code}/syllabus` sections through the
  existing markdown renderer; show `updatedAt`; if `syllabusAcceptancePending` show notice
  banner (read-only acknowledgement happens on web — no accept POST from mobile v1).
  Falls back to course description when no syllabus content.
- [x] **C3. Grades tab.** `GET /my-grades`: overall summary card (computed weighted total when
  `assignmentGroups` provide weights — mirror web logic *simplified*: sum earned/possible per
  group, apply group weights when present; show "—" when nothing graded). Rows per column:
  title, due date, score `earned / maxPoints`, display grade, held badge (amber) when id in
  `heldGradeItemIds`, dropped strikethrough when `droppedGrades[id]`, excused chip via
  `gradeStatuses`.
- [x] **C4. Assignment detail: my submission & grade.** On `ItemDetailView` for kind
  `assignment` + `viewerIsStudent`: card showing submission status (`submissions/mine`):
  not submitted / submitted at + version + filename; if grade exists (`submissions/{id}/grade`
  with `posted`), show points + instructor comment. Coral "revision requested" banner when
  `resubmissionRequested`.
- [x] **C5. Attendance tab.** Student: list sessions; open sessions with
  `collectionMethod == 'self_report'` + `canSelfReport` → status picker (present/tardy) posting
  self-report; always show `myRecord` status chip per session. Staff: per-session record list
  (read-only v1).

### Phase D — Home (Dashboard) redesign

- [x] **D1. Header**: serif greeting + first name (from `/me` displayName, fallback email
  local-part), avatar → Profile, bell icon → Notifications with unread dot.
- [x] **D2. "Continue learning" carousel**: horizontal cards (cover gradient, course title,
  progress ring % of items in published modules; tap → course). Replaces the plain
  first-5 list.
- [x] **D3. "Due soon" rail** stays but restyled as the inspiration's schedule cards (coral
  accent stripe, course chip, relative due time); tapping a due item deep-links to the item
  detail (wire `navigationDestination` / nav callback through Courses tab).
- [x] **D4. Announcements strip** (from B4) between hero and stats.
- [x] **D5. Teacher snapshot card** (staff in ≥1 course): "Needs grading" total across staff
  courses (sum of backlog counts, fetched lazily) linking to per-course backlog (E1).

### Phase E — Teacher / TA tools

- [x] **E1. Grading backlog screen** per staff course (entry: course detail chip row gains
  **Grading** for staff): `GET /grading-backlog` list → assignment rows with ungraded count
  badges.
- [x] **E2. Submissions list** per assignment (`?graded=ungraded` default, chips:
  Ungraded / Graded / All) showing submitter (respect `blindLabel`), submitted time, version.
- [x] **E3. Grade sheet**: points stepper/field (validated ≤ maxPoints from grade GET),
  instructor comment, save via `PUT .../grade`. Show text submission content
  (`attachmentContentPath` text fetch is out of scope v1 — show filename + link out to web for
  file review).
- [x] **E4. Attendance (staff)**: covered read-only in C5 v1. (Roll-call editing = v2.)

### Phase F — Polish & parity sweep

- [x] **F1. Inbox folders → chip row** (replace Menu picker with `LMSSegmentedChips`).
- [x] **F2. Unify unread badge plumbing** (single source on shell; Home stat card, tab badge,
  bell dot all read it).
- [x] **F3. Empty states audit** — every new screen has icon+title+message empty state.
- [x] **F4. Dark mode audit** — run both apps in dark appearance; no hardcoded grays.
- [x] **F5. Update memory file** `project_mobile_brand_system.md` (new components, 5-tab shell).
- [x] **F6. Final builds**: `generate_xcodeproj.py`, iOS `xcodebuild build`, Android
  `./gradlew :app:compileDebugKotlin` — all green.

---

## 3.5 Progress log (2026-06-11)

All Phase A–F items landed on both platforms; iOS `xcodebuild build` and Android
`:app:compileDebugKotlin` are green. Deviations from the original spec:

- **D2 (carousel ring):** there is no per-student progress endpoint for the student role, so the
  course carousel shows honest "N modules · M items" counts instead of a fake completion ring.
  The progress ring component exists and is used for the overall grade on the Grades tab.
- **F4 (dark mode):** code-level audit — every new component reads scheme-aware theme helpers
  (`LexturesTheme.*(for:)` / `isDarkTheme()` helpers); no hardcoded grays were introduced. A
  visual pass on simulators is still worth doing when convenient.
- **E4:** staff attendance is read-only (roster list) as planned; roll-call editing is v2.
- New iOS files: `Core/LMS/LMSFeatureModels.swift`, `Core/LMS/LMSAPIFeatures.swift`,
  `Features/Profile/{ProfileView,NotificationsView}.swift`, `Features/Home/AnnouncementsViews.swift`,
  `Features/Courses/{CourseSyllabusView,CourseGradesView,CourseAttendanceView}.swift`,
  `Features/Grading/GradingBacklogView.swift` (project regenerated via `generate_xcodeproj.py`).
- New Android files: `core/lms/LmsFeatureModels.kt`, `features/profile/{ProfileTab,NotificationsScreen}.kt`,
  `features/courses/CourseSections.kt`, `features/grading/GradingScreens.kt`; new endpoints appended
  to `core/lms/LmsApi.kt`.
- Breaking signature changes: iOS `ItemDetailView(course:item:)` (was `courseCode:`), iOS
  `DashboardView()`/`InboxView()` now read `AppShellModel` from the environment; Android
  `ItemDetailScreen(course=…)` (was `courseCode=`), `DashboardTab(session, shell, onOpenProfile)`.

## 4. Out of scope (v1) — revisit later

- Quiz taking on mobile (start/advance/submit exist server-side; large UX surface — phase G).
- Student submission upload (blocked server-side; no endpoint).
- Course feed / discussions / forums / groups (endpoints exist; big surface).
- Course files browser, meetings/calendar ICS, parent role, flashcards, AI tutor.
- Push notifications (`/me/push-subscriptions` is web-push VAPID; native APNs/FCM needs server work).

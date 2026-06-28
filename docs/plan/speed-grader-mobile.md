# Speed Grader — Mobile (iOS + Android)

> Implementation plan for a SpeedGrader-style grading flow in the Lextures mobile
> apps. Lets staff with grading permission pick an assignment, page through
> students, read each submission, and enter a grade/feedback without bouncing
> back to a list between students.

## Metadata

| Field | Value |
|---|---|
| **Feature** | Speed Grader (mobile) |
| **Section** | 21 — Mobile / Cross-platform |
| **Severity** | MAJOR |
| **Status (today)** | PARTIAL — backlog → submissions list → single grade sheet exists; no per-student paging, text submissions are "open the web app" |
| **Estimated effort** | M |
| **Platforms** | iOS (SwiftUI), Android (Compose) |
| **Backend** | No changes required — endpoints already return everything we need |

---

## 1. Problem Statement

Mobile graders today can open a course's grading backlog, drill into one
assignment, see its submissions, and tap a single submission to open a grade
sheet (points + comment). That sheet is a dead end: after saving you return to
the list and must re-tap the next student. Text submissions show only the
filename with a "open the web app to review" note, so the grader can't actually
read the work on device. The result is that real grading still has to happen on
the web. A SpeedGrader-style flow — pick assignment, pick a starting student,
then swipe/Next through the roster grading each one inline — makes mobile a
first-class grading surface.

## 2. Goals

- Staff with grading permission can grade an assignment's submissions end to end
  on mobile, paging student-to-student without returning to the list.
- The submission content (text body + attachment metadata) is visible inline so
  the grader has context for the score.
- Saving a grade advances to the next ungraded student automatically (with the
  option to go back/forward manually).
- Progress is visible ("3 of 12", count remaining) so the grader knows how much
  is left.
- Reuse the existing backlog/submissions/grade endpoints and the existing entry
  points (course-detail Grading section + dashboard teacher snapshot). No new
  backend work.

## 3. Non-Goals

- Quiz / per-question grading stays read-only on mobile (unchanged: keep the
  existing "grade on web" info screen). Speed Grader covers assignment
  submissions only in v1.
- No inline file/PDF/image rendering of attachments in v1 — show filename, type,
  and (optionally) a "view attachment" affordance. Text bodies ARE rendered.
- No rubric grading, no SpeedGrader-style annotation/markup, no offline queueing.
- No new permissions model — reuse whatever already gates the backlog/grade
  endpoints server-side.

## 4. User Stories

- **As an instructor**, I open an assignment from the grading backlog, tap a
  student, and land in a grader where I can read their submission and enter a
  score, then advance to the next student with one tap.
- **As a TA**, I want to see how many submissions remain ungraded so I can pace
  a grading session.
- **As a grader**, I can jump backward to a student I already graded to fix a
  score.

## 5. What Already Exists (reuse)

Backlog → list → single grade sheet is implemented on both platforms:

- iOS: `clients/ios/Lextures/Features/Grading/GradingBacklogView.swift`
  (`GradingBacklogSection`, `SubmissionsListView`, `GradeSubmissionSheet`,
  `QuizAttemptInfoSheet`).
- Android:
  `clients/android/app/src/main/kotlin/com/lextures/android/features/grading/GradingScreens.kt`
  (`GradingBacklogScreen`, `SubmissionsListScreen`, `GradeSubmissionScreen`,
  `QuizAttemptInfoScreen`).

API + models already present:

- iOS `LMSAPI.fetchGradingSubmissions / fetchSubmissionGrade / putSubmissionGrade`
  in `Core/LMS/LMSAPIFeatures.swift`; models in `Core/LMS/LMSFeatureModels.swift`.
- Android `LmsApi.fetchGradingSubmissions / fetchSubmissionGrade / putSubmissionGrade`;
  models in `core/lms/LmsFeatureModels.kt`.

Backend response (`submissionToJSON` in
`server/internal/httpserver/assignment_submissions_http.go`) already returns:
`bodyText`, `attachmentFilename`, `attachmentMimeType`, `attachmentContentPath`,
`attachments[]`, `versionNumber`, plus grading state. The clients currently
decode only a subset — they drop `bodyText`, `attachmentMimeType`, and
`attachmentContentPath`.

## 6. Functional Requirements

- **FR-1.** Tapping a (non-quiz) submission in the submissions list MUST open a
  **Speed Grader** screen positioned on that student instead of the current
  single-student sheet.
- **FR-2.** The Speed Grader MUST present, for the current student: display name
  (respecting blind-grading `blindLabel`), submitted-at, attempt/version,
  submission **text body** (rendered) when present, and attachment filename +
  type when present.
- **FR-3.** The Speed Grader MUST provide a points field (validated `0…maxPoints`)
  and an optional feedback comment, pre-filled from the existing grade via
  `fetchSubmissionGrade`, and save via `putSubmissionGrade` — identical
  semantics to today's sheet.
- **FR-4.** The grader MUST be able to move to the previous / next submission in
  the loaded list. Saving SHOULD advance to the next **ungraded** student in the
  list; if none remain, show a "done" state.
- **FR-5.** The grader MUST show progress: current index, total, and remaining
  ungraded count.
- **FR-6.** Per-student grade/comment/maxPoints MUST load when that student
  becomes current (lazy per-student fetch), and unsaved edits MUST NOT leak
  between students.
- **FR-7.** Quiz items MUST keep the existing read-only "grade on web" behavior
  (no Speed Grader).
- **FR-8.** The grading-backlog and submissions-list entry points and their
  navigation MUST remain functional; Speed Grader replaces only the leaf grade
  sheet for assignments.

## 7. Non-Functional

- **Permissions** — unchanged; server already authorizes backlog/grade calls.
- **Blind grading** — keep honoring `blindLabel`; never reveal identity the API
  redacted.
- **Accessibility** — fields labelled; Prev/Next reachable via VoiceOver /
  TalkBack; respect Dynamic Type / font scaling (use existing `LexturesTheme` /
  `LexturesType`).
- **Resilience** — a failed save keeps the grader on the current student and
  surfaces the error (reuse `session.mapError` / `LMSErrorBanner`); navigating
  away from an unsaved edit is allowed (auto-advance only after a successful save).

## 8. Design / Approach

Keep the backlog and submissions list exactly as-is. Replace the leaf:

- **iOS:** `SubmissionsListView`'s `.sheet(item: $grading)` for the non-quiz
  branch opens a new `SpeedGraderView` (full-screen `NavigationLink`/sheet)
  seeded with the loaded `[AssignmentSubmission]` and the tapped index, plus
  `course` and `assignmentId`. `SpeedGraderView` owns: current index, a per-index
  cache of `SubmissionGrade` (points/comment/maxPoints), header card,
  body/attachment card, score + feedback cards (lifted from `GradeSubmissionSheet`),
  a progress chip, and a Prev / Save & Next bar. On successful save, mark the
  student graded locally, recompute remaining, and advance to the next ungraded
  index. New file: `Features/Grading/SpeedGraderView.swift` (added to
  `project.pbxproj`). `GradeSubmissionSheet` stays for any single-shot callers but
  the list now routes to the grader.
- **Android:** `SubmissionsListScreen`'s non-quiz tap sets a
  `speedGrader = (submissions, index)` state and renders a new
  `SpeedGraderScreen` (same back-stack-as-composable pattern already used for
  `GradeSubmissionScreen`). It holds index + a `Map<submissionId, SubmissionGrade>`
  cache and mirrors the iOS layout. New file:
  `features/grading/SpeedGraderScreen.kt`. Reuse `formatPoints`, `LmsCard`,
  `LmsErrorBanner`.

Shared submission-content surfacing (small, additive):

- Add `bodyText`, `attachmentMimeType`, `attachmentContentPath` to the
  `AssignmentSubmission` model on both platforms (iOS `LMSFeatureModels.swift`,
  Android `LmsFeatureModels.kt`). All optional/defaulted → backward compatible.
- Quiz-mapped submissions leave these nil (already the case).

## 9. Files to Touch

**iOS**
- `clients/ios/Lextures/Features/Grading/SpeedGraderView.swift` *(new)* — grader UI + paging/state.
- `clients/ios/Lextures/Features/Grading/GradingBacklogView.swift` — route non-quiz taps to the grader.
- `clients/ios/Lextures/Core/LMS/LMSFeatureModels.swift` — add `bodyText` / attachment fields to `AssignmentSubmission`.
- `clients/ios/Lextures.xcodeproj/project.pbxproj` — register the new file.

**Android**
- `.../features/grading/SpeedGraderScreen.kt` *(new)* — grader UI + paging/state.
- `.../features/grading/GradingScreens.kt` — route non-quiz taps to the grader.
- `.../core/lms/LmsFeatureModels.kt` — add `bodyText` / attachment fields to `AssignmentSubmission`.

**Backend** — none.

## 10. Acceptance Criteria

1. From a course's Grading section (or dashboard teacher snapshot), opening an
   assignment and tapping a student opens the Speed Grader on that student.
2. The grader shows name, submitted time, attempt, and the submission's text body
   (when present) plus attachment info; quizzes still show the read-only web note.
3. Entering valid points + optional comment and tapping Save persists the grade
   (verified by reopening) and auto-advances to the next ungraded student.
4. Prev/Next move between students; previously-graded students show their saved
   score pre-filled; edits don't bleed across students.
5. Progress indicator reflects index/total and remaining ungraded; reaching the
   end shows a "caught up" state.
6. Blind-graded assignments still show only the blind label.
7. Both apps build; existing backlog/list flows and quiz read-only flow unchanged.

## 11. Phasing

1. Model fields (`bodyText` + attachment) on both platforms.
2. iOS `SpeedGraderView` + routing.
3. Android `SpeedGraderScreen` + routing.
4. Progress/auto-advance polish + empty/done states.
5. Manual verification on each platform; update this doc with any deltas.

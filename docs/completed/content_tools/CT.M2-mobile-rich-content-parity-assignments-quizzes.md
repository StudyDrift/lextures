# CT.M2 — Mobile Rich Content Parity: Assignments, Quizzes, Syllabus & Discussions

> Implementation plan. Source: mobile parity for authored rich content. Folder overview: [README](README.md). Consumes the engine from [CT.M1](../../completed/content_tools/CT.M1-mobile-markdown-engine-tables-code-math.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.M2 |
| **Section** | Content Tools (CT) — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | CT.M1 |
| **Unblocks** | CT.M3 |

---

## 1. Problem Statement

CT.M1 builds the engine; CT.M2 makes every reading surface use it. Today the apps pick a renderer
almost at random: iOS assignment instructions, the syllabus and item detail go through the
headings-and-bullets-only `MarkdownTextView`, while content pages and quiz prompts go through the
richer `CourseMarkdownContentView`. Android is worse — its `MarkdownText` deletes bold, italic, code
and link syntax, and the **quiz screen does not use a markdown renderer at all**: `QuizQuestion.kt:42`
renders `Text(question.prompt)`, choices render `Text(choice)`, and the review screen renders
`Text(promptSnapshot)`. A physics question authored with an equation and a data table is unanswerable
on an Android phone. CT.M2 routes every one of these call sites through the CT.M1 renderer and deletes
the legacy ones.

## 2. Goals

- Every authored body on mobile — assignment instructions, quiz intro/instructions, question prompts,
  answer choices, feedback and review snapshots, syllabus sections, discussions, announcements,
  portfolio artifacts, tutor and review sessions — renders through the single CT.M1 renderer.
- Delete `MarkdownTextView` (iOS) and `MarkdownText`/`stripInline` (Android); one renderer remains.
- Close the specific gaps that block assessment: markdown in **answer choices**, **feedback/rationale**
  and **graded review snapshots**, which have never rendered on mobile.
- No regression on any screen that renders acceptably today, proven by snapshot tests per surface.
- Keep read-aloud, translation and reader-preference chrome working on every migrated surface.

## 3. Non-Goals

- New engine capability — anything the renderer cannot do is a CT.M1 change, not a CT.M2 one.
- Content Tools behaviour: on these surfaces a `lex-tool` fence still shows the CT.M1 placeholder;
  CT.M3 activates it.
- Composers and editors (announcement composer, notebook editor, discussion reply box) — CT.M2 changes
  what is *rendered*, not what is *typed*. Live preview inside composers is a fast-follow.
- Desktop and CLI clients.
- Changing any server response shape.

## 4. Personas & User Stories

- **As a student**, I want the data table in my assignment instructions to render so I know what to
  submit.
- **As a student taking a quiz**, I want the equation in the question and the code in the answer
  choices to render, so I am not guessing at LaTeX.
- **As a student reviewing a graded quiz**, I want the question snapshot and the instructor's feedback
  to render the same way the live question did.
- **As an instructor**, I want the syllabus table of due dates I wrote to be readable on a phone.
- **As a TA**, I want discussion posts with code blocks to be legible while I moderate from my phone.
- **As a homeschool parent**, I want the weekly plan table in the syllabus to render on my tablet.

## 5. Functional Requirements

- **FR-1.** Assignment instructions MUST render through the CT.M1 renderer
  (`AssignmentDetailView.swift:172`, `AssignmentDetailScreen.kt:411`), replacing the reduced renderers.
- **FR-2.** Quiz **question prompts** MUST render through the CT.M1 renderer on both platforms;
  `QuizQuestion.kt:42`'s raw `Text(question.prompt)` MUST be replaced.
- **FR-3.** Quiz **answer choices** (multiple choice, multiple answer, true/false, matching pairs,
  ordering items) MUST render inline markdown — bold, italic, inline code, inline math and links — on
  both platforms, while remaining fully tappable with a ≥ 44 pt / 48 dp target.
- **FR-4.** Quiz **intro/instructions** MUST render through the CT.M1 renderer
  (`QuizIntroView.swift:47`, `QuizIntroScreen.kt`).
- **FR-5.** Quiz **results/review** MUST render `promptSnapshot`, per-question feedback and rationale
  through the CT.M1 renderer (`QuizQuestion.kt:153` and the iOS equivalent).
- **FR-6.** Assignment **rubric criteria/descriptors** and **submission feedback comments** MUST render
  inline markdown where the server returns authored text.
- **FR-7.** Syllabus sections MUST render through the CT.M1 renderer (`CourseSyllabusView.swift:62`,
  `CourseSections.kt:143`).
- **FR-8.** Module item detail bodies MUST render through the CT.M1 renderer
  (`ItemDetailView.swift:242`, `ItemDetailScreen.kt:259`).
- **FR-9.** Discussion posts/replies, announcements, board post bodies (`BoardPostCard.swift:259`,
  `BoardPostCard.kt:224`), portfolio artifacts (`ArtifactDetailView.swift:51`,
  `ArtifactDetailScreen.kt:164`), tutor messages (`TutorChatView.swift:243`) and review-session stems
  (`ReviewSessionView.swift:166,289`) MUST all use the same renderer.
- **FR-10.** `MarkdownTextView` (iOS) and `MarkdownText` + `stripInline` (Android) MUST be deleted once
  every call site is migrated; CI MUST fail on reintroduction (lint rule or grep gate).
- **FR-11.** Surfaces with a constrained layout (chat bubbles, board cards, list rows) MUST pass a
  `compact` rendering mode that caps heading sizes and block spacing while using the same parser —
  never a second parser.
- **FR-12.** Reader chrome that exists today (read-aloud, translation, reader preferences,
  `lexturesReadableText()`) MUST keep working on every migrated surface, and read-aloud MUST receive
  the plain-text projection of new block kinds rather than their source markup.
- **FR-13.** Quiz surfaces MUST NOT regress the lockdown/proctoring path: nothing the renderer adds
  (copy button, full-screen table viewer, link taps) may open an escape hatch while a lockdown quiz is
  in progress — those affordances MUST be suppressed in that mode.
- **FR-14.** Offline-cached bodies MUST render identically to live ones; the renderer MUST never
  require a network call to lay out text.

## 6. Non-Functional Requirements

- **Performance** — No measurable regression in screen open time; quiz question paging stays under
  100 ms per question on a mid-range device. Bodies parsed once per value, not per recomposition.
- **Security** — Link scheme allowlist enforced identically on every surface; lockdown suppression per
  FR-13; no HTML execution anywhere.
- **Privacy & Compliance** — Rendering only. Quiz prompts, feedback and submissions remain education
  records; nothing new is logged or cached beyond the existing offline caches.
- **Accessibility** — WCAG 2.1 AA. Answer choices remain a single accessible control per choice with a
  complete label (formatting must not fragment the label into pieces); focus order unchanged; the
  full-screen table viewer is reachable and dismissible on every surface that allows it.
- **Scalability** — Client-only.
- **Reliability** — Migration is per-surface behind one flag; any surface can fall back independently
  during rollout.
- **Observability** — Reuse CT.M1's parse-fallback counter, labelled by surface, so we learn which
  screens still hit unsupported constructs.
- **Maintainability** — After CT.M2 there is exactly one markdown renderer per platform. New screens
  get rich content for free.
- **Internationalization** — No new user-facing strings beyond those CT.M1 introduced; RTL verified on
  quiz choices and tables inside assignments.
- **Backward compatibility** — Purely additive to what renders. Any body that rendered before renders
  at least as well after; snapshot tests are the contract.

## 7. Acceptance Criteria

- **AC-1.** *Given* an assignment whose instructions contain a table, code block and equation, *When*
  a student opens it on iOS and Android, *Then* all three render correctly.
- **AC-2.** *Given* a quiz question whose prompt contains a table and inline math, *When* it renders on
  Android, *Then* the table and math render (not raw markdown/LaTeX text).
- **AC-3.** *Given* multiple-choice options containing inline code and bold text, *When* they render,
  *Then* the formatting is applied, each option is one tappable control with a ≥ 48 dp target, and the
  accessibility label reads the full option text once.
- **AC-4.** *Given* a graded quiz review, *When* the student opens it, *Then* the question snapshot and
  feedback render identically to the live question.
- **AC-5.** *Given* a syllabus with a weekly-schedule table, *When* it renders on a phone, *Then* the
  table renders and the page does not scroll horizontally.
- **AC-6.** *Given* a lockdown quiz in progress, *When* a question contains a link and a code block,
  *Then* the link is not tappable and the copy affordance is hidden.
- **AC-7.** *Given* the repo after CT.M2, *When* CI greps for `MarkdownTextView`, `MarkdownText(` or
  `stripInline`, *Then* no production references remain and the gate passes.
- **AC-8.** *Given* every migrated surface, *When* snapshot tests run in light and dark themes,
  *Then* they match approved baselines.
- **AC-9.** *Given* a body with a `lex-tool` fence on any migrated surface, *When* it renders, *Then*
  the CT.M1 placeholder appears — never raw JSON, never a blank gap.
- **AC-10.** *Given* read-aloud started on an assignment containing a table, *When* it plays, *Then* it
  speaks a sensible linearisation rather than pipe characters.
- **AC-11.** *Given* offline mode with a cached quiz, *When* questions render, *Then* output matches
  the online rendering.
- **AC-12.** *Given* CI, *When* it runs, *Then* iOS build, Android compile and both unit suites pass.

## 8. Data Model

**No server schema change, no migration, no new client models.** CT.M2 rewires existing view code to a
renderer that already exists after CT.M1.

## 9. API Surface

**No new or changed endpoints.** Every body consumed here is already returned by the assignment, quiz,
syllabus, item, discussion, board, portfolio and tutor endpoints the apps call today. No OpenAPI
change, no rate-limit change.

## 10. UI / UX

- **Modified (iOS)** — `Features/Assignments/AssignmentDetailView.swift`,
  `Features/Quiz/QuizQuestionViews.swift` (prompt + `choiceButton` label),
  `Features/Quiz/QuizIntroView.swift`, `Features/Quiz/QuizResults*` (review snapshots),
  `Features/Courses/{ItemDetailView,CourseSyllabusView,VibeActivityView}.swift`,
  `Features/Boards/BoardPostCard.swift`, `Features/Portfolio/ArtifactDetailView.swift`,
  `Features/Tutor/TutorChatView.swift`, `Features/Review/ReviewSessionView.swift`,
  `Features/Discussions/*`, `Features/Home/AnnouncementComposerView.swift` (render side only).
- **Modified (Android)** — `features/assignments/AssignmentDetailScreen.kt`,
  `features/quiz/{QuizQuestion,QuizIntroScreen}.kt`, `features/courses/{ItemDetailScreen,
  CourseSections,ContentPageScreen,VibeActivityScreen}.kt`, `features/boards/BoardPostCard.kt`,
  `features/portfolio/ArtifactDetailScreen.kt`, `features/tutor/TutorChatScreen.kt`,
  `features/discussions/*`, `features/feed/FeedScreens.kt`.
- **Deleted** — `MarkdownTextView` (iOS `ItemDetailView.swift:372`), `MarkdownText`, `MdBlock`,
  `parseMarkdownBlocks`, `stripInline` (Android `ItemDetailScreen.kt:412+`).
- **Key flows** — (1) Student opens an assignment → instructions render fully → submits. (2) Student
  takes a quiz → each question and its choices render → answers → reviews the graded attempt with the
  same rendering. (3) Instructor opens the syllabus on a phone and reads the schedule table.
- **States** — unchanged per screen (loading skeletons, error banners, staleness chips, offline
  indicators all stay); only the body rendering changes. A `compact` mode is used inside chat bubbles,
  board cards and list rows.
- **Accessibility annotations** — quiz choices keep `Button` semantics with one merged label; nothing
  inside a choice becomes independently focusable; table viewers inside quizzes return focus to the
  question on dismiss.
- **Copy & i18n** — no new keys beyond CT.M1's `mobile.markdown.*`.

## 11. AI / ML Considerations

Not AI-touching. (AI-generated assignment and quiz content is a major source of tables with blank
lines — which is precisely why CT.M1's healing rule matters here.)

## 12. Integration Points

- **Internal (iOS)** — the CT.M1 renderer, `Core/Accessibility/ReadAloud.swift`,
  `Core/LMS/QuizLogic.swift` (`visibleChoices`), the lockdown/proctoring controller, reader toolbar.
- **Internal (Android)** — the CT.M1 renderer, `core/accessibility/ReadAloud.kt`,
  `core/lms/QuizLogic.kt`, `features/quiz/LockdownController.kt`.
- **External** — none.
- **Events** — none.

## 13. Dependencies & Sequencing

- Must ship after: **CT.M1** (the renderer must exist).
- Must ship before: **CT.M3** — the tool host mounts inside these same surfaces, so they must be on
  the shared renderer first or the host would need N integrations instead of one.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A migrated screen regresses visually in a way tests miss | M | M | Snapshot test per surface in light/dark before the swap; migrate one surface per PR |
| Rich choices break quiz tap targets or a11y labels | M | H | FR-3 explicitly keeps one control per choice with a merged label; scripted screen-reader pass on choices |
| Renderer affordances leak an escape hatch during lockdown quizzes | L | H | FR-13 suppression, verified by an explicit AC and a lockdown test case |
| Compact surfaces (chat, cards) look wrong with full block spacing | H | L | `compact` mode parameter (FR-11), same parser, different spacing tokens |
| Read-aloud starts reading pipe characters | M | M | Plain-text projection updated for tables/code/math as part of FR-12 |
| Legacy renderer creeps back in a parallel PR | M | L | CI grep gate (AC-7) |

## 15. Rollout Plan

- **Feature flag** — reuse CT.M1's `mobileRichMarkdownEnabled`, with per-surface constants so a single
  screen can fall back without reverting the release.
- **Sequencing** — assignments → quiz (intro, prompt, choices, review) → syllabus & item detail →
  discussions/boards/portfolio/tutor → delete legacy renderers and add the CI gate.
- **Dogfood** — internal courses with table-heavy assignments and math-heavy quizzes.
- **GA criteria** — all ACs green; snapshot baselines approved; one full dogfood quiz cycle
  (take → submit → review) on both platforms with no P1.
- **Rollback** — per-surface flag constants; the deleted renderers stay recoverable in git history but
  the flag path, not a revert, is the intended rollback.

## 16. Test Plan

- **Unit** — choice-label span rendering; compact-mode spacing selection; lockdown suppression logic;
  plain-text projection used by read-aloud.
- **Integration** — snapshot tests per surface (assignment instructions, quiz prompt, quiz choices,
  quiz review, syllabus section, item detail, board card, tutor bubble) in light and dark, LTR and RTL.
- **End-to-end (device)** — full quiz cycle on a seeded quiz containing a table, math and code in the
  prompt and in the choices; open the graded review; open an assignment with the same content and
  submit.
- **Security** — lockdown quiz with links/code present: verify no navigation, no clipboard, no
  full-screen escape.
- **Accessibility** — screen-reader pass over quiz choices (label completeness, no fragmentation),
  assignment instructions with a table, and the review screen; 200% font-scale pass on quiz choices.
- **Performance / load** — question paging benchmark across a 40-question quiz with rich prompts.
- **Manual exploratory** — very long choices, choices that are pure code, prompts with only an image,
  RTL quiz, offline quiz, tablet split view.

## 17. Documentation & Training

- Instructor docs: remove any "avoid tables/math because mobile can't show them" guidance; state the
  supported constructs.
- End-user: note that graded review now renders the same as the live question.
- Internal: update `clients/ios/README.md` / `clients/android/README.md` with the one-renderer rule and
  the CI gate.
- No API reference change.

## 18. Open Questions

1. Do matching and ordering question types need per-item rich rendering, or is inline-only enough for
   their compact rows? (Recommendation: inline-only in v1, with a follow-up if authors complain.)
2. Should composers gain a live markdown preview in this story or a fast-follow? (Recommendation:
   fast-follow — CT.M2 is a render-path change and mixing in editors widens the regression surface.)
3. Is there any surface where the reduced renderer was a deliberate product choice rather than an
   accident? (Audit with design before deleting; boards cards are the likeliest candidate and are
   covered by `compact` mode.)

## 19. References

- Existing mobile call sites: `AssignmentDetailView.swift:172`, `QuizQuestionViews.swift:47`,
  `QuizIntroView.swift:47`, `ItemDetailView.swift:242,372`, `CourseSyllabusView.swift:62`,
  `BoardPostCard.swift:259`, `ArtifactDetailView.swift:51`, `TutorChatView.swift:243`,
  `ReviewSessionView.swift:166,289`; `AssignmentDetailScreen.kt:411`, `QuizQuestion.kt:42,153`,
  `CourseSections.kt:143`, `ItemDetailScreen.kt:259,412`, `BoardPostCard.kt:224`,
  `ArtifactDetailScreen.kt:164`.
- Web reference: `clients/web/src/components/quiz/quiz-response-display.tsx`,
  `clients/web/src/components/quiz/quiz-student-preview-modal.tsx`,
  `clients/web/src/components/content-page/content-page-reader.tsx`.
- Related plans: [CT.M1](../../completed/content_tools/CT.M1-mobile-markdown-engine-tables-code-math.md),
  [CT.M3](CT.M3-mobile-content-tool-host-and-state.md).
- Standards: WCAG 2.1 AA §1.3.1, §2.5.5 (target size), §4.1.2 (name, role, value for choices).

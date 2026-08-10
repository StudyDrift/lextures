# UX.10 — Course Home and the Learning Flow

> Implementation plan. Source: [audit.md](audit.md) §5 G-15, §1.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.10 |
| **Section** | UI/UX — Core Surfaces |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — functional but undesigned; god components block change |
| **Estimated effort** | XL (>2mo) |
| **Owner (proposed)** | Product Design + Web + Learning Design |
| **Depends on** | UX.1, UX.2, UX.3, UX.7, UX.12, TD.14 |
| **Unblocks** | UX.16; the core learner value proposition |

---

## 1. Problem Statement

The learner's core loop — open a course, find the next thing, read/watch/do it,
submit, see feedback — runs through the least-maintainable code in the product.
`course-modules.tsx` is **3,284 lines with 100 `useState` hooks**;
`course-module-quiz-page.tsx` is 3,403 lines; `quiz-student-take-panel.tsx` is
2,196. These surfaces cannot be redesigned, only rewritten. Consequently the
learning flow has never had a designed reading experience: no measure constraint,
no persistent progress indicator, no consistent "next" affordance, and a module
list that presents every item at equal weight regardless of state. For a product
whose thesis is "the learning environment that adapts", the surface where learning
actually happens is the least designed one — and per **R-1**, every unit of
extraneous load here is subtracted directly from learning.

## 2. Goals

- Design a **continuous learning flow**: always obvious what to do next, always
  possible to resume, never a dead end at the end of an item.
- Give course home a **role-aware, prioritised** shape (the same contract as
  [UX.9](UX.9-role-aware-dashboard.md), applied per course).
- Make **progress and mastery legible** at course, module and item level —
  supporting competence, the need gamification does not serve (**R-5**).
- Deliver a genuine **reading experience** for content pages: measure, typography,
  focus mode, annotation, resume position.
- Decompose the god components so the surface becomes maintainable.

## 3. Non-Goals

- Changing the underlying content model (modules, items, prerequisites,
  conditional release).
- Redesigning the authoring/editing experience (a separate future plan).
- The adaptive content engine itself (`../adaptive/` AC.*) — consumed, not rebuilt.
- Content Tools (`../content_tools/` CT.*) — rendered inside these surfaces via the
  existing manifest/renderer contract.
- Quiz *authoring*; quiz *taking* is in scope.
- Gradebook — that is [UX.11](UX.11-data-table-and-gradebook-system.md).

## 4. Personas & User Stories

- **As a student**, I want to open a course and immediately continue where I left
  off, without hunting through modules.
- **As a student**, I want to see how far through a module I am and what remains.
- **As a student**, I want to finish an item and go straight to the next one
  without returning to a list.
- **As a student reading a long content page**, I want a comfortable line length,
  a progress indicator, and my position remembered.
- **As a student**, I want to know *why* something is locked and what unlocks it.
- **As a student taking a quiz**, I want a calm, distraction-free surface with
  unambiguous save state.
- **As an instructor**, I want course home to show what needs my attention, not
  the student view.
- **As an instructor**, I want to see at a glance where students are getting stuck.

## 5. Functional Requirements

### Course home

- **FR-1.** Course home MUST be role-aware, using the [UX.9](UX.9-role-aware-dashboard.md)
  widget contract scoped to one course.
- **FR-2.** For students, the primary region MUST be **"Continue"** — the resume
  point (last visited item, or the next incomplete item), with module context and
  due date.
- **FR-3.** For instructors, the primary region MUST be **"Needs attention"** —
  ungraded submissions, at-risk students, unanswered discussion posts, and
  outstanding checklist items.
- **FR-4.** Course home MUST show a **course-level progress summary** for students:
  items complete / total, current grade if released, and upcoming due dates.

### Module list

- **FR-5.** The module list MUST convey **item state at a glance** — not started,
  in progress, complete, overdue, locked, graded — using the shared status
  vocabulary with icon + text, never colour alone.
- **FR-6.** The **current/next item MUST be visually distinguished** from all
  others; the list must not present 40 equal rows.
- **FR-7.** Completed modules MUST collapse by default (**R-9**), with an
  indicator, and MUST remember expansion state per user.
- **FR-8.** Locked items MUST state the unlock condition in plain language
  ("Available after you complete *Quiz 2*" / "Opens 14 March"), never just a
  padlock.
- **FR-9.** The list MUST support keyboard navigation and single-pointer reordering
  for instructors ([UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) FR-5/FR-6).
- **FR-10.** Module and item counts, durations and point values MUST be shown
  consistently so learners can plan.

### Item flow

- **FR-11.** Every module item page MUST render a consistent **item shell**:
  breadcrumb (course → module → item), title, state, and **previous/next**
  navigation.
- **FR-12.** Completing an item MUST offer the **next item immediately** — no
  dead ends, no forced return to the list.
- **FR-13.** Item type differences (content page, assignment, quiz, external link,
  LTI, SCORM, H5P, textbook resource, vibe activity) MUST share the shell and
  differ only in body.
- **FR-14.** Progress within a module MUST be persistently visible during an item.

### Reading experience

- **FR-15.** Content pages MUST constrain measure to 65ch
  ([UX.3](UX.3-typography-and-reading-system.md) FR-4) and use the `body-lg` role.
- **FR-16.** Long content MUST show a **reading-progress indicator** and remember
  scroll position across sessions.
- **FR-17.** The existing reading focus mode MUST be discoverable from any content
  page, not only via the top bar.
- **FR-18.** Annotation, highlight and note-taking (already shipped) MUST be
  available consistently on every reading surface, with a single entry point.
- **FR-19.** Embedded media (video, audio, PDF, H5P) MUST have consistent controls,
  captions on by default where available, and MUST not break the reading column.

### Quiz taking

- **FR-20.** Quiz taking MUST use a focused shell (the existing
  `quiz-focus-top-bar` pattern), suppressing non-essential chrome.
- **FR-21.** Save state MUST be **explicit and continuous** ("All answers saved"
  / "Saving…" / "Not saved — retrying"), never ambiguous.
- **FR-22.** Time remaining, question progress and flagged questions MUST be
  persistently visible.
- **FR-23.** Connection loss MUST NOT lose answers; answers MUST persist locally
  (Dexie is already available) and sync on reconnect.

### Decomposition

- **FR-24.** `course-modules.tsx`, `course-module-quiz-page.tsx` and
  `quiz-student-take-panel.tsx` MUST be decomposed so that no resulting component
  exceeds 300 lines or 10 `useState` hooks, per
  [`TD.14`](../tech_debt/TD.14-decompose-god-components.md).

## 6. Non-Functional Requirements

- **Performance** — Course home LCP ≤2.0 s p75; module list renders 200 items
  without jank (virtualise beyond 100); item navigation INP ≤200 ms; quiz answer
  save ≤500 ms p95 and never blocks input.
- **Security** — Locked-item reasons MUST NOT leak content or answers. Client-side
  progress is advisory; server remains authoritative for completion, unlocking and
  grading. Local answer persistence MUST be cleared on submit and on sign-out.
- **Privacy & Compliance** — Reading position and time-on-page are behavioural
  data; covered by the RoPA and the existing research-consent framework. Instructor
  "where students get stuck" views MUST respect existing at-risk governance.
- **Accessibility** — Item shell provides consistent landmarks and heading
  structure; progress indicators are text + visual; quiz timer announcements are
  polite and non-disruptive; focus mode does not trap focus; reading position
  restore MUST not steal focus.
- **Scalability** — Item shell must accommodate new item types (Content Tools adds
  them continuously) without shell changes.
- **Reliability** — Offline: previously-visited content readable; quiz answers
  queued; submission retried. Never silently lose learner work.
- **Observability** — Emit `item_start`, `item_complete`, `item_abandon`,
  `next_item_click`, `resume_click`, `reading_progress`, `quiz_answer_saved`,
  `quiz_save_failed`. `item_abandon` by item type is the primary signal of a
  broken surface.
- **Maintainability** — FR-24 decomposition; one file per item type body.
- **Internationalization** — All copy from i18n keys; RTL reading column;
  locale-aware durations and dates (the existing `DeadlineDateTime` and
  `locale-time` components are reused).
- **Backward compatibility** — All routes preserved. Deep links from the CC
  checklist, notifications, calendar and native clients must continue to work.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student opens a course, *When* it loads, *Then* a
  "Continue" action naming a specific item is visible without scrolling.
- **AC-2.** *Given* an instructor opens a course, *When* it loads, *Then* the
  primary region shows items needing attention, not student progress widgets.
- **AC-3.** *Given* a module list with mixed item states, *When* rendered, *Then*
  each state is distinguishable by icon **and** text, and the next item is
  visually dominant.
- **AC-4.** *Given* a locked item, *When* viewed, *Then* the unlock condition is
  stated in plain language.
- **AC-5.** *Given* a student completes an item, *When* it is marked complete,
  *Then* the next item is offered in the same view.
- **AC-6.** *Given* a student returns to a content page, *When* it loads, *Then*
  their previous scroll position is restored without stealing focus.
- **AC-7.** *Given* a content page at 1,920px, *When* measured, *Then* the prose
  column is 45–75ch.
- **AC-8.** *Given* a student taking a quiz, *When* they answer, *Then* save state
  is explicitly shown and never ambiguous.
- **AC-9.** *Given* a network drop mid-quiz, *When* answers are entered, *Then*
  they persist locally and sync on reconnect with no data loss.
- **AC-10.** *Given* a module with 200 items, *When* rendered, *Then* scrolling is
  smooth (no long task >50 ms) and INP ≤200 ms.
- **AC-11.** *Given* the decomposition, *When* measured, *Then* no component in
  the learning flow exceeds 300 lines or 10 `useState` hooks.
- **AC-12.** *Given* the learning-flow routes, *When* axe runs in all four themes,
  *Then* 0 violations; *And* a screen-reader user can complete a full item →
  next-item cycle.
- **AC-13.** *Given* moderated testing with ≥10 students, *When* asked to resume
  and complete an item, *Then* ≥90% succeed without assistance and report the flow
  as clear.
- **AC-14.** *Given* an offline learner with previously-visited content, *When*
  they open it, *Then* it renders from cache with a clear offline indicator.

## 8. Data Model

```sql
-- server/migrations/NNN_learner_item_position.sql
-- Resume position and reading progress (FR-16). Completion already exists.
CREATE TABLE learner_item_positions (
  enrollment_id  uuid        NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
  item_id        uuid        NOT NULL REFERENCES module_items(id) ON DELETE CASCADE,
  scroll_ratio   real        NOT NULL DEFAULT 0,   -- 0..1
  last_seen_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (enrollment_id, item_id),
  CONSTRAINT learner_item_positions_ratio_chk CHECK (scroll_ratio BETWEEN 0 AND 1)
);
CREATE INDEX learner_item_positions_recent_idx
  ON learner_item_positions (enrollment_id, last_seen_at DESC);

-- Module expansion state (FR-7)
ALTER TABLE enrollments
  ADD COLUMN module_ui_state jsonb NOT NULL DEFAULT '{}'::jsonb;
```

- The existing `last-visited-module-item` client utility is superseded by
  server-side positions so resume works across devices.
- **Backfill** — none; absent row means "start at the top".
- Cascade deletes satisfy `../standards/S02-data-retention-deletion-engine.md`.

## 9. API Surface

```ts
// GET /api/v1/courses/{code}/home?audience=student     (auth: enrolled)
type CourseHome = {
  audience: 'student' | 'instructor'
  primary: ContinueAction | NeedsAttention | null
  progress: { itemsComplete: number; itemsTotal: number; gradePercent: number | null }
  widgets: Record<string, WidgetPayload | { error: string }>
}

// GET  /api/v1/courses/{code}/items/{itemId}/position   (auth: enrolled self)
// PUT  /api/v1/courses/{code}/items/{itemId}/position   (auth: enrolled self)
type ItemPosition = { scrollRatio: number; lastSeenAt: string }

// GET /api/v1/courses/{code}/items/{itemId}/next        (auth: enrolled)
type NextItem = { itemId: string; title: string; kind: string; href: string } | null

// Locked-item reason is added to the existing structure payload:
type LockReason =
  | { kind: 'prerequisite'; itemId: string; itemTitle: string }
  | { kind: 'date'; availableAt: string }
  | { kind: 'requirement'; description: string }
```

- Position `PUT` MUST be throttled client-side (≤1 write / 5 s) and MUST be
  fire-and-forget — never blocking scroll.
- Quiz answer autosave uses the **existing** quiz endpoints; this plan changes only
  how state is displayed and locally queued.
- **OpenAPI** — all new routes documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none; existing routes are redesigned.
- **Modified pages** — `course-detail.tsx` (course home), `course-modules.tsx`,
  all nine `course-module-*-page.tsx` item types, `quiz-student-take-panel.tsx`,
  `course-syllabus.tsx`.
- **Key user flows**
  1. Open course → "Continue: *Module 4 — Cell Division*" → item opens at last
     position → complete → "Next: *Quiz 4*" → straight in.
  2. Browse modules → completed modules collapsed → next item highlighted →
     locked item explains itself.
  3. Read a long page → progress bar fills → leave → return → resume exactly.
  4. Take a quiz → focused shell → explicit save state → connection drops →
     "Not saved — retrying" → reconnect → "All answers saved".
- **States** — module list: loading (skeleton matching row count), empty (no
  modules yet → instructor sees "Add your first module", student sees "Your
  instructor hasn't published content yet"), error (retry), offline (cached with
  indicator). Item: loading, locked, unavailable, complete, overdue, submitted,
  graded.
- **Mobile/responsive** — item shell is single-column; prev/next become a sticky
  bottom bar; module list rows are ≥44px targets; quiz shell is full-bleed.
- **Accessibility annotations** — item shell: `nav` for prev/next, `main` for
  body, `h1` for item title; progress uses `role="progressbar"` with text
  equivalent; quiz timer announces at intervals (polite), not continuously;
  focus mode preserves normal tab order; scroll restore uses
  `scrollTo` without focus change.
- **Copy & i18n** — new keys under `course.*` and `learning.*`, at parity across
  four locales. Lock reasons and save states are the highest-stakes copy in this
  plan — they must be specific, calm and actionable.

## 11. AI / ML Considerations

Course home's "Continue" consumes the existing recommendation and adaptive-content
engines (`../adaptive/` AC.*, `fetchLearnerRecommendations`).

- **Model** — existing; none introduced.
- **Prompts** — n/a for this plan; adaptive rewriting is owned by AC.*.
- **Eval metric** — resume click-through rate and `item_complete` rate following a
  "Continue" click, vs a simple last-visited baseline.
- **Fallback path** — if recommendations are unavailable, "Continue" MUST fall back
  to last-visited, then to first-incomplete. The learner must never see a broken
  primary action.
- **Explainability** — where an item is adaptively selected or rewritten, the
  existing rationale chip and AC fidelity indicators MUST be shown. A learner must
  always be able to see the standard version.
- **PII redaction / cost** — no new inference at render; positions and next-item
  are deterministic queries.

## 12. Integration Points

- **External** — LTI, SCORM, H5P players (wrapped in the item shell, not
  rewritten); `hls.js`, `pdfjs-dist`, `katex` for media/math rendering.
- **Internal**
  - `clients/web/src/pages/lms/course-detail.tsx`, `course-modules.tsx`,
    `course-module-*-page.tsx` (9 files), `course-layout.tsx`
  - `clients/web/src/components/quiz/**`, `components/modules/**`,
    `components/content-page/**`, `components/annotation/**`,
    `components/media/**`, `components/video-player/**`, `components/markdown/**`
  - `clients/web/src/components/layout/{quiz-focus-top-bar,reading-focus-top-bar}.tsx`
  - `clients/web/src/lib/last-visited-module-item.ts` — superseded
  - `clients/web/src/db/` (Dexie) — offline answer queue
  - `clients/web/src/components/content-tools/**` — CT tools render in the item body
  - `clients/web/src/components/lms/adaptive-content/**` — AC integration
  - `server/internal/httpserver` — home, position, next-item routes
- **Events** — learning telemetry into `server/internal/telemetry`; xAPI emission
  where enabled.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md),
  [UX.2](../../completed/ui-ux/UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.3](UX.3-typography-and-reading-system.md) (the reading experience *is* the
  type system), [UX.7](UX.7-navigation-information-architecture.md),
  [UX.12](UX.12-loading-empty-error-offline-states.md), and
  [`TD.14`](../tech_debt/TD.14-decompose-god-components.md) — **TD.14 is a hard
  prerequisite**; these components cannot be safely changed as they stand.
- **Must ship before** — [UX.16](UX.16-progress-motivation-and-learner-agency.md).
- **Coordinates with** — `../adaptive/` (AC.7–AC.9), `../content_tools/` (CT.*).
- **Shared infra** — offline sync (Workbox + Dexie, already present); student
  participant recruitment.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Rewriting 3,284- and 3,403-line components introduces regressions in the highest-stakes flow (quiz taking, submission) | **H** | **H** | TD.14 first; characterisation tests captured **before** any change; feature-flagged per surface; quiz taking migrated last and with the most testing |
| Losing learner work during the quiz migration | L | **Critical** | Local persistence (FR-23) shipped and proven *before* any quiz UI change; explicit save state; E2E network-partition tests; no flag rollout beyond 10% until zero loss events for 30 days |
| Scope explosion — this touches most of the learner surface | **H** | **H** | Ship in four independently-valuable slices: (1) item shell + prev/next, (2) module list states, (3) course home, (4) quiz shell. Each is releasable alone |
| Resume position feels intrusive or lands users mid-sentence | M | M | Restore to the start of the nearest block, not the exact pixel; offer "back to top"; never move focus |
| Collapsing completed modules hides content students want | M | M | State persists per user; a single click expands; test in moderated sessions |
| Content Tools and adaptive content assume the current DOM | M | M | Item shell defines a stable body contract; CT/AC teams review the contract before implementation |

## 15. Rollout Plan

- **Feature flag** — one flag per slice: `ffItemShell`, `ffModuleListV2`,
  `ffCourseHomeV2`, `ffQuizShellV2`. All default off.
- **Sequencing**
  1. **TD.14 decomposition** of the three god components, with characterisation
     tests. No visible change.
  2. Item shell + prev/next + resume position (`ffItemShell`).
  3. Module list states, collapse, lock reasons (`ffModuleListV2`).
  4. Course home role-aware primary region (`ffCourseHomeV2`).
  5. Reading experience: measure, progress, annotation entry point.
  6. Quiz shell + explicit save state + offline queue (`ffQuizShellV2`) —
     **last**, with the longest bake.
  7. Internal → 10% → 50% → GA per slice independently.
- **Dogfood** — internal org plus a volunteer pilot course per slice; quiz slice
  requires a live pilot course with instructor consent.
- **GA criteria** — AC-1…AC-14 green; AC-13 ≥90% task success; zero learner-work
  loss events; `item_abandon` rate flat or improved per item type.
- **Rollback** — per-slice flag off. Local answer persistence is additive and
  stays on regardless.

## 16. Test Plan

- **Unit** — item state derivation (not started / in progress / complete /
  overdue / locked / graded); next-item resolution including conditional release;
  lock-reason formatting per kind; scroll-ratio throttling; offline answer queue
  enqueue/dedupe/flush; module collapse state.
- **Integration** — course home authz per audience; position read/write scoped to
  the caller's enrollment; next-item respects prerequisites and differentiated
  assignments; quiz autosave against the existing endpoints.
- **End-to-end** — Playwright: full learner journey (open → continue → read →
  complete → next → quiz → submit → see feedback); network-partition mid-quiz with
  answer recovery; resume across sessions; locked item messaging; instructor course
  home.
- **Security** — lock reasons do not leak content; position writes cannot target
  another enrollment; local answer store cleared on sign-out; offline cache does
  not retain content after unenrollment.
- **Accessibility** — axe on all learning-flow routes × 4 themes (AC-12);
  screen-reader scripts: complete an item and advance; hear quiz progress and save
  state; understand why an item is locked; keyboard-only quiz completion;
  reduced-motion honoured.
- **Performance / load** — 200-item module list (AC-10); Lighthouse CI on course
  home and a content page; quiz save p95; long-task profiling during quiz typing.
- **User research** — 10+ moderated student sessions (AC-13); 5 instructor
  sessions on course home; diary study on resume behaviour over two weeks.
- **Manual exploratory** — QA matrix of 9 item types × states × offline/online ×
  student/instructor; K-2 and elementary UI modes; RTL.

## 17. Documentation & Training

- **End-user** — help-centre: "Working through a course", "Reading and annotating",
  "Taking a quiz (including what happens if you lose connection)".
- **Admin / instructor** — "Designing a module students can follow", cross-linked
  from the CC course checklist; explain what students see for locked items.
- **Engineer** — `docs/guides/item-shell.md`: the shell contract, how to add an
  item type, the body contract that Content Tools and adaptive content rely on,
  the offline answer queue.
- **API reference** — OpenAPI for home/position/next-item.
- **Runbook** — "A student reports lost quiz answers": how to inspect the local
  queue and server-side autosave history.

## 18. Open Questions

1. Should "Continue" prefer the **adaptive recommendation** or **last visited**
   when they disagree? *Recommendation: last-visited if within 7 days, else
   recommendation — validate in the diary study.*
2. Do completed modules collapse by default for **all** learners, or only past a
   threshold (e.g. >5 modules)? Test.
3. Is the quiz focus shell mandatory or instructor-configurable per quiz? Some
   institutions require lockdown-like behaviour; coordinate with the integrity
   plans in `../../completed/03-submissions-grading-integrity/`.
4. How long does offline content remain cached after unenrollment, and does that
   need a retention entry?
5. Does the item shell replace or wrap the existing LTI/SCORM/H5P containers?
   *Recommendation: wrap — those players have their own lifecycle.*
6. Should reading position sync to native clients in this cycle?

## 19. References

- Existing files: `clients/web/src/pages/lms/course-modules.tsx` (3,284 lines,
  100 `useState`), `course-module-quiz-page.tsx` (3,403 lines),
  `course-detail.tsx`, `course-layout.tsx`, the nine `course-module-*-page.tsx`
  files, `clients/web/src/components/quiz/quiz-student-take-panel.tsx` (2,196
  lines), `clients/web/src/components/layout/quiz-focus-top-bar.tsx`,
  `reading-focus-top-bar.tsx`, `clients/web/src/lib/last-visited-module-item.ts`,
  `clients/web/src/db/`
- Research: [research.md](research.md) R-1, R-2, R-3, R-4, R-5, R-9, R-16, R-18,
  R-19, R-23
- Audit: [audit.md](audit.md) G-15, G-9, G-7, §1
- Related plans: [UX.3](UX.3-typography-and-reading-system.md),
  [UX.12](UX.12-loading-empty-error-offline-states.md),
  [UX.16](UX.16-progress-motivation-and-learner-agency.md),
  [`../tech_debt/TD.14-decompose-god-components.md`](../tech_debt/TD.14-decompose-god-components.md),
  `../adaptive/`, `../content_tools/`, `../../completed/07-mobile-offline-cross-platform/`

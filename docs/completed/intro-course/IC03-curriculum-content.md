# IC03 — Curriculum & Content: The "Welcome to Lextures" Course

> Implementation plan. Source: product direction — *"cover the features, learning patterns, the
> learner profile, the mobile application, importing courses from Canvas, and the rest of the
> relevant features."* Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC03 |
| **Section** | Intro Course |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Product/content + backend platform |
| **Depends on** | IC01 (course + system instructor) |
| **Unblocks** | IC04 (graded items attach to this structure), IC05, IC06, IC07 |

---

## 1. Problem Statement

The intro course shell (IC01) has no teaching in it. This plan authors the actual curriculum —
the modules, content pages, and per-module knowledge checks that teach a new user what Lextures
is and how to succeed on it: the core features, the platform's learning patterns, the autonomous
learner profile, the mobile app, and importing existing courses from Canvas. Content is written
as **versioned markdown fixtures in the repo** and synced idempotently into the course, so it is
code-reviewed, localizable (IC08), and safe to redeploy without clobbering learner data.

## 2. Goals

- Ship a **7-module curriculum** (below) covering features, learning patterns, learner profile,
  mobile, Canvas import, and grades — pitched to a first-time user of any market.
- Author content as **repo-versioned markdown fixtures** synced idempotently into
  `course.module_content_pages` / `course.module_quizzes` (assignments in IC04), keyed by stable
  item slugs so edits update in place and reordering is deterministic.
- Make it **short and completable** (~30–45 min): concise pages, one knowledge-check quiz per
  module, "try it" links that deep-link into the real product surface being taught.
- Keep content **market-neutral** and **feature-flag-aware**: sections about flagged features
  (mobile, Canvas import, learner profile, AI) render conditionally so the course never teaches a
  feature the deployment has disabled.
- Generate genuine behavioural signal (page views, quiz attempts) so a new account's
  [learner profile](../learner-profile/README.md) and adaptive engines have data on day one.

## 3. Non-Goals

- Grading logic and the gradebook wiring (IC04) — this plan defines *where* graded items sit,
  IC04 defines *how* they score.
- Completion detection and the certificate (IC05).
- Discovery/entry-point UI (IC06) and mobile rendering specifics (IC07).
- Translation of the content into other languages (IC08 owns the i18n pipeline; this plan writes
  the English source and externalizes strings).
- Video production (v1 uses text + images + deep links; video is an IC08/future enhancement).

## 4. Personas & User Stories

- **As a brand-new student**, I want a short guided tour that shows me exactly where things are
  and lets me try them, so I'm productive on day one.
- **As a self-learner**, I want to understand how Lextures adapts to me (learning patterns +
  learner profile), so I trust and use the adaptive features.
- **As a K-12/HE instructor evaluating the platform** (also auto-enrolled), I want to see the
  feature set demonstrated inside a real course, so I can judge fit quickly.
- **As a user migrating from Canvas**, I want a clear walkthrough of importing my courses, so I
  don't abandon during migration.
- **As a content maintainer**, I want the curriculum in version control, so changes are reviewed
  and shipped like code, not lost on the next sync.

## 5. Functional Requirements

- **FR-1.** The course MUST contain the 7 modules in §10, each `published`, ordered, with a
  stable `item_slug` per structure item so the sync is idempotent (update-in-place, not
  recreate).
- **FR-2.** Content pages MUST be authored as markdown fixtures under
  `server/internal/service/introcourse/content/<locale>/<module>/<page>.md` with front-matter
  (`slug`, `title`, `sort_order`, optional `requires_flag`). The sync MUST upsert them into
  `course.course_structure_items` + `course.module_content_pages`.
- **FR-3.** Each module MUST end with one short **knowledge-check quiz** (3–5 questions,
  auto-scored — IC04) authored as a fixture into `course.module_quizzes.questions_json`.
- **FR-4.** Pages MAY contain **deep links** into the live product ("Open your dashboard", "Go to
  the mobile download page", "Start a Canvas import") using relative in-app routes so the tour is
  interactive, not just descriptive.
- **FR-5.** Any page/quiz/module whose front-matter names a `requires_flag` MUST be **omitted**
  from the synced course when that platform flag is off (e.g. the Canvas-import module hidden when
  Canvas import is unavailable; the learner-profile module hidden when `learner_profile_enabled`
  is off; AI pages hidden when AI features are off).
- **FR-6.** The sync MUST be **idempotent and non-destructive to learner data**: it may
  update/add/remove *content* items but MUST NOT delete a student's grades/submissions for items
  that persist; removing an item MUST be a soft archive (`archived`) not a hard delete, to avoid
  cascading away `course_grades`.
- **FR-7.** All human-readable strings MUST be externalized to i18n keys (IC08 resolves
  translations); the English fixture is the source of truth.
- **FR-8.** The sync MUST run as part of IC01's `EnsureProvisioned` / admin re-sync, versioned by
  a `content_version` so an unchanged version is a fast no-op.

## 6. Non-Functional Requirements

- **Performance** — Content sync completes in < 2 s for the full curriculum; no-op (version
  unchanged) in < 50 ms. Rendered pages reuse existing content-page rendering (no new hot path).
- **Security** — Content is public-within-course; markdown MUST pass the existing sanitizer (no
  raw HTML/script injection). Deep links MUST be same-origin relative routes only.
- **Privacy & Compliance** — No PII in content. AI-feature pages MUST carry the standard AI
  disclosure copy when `ai_disclosure_enabled`. Content that describes data use (learner profile
  module) MUST link to the Privacy Center, not restate policy incorrectly.
- **Accessibility** — All pages WCAG 2.1 AA: heading hierarchy, alt text on every image
  (enforced if `alt_text_enforcement`), no color-only meaning, readable at 200% zoom. Quizzes
  keyboard-operable. Target reading level ≈ grade 8 for market-neutrality.
- **Scalability** — Content is static per deploy; shared by all learners; no per-user copies.
- **Reliability** — Sync is transactional per module; a bad fixture fails that module's sync
  without corrupting others; validation runs in CI before deploy.
- **Observability** — Emit `intro_course_content_sync_total{result}`, `..._duration_seconds`,
  and a `intro_course_content_version` info gauge. Page-view + quiz-attempt events flow through
  existing engagement instrumentation (feeds LP).
- **Maintainability** — Fixtures are plain markdown + JSON; a `make intro-course-validate` lint
  checks front-matter, slugs uniqueness, flag references, link validity, and quiz schema.
- **Internationalization** — Locale-partitioned fixture dirs; English is default; missing-locale
  falls back to English (IC08).
- **Backward compatibility** — Additive; edits are versioned and update-in-place.

## 7. Acceptance Criteria

- **AC-1.** *Given* a provisioned course, *when* content sync runs, *then* the 7 modules and
  their pages/quizzes exist with correct order and titles, each `published`.
- **AC-2.** *Given* an edit to a page fixture and a bumped `content_version`, *when* sync re-runs,
  *then* that page's markdown is updated in place (same `structure_item_id`), and no student
  grade/submission for other items is affected.
- **AC-3.** *Given* `learner_profile_enabled=false`, *when* sync runs, *then* the "Your Learner
  Profile" module is omitted (or archived) and no broken links reference it.
- **AC-4.** *Given* the Canvas-import feature is unavailable, *when* sync runs, *then* the
  "Bringing Your Courses In" module is omitted.
- **AC-5.** *Given* a deep link on a page (e.g. "Open the mobile download page"), *when* a
  logged-in student clicks it, *then* they navigate to the live in-app surface.
- **AC-6.** *Given* a malformed fixture (bad front-matter/quiz schema), *when* CI validation runs,
  *then* the build fails with a clear error and the bad content never deploys.
- **AC-7.** *Given* a student reads pages and takes a module quiz, *then* engagement events and a
  quiz attempt are recorded (visible to the LP engine).

## 8. Data Model

No new schema — reuses IC01's course and existing content tables:
`course.course_structure_items` (kinds `module`/`heading`/`content_page`/`quiz`/`assignment`),
`course.module_content_pages`, `course.module_quizzes`. IC04 adds `assignment` bodies.

New mapping needed: a stable **item slug → structure_item_id** map so sync is idempotent. Add a
nullable `content_slug` column scoped to the intro course (or a side table
`settings.intro_course_items(slug PK, structure_item_id, content_version)`), so re-sync locates
the row to update:

```sql
-- server/migrations/372_intro_course_items.sql  (renumber on merge)
CREATE TABLE settings.intro_course_items (
    slug             TEXT PRIMARY KEY,           -- e.g. 'm3.learning-patterns.spaced-practice'
    structure_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    content_version  INTEGER NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Backfill: none (populated on first sync).

## 9. API Surface

No new endpoints — content is read through existing course/module/page/quiz APIs. Sync is invoked
by IC01's `EnsureProvisioned` and the IC08 admin re-sync endpoint. Internal:

```go
// server/internal/service/introcourse/content_sync.go
func SyncContent(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, cfg config.Config) (Report, error)
//go:embed content/**   // fixtures embedded so deploys are self-contained
```

Fixtures embedded via `embed.FS` so no runtime filesystem dependency.

## 10. UI / UX — the curriculum

The course renders through the **existing** course/module/content-page/quiz UI; no new components.
The 7 modules (each: 1–3 content pages + a 3–5 Q knowledge-check quiz; graded items per IC04):

1. **Welcome & Getting Oriented** — What Lextures is; the dashboard tour; how a course is
   structured (modules, pages, assignments, quizzes); navigation & settings. *Try it: open your
   dashboard.* (Always shown.)
2. **Core Features You'll Use Every Day** — Assignments & submissions, quizzes, the notebook,
   the course feed & announcements, calendar & due dates, messaging/inbox, files. Deep links to
   each. (Sections conditioned on the relevant flags.)
3. **How Learning Works on Lextures (Learning Patterns)** — The platform's pedagogy: mastery &
   the learner model, adaptive learning paths, spaced practice / spaced repetition, recommended
   next steps, diagnostics, self-reflection & coaching. Explains *why* the platform nudges what it
   nudges. (Pages conditioned on `adaptive_learner_model_enabled`, `srs_practice_enabled`,
   `diagnostic_assessments_enabled`, `self_reflection_enabled`.)
4. **Your Learner Profile** — What the autonomous [learner profile](../learner-profile/README.md)
   is, that it's derived (never typed in), how to read its facets and provenance, and the privacy
   controls (view/export/pause/reset) at *Settings → Learner Profile*. *Try it: open your Learner
   Profile.* (Whole module `requires_flag: learner_profile_enabled`.)
5. **Lextures on the Go (Mobile App)** — Installing iOS/Android, signing in, what works offline,
   push notifications, and mobile-specific gestures. *Try it: open the mobile download page / scan
   the QR.* (`requires_flag: push_notifications_enabled` gates the notifications page; module shown
   generally.)
6. **Bringing Your Courses In (Importing from Canvas)** — Why/when to import, what transfers
   (modules, pages, assignments, quizzes, rubrics, announcements, submissions/grades), starting an
   import, monitoring progress, and verifying results. *Try it: start a Canvas import.* (Whole
   module `requires_flag` on Canvas import availability.)
7. **Grades, Assignments & Finishing Up** — How grades and the gradebook work, where to see your
   grades, what "auto-graded" means here, a short capstone reflection assignment (IC04), and a
   completion recap (IC05). *Try it: open your gradebook.* (Always shown.)

Empty/loading/error states are the existing course UI's. IC06 adds the *entry points* to this
course; IC07 verifies mobile rendering.

## 11. AI / ML Considerations

The course *describes* AI features (adaptive paths, recommendations, AI study buddy, grader
agent) but this plan performs **no** model calls. AI-describing pages MUST include the standard AI
disclosure when `ai_disclosure_enabled`. (Optional future: an LLM-authored per-market variant —
out of scope; content stays human-authored + reviewed for v1.)

## 12. Integration Points

- **Content tables / rendering:** `course.module_content_pages`, `course.module_quizzes`,
  `course.course_structure_items`; existing web/mobile content-page + quiz renderers.
- **Sync:** IC01 `introcourse` service (`content_sync.go`, `embed.FS`).
- **Flags for conditional content:** `learner_profile_enabled`, Canvas-import availability,
  `push_notifications_enabled`, adaptive/SRS/diagnostic/self-reflection flags, `ai_disclosure_enabled`,
  `alt_text_enforcement`.
- **Sanitizer:** existing markdown sanitizer used by content pages.
- **Engagement instrumentation:** existing page-view/quiz-attempt events (feeds LP02/LP03).
- **CI:** `make intro-course-validate` lint in the build pipeline.

## 13. Dependencies & Sequencing

- **After:** IC01 (course + structure to attach to).
- **Before:** IC04 (grades attach to these items), IC05 (completion over these items), IC06/IC07
  (surfaces render this content).
- **Shared infra:** markdown rendering, quiz engine, CI — present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Content teaches a disabled feature (broken/confusing) | M | M | `requires_flag` front-matter + omit/archive on sync (FR-5); link-validity lint |
| Sync hard-deletes an item and cascades away student grades | L | H | Soft-archive on removal (FR-6); never `DELETE` structure items with grades |
| Content drifts from the real UI over releases | M | M | Deep links use canonical routes; screenshots dated; content-review checklist per release |
| Fixtures edited in prod DB and lost on next sync | M | M | Content is code-owned; admin edits documented as unsupported; re-sync warns |
| Too long → users abandon | M | M | Hard cap ~30–45 min; concise pages; one quiz/module; optional modules gated |

## 15. Rollout Plan

- **Flag:** gated by IC01's `intro_course_enabled`; individual modules gated by their feature
  flags.
- **Sequencing:** author English fixtures → CI validation green → sync on staging → content review
  (product + a11y + market reviewers) → enable in prod.
- **Dogfood:** internal reviewers walk the full course on web + mobile; a11y audit (axe + screen
  reader) on every page.
- **GA criteria:** all 7 modules pass content + a11y review; conditional gating verified for each
  optional module; completable in target time.
- **Rollback:** revert fixture version + re-sync (content reverts in place); or disable
  `intro_course_enabled`.

## 16. Test Plan

- **Unit** — front-matter parser; flag-conditioned inclusion; slug uniqueness; version no-op.
- **Integration (DB)** — sync creates the 7 modules idempotently; edit+version-bump updates in
  place; disabling a flag archives its module without touching other grades.
- **End-to-end** — student reads pages, follows a deep link into the live surface, takes a module
  quiz; engagement + attempt recorded.
- **Security** — sanitizer rejects script in a fixture; deep links are same-origin only.
- **Accessibility** — automated axe on every page + manual screen-reader script for one full
  module; images have alt text; quizzes keyboard-navigable.
- **Content/lint** — `make intro-course-validate` catches bad front-matter, dead links, bad quiz
  schema in CI.
- **Manual exploratory** — toggle each optional module's flag; confirm graceful omission and no
  dangling links.

## 17. Documentation & Training

- Content-authoring guide: `docs/guides/intro-course-content.md` — fixture format, front-matter,
  flags, deep-link routes, the validation lint, how to add/edit a module.
- Release checklist item: "review intro-course content against UI changes this release."
- Help-center article mirroring the course for search.

## 18. Open Questions

1. Text-only v1, or include short screen-capture videos? (Leaning text + images + deep links for
   v1; video in IC08/backlog.)
2. Should optional modules be **omitted** or **shown as locked with an explanation** when their
   flag is off? (Leaning omit, to avoid teasing unavailable features.)
3. How market-specific should copy be (K-12 vs HE vs self-learner tone)? (v1 one neutral voice;
   IC08 could add per-market variants via the locale/variant mechanism.)
4. Do we deep-link into features that require additional setup (e.g. Canvas import needs
   credentials), and how do we handle the not-configured state gracefully?

## 19. References

- Existing files: `server/migrations/021_module_content_pages.sql`,
  `.../033_module_quizzes.sql`, `.../014_course_structure.sql`,
  `server/internal/httpserver/canvas_import_ws.go` (the subject of module 6),
  learner-profile settings UI ([LP07](../learner-profile/LP07-settings-page-transparency-ui.md)).
- Related plans: [IC01](IC01-foundation-provisioning-flag.md),
  [IC04](IC04-graded-assessments-autograding.md), [IC08](IC08-admin-governance-localization.md),
  [learner-profile epic](../learner-profile/README.md).

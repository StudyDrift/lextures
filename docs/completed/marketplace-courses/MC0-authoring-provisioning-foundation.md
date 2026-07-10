# MC0 — Official-Course Authoring & Provisioning Foundation

> Implementation plan. Source: [docs/plan/marketplace-courses/README.md](../../plan/marketplace-courses/README.md). Generalizes the shipped intro-course harness (`server/internal/service/introcourse`) to publish first-party, free marketplace courses.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC0 |
| **Section** | Marketplace Courses |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform team |
| **Depends on** | MKT4 (free-claim flow, shipped); introcourse harness (shipped) |
| **Unblocks** | MC1 (AI Essentials), MC2 (Introduction to Python), MC3 (Personal Finance) |

---

## 1. Problem Statement

The in-app course marketplace (MKT1–MKT5) shipped with the *mechanics* to list, discover, and claim courses, but with **no first-party inventory**. An empty storefront reads as a broken feature and gives instructors no reference for what a "good" listing looks like. We need a repeatable way to author, review, localize, and **idempotently provision** official courses — the way the intro course is provisioned — so the platform can ship (and maintain) a small set of high-quality free courses without hand-editing production data. This story builds that harness; MC1–MC3 are its first payloads.

## 2. Goals

- A **content-as-code** authoring format for a full course (metadata, syllabus, modules, pages, assignments, quizzes) that a subject-matter expert can edit in the repo and a reviewer can diff.
- An **idempotent provisioner** (`EnsureProvisioned`) + CLI that creates/reconciles a course and its structure items from embedded content, safe to re-run on every deploy.
- Provisioned courses land **published, free (`price_cents = 0`), and `marketplace_listed = true`**, discoverable via MKT3 and claimable via MKT4's free path.
- A **validation CLI + CI gate** that fails on malformed content, broken internal references, unresolved quiz answers, or **dead external links**.
- **Localization-ready** (`content/<locale>/…`) with English as the source of truth and graceful fallback, matching introcourse.

## 3. Non-Goals

- No new commerce, pricing, checkout, or entitlement code — MKT4 already covers free claims.
- No in-app WYSIWYG course builder for these courses (they are code-authored on purpose). The existing instructor authoring UI is untouched.
- No auto-enrollment. Unlike the intro course, official marketplace courses are **opt-in** (learner claims them).
- No certificates/credentials issuance in this story (tracked separately; see §18).
- Not a general CMS — the harness targets a curated handful of first-party courses, not arbitrary bulk import.

## 4. Personas & User Stories

- **As a content author (SME)**, I want to write a course as reviewable Markdown/JSON in the repo so that changes go through normal code review and CI.
- **As a platform engineer**, I want a re-runnable `provision-marketplace-courses` command so that deploys converge the catalog to the checked-in content without manual DB edits.
- **As a learner**, I want these official courses to appear in the marketplace and claim in one click (free) so that I can start immediately.
- **As an admin**, I want official courses clearly attributed to the platform and toggleable so that I can hide them per-tenant if policy requires.
- **As a reviewer/QA**, I want a validator that flags broken links and unanswerable quizzes so that a course cannot ship inaccurate.

## 5. Functional Requirements

- **FR-1.** The system MUST define a per-course, per-locale on-disk content layout embedded via `//go:embed`, parsed into an in-memory `CourseSpec` (metadata + ordered modules + pages/assignments/quizzes + syllabus). It MUST reuse introcourse's fixture parsing (`ModuleMeta`, `PageFixture`, `AssignmentFixture`, `QuizFixture`, `GradingConfig`) where possible.
- **FR-2.** Each course MUST carry a `course.yaml` manifest: `code`, `title`, `catalog_slug`, `catalog_category`, `difficulty_level` (`beginner`|`intermediate`|`advanced`), `catalog_language`, `summary`, `outcomes[]`, `estimated_minutes`, `price_cents` (MUST be `0` for this epic), `is_public`, `marketplace_listed` (MUST be `true`), `hero_image`, and `content_version`.
- **FR-3.** `EnsureProvisioned(ctx, courseSlug)` MUST idempotently upsert the course row, its catalog/marketplace columns, its syllabus, and all structure items + bodies + quizzes, keyed by stable slugs, in a single transaction per course.
- **FR-4.** The provisioner MUST set `published = true`, `marketplace_listed = true`, `marketplace_listed_at = NOW()` (on first list), `price_cents = 0`, `catalog_*` from the manifest, and MUST enable self-paced enrollment so MKT4's free claim applies.
- **FR-5.** A slug→structure-item mapping table MUST persist so re-runs reconcile edits in place (rename/re-order/edit body) rather than duplicating items — mirroring `settings.intro_course_items`.
- **FR-6.** A per-item `content_version` MUST allow the provisioner to skip unchanged items and re-sync changed ones without clobbering learner progress/attempts.
- **FR-7.** The system MUST provide a `provision-marketplace-courses` CLI (optionally `--only <slug>`, `--migrate`) and a `marketplace-courses-validate` CLI that parses all content and exits non-zero on any error.
- **FR-8.** Validation MUST check: every module has ≥1 page and a knowledge check; every quiz question has exactly the required correct answer(s); every internal link resolves; front-matter is well-formed; `price_cents == 0`; and (in CI) every external URL returns 2xx/3xx.
- **FR-9.** Official courses MUST be attributable and centrally ownable (a platform "publisher" identity), not owned by an arbitrary human instructor account, and MUST be hideable per-tenant via existing catalog hide (`catalog_user_prefs`) / an admin toggle.
- **FR-10.** Provisioning MUST be safe to run repeatedly and concurrently-guarded (advisory lock or `ON CONFLICT`), and MUST NOT reset `enrollment_count` or `average_rating`.

## 6. Non-Functional Requirements

- **Performance** — Provisioning all three courses MUST complete in < 10s on a warm DB; it runs at deploy time, not on the request path.
- **Security** — CLI requires `DATABASE_URL`; no external network at provision time (link-checking is a separate CI step). No user input is parsed at runtime.
- **Privacy & Compliance** — Content is public educational material; no PII. Linked third-party resources are referenced, not copied (see §14). Course prose is original or fair-use quotation with attribution.
- **Accessibility** — Authored Markdown MUST meet WCAG 2.1 AA when rendered: images require alt text (validator-enforced), headings are ordered, links have descriptive text (no "click here"), color is not the sole signal. Quizzes rely on the existing accessible quiz player.
- **Scalability** — Handful of courses; no partitioning concerns. Format MUST support adding courses by dropping a new content dir + manifest.
- **Reliability** — Idempotent + transactional per course; a failed course provision MUST NOT leave partial structure (rollback).
- **Observability** — Emit a provisioning summary (created/reconciled/skipped counts per course) and a metric/log per run; reuse the telemetry layer (`server/internal/telemetry`).
- **Maintainability** — Reuse introcourse fixture code; keep the new package (`marketplacecourses`) parallel and small.
- **Internationalization** — `content/<locale>/`; English canonical; missing locale falls back to English (as introcourse's `LoadCurriculum`).
- **Backward compatibility** — New tables/columns are additive; the harness is opt-in per course slug.

## 7. Acceptance Criteria

- **AC-1.** *Given* checked-in content for a course slug, *When* `provision-marketplace-courses --only <slug>` runs on an empty DB, *Then* the course exists with `published=true`, `marketplace_listed=true`, `price_cents=0`, a syllabus, and all modules/pages/quizzes, and appears in the MKT3 storefront.
- **AC-2.** *Given* an already-provisioned course, *When* the command re-runs with no content change, *Then* zero items are created or modified (idempotent) and learner enrollments/attempts are untouched.
- **AC-3.** *Given* a content edit that bumps an item's `content_version`, *When* re-provisioned, *Then* only that item's body is updated and its `structure_item_id` is unchanged.
- **AC-4.** *Given* a quiz question with no correct answer marked, *When* `marketplace-courses-validate` runs, *Then* it exits non-zero naming the offending file.
- **AC-5.** *Given* a content page with a dead external link, *When* the CI link-check runs, *Then* the job fails naming the URL.
- **AC-6.** *Given* a provisioned free course, *When* a learner clicks "Get" in the storefront, *Then* MKT4's free-claim path grants an entitlement + enrollment and the course shows as "Purchased" (MKT5).

## 8. Data Model

Reuse existing structure tables (`course.course_structure_items`, `module_content_pages`, `module_assignments`, `module_quizzes`, `course_syllabus`) and catalog/marketplace columns (`276_public_course_catalog.sql`, `368_course_marketplace.sql`). New:

- `settings.marketplace_course_items` — `(course_slug TEXT, slug TEXT PRIMARY KEY, structure_item_id UUID, content_version INT, grade_policy TEXT NULL, updated_at)` — the slug→item map for idempotent reconcile (mirrors `settings.intro_course_items`).
- `settings.marketplace_courses` — `(slug TEXT PRIMARY KEY, course_id UUID, content_version INT, provisioned_at, updated_at)` — course-level provisioning ledger.
- Migration naming: `server/migrations/NNN_marketplace_course_provisioning.sql` (+ `.down.sql`), next free number.
- **Backfill:** none — courses are created by the provisioner, not backfilled.
- No change to entitlement/enrollment tables (MKT4 owns those).

## 9. API Surface

- **No new runtime HTTP routes.** These courses are served by existing course/marketplace endpoints (MKT3 storefront list, MKT4 claim, course structure/quiz APIs).
- **New CLI commands** (not HTTP): `server/cmd/provision-marketplace-courses`, `server/cmd/marketplace-courses-validate`.
- OpenAPI: no additions (documented as "no API surface" in the story's PR).
- Rate-limit/quota: N/A (CLI at deploy time).

## 10. UI / UX

- **No new UI.** Courses render through the existing storefront cards (MKT3), course landing/detail, syllabus view, module/page reader, and quiz player. A small **"Official"/platform badge** on official-course cards is a nice-to-have (reuse the existing verified/official badge pattern if present; otherwise Open Question §18).
- Empty/loading/error states are the existing course + storefront states.
- Mobile: covered by MKT6 once it ships; content is responsive Markdown.
- i18n: course strings live in the content payload per locale; UI chrome already localized.

## 11. AI / ML Considerations

- Reflection/short-answer assignments MAY use the existing `grader_agent` grade policy (already used by the intro course capstone) — full credit for a good-faith submission, optional AI feedback when enabled. No new model integration.
- Quiz generation is authored by hand for accuracy (not AI-generated), though the existing quiz-generation tooling MAY assist drafting. Cost budget: negligible (grading only, reuses existing path).
- PII: submissions are learner free-text; handled by existing grading/redaction paths.

## 12. Integration Points

- **Internal:** `server/internal/service/introcourse` (fixture parser to generalize), `server/internal/repos/course` (create, marketplace, catalog_listing, syllabus, self_paced), `server/internal/repos/coursemodulequizzes`, `server/internal/repos/billing/entitlements` (via MKT4), `server/internal/telemetry`.
- **Deploy:** run `provision-marketplace-courses --migrate` in the same release step that runs `provision-intro-course`.
- **CI:** new `marketplace-courses-validate` step + external link-checker (e.g. `lychee`) over `content/**`.
- **Events:** none new (enrollment/entitlement events already emitted by MKT4).

## 13. Dependencies & Sequencing

- Must ship after: MKT4 (free claim) — shipped.
- Must ship before: MC1, MC2, MC3 (they are payloads).
- Shared infra: object storage for hero images (existing `course-files` bucket, as introcourse uses `data/course-files/`), CI runner with network egress for link-check.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| External links rot over time | H | M | CI link-checker on every PR + scheduled weekly job; prefer stable official domains |
| Copying third-party content raises licensing/IP issues | M | H | Original prose only; link out; validator flags large verbatim blocks; legal review of §19 lists |
| Re-provision clobbers learner attempts/progress | L | H | Idempotent reconcile keyed by slug + `content_version`; AC-2/AC-3 tests; never delete items with attempts |
| Duplicate courses from non-idempotent runs | L | H | Slug ledger + advisory lock; `ON CONFLICT` upserts |
| Official course owned by a real user account gets deleted | L | M | Platform publisher identity (FR-9); protect from permanent-delete |
| Content factually wrong | M | H | SME sign-off checklist gate (§16) + cited sources (each course §19) |

## 15. Rollout Plan

- **Feature flag:** reuse `FFCourseMarketplace` (default ON). API startup runs
  `EnsureDeployProvisioned` when the flag is on (skips the harness-smoke fixture).
- **Sequencing:** migration → deploy code (startup provisions official courses) → verify storefront → (per course) flip `is_public` if used for SEO.
  Optional explicit CLI: `provision-marketplace-courses --migrate`.
- **Dogfood:** provision to staging; internal claim + complete run-through of one course before GA.
- **GA criteria:** all three courses validate green (incl. link-check), claim + quiz + assignment work end-to-end, a11y (axe) pass on rendered pages.
- **Rollback:** set `marketplace_listed=false` (hides from storefront) without deleting; full rollback drops the two ledger tables (down migration) and unlists.

## 16. Test Plan

- **Unit** — fixture parsing (manifest, front-matter, quiz JSON), version-diff logic, validator rules (missing correct answer, missing alt text, `price_cents != 0` rejected).
- **Integration (DB)** — `EnsureProvisioned` create → reconcile idempotency (AC-2), single-item re-sync (AC-3), transactional rollback on injected failure, slug-ledger correctness. Follow introcourse's `*_db_test.go` patterns.
- **End-to-end (Playwright)** — provision a course in test env → it appears in storefront → free claim → shows "Purchased" (MKT5) → open module → pass knowledge check → submit assignment. Extend `e2e/tests/course-marketplace-*.spec.ts`.
- **Security** — official course not editable/deletable by non-admins; provisioning requires DB creds only.
- **Accessibility** — axe over rendered syllabus + a sample page + quiz; validator enforces alt text and heading order at author time.
- **Content/CI** — `marketplace-courses-validate` + external link-checker as required checks.
- **Performance** — assert full provision < 10s in CI DB.

## 17. Documentation & Training

- **Authoring guide** (`docs/`): the content layout, manifest schema, front-matter fields, quiz JSON schema, grading policies, how to add a course, how to run the validator locally.
- **Runbook:** how/when provisioning runs on deploy; how to unlist a course; how to re-sync after an edit.
- **API reference:** note "no new API surface."
- **Admin docs:** how to hide official courses per tenant.

## 18. Open Questions

1. **Official/publisher identity** — a dedicated platform account vs. a nullable `owner` + `is_official` flag on `course.courses`? (Recommend an `is_official` boolean for badge + delete-protection.)
2. **"Official" badge** — reuse an existing badge component or add one? (UX to confirm.)
3. **Certificates** — do free official courses issue a completion credential now or later? (Defer; note the intro course already contemplates credentials.)
4. **`is_public` default** — list officially in the SEO catalog (15.1) too? (Recommend ON per course for marketing; decided per course.)
5. **Localization scope** — ship EN only first, or EN+ES like the intro course? (Recommend EN at GA, ES fast-follow.)

## 19. References

- Harness to generalize: `server/internal/service/introcourse/{fixtures.go,service.go,content_sync.go,validate.go}`, `server/internal/repos/introcourse/content_items.go`, `server/cmd/provision-intro-course/main.go`, `server/cmd/intro-course-validate`.
- Structure model: `server/migrations/{014,020,021,024,025,033}_*.sql`; `server/internal/repos/course/{create.go,syllabus.go,marketplace.go,catalog_listing.go,self_paced.go}`.
- Quiz model: `server/internal/models/coursemodulequiz/types.go`; `server/internal/repos/coursemodulequizzes`.
- Commerce reuse: `../../completed/marketplace/` (MKT1–MKT5), `docs/completed/marketplace/MKT4-course-purchase-entitlement-flow.md`, `server/internal/repos/billing/entitlements.go`.
- Catalog columns: `server/migrations/276_public_course_catalog.sql`, `368_course_marketplace.sql`.
- Course payloads that consume this harness: [MC1](MC1-ai-essentials.md), [MC2](MC2-introduction-to-python.md), [MC3](MC3-personal-finance.md).
- Authoring guide: [docs/marketplace-courses-authoring.md](../../marketplace-courses-authoring.md).

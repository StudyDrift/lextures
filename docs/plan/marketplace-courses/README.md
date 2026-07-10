# Marketplace Courses — Official Free Catalog

> Epic: author and provision three **official, free, self-paced courses** into the in-app course marketplace — **AI Essentials**, **Introduction to Python**, and **Personal Finance**. These are first-party, evergreen courses the platform ships and maintains (like the intro course), listed at **$0** so any learner can claim them in one click. They give the marketplace real, credible inventory on day one and act as reference examples for third-party instructors.

Each story follows [`../_TEMPLATE.md`](../_TEMPLATE.md). This folder is the working set; move stories to `docs/completed/marketplace-courses/` when shipped.

## Relationship to the marketplace epic

This epic **consumes** the shipped commerce + storefront layer from [`../marketplace/`](../marketplace/README.md) (MKT1–MKT5). It adds **no** new commerce mechanics. What is new here is a **content-authoring + provisioning harness** for first-party courses and the **course content itself** (syllabi, pages, assessments, cited resources).

The pattern is deliberately modeled on the already-shipped **intro course** (`server/internal/service/introcourse`, `server/cmd/provision-intro-course`): embedded, version-controlled content parsed into `course.course_structure_items` + body/quiz rows by an idempotent `EnsureProvisioned`-style provisioner. See [MC0](../../completed/marketplace-courses/MC0-authoring-provisioning-foundation.md).

## What already exists (reuse, do not rebuild)

| Capability | Where |
|---|---|
| Course structure model — `course_structure_items` (`module`/`heading`/`content_page`/`assignment`), `module_content_pages`, `module_assignments`, `module_quizzes`, `course_syllabus` | `server/migrations/014,020,021,024,025,033`, `server/internal/repos/course` |
| Quiz question model (`multiple_choice`, `true_false`, `short_answer`, …) + auto-scoring, attempts, review | `server/internal/models/coursemodulequiz`, `server/internal/repos/coursemodulequizzes` |
| Catalog metadata columns — `is_public`, `catalog_slug`, `catalog_category`, `difficulty_level` (`beginner`/`intermediate`/`advanced`), `catalog_language`, `price_cents`, `enrollment_count`, `average_rating` | `server/migrations/276_public_course_catalog.sql` |
| Marketplace listing — `marketplace_listed`, `marketplace_listed_at`, `FFCourseMarketplace` (default ON) | `server/migrations/368_course_marketplace.sql`, `server/internal/repos/course/marketplace.go` |
| Free claim = entitlement (`course_purchase`, price 0) + enrollment + role grant | `server/internal/repos/billing/entitlements.go`, `httpserver/course_self_paced.go` (MKT4) |
| Content-embed → provision harness (idempotent, i18n, validate CLI, grading front-matter, `grader_agent`/`completion_full` policies) | `server/internal/service/introcourse`, `server/cmd/provision-intro-course`, `server/cmd/intro-course-validate` |
| Storefront discovery, "Purchased" indicator, My purchases | MKT3/MKT5, `clients/web/src/pages/lms/*` |

Net: this epic is **~80% content authoring + a thin generalized provisioner**, and **~0% new commerce**.

## Stories

| ID | Title | Effort | Depends on |
|---|---|---|---|
| [MC0](../../completed/marketplace-courses/MC0-authoring-provisioning-foundation.md) | Official-course authoring & provisioning foundation | M | MKT4 (shipped), introcourse harness |
| [MC1](../../completed/marketplace-courses/MC1-ai-essentials.md) | **AI Essentials** — course content & assessments | M | MC0 |
| [MC2](../../completed/marketplace-courses/MC2-introduction-to-python.md) | **Introduction to Python** — course content & assessments | M | MC0 |
| [MC3](../../completed/marketplace-courses/MC3-personal-finance.md) | **Personal Finance** — course content & assessments | M | MC0 |

## Sequencing

```
MKT4 (shipped) ──> MC0 ──┬─> MC1  (AI Essentials)
                         ├─> MC2  (Introduction to Python)
                         └─> MC3  (Personal Finance)
```

Ship order: **MC0 first** (the harness + one course end-to-end proves the pipeline), then **MC1 / MC2 / MC3 in parallel** — each is an independent content payload once the harness exists. Recommend MC1 be the harness's first payload since AI Essentials is the strongest marketing hook.

## Cross-cutting decisions

1. **All three courses are free.** `price_cents = 0`, so MKT4's *free-claim* path applies: one click yields an entitlement + enrollment, no Stripe. No paid tier is in scope (see each story's Non-Goals).
2. **First-party, evergreen, and version-controlled.** Content is embedded (`//go:embed`) and provisioned by a CLI, not hand-authored in the UI — so it is reviewable, testable, localizable, and re-runnable, exactly like the intro course. A `content_version` per item drives safe re-sync of edits (MC0 §8).
3. **Distinct from the intro course.** The intro course auto-enrolls every account and teaches *the platform*. These are *opt-in* marketplace courses that teach *a subject*. They share the harness pattern but not the auto-enroll or onboarding hooks.
4. **`marketplace_listed = true` + `published = true`; `is_public` optional.** All three are listed in the authenticated in-app storefront (MKT3). Making them `is_public` (SEO catalog, plan 15.1) is a per-course toggle called out in each story's Open Questions — recommended ON to double as marketing landing pages.
5. **Accuracy is a release gate.** Every substantive claim in a content page is backed by a cited, link-checked, authoritative source (each story §19). A CI link-checker + a factual-review checklist gate provisioning (MC0 §16). Prefer primary/official sources (docs.python.org, NIST, SEC/CFPB, university courses) over blogs.
6. **Assessments are auto-scored and low-stakes.** Each module ends with a knowledge-check quiz (auto-graded via `coursemodulequiz`); each course has 1–2 applied assignments graded by `completion_full` or `grader_agent`. No human grading, no proctoring — these are self-paced.
7. **Licensing hygiene.** We *link to* external resources; we do **not** copy third-party text/media into course pages beyond fair-use quotation with attribution. Original prose + our own examples fill the pages. See each story §14 (Risks) and §6 (Compliance).

## Ownership

Content teams (subject-matter reviewers) own the `.md`/`.json` payloads; the platform team owns MC0 (the harness). Each course names a proposed SME reviewer in its Metadata block.

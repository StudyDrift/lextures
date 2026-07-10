# Official marketplace courses — authoring guide (MC0)

First-party free marketplace courses are authored as embedded content under
`server/internal/service/marketplacecourses/content/<course-dir>/` and provisioned
idempotently by `provision-marketplace-courses`.

## Layout

```
content/<course-dir>/
  course.yaml                 # manifest (required)
  en/                         # English (canonical)
    syllabus.json
    m1-…/
      module.yaml
      page.md
      assignment.md           # front-matter kind: assignment
      knowledge-check.json
  es/                         # optional locale overlay (falls back to en)
```

Directory name is usually the `catalog_slug` (e.g. `ai-essentials`).

## Manifest (`course.yaml`)

| Field | Notes |
|---|---|
| `code` | Course code `C-[A-Z0-9]{6}` |
| `title` | Display title |
| `catalog_slug` | Storefront / claim slug |
| `catalog_category` | Browse category |
| `difficulty_level` | `beginner` \| `intermediate` \| `advanced` |
| `catalog_language` | BCP-47-ish (default `en`) |
| `summary` | Course description |
| `outcomes` | YAML list of learning outcomes |
| `estimated_minutes` | Approximate duration |
| `price_cents` | **Must be `0`** for this epic |
| `is_public` | SEO catalog listing (optional) |
| `marketplace_listed` | **Must be `true`** |
| `hero_image` | Optional path to an embedded banner under `assets/` (e.g. `assets/ai-essentials-banner.jpg`). When set for a known course, provisioning writes the file into course-files and sets `hero_image_url`. |
| `content_version` | Course-level ledger version |
| `short_code` | Stable idempotency key (`LEX-MC-…`) |

## Item front-matter

Pages and assignments use Markdown with YAML front-matter:

```md
---
slug: m1.topic.page
title: Page title
sort_order: 0
content_version: 1
---
```

Assignments add `kind: assignment`, `points`, `group`, `grade_policy`
(`completion_full` or `grader_agent`), and `submission_modes`.

Quizzes are JSON with `slug`, `title`, `sort_order`, `questions[]`, and optional
grading fields. Every `multiple_choice` / `true_false` question needs a correct answer.

Bump an item's `content_version` when its body changes so re-provision updates that
item in place without touching unchanged siblings or learner attempts.

## Commands

```bash
# Lint all embedded courses (no DB)
make marketplace-courses-validate

# Provision (requires DATABASE_URL)
cd server && go run ./cmd/provision-marketplace-courses --migrate
cd server && go run ./cmd/provision-marketplace-courses --deploy
cd server && go run ./cmd/provision-marketplace-courses --only harness-smoke
cd server && go run ./cmd/provision-marketplace-courses --only ai-essentials
cd server && go run ./cmd/provision-marketplace-courses --only introduction-to-python
cd server && go run ./cmd/provision-marketplace-courses --only personal-finance
```

`--deploy` matches API startup (official courses only; skips `harness-smoke`).
With no `--only` / `--deploy`, every embedded course including the smoke fixture is provisioned.
## Runbook

- **Deploy:** API startup provisions official courses automatically when
  `FFCourseMarketplace` is on (default), via `EnsureDeployProvisioned` (skips
  `harness-smoke`). You can also run the CLI explicitly:
  `provision-marketplace-courses --migrate` (alongside `provision-intro-course`).
- **Unlist:** `UPDATE course.courses SET marketplace_listed = false, marketplace_listed_at = NULL WHERE catalog_slug = '…';`
- **Re-sync after edit:** bump item `content_version`, merge, re-run provisioner (or restart API).
- **Hide per tenant/user:** existing catalog hide (`catalog_user_prefs` / W07) applies.
- **Attribution:** courses are `is_official = true` and owned by the system publisher
  (`publisher@system.lextures.invalid`).

External link reachability is a separate CI step (e.g. `lychee` over URLs printed by
`marketplace-courses-validate`). Provisioning itself does not call the network.

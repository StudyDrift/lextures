# AC.1 — Adaptive Content Engine: Foundations, Course Feature Flag & Data Model

> Implementation plan. Source: realizes the Lextures tagline *"the learning environment that adapts"*; extends the shipped adaptive-learning core (`docs/completed/01-adaptive-learning-core/`). Folder overview: [README](../../plan/adaptive/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.1 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | 1.1 (learner model), 1.2 (concept graph) — already shipped |
| **Unblocks** | AC.2, AC.3, AC.4, AC.5, AC.6, AC.7, AC.8, AC.9 |

---

## 1. Problem Statement

Lextures adapts the learning *path* (which module comes next) and the *questions* (adaptive quizzes / CAT), but never the *teaching content* a learner reads. A student who fails a pre-check and a student who aces it see the byte-identical content page. There is no schema to bind a pre-assessment to a body of content, to hold a per-learner adaptation decision, to store a generated content variant, or to record which variant a learner was served — so none of AC.2–AC.9 can exist. This story lays the backbone: the per-course feature flag, the core relational model, the configuration API, and the emergency kill-switch, with nothing yet generating or serving content.

## 2. Goals

- Ship a **per-course** `adaptive_content_enabled` flag wired through the identical path as `adaptive_paths_enabled`, with **no** required global platform on-switch.
- Define the core relational model: adaptation *units*, *profiles*, *variants*, *servings*, *outcomes*, and an append-only *event* log — additive migrations that touch no existing table's semantics.
- Expose a course-scoped configuration read/write API (`.../adaptive-content/settings`) and a units CRUD skeleton, all instructor-gated.
- Provide an ops-only emergency kill-switch that, when disengaged (default), never affects a course.
- Guarantee the feature is inert for every existing course and every course that leaves the flag off.

## 3. Non-Goals

- Computing adaptation profiles (AC.2), generating variants (AC.3), serving them (AC.6), or measuring lift (AC.7).
- Any AI/model call — this story writes zero prompts and makes zero provider calls.
- Instructor authoring UI beyond a flag toggle (AC.5 owns the authoring workspace).
- Analytics dashboards (AC.9).

## 4. Personas & User Stories

- **As an instructor**, I want to turn "Adaptive Content" on for *my* course from course settings, without asking an admin to flip a platform switch.
- **As an instructor**, I want to configure course-wide defaults (which adaptation axes are allowed, cost budget, holdout %) in one place before I author any unit.
- **As a platform admin**, I want an emergency kill-switch I can engage during an incident to halt all adaptive-content generation/serving without editing every course.
- **As a data-protection officer**, I want every adaptation artifact (profile, variant, serving) to be a first-class, queryable, deletable row so it is covered by DSAR/retention.
- **As a backend developer**, I want a clean schema and repo layer so AC.2–AC.9 build on stable contracts.

## 5. Functional Requirements

- **FR-1.** The system MUST add `course.courses.adaptive_content_enabled BOOLEAN NOT NULL DEFAULT FALSE` and surface it in the course model, `GET`/list payloads, and the `PATCH /api/v1/courses/{course_code}/features` handler exactly like existing per-course flags.
- **FR-2.** The system MUST expose `GET`/`PUT /api/v1/courses/{course_code}/adaptive-content/settings` returning/accepting per-course config: `allowed_axes` (subset of `emphasis`, `scaffolding`, `reading_level`, `misconception`, `modality`), `default_strategy`, `holdout_percent` (0–50), `monthly_token_budget`, `require_instructor_approval` (bool), `student_optout_allowed` (bool).
- **FR-3.** The system MUST create the core tables (see §8): `adaptive_content_units`, `adaptation_profiles`, `content_variants`, `adaptation_servings`, `adaptation_outcomes`, `adaptive_content_events`, and `adaptive_content_settings`.
- **FR-4.** The system MUST expose a units CRUD skeleton (`GET`/`POST`/`PATCH`/`DELETE .../adaptive-content/units`) that validates the target scope and referenced structure items belong to the course; unit *activation* is gated on downstream stories but the row MUST persist.
- **FR-5.** The system MUST expose an effective-state helper `adaptivecontent.ActiveForCourse(courseFlag bool) bool` returning `courseFlag && !killSwitchEngaged`; there MUST be **no** requirement for a separate global "enabled" flag.
- **FR-6.** The system MUST honor an ops-only kill-switch read from env `ADAPTIVE_CONTENT_KILL_SWITCH` (default *disengaged*); when engaged, all ACE write/generate/serve endpoints return `503` with a stable error code, while config reads still succeed.
- **FR-7.** The system MUST write an `adaptive_content_events` row for every settings change, unit create/update/delete, and flag toggle (append-only audit).
- **FR-8.** All ACE mutation endpoints MUST require the `course:{code}:item:create` permission (instructor/TA), reusing `requireCourseItemCreate`.
- **FR-9.** The system SHOULD include ACE tables in the course-export/import model so course duplication carries units and settings (but never per-student profiles/variants/servings).

## 6. Non-Functional Requirements

- **Performance** — Settings/units reads p95 ≤ 50 ms (single indexed query). The flag adds one boolean to existing course payloads; no extra round-trips.
- **Security** — All mutations instructor-gated; config reads course-member-gated. Kill-switch is env/ops-only, never user-writable. No PII in `adaptive_content_settings` or `adaptive_content_units`.
- **Privacy & Compliance** — `adaptation_profiles`, `content_variants`, `adaptation_servings`, `adaptation_outcomes` are per-student education records (FERPA) — modeled with `ON DELETE CASCADE` from `course_enrollments`/`users` so DSAR erasure (S01) and retention (S02) reach them. Documented in the data inventory (S05).
- **Accessibility** — N/A at data layer; the settings toggle inherits the existing accessible course-features form (AC.5 covers authoring UI a11y).
- **Scalability** — Tables partition-ready: `adaptive_content_events` and `adaptation_servings` carry `created_at` for future BRIN/range partitioning (mirrors `learner_path_events`).
- **Reliability** — Migrations are additive and idempotent (`ADD COLUMN IF NOT EXISTS`, `CREATE TABLE IF NOT EXISTS`). Feature-off is the safe default; any ACE code path guards on `ActiveForCourse`.
- **Observability** — Gauge `adaptive_content.courses_enabled`; counter `adaptive_content.settings_updated`. Kill-switch state exported as gauge `adaptive_content.kill_switch_engaged`.
- **Maintainability** — Follows the established package layout (`service/adaptivecontent`, `repos/adaptivecontent`, `httpserver/adaptive_content_*.go`), no SQL in service layer.
- **Internationalization** — Config enums are locale-independent; any user-facing strings use i18n keys.
- **Backward compatibility** — Every existing course has `adaptive_content_enabled = FALSE`; behavior is unchanged until an instructor opts in. No existing table's columns or constraints change.

## 7. Acceptance Criteria

- **AC-1.** *Given* a fresh course, *When* its `GET /features` payload is read, *Then* `adaptiveContentEnabled` is `false` and no ACE rows exist.
- **AC-2.** *Given* an instructor, *When* they `PATCH .../features` with `adaptiveContentEnabled=true`, *Then* the column flips, an `adaptive_content_events` audit row is written, and the response echoes the new value.
- **AC-3.** *Given* the flag is on, *When* an instructor `PUT`s settings with `holdout_percent=100`, *Then* the API rejects with `400` (max 50) and no row is written.
- **AC-4.** *Given* a unit `POST` whose `pre_assessment_item_id` belongs to a *different* course, *When* submitted, *Then* the API returns `400` and no unit is created.
- **AC-5.** *Given* `ADAPTIVE_CONTENT_KILL_SWITCH=on`, *When* any ACE mutation endpoint is called, *Then* it returns `503 CodeServiceUnavailable`, but `GET .../settings` still returns `200`.
- **AC-6.** *Given* a non-instructor course member, *When* they call any ACE mutation, *Then* they receive `403`.
- **AC-7.** *Given* a course with units and settings, *When* it is duplicated via course export/import, *Then* units and settings copy over and no profile/variant/serving rows are copied.

## 8. Data Model

Migrations reserve `439_adaptive_content_core.sql` (+ `.down.sql`). Schema `course` for authoring/runtime rows; `analytics` reserved for AC.9 rollups.

```sql
-- 439_adaptive_content_core.sql

-- Per-course flag (mirrors adaptive_paths_enabled / misconception_detection_enabled).
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS adaptive_content_enabled BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN course.courses.adaptive_content_enabled IS
    'When true, the Adaptive Content Engine (ACE) may generate and serve per-learner content variants for this course.';

-- Per-course configuration (one row per course; created lazily on first PUT).
CREATE TABLE course.adaptive_content_settings (
    course_id UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    allowed_axes TEXT[] NOT NULL DEFAULT ARRAY['emphasis','scaffolding','reading_level','misconception'],
    default_strategy TEXT NOT NULL DEFAULT 'balanced',
    holdout_percent SMALLINT NOT NULL DEFAULT 0 CHECK (holdout_percent BETWEEN 0 AND 50),
    monthly_token_budget BIGINT NOT NULL DEFAULT 0 CHECK (monthly_token_budget >= 0),
    require_instructor_approval BOOLEAN NOT NULL DEFAULT FALSE,  -- auto-serve once gates pass; instructor/org/jurisdiction can force TRUE
    student_optout_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- An authorable unit: bind a target scope + its base content + pre/post assessments.
CREATE TABLE course.adaptive_content_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    target_kind TEXT NOT NULL CHECK (target_kind IN ('module', 'outcome')),
    target_module_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    target_outcome_id UUID REFERENCES course.course_learning_outcomes (id) ON DELETE CASCADE,
    base_content_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    pre_assessment_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL,
    post_assessment_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL,
    allowed_axes TEXT[] NOT NULL DEFAULT '{}',      -- empty = inherit course settings
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused', 'archived')),
    created_by UUID NOT NULL REFERENCES "user".users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ac_units_target_shape CHECK (
        (target_kind = 'module'  AND target_module_item_id IS NOT NULL AND target_outcome_id IS NULL) OR
        (target_kind = 'outcome' AND target_outcome_id     IS NOT NULL AND target_module_item_id IS NULL)
    )
);
CREATE INDEX idx_ac_units_course ON course.adaptive_content_units (course_id, status);
CREATE INDEX idx_ac_units_base_item ON course.adaptive_content_units (base_content_item_id);

-- Per-learner adaptation decision (populated by AC.2). Kept minimal here; AC.2 adds columns.
CREATE TABLE course.adaptation_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    profile_signature TEXT NOT NULL,                -- stable hash of adaptation inputs (cache key)
    emphasis_mode TEXT,                             -- introduce | reinforce | compress | remediate
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,-- concept gaps, misconceptions, bloom targets (AC.2)
    source_attempt_id UUID REFERENCES course.quiz_attempts (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (unit_id, enrollment_id)
);
CREATE INDEX idx_ac_profiles_signature ON course.adaptation_profiles (unit_id, profile_signature);

-- Generated content variant (populated by AC.3; approvable in AC.5). Cache key = (unit, signature).
CREATE TABLE course.content_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    profile_signature TEXT NOT NULL,
    axes_applied TEXT[] NOT NULL DEFAULT '{}',
    variant_markdown TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    fidelity_score REAL,                            -- 0..1 semantic-fidelity vs base (AC.3)
    safety_flags JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','pending_review','approved','rejected','auto_served','superseded')),
    approved_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (unit_id, profile_signature)
);
CREATE INDEX idx_ac_variants_unit_status ON course.content_variants (unit_id, status);

-- What a learner was actually served (populated by AC.6). NULL variant = base/control.
CREATE TABLE course.adaptation_servings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    profile_id UUID REFERENCES course.adaptation_profiles (id) ON DELETE SET NULL,
    variant_id UUID REFERENCES course.content_variants (id) ON DELETE SET NULL,
    was_holdout BOOLEAN NOT NULL DEFAULT FALSE,     -- control group (AC.7)
    was_fallback BOOLEAN NOT NULL DEFAULT FALSE,    -- base served due to error/opt-out
    served_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ac_servings_unit ON course.adaptation_servings (unit_id, served_at DESC);
CREATE INDEX idx_ac_servings_enrollment ON course.adaptation_servings (enrollment_id);

-- Loop closure: pre/post scores per serving (populated by AC.7).
CREATE TABLE course.adaptation_outcomes (
    serving_id UUID PRIMARY KEY REFERENCES course.adaptation_servings (id) ON DELETE CASCADE,
    pre_score_pct REAL,
    post_score_pct REAL,
    mastery_before REAL,
    mastery_after REAL,
    lift REAL,                                      -- post - pre (or mastery delta)
    measured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Append-only audit for all ACE actions (settings, unit CRUD, generation, approval, serving, opt-out).
CREATE TABLE course.adaptive_content_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    unit_id UUID REFERENCES course.adaptive_content_units (id) ON DELETE SET NULL,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    subject_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ac_events_course ON course.adaptive_content_events (course_id, created_at DESC);
CREATE INDEX idx_ac_events_created_brin ON course.adaptive_content_events USING BRIN (created_at);
```

**Backfill strategy:** none required — all tables are new and empty; the boolean column defaults to `FALSE` so existing rows are correct without a data migration.

## 9. API Surface

All routes under the course scope; auth column shows the minimum role.

```
GET  /api/v1/courses/{course_code}/adaptive-content/settings         member
PUT  /api/v1/courses/{course_code}/adaptive-content/settings         instructor
GET  /api/v1/courses/{course_code}/adaptive-content/units            instructor
POST /api/v1/courses/{course_code}/adaptive-content/units            instructor
PATCH  /api/v1/courses/{course_code}/adaptive-content/units/{id}     instructor
DELETE /api/v1/courses/{course_code}/adaptive-content/units/{id}     instructor
PATCH /api/v1/courses/{course_code}/features                         instructor  (adds adaptiveContentEnabled)
```

```ts
// Settings shape (courses-api-schemas.ts additions)
type AdaptiveContentSettings = {
  allowedAxes: ('emphasis'|'scaffolding'|'reading_level'|'misconception'|'modality')[];
  defaultStrategy: 'gentle'|'balanced'|'aggressive';
  holdoutPercent: number;        // 0..50
  monthlyTokenBudget: number;    // 0 = unlimited
  requireInstructorApproval: boolean;
  studentOptoutAllowed: boolean;
};
type AdaptiveContentUnit = {
  id: string; targetKind: 'module'|'outcome';
  targetModuleItemId?: string; targetOutcomeId?: string;
  baseContentItemId: string;
  preAssessmentItemId?: string; postAssessmentItemId?: string;
  allowedAxes: string[]; status: 'draft'|'active'|'paused'|'archived';
};
```

- **Rate limits:** config/units mutations inherit the standard authenticated write limiter; no new quota.
- **OpenAPI:** all new routes documented in `server/internal/openapi/openapi.go`; `503` kill-switch response registered.

## 10. UI / UX

- **Course settings → Features:** add an "Adaptive Content" toggle in the existing features form (`clients/web/src/pages/lms/course-settings.tsx`), styled identically to the Adaptive Paths / Misconception toggles, with helper copy: *"Rewrite content per learner based on a pre-check, then measure improvement."*
- **No dedicated authoring surface in this story** — the settings drawer (allowed axes, holdout, budget, approval) is a minimal read-only-until-flag-on form; the full authoring workspace is AC.5.
- **States:** flag off → settings drawer disabled with a "Turn on Adaptive Content to configure" hint; flag on → editable. Loading/error use the existing course-settings patterns.
- **Mobile:** inherits the responsive features list; the settings drawer stacks vertically.
- **Accessibility:** toggle is a labeled `switch` with `aria-describedby` pointing at helper copy; no new patterns.

## 11. AI / ML Considerations

None in this story — AC.1 makes zero model calls. It only *reserves* the `adaptive_content` feature id in `aigateway` (constant `FeatureAdaptiveContent = "adaptive_content"`) so AC.3 can disclose/budget/log against it.

## 12. Integration Points

- `server/migrations/439_adaptive_content_core.sql` (+ down).
- `server/internal/repos/course/` — extend course model + `PatchFeatures` for the new boolean (mirrors `MisconceptionDetectionEnabled` wiring in `list_enrolled.go`, `search_index.go`, `models/course/types.go`).
- `server/internal/httpserver/course_features.go` — add `AdaptiveContentEnabled *bool`.
- `server/internal/service/adaptivecontent/` (new) — `ActiveForCourse`, kill-switch reader, settings/units validation.
- `server/internal/repos/adaptivecontent/` (new) — settings + units + events repo.
- `server/internal/httpserver/adaptive_content_settings.go`, `adaptive_content_units.go` (new).
- `server/internal/service/aigateway/service.go` — add `FeatureAdaptiveContent`.
- `clients/web/src/lib/courses-api-schemas.ts` + `courses-api.ts` — `adaptiveContentEnabled` + settings/units schemas.
- `server/internal/models/courseexport/types.go` — carry units + settings on duplication.

## 13. Dependencies & Sequencing

- **Must ship after:** 1.1 (learner model), 1.2 (concept graph), 1.10 (misconceptions), 072/074 (outcomes) — all shipped.
- **Must ship before:** AC.2–AC.9 (every ACE story reads these tables).
- **Shared infra:** none beyond the existing Postgres + course-features plumbing.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Schema churn as AC.2/AC.3 discover needs | M | M | `payload_json`/`detail_json` JSONB escape hatches; additive later migrations |
| Flag confused with adaptive *paths* | M | L | Distinct column, distinct UI copy, distinct docs; README disambiguation table |
| Kill-switch mis-set blocks a course silently | L | M | Kill-switch state exported as metric + shown in admin console; disengaged default |
| Orphan units after content-page deletion | M | L | `ON DELETE CASCADE` from `course_structure_items`; unit list filters archived |

## 15. Rollout Plan

- **Feature flag:** `adaptive_content_enabled` — per-course, default `FALSE`. **Deliberately no required global on-switch** (honors the "course-level, not platform-level" mandate). Only control above the course is the ops kill-switch `ADAPTIVE_CONTENT_KILL_SWITCH` (default disengaged; engaging it is an incident action, not the normal gate).
- **Sequencing:** deploy migration → ship flag + settings/units API behind it → enable on one internal test course → confirm inert for all others → hand off to AC.2.
- **Pilot cohort:** internal QA course only (no learners see anything yet — nothing is served).
- **GA criteria:** migration applied cleanly; existing course-features e2e green; toggling the flag writes an audit row; kill-switch verified in staging.
- **Rollback:** flip flag off (course reverts to static content instantly); migration `.down` drops ACE tables + column (data-loss acknowledged, restore-from-backup policy per repo convention).

## 16. Test Plan

- **Unit** — `ActiveForCourse` truth table (flag×kill-switch); settings validators (holdout bounds, axis enum); unit target-shape constraint; signature stability helper.
- **Integration** — migration up/down; `PATCH /features` flips column + writes event; `PUT settings` upsert; unit `POST` cross-course rejection; kill-switch `503` on mutations but `200` on settings read.
- **End-to-end** — Playwright: instructor toggles Adaptive Content in course settings, sees the config drawer become editable, saves settings.
- **Security** — authz matrix: student/guest `403` on all mutations; member `200` on settings read; only env sets kill-switch.
- **Accessibility** — axe on the settings form; toggle keyboard-operable, labeled.
- **Performance / load** — settings/units reads p95 ≤ 50 ms at 500 rps (k6).
- **Manual exploratory** — duplicate a course, confirm units/settings copy and no per-student rows copy.

## 17. Documentation & Training

- Instructor help: "What is Adaptive Content and how do I turn it on for my course?"
- Admin runbook: "Engaging the Adaptive Content emergency kill-switch."
- API reference: new settings/units routes + `adaptiveContentEnabled` field.
- Internal architecture note: ACE entity model + how the loop tables relate (link this file).

## 18. Open Questions

1. Should a *unit* target a single content page or a set of pages under a module? (v1: single `base_content_item_id`; multi-page grouping is a later enhancement.)
2. Should `adaptive_content_settings` live on the `courses` row (like the flags) or a side table? (Chosen: side table, to keep the wide `courses` row stable and allow richer config.)
3. Do we need per-section (not just per-course) settings for cross-listed sections? (Deferred; sections can override in a later story if demand appears.)
4. Should course export copy *approved* variants as reusable templates? (Deferred to AC.5.)

## 19. References

- Existing files: `server/internal/httpserver/course_features.go`, `server/internal/repos/course/list_enrolled.go`, `server/internal/models/course/types.go`, `clients/web/src/lib/courses-api-schemas.ts`, `server/migrations/090_adaptive_paths.sql`, `server/migrations/096_misconceptions.sql`.
- Related plans: [AC.2](../../plan/adaptive/AC.2-pre-assessment-and-adaptation-profile.md), [AC.8](../../plan/adaptive/AC.8-governance-safety-fairness-privacy.md), `../01-adaptive-learning-core/1.4-adaptive-paths-across-modules.md`.
- External: FERPA 34 CFR Part 99 (education-record scope of the new tables).

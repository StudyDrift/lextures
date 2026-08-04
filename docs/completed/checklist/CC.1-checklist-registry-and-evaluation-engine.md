# CC.1 — Checklist Rule Registry & Evaluation Engine

> Implementation plan. Source: Course Checklist product request (a course-scoped "is this course actually
> ready and well designed?" surface). Folder overview: [README](../../plan/checklist/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.1 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server / platform team |
| **Depends on** | — |
| **Unblocks** | CC.2, CC.3, CC.4, CC.5, CC.6, CC.7, CC.8, CC.9, CC.10 |

---

## 1. Problem Statement

Lextures has ~80 course-authoring surfaces (modules, syllabus, outcomes, rubrics, assignment groups,
accommodations, features, dates, enrollments) and no single place that tells an instructor which of them
are still unfinished. A teacher who has never taught online cannot know that "every assessment should map
to an outcome" or "state your feedback turnaround time" is expected — those expectations live in external
rubrics (Quality Matters, SUNY OSCQR, NSQ) that we do not surface. The result is courses that ship with
empty modules, unpublished items past their start date, orphan outcomes and no welcome message, and
instructors who discover the gap only from student complaints. CC.1 builds the engine: a declarative rule
registry plus an evaluator that turns one course into a list of pass/fail/not-applicable findings.

## 2. Goals

- One **declarative registry** where a checklist item is a data descriptor (stable ID, category, copy,
  applicability predicate, evaluator, navigation target, evidence shape) — adding item #81 is one registry
  entry plus one evaluator function, with **no migration, no table, no route**.
- An **evaluator** that computes the full checklist for a course in a single bounded pass over the DB,
  with a hard latency budget, so the page and the nav badge can both be served from it.
- A **stable ID contract** so dismissals (CC.2), telemetry (CC.10) and mobile parity (CC.9) can persist a
  reference to an item across releases.
- **Structured evidence**: a failing item can carry the list of offending entities (e.g. the eleven
  assignments with no outcome mapping) so the UI can expand a table instead of just saying "no".
- **Graceful degradation**: one broken evaluator degrades that item to `unknown`, never the page.

## 3. Non-Goals

- No persistence of state or dismissals — that is CC.2.
- No HTTP routes — CC.2 owns the API surface.
- No specific checklist items — the catalog is CC.3 / CC.4 / CC.5 / CC.6. CC.1 ships the engine plus two
  trivial reference rules used only by its own tests.
- No UI of any kind (CC.7 web, CC.9 mobile).
- No auto-remediation ("fix it for me"); CC.10 covers guidance and AI assist.
- No cross-course or org-level rollup ("which of my 40 courses are unready") — see §18 Q4.

## 4. Personas & User Stories

- **As an instructor**, I want the platform to know what "done" means for each part of my course, so that I
  am not guessing which of thirty settings pages still needs me.
- **As an instructional designer**, I want the rules to reflect recognised quality rubrics rather than one
  person's opinion, so that I can defend the checklist to a curriculum committee.
- **As a K-12 teacher**, I want rules that do not apply to my course (sections, SBG, evaluations) to stay
  silent, so that the list is short and honest.
- **As a homeschool parent**, I want a small credible list of things that make a course good, so that I get
  the benefit of an instructional-design review I could never afford.
- **As a platform engineer**, I want to add a rule without touching the schema or the router, so that the
  catalog can grow without a release-coordination cost.
- **As an SRE**, I want a single evaluator with one latency budget and one metric, so that a slow rule is
  attributable and can be disabled without a deploy.

## 5. Functional Requirements

- **FR-1.** The system MUST define package `server/internal/service/coursechecklist` exporting a
  `Registry` of `ItemDescriptor` values, keyed by a stable `ItemID`.
- **FR-2.** An `ItemDescriptor` MUST carry: `ID`, `Category`, `Title`, `Why` (one-sentence rationale),
  `HelpRef` (docs anchor), `Tier` (`essential | recommended`), `Sources []string` (e.g. `"QM 1.2"`,
  `"OSCQR 41"`, `"NSQ A"`), `Applies func(Context) bool`, `Evaluate func(ctx, Context) (Finding, error)`,
  `Target NavTarget`, and `EvidenceShape` (nil, or column headers for the expandable table).
- **FR-3.** `ItemID` MUST match `^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$` (e.g. `outcomes.assessment-mapping`)
  and MUST be treated as a persisted contract: once shipped, an ID is never re-pointed at a different rule.
- **FR-4.** The registry MUST expose `ITEM_ID_ALIASES map[ItemID]ItemID` and `RETIRED_ITEM_IDS`, plus
  `ResolveItemID(string) (ItemID, bool)` returning the canonical ID, or `false` for unknown/retired IDs.
- **FR-5.** A `Finding` MUST carry `Status` (`done | todo | in_progress | not_applicable | unknown`),
  `Detail` (short human string), `Progress` (`done`/`total` counters, optional) and
  `Evidence []EvidenceRow` where each row has `Label`, `Sublabel`, `TargetOverride *NavTarget` and
  `Status`.
- **FR-6.** `Evaluate(ctx, CourseSnapshot)` MUST run against an **in-memory `CourseSnapshot`** loaded once
  per evaluation; individual evaluators MUST NOT issue their own queries except through a declared
  `LazyLoader` (FR-8).
- **FR-7.** The snapshot loader MUST load, in one bounded batch: the course row and its feature columns,
  structure items (+ published/due/parent), module content pages / assignments / quizzes / surveys metadata,
  syllabus sections, learning outcomes and outcome links, assignment groups, grading scheme, enrollments
  aggregated by role, feed channels + latest message per channel, course files metadata, sections, and
  accommodations counts.
- **FR-8.** Rules that need data too expensive for every evaluation (e.g. external link health) MUST declare
  a `LazyLoader` that the engine invokes at most once per evaluation and only when at least one *applicable*
  rule requests it.
- **FR-9.** `Evaluate` MUST run every applicable rule; a rule returning an error or panicking MUST be
  recovered, logged with `item_id`, recorded on the `coursechecklist_rule_errors_total` counter, and reported
  as `Status = unknown` without failing the evaluation.
- **FR-10.** The engine MUST expose `CatalogVersion() string` — a hash over the sorted set of
  `(ItemID, Tier, Category)` — and `EngineVersion() int`, both used by CC.2 for snapshot invalidation.
- **FR-11.** `Result` MUST include per-category and overall counts: `Total`, `Done`, `Todo`,
  `NotApplicable`, `Unknown`, plus `OutstandingEssential` (essential-tier items in `todo`/`in_progress`,
  excluding dismissed — dismissal filtering happens in CC.2).
- **FR-12.** Category ordering and item ordering within a category MUST be deterministic and declared in
  the registry, not derived from map iteration.
- **FR-13.** The engine MUST accept an `EvaluateOptions{ Only []ItemID }` so a single item can be
  re-evaluated cheaply after a user acts on it.
- **FR-14.** A registry integrity test MUST assert: IDs unique and regex-conformant; every alias resolves;
  every descriptor has non-empty `Title`, `Why` and at least one `Source`; every `Target` names a route that
  exists in the web route table fixture; `EvidenceShape` present iff the evaluator can emit evidence.
- **FR-15.** The engine MUST be **read-only**: no evaluator may write to the database.

## 6. Non-Functional Requirements

- **Performance** — Full evaluation p95 < 400 ms and p99 < 900 ms for a course with 40 modules, 300
  structure items and 500 enrollments, measured server-side excluding lazy loaders. Snapshot load MUST be
  ≤ 18 queries. `EvaluateOptions.Only` for a single item MUST be p95 < 120 ms. Lazy loaders MUST have their
  own 5 s budget and MUST NOT block the base result (they yield `unknown` on timeout).
- **Security** — The engine takes an already-authorised `CourseSnapshot`; it performs no authz itself and
  MUST NOT be callable with a course the caller has not been authorised for (enforced by CC.2). Evidence
  rows MUST NOT contain data a course staff member could not already see.
- **Privacy & Compliance** — Evidence may name enrolled users (e.g. "3 students with no guardian link"); it
  MUST expose only display name + opaque ID, never email or DOB, and MUST honour the existing FERPA
  directory-info suppression used by the enrollments API. No learner performance data enters the checklist.
- **Accessibility** — N/A (no UI). Copy fields MUST be plain-language, ≤ 90 chars for `Title`.
- **Scalability** — Registry design MUST tolerate 250 descriptors without a lookup-strategy change. Snapshot
  memory MUST stay under 8 MB for the p99 course; evaluation MUST stream/limit evidence to 200 rows per item
  with a `TruncatedAt` marker.
- **Reliability** — Evaluation is pure and idempotent. Rule panics are contained (FR-9). A missing optional
  table (feature not provisioned) MUST yield `not_applicable`, never an error.
- **Observability** — Metrics: `coursechecklist_evaluate_duration_seconds` (histogram, label `mode=full|single`),
  `coursechecklist_rule_duration_seconds` (histogram, label `item_id`), `coursechecklist_rule_errors_total`
  (counter, labels `item_id`, `kind=error|panic|timeout`), `coursechecklist_snapshot_query_duration_seconds`.
  Traces: one span per evaluation, child span per lazy loader. Logs include `course_id`, `catalog_version`.
- **Maintainability** — Rules live in `rules_*.go` files grouped by category (CC.3–CC.6). Each rule is a
  pure function of the snapshot and is unit-tested in isolation with a table-driven fixture.
- **Internationalization** — `Title`, `Why` and `Detail` MUST be i18n **keys plus English defaults**;
  the engine returns both so clients can localise (`coursechecklist.item.<id>.title`). Any date/number in
  `Detail` MUST be emitted as structured fields, not pre-formatted strings.
- **Backward compatibility** — Removing a rule MUST go through `RETIRED_ITEM_IDS` so persisted dismissals
  resolve to a tombstone rather than erroring. Bumping `EngineVersion` invalidates all snapshots.

## 7. Acceptance Criteria

- **AC-1.** *Given* a registry with two rules and a fixture course, *When* `Evaluate` runs, *Then* the result
  contains exactly two findings in declared order with correct counts.
- **AC-2.** *Given* a rule whose evaluator panics, *When* `Evaluate` runs, *Then* that item's status is
  `unknown`, `coursechecklist_rule_errors_total{kind="panic"}` increments by 1, and every other item still
  evaluates.
- **AC-3.** *Given* a course where `sectionsEnabled` is false, *When* a sections-scoped rule is evaluated,
  *Then* its status is `not_applicable` and it is excluded from `Total` for progress purposes.
- **AC-4.** *Given* an evaluator that emits 500 evidence rows, *When* the result is built, *Then* exactly 200
  rows are returned and `TruncatedAt = 200`.
- **AC-5.** *Given* `EvaluateOptions{Only: []ItemID{"course.dates"}}`, *When* evaluation runs, *Then* exactly
  one rule executes and the snapshot loader issues only the queries that rule's data dependencies declare.
- **AC-6.** *Given* the registry, *When* the integrity test runs, *Then* it fails if any ID is duplicated,
  malformed, aliased to a missing ID, or missing a `Sources` entry.
- **AC-7.** *Given* two identical courses, *When* each is evaluated, *Then* `CatalogVersion()` is equal and
  the serialized results are byte-identical (determinism).
- **AC-8.** *Given* a lazy loader that exceeds its 5 s budget, *When* evaluation completes, *Then* the base
  result is returned within the normal budget and the dependent items are `unknown`.
- **AC-9.** *Given* a course with 300 structure items, *When* evaluation runs against a seeded database,
  *Then* the benchmark test asserts ≤ 18 queries and p95 < 400 ms.

## 8. Data Model

CC.1 introduces **no schema changes**. It reads existing tables:

| Concern | Source |
|---|---|
| Course row, feature switches, dates, timezone, schedule mode | `course.courses` |
| Modules / headings / items, published, due, parent | `course.course_structure_items` |
| Item detail | `course.module_content_pages`, `course.module_assignments`, `course.module_quizzes`, `course.module_surveys`, `course.module_external_links` |
| Syllabus | `course.course_syllabus` |
| Outcomes & mapping | `course.course_learning_outcomes`, `course.course_outcome_links` |
| Grading | `course.assignment_groups`, `course.grading_schemes` |
| People | `course.course_enrollments`, `course.enrollment_roles`, `course.course_sections` |
| Announcements / welcome | `course.feed_channels`, `course.feed_messages` |
| Files | `course.course_files`, `course.file_items` |
| Accommodations | `course.student_accommodations` |
| Standards (K-12) | `course.course_standards`, `course.question_standard_alignments` |

Snapshot loading MUST use existing repo packages (`repos/course`, `repos/coursestructure`,
`repos/courseoutcomes`, `repos/enrollment`, `repos/coursegrading`) and MUST add read-only batch helpers
rather than new ad-hoc SQL in the service package.

No migration file is created by CC.1. Persistence lands in CC.2 (`server/migrations/461_course_checklist.sql`).

## 9. API Surface

None. CC.1 is a Go package consumed by CC.2's handlers. Public Go surface:

```go
type ItemID string

type ItemDescriptor struct {
    ID            ItemID
    Category      CategoryID
    TitleKey      string; TitleDefault string
    WhyKey        string; WhyDefault   string
    HelpRef       string
    Tier          Tier          // essential | recommended
    Sources       []string      // "QM 2.1", "OSCQR 41", "NSQ C", "WCAG 1.1.1"
    DataNeeds     []DataNeed    // drives snapshot loading + Only-mode
    Applies       func(CourseSnapshot) bool
    Evaluate      func(context.Context, CourseSnapshot) (Finding, error)
    Target        NavTarget
    EvidenceShape *EvidenceShape
}

type NavTarget struct {
    Surface   string // "web" | "ios" | "android" | "all"
    Route     string // "/courses/{courseCode}/settings/general"
    Anchor    string // CC.8 focus token, e.g. "course.general.dates"
    EntityKey string // optional; substituted from evidence row
}

func Evaluate(ctx context.Context, snap CourseSnapshot, opt EvaluateOptions) Result
func LoadSnapshot(ctx context.Context, pool *pgxpool.Pool, courseCode string, needs []DataNeed) (CourseSnapshot, error)
func CatalogVersion() string
func EngineVersion() int
func ResolveItemID(raw string) (ItemID, bool)
```

## 10. UI / UX

None in CC.1. The engine defines the **contract the UI renders**:

- `Status` values map to: `done` → struck-through row with check; `todo` → actionable row; `in_progress` →
  actionable row with `x / y` progress; `not_applicable` → hidden; `unknown` → muted row with "couldn't
  check" microcopy and a retry affordance.
- `EvidenceShape.Columns` names the header cells of the expandable table CC.7 renders.
- `Target` is what CC.8 turns into a focus-and-highlight navigation.

## 11. AI / ML Considerations

None in CC.1 — every rule is deterministic. Two rules planned in later packs (outcome verb quality in CC.4,
"welcome message is substantive" in CC.3) are specified as **heuristic, non-AI** checks precisely so the
engine stays pure and cheap. AI-assisted remediation is scoped to CC.10 and sits outside the evaluator.

## 12. Integration Points

- Internal: `server/internal/repos/course`, `.../coursestructure`, `.../courseoutcomes`, `.../enrollment`,
  `.../coursegrading`, `.../coursesyllabus`, `.../coursefeed`, `.../coursefiles`, `.../coursesections`.
- `server/internal/telemetry` for metrics/traces registration.
- `server/internal/l10n` for the key/default pairing convention.
- Consumed by `server/internal/httpserver/course_checklist*.go` (CC.2).
- No external services. No webhook emissions in CC.1 (CC.10 adds the analytics event stream).

## 13. Dependencies & Sequencing

- Must ship before: CC.2 (API), CC.3–CC.6 (rule packs), CC.7 (web), CC.9 (mobile).
- Must ship after: nothing.
- Shared infra: existing pgx pool, telemetry registry. No queue, no object storage, no email.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Snapshot query fan-out makes the course page slow | M | H | Hard ≤ 18-query budget asserted by test; `DataNeeds` prunes loads in `Only` mode; CC.2 caches snapshots |
| A rule is wrong and nags every instructor | M | H | `Tier` split (essential vs recommended); dismissal (CC.2); rules land dark-then-promoted (§15) |
| Registry becomes a dumping ground of 300 rules | M | M | `Sources` field is mandatory — a rule with no external-rubric or product basis fails review |
| Evidence leaks learner data | L | H | FR-5 + §6 privacy rule; test asserts no email/DOB fields in serialized evidence |
| Rule authors write ad-hoc SQL, fan-out grows silently | M | M | Evaluators receive only the snapshot; a lint test forbids `pgxpool` imports in `rules_*.go` |
| Determinism breaks via map iteration | M | M | AC-7 byte-identical test; explicit ordering in registry |

## 15. Rollout Plan

**There is no feature flag for the Course Checklist** — the product decision is that the checklist is
always on for staff. CC.1's safety valves are therefore structural, not toggle-based:

1. Ship the engine with only its two reference rules; real rules arrive per-pack in CC.3–CC.6.
2. Each new rule lands `Tier: recommended` first; promotion to `essential` (which drives the nav badge)
   happens in a follow-up change once dogfooding shows a low false-positive rate.
3. A rule can be **retired without a deploy of clients** by moving its ID into `RETIRED_ITEM_IDS`; the API
   (CC.2) filters retired IDs out of responses, so a bad rule needs one server release, not a client one.
4. `EngineVersion` bump invalidates all cached snapshots (CC.2) — the migration sequence is schema (CC.2)
   → engine → rules → clients.
5. Rollback: revert the rules file; the engine returns an empty catalog and CC.7 renders its
   "nothing to check right now" empty state.

## 16. Test Plan

- **Unit** — Per-rule table tests over hand-built snapshots (done / todo / not-applicable / evidence).
  Registry integrity (FR-14). Alias + retired-ID resolution. Panic containment. Evidence truncation.
  Determinism (AC-7).
- **Integration** — `LoadSnapshot` against a seeded Postgres: query-count assertion, field-mapping
  assertions for every `DataNeed`, and behaviour when optional tables are empty.
- **End-to-end** — None in CC.1 (no surface).
- **Security** — Test that `LoadSnapshot` never returns suppressed directory info; lint test that
  `rules_*.go` files import no database packages.
- **Accessibility** — N/A.
- **Performance / load** — Go benchmark on a 300-item / 500-enrollment fixture asserting the §6 budgets;
  runs in CI as a non-blocking report with a blocking regression threshold of +50%.
- **Manual exploratory** — Run the evaluator against the three seeded demo courses (K-12, HE, homeschool)
  and eyeball every finding for plausibility before CC.3 promotion.

## 17. Documentation & Training

- `docs/dev/course-checklist-engine.md` — how to add a rule (registry entry → evaluator → fixture → test),
  the ID contract, and the `DataNeeds` mechanism.
- Godoc on the package covering `ItemDescriptor` semantics and the read-only invariant.
- ADR in `docs/adr/` recording: (a) code registry over DB-driven rules, (b) no feature flag,
  (c) evidence-in-result rather than a second round trip.
- Runbook stub `docs/runbooks/course-checklist.md` — how to retire a misbehaving rule fast.

## 18. Open Questions

1. Should `Tier` be a 3-level scale (`essential | recommended | optional`) to keep the badge count small
   while still surfacing polish items? Proposed: start with 2, revisit after CC.10 telemetry.
2. Does the snapshot need per-section scoping for cross-listed courses, or is course-level sufficient?
   (`course.cross_list_groups` exists.) Proposed: course-level in CC.1; revisit in CC.4.
3. Should `unknown` items count against progress, or be excluded like `not_applicable`? Proposed: excluded
   from the denominator but surfaced with a retry affordance.
4. Is an org-level rollup ("checklist health across my department's courses") in scope for a later CC story
   or for section 18 (Admin Experience)? Proposed: defer, and design `Result` to be aggregatable.
5. Do blueprint child courses inherit dismissals from the parent? (`course.courses.blueprint_parent_id`.)
   Proposed: no in CC.2; flag for CC.10.

## 19. References

- Existing files this work touches: `server/internal/service/` (new `coursechecklist` package),
  `server/internal/repos/course/`, `server/internal/repos/coursestructure/`,
  `server/internal/repos/courseoutcomes/`, `server/internal/repos/enrollment/`,
  `server/internal/telemetry/`.
- Precedent in-repo: [PS.1 settings registry](../settings/PS.1-settings-registry-and-addressable-controls.md)
  (stable string IDs, aliases, integrity tests) and
  [CT.1 content-tools registry](../content_tools/CT.1-foundations-registry-and-data-model.md)
  (declarative manifest, no migration per tool).
- External standards: [Quality Matters Higher Ed Rubric](https://www.qualitymatters.org/qa-resources/rubric-standards/higher-ed-rubric),
  [SUNY OSCQR](https://oscqr.suny.edu/),
  [National Standards for Quality Online Courses](https://nsqol.org/the-standards/quality-online-courses/),
  [CAST UDL Guidelines 3.0](https://udlguidelines.cast.org/), WCAG 2.1 AA.
- Related plans: [CC.2](../../plan/checklist/CC.2-checklist-state-api-and-dismissals.md), [CC.3](../../plan/checklist/CC.3-rule-pack-foundations-and-orientation.md),
  [CC.7](../../plan/checklist/CC.7-web-checklist-page-and-nav-badge.md), [CC.8](../../plan/checklist/CC.8-deep-link-and-highlight-targeting.md).

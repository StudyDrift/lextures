# Course Checklist engine (CC.1)

How to add a checklist rule to the declarative registry in
`server/internal/service/coursechecklist`.

## Mental model

1. A rule is an `ItemDescriptor` (stable ID, copy, applicability, evaluator, nav target).
2. `LoadSnapshot` loads an in-memory `CourseSnapshot` once per evaluation (≤ 18 SQL queries).
3. `Evaluate` runs applicable rules against that snapshot. Rules never query the DB.

There is **no feature flag**. Retire a bad rule via `RETIRED_ITEM_IDS` (server release only).

## Adding a rule

1. Pick a stable `ItemID` matching `^[a-z][a-z0-9]*(\.[a-z0-9-]+){1,3}$`
   (e.g. `outcomes.assessment-mapping`). Once shipped, never re-point an ID at a different rule.
2. Add the descriptor in the appropriate `rules_*.go` file (grouped by category; CC.3–CC.6 packs).
3. Declare `DataNeeds` so Only-mode loads only what the rule needs.
4. Implement `Applies` (or `nil` if always applicable) and `Evaluate` as pure functions of `CourseSnapshot`.
5. Set `Target` to a route present in `testdata/web_routes.json`. Prefer an `Anchor` from the
   [focus-anchors registry](focus-anchors.md) (CC.8) so the web client can highlight the control.
6. If the evaluator can emit evidence rows, set `EvidenceShape` and keep evidence ≤ 200 rows
   (the engine truncates and sets `TruncatedAt`).
7. Cite at least one `Sources` entry (`"QM 2.1"`, `"OSCQR 41"`, `"NSQ C"`, `"Product"`, …).
8. Add a table-driven unit test over hand-built snapshots (done / todo / not_applicable / evidence).
9. Ship new rules as `Tier: recommended`. Promote to `essential` only after dogfooding (CC.10).

### I18n keys

`TitleKey` / `WhyKey` / detail keys use `coursechecklist.item.<id>.*` with English defaults on the
descriptor / finding. Clients localise; the engine returns both.

### Lazy loaders

For data too expensive for every evaluation, declare `LazyNeeds` and register a `LazyLoader` on
`EvaluateOptions`. The engine invokes each loader at most once, only when an applicable rule needs it,
with a 5s budget. On timeout the dependent item is `unknown`.

## ID contract

| Action | How |
|---|---|
| Rename | `ITEM_ID_ALIASES[old] = canonical` |
| Remove | add to `RETIRED_ITEM_IDS` |
| Resolve | `ResolveItemID(raw) (ItemID, bool)` |

`CatalogVersion()` hashes sorted `(ItemID, Tier, Category)`. `EngineVersion()` bumps when Result
semantics change (invalidates CC.2 caches).

## DataNeeds

| Need | Loads |
|---|---|
| `course` | Course row + features + dates |
| `structure` | Structure items |
| `item_meta` | Content/assignment/quiz/survey/link metadata |
| `syllabus` | Syllabus sections |
| `outcomes` | Outcomes + links |
| `grading` | Groups + scale |
| `enrollments` | Role aggregates |
| `feed` | Channels + latest root message (read-only) |
| `files` | File metadata |
| `sections` | Sections |
| `accommodations` | Active count |
| `standards` | Course standards + aligned structure item IDs |
| `content_tool_counts` | Active content-tool instance item IDs |
| `module_prerequisites` | Module prerequisite edges + item completion-rule presence |

`DataNeedsForEvaluate(reg, opt)` computes the union for Only-mode pruning.

### Evidence tables with per-row targets (CC.4)

Some rules (notably `outcomes.assessment-mapping`) emit evidence where each row has its own
`TargetOverride` (assignment/quiz editor + settings-registry anchor). Clients MUST prefer the row
target over the item-level `Target` when present. Column headers come from `EvidenceShape.Columns`
and MUST be rendered as real `<th scope="col">` cells (CC.7).

## Guards

- `rules_*.go` must not import `pgx` / `database/sql` / `repos/*` (enforced by test).
- Evidence must not carry email or DOB — display name + opaque user ID only.
- One broken evaluator → that item `unknown`; evaluation continues.

## Writing a text-heuristic rule (CC.3)

Orientation and syllabus rules match learner-facing prose with **locale-aware lexicons**,
not English-only inline regexes.

1. Add keywords / patterns under `server/internal/service/coursechecklist/lexicons/{locale}.json`
   (`en`, `es`, `fr`, `ar` ship today). Missing locales fall back to English (never `unknown`
   solely for a missing lexicon).
2. Load via `lexiconForSnap(snap)` (uses `CatalogLanguage`) and call the compiled matchers
   (`matchResponseTime`, `matchContact`, …). Regexes are compiled once at package init.
3. Scan syllabus markdown capped at **512 KB**; when truncated, set a detail note
   (`checked first 512 KB`).
4. Malformed syllabus JSON → syllabus text rules return `unknown`; other packs keep running.
5. Prefer `not_applicable` when the inspected feature is disabled on the course.
6. Keep evaluators pure snapshot functions — no lazy loaders in the CC.3 pack.

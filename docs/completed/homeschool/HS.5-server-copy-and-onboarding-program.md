# HS.5 — Server: copy, flag descriptions, course content & the onboarding `program` value

> Implementation plan. **Status: DONE** (moved from `docs/plan/homeschool/`). Source: product
> rebrand of the **self-learner** segment to **Homeschool**.
> Terminology and copy are fixed by [HS.1](HS.1-terminology-copy-deck-and-guardrails.md).
> Code references: `server/internal/httpserver/onboarding_http.go`,
> `server/migrations/142_onboarding_events.sql`, `server/internal/aidisclosure/disclosure.go`,
> `server/internal/service/studybuddy/prompt.go`, `server/internal/config/config.go`,
> `server/internal/service/{introcourse,marketplacecourses}/content/**`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | HS.5 |
| **Section** | Go API (`server/`) |
| **Severity** | MINOR (copy) / MAJOR (the `program` enum, which today silently drops events) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE — `program` accepts `homeschool`+`school`, insert failures are logged/metered, AI disclosure + study-buddy prompt + syllabus audience lines use Homeschool terms, content versions bumped, www beacon flipped |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend |
| **Depends on** | HS.1 (copy deck) |
| **Unblocks** | HS.2's `program` flip · HS.6 (guard flip) |

---

## 1. Problem Statement

Three user-visible server strings still name the old segment: the AI-disclosure catalogue entry for
the study buddy ("Standalone study companion for self-learners."), the study-buddy **system prompt**
sent to the model on every turn, and the audience lines in three seeded course syllabi. Separately,
and more consequentially, `www`'s `/get-started` page reports which path a visitor chose to
`POST /api/v1/public/onboarding/track`, and that value lands in `onboarding_events.program` — a
column whose CHECK constraint from migration 142 allows only `'k-12' | 'higher-ed' | 'self-learner'`.
The handler accepts a fourth value, `'school'`, and the insert path deliberately swallows database
errors, so **every "School" choice made on `/get-started` since that path shipped has been silently
discarded**. [HS.2](HS.2-www-marketing-site-rebrand.md) is about to start sending `'homeschool'`,
which would be discarded the same way. This plan fixes the constraint, adds the new value, keeps the
old one for history, and cleans up the copy.

## 2. Goals

- Accept and persist `program = 'homeschool'`, and fix the pre-existing `'school'` drop, in one
  migration.
- Preserve `'self-learner'` as a readable historical value — no rewriting of past analytics rows.
- Make the silent-drop failure mode observable so a future enum mismatch is not invisible again.
- Update the three user-visible server strings and the LLM system prompt to the HS.1 terms.
- Update Go doc comments and feature-flag comments, leaving applied migration files untouched.

## 3. Non-Goals

- No feature-flag **key** renames (`FFOnboardingFlow`, `FFStripeBilling`, `FFGamification`,
  `FFAIStudyBuddy`, `FFLearningPaths`, `FFSelfPacedMode` all keep their names and their
  `Settings → Global platform` keys). Admin **labels** live in the web client — see
  [HS.3](HS.3-web-client-rebrand.md).
- No change to `self.lextures.com` anywhere in server config, deploy workflow, or CORS origins.
- No backfill or rewrite of existing `onboarding_events` rows.
- No edits to previously applied migration files (`277`, `278`, `279`, `284`, `287`, `289`, `292`,
  `296`), even though their **comments** mention self-learners. Applied migrations are immutable by
  convention; those comment mentions are allowlisted in HS.1.
- No change to "self-paced" as a product term — `FFSelfPacedMode`, migration `277_self_paced_mode`,
  and the `self-paced` course setting are unrelated to this rename.
- No new endpoint, no auth change.

## 4. Personas & User Stories

- **As a growth analyst**, I want every `/get-started` choice — including School and Homeschool — to
  land in `onboarding_events` so funnel numbers are not silently wrong.
- **As a homeschooling learner using the AI study buddy**, I want the assistant's framing to match how
  the product describes me.
- **As a compliance reviewer** reading the AI-disclosure page, I want the feature descriptions to use
  current product vocabulary.
- **As a learner opening the intro course**, I want the "Who it's for" line to name homeschoolers.
- **As a backend engineer**, I want a failed analytics insert to show up in logs/metrics rather than
  disappear.

## 5. Functional Requirements

### Onboarding program value

- **FR-1.** A new migration `server/migrations/433_onboarding_program_homeschool.sql` MUST replace the
  `onboarding_events_program_check` constraint so the allowed set is
  `('k-12', 'higher-ed', 'self-learner', 'homeschool', 'school')` — adding `'homeschool'` and
  `'school'`, retaining `'self-learner'` for history.
- **FR-2.** A companion `433_onboarding_program_homeschool.down.sql` MUST restore the previous
  constraint, and MUST be written so it fails loudly rather than silently dropping rows that use the
  new values (documented in the file header: the down migration requires the operator to purge or
  remap `'homeschool'`/`'school'` rows first).
- **FR-3.** `onboarding_http.go:97` MUST accept `"homeschool"` in addition to the current four values.
  `"self-learner"` MUST remain accepted until [HS.2](HS.2-www-marketing-site-rebrand.md) has been
  deployed for one full analytics reporting period, then MAY be removed from the accept list (it
  stays allowed in the CHECK constraint regardless).
- **FR-4.** The insert path (`onboarding_http.go:130`) MUST keep swallowing errors from the client's
  point of view — an unauthenticated endpoint must not leak internal state — but MUST NOT swallow
  them from ours: it MUST log at `warn` with the rejected `program` value and increment a counter
  metric.
- **FR-5.** The metric MUST be registered in `server/internal/telemetry` alongside the existing
  counters, named `onboarding_event_insert_failed_total`, labelled by `program`.
- **FR-6.** Reporting queries and any dashboard over `onboarding_events` MUST treat `'self-learner'`
  and `'homeschool'` as the same segment for cross-cutover comparisons; document the mapping in the
  migration header comment.

### Copy

- **FR-7.** `aidisclosure/disclosure.go:77` description MUST read
  `Standalone study companion for homeschoolers.`
- **FR-8.** `studybuddy/prompt.go:10` MUST replace "You help self-learners understand course
  material…" with homeschool-neutral phrasing. Proposed:
  `You help learners understand course material, review concepts, and stay on track with their goals.`
  — dropping the segment noun entirely, since the study buddy also serves school learners.
- **FR-9.** Package doc comments MUST be updated: `service/onboarding/onboarding.go:1`,
  `service/coursereviews/service.go:2`, `service/studybuddy/service.go:1`,
  `repos/learnergoals/learnergoals.go:1`.
- **FR-10.** `config.go` flag comments at lines 518, 576, 591, 600, 606 MUST say "homeschool" where
  they currently say "self-learner". Flag **identifiers** do not change.

### Seeded course content

- **FR-11.** The "Who it's for" lines MUST be updated in
  `service/introcourse/content/en/syllabus.json`,
  `service/marketplacecourses/content/ai-essentials/en/syllabus.json`, and
  `service/marketplacecourses/content/introduction-to-python/en/syllabus.json`
  (`self-learners` → `homeschoolers`). Non-English locale variants of those files MUST be updated in
  the same PR where they exist.
- **FR-12.** `introcourse.ContentVersion` MUST be bumped (`3` → `4`) — content sync no-ops when the
  stored version matches, so an unbumped edit never reaches existing installs.
- **FR-13.** `content_version` in `content/ai-essentials/course.yaml` and
  `content/introduction-to-python/course.yaml` MUST be bumped (`1` → `2`) for the same reason.
- **FR-14.** `harness-smoke` and `personal-finance` MUST NOT be touched (no self-learner mention).

## 6. Non-Functional Requirements

- **Performance** — the migration is a constraint swap on a small analytics table; it takes an
  `ACCESS EXCLUSIVE` lock briefly. Use `ADD CONSTRAINT … NOT VALID` + `VALIDATE CONSTRAINT` if the
  table has grown beyond ~1M rows at deploy time; otherwise a plain drop/add is fine.
- **Security** — the tracking endpoint stays unauthenticated and rate-limited (`onboardingCheckRate`,
  10-minute window). The new log line MUST NOT include IP, referrer, or user agent — only the
  rejected `program` value, which is server-validated to a closed set.
- **Privacy & Compliance** — no new personal data. `onboarding_events` already stores IP, UA, and
  referrer; retention and RoPA entries are unchanged. The AI-disclosure text change is a
  transparency-surface edit — note it in the AI disclosure changelog if one is maintained.
- **Accessibility** — n/a (no UI).
- **Scalability** — n/a.
- **Reliability** — content resync (FR-12/FR-13) runs under an advisory lock and is idempotent;
  re-running provisioning must not duplicate courses.
- **Observability** — FR-4/FR-5 are the observability deliverable: a rejected insert becomes a warn
  log plus a counter, so the next enum drift is visible within one dashboard refresh.
- **Maintainability** — the allowed-value list now exists in two places (Go switch + SQL CHECK);
  the migration header MUST cross-reference `onboarding_http.go` and vice versa.
- **Internationalization** — the study-buddy prompt is English; localized syllabi are updated per
  FR-11 where they exist.
- **Backward compatibility** — old clients still send `'self-learner'` and still succeed; the value
  stays in the CHECK constraint permanently.

## 7. Acceptance Criteria

- **AC-1.** *Given* migration 433 applied, *When* `POST /api/v1/public/onboarding/track` is called
  with `program: "homeschool"`, *Then* the response is `204` **and** a row exists in
  `onboarding_events` with that program.
- **AC-2.** *Given* the same, *When* called with `program: "school"`, *Then* a row is persisted —
  closing the pre-existing silent drop.
- **AC-3.** *Given* the same, *When* called with `program: "self-learner"`, *Then* a row is persisted
  (backward compatibility for the pre-HS.2 site).
- **AC-4.** *Given* `program: "bogus"`, *Then* the response is `400` and no row is written.
- **AC-5.** *Given* an insert that violates the constraint (simulated), *Then* the client still gets
  `204`, a `warn` log names the program, and
  `onboarding_event_insert_failed_total{program="…"}` increments.
- **AC-6.** *Given* the AI-disclosure endpoint/page, *Then* the study-buddy entry reads "Standalone
  study companion for homeschoolers."
- **AC-7.** *Given* a study-buddy conversation, *Then* the system prompt contains no "self-learner"
  substring.
- **AC-8.** *Given* an install with the intro course already provisioned, *When* the API restarts
  after this change, *Then* the syllabus resyncs (version 3 → 4) and the overview shows the new
  audience line.
- **AC-9.** *Given* `go test ./... -count=1` and `golangci-lint run ./...` in `server/`, *Then* both
  pass.
- **AC-10.** *Given* `rg -i 'self.?learn' server/` excluding `server/migrations/[0-9]*_*.sql`,
  *Then* there are zero matches.

## 8. Data Model

**Changed table** — `onboarding_events` (from `142_onboarding_events.sql`).

```sql
-- server/migrations/433_onboarding_program_homeschool.sql
-- 'self-learner' is retained for historical rows written before the Homeschool rebrand (HS.5).
-- 'school' was accepted by onboarding_http.go but rejected by the original CHECK, so those
-- events were silently dropped; adding it here closes that gap.
-- Keep this list in sync with the switch in server/internal/httpserver/onboarding_http.go.
ALTER TABLE onboarding_events DROP CONSTRAINT onboarding_events_program_check;
ALTER TABLE onboarding_events ADD CONSTRAINT onboarding_events_program_check
  CHECK (program IN ('k-12', 'higher-ed', 'self-learner', 'homeschool', 'school'));
```

- Indexes/constraints: `onboarding_events_program_idx` unchanged.
- Naming convention: `server/migrations/NNN_*.sql` + `NNN_*.down.sql` (next free number is **433**;
  confirm at implementation time).
- **Backfill: none.** Existing `'self-learner'` rows stay as written; rewriting them would corrupt
  the historical record of what the site actually offered on those dates.

## 9. API Surface

No new routes. One widened enum on an existing route:

```
POST /api/v1/public/onboarding/track      auth: none (rate-limited, 2 KB max body)
  request:  { program, school_name?, language?, timezone?,
              screen_width?, screen_height?, referrer? }
  program:  'k-12' | 'higher-ed' | 'school' | 'homeschool'
            | 'self-learner'   (deprecated, still accepted)
  response: 204 No Content | 400 invalid program | 429 rate limited
```

- Rate limit unchanged (`onboardingRateLimit` per IP per `onboardingRateWindow`).
- OpenAPI: update the `program` enum and mark `self-learner` deprecated with a note pointing at this
  plan.

## 10. UI / UX

No server-rendered UI. Three strings surface in clients:

| String | Surfaces in |
|---|---|
| AI-disclosure study-buddy description | web + mobile AI transparency screens |
| Study-buddy system prompt | not shown to users; shapes assistant tone |
| Syllabus "Who it's for" lines | intro course + two marketplace course overviews, web + mobile |

No empty/loading/error state changes. Copy comes from the
[HS.1 deck](HS.1-terminology-copy-deck-and-guardrails.md#10-ui--ux).

## 11. AI / ML Considerations

- **Model(s)** — whatever the tenant's configured provider is (BYOK; see
  [AP.2](../ai-providers/AP.2-credential-store-and-byok.md)). No model change.
- **Prompt** — `studybuddy/prompt.go` `systemPromptTemplate` is edited (FR-8). This is model-visible
  context on every turn, so it is a behavioural change, not a copy change.
- **Eval** — before/after comparison on the existing study-buddy prompt fixtures: run the same set of
  learner turns through old and new prompts and confirm no regression in groundedness or refusal
  behaviour. Dropping the segment noun is the lowest-risk edit precisely because it removes an
  audience assumption rather than substituting a narrower one.
- **Fallback** — unchanged; prompt assembly has no new failure mode.
- **PII redaction** — unchanged; the prompt still interpolates display name and course title only.
- **Cost budget** — the replacement is shorter than the original; token cost is neutral-to-lower.

## 12. Integration Points

- Internal: `httpserver/onboarding_http.go`, `repos/onboardingevent/onboardingevent.go`,
  `aidisclosure/disclosure.go`, `service/studybuddy/{prompt,service}.go`,
  `service/{onboarding,coursereviews}/`, `repos/learnergoals/`, `config/config.go`,
  `service/introcourse/**`, `service/marketplacecourses/**`, `internal/telemetry` (new counter).
- External: none.
- Emissions: no new events; one new metric.
- Consumers: `www` `/get-started` beacon ([HS.2](HS.2-www-marketing-site-rebrand.md)).

## 13. Dependencies & Sequencing

- Must ship after: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md).
- Must ship **before** [HS.2](HS.2-www-marketing-site-rebrand.md) flips its `program` constant —
  otherwise the new value is accepted by neither the handler nor the constraint and the events
  vanish. HS.2 may ship its copy changes first (FR-11 there keeps the old value behind a constant).
- Must ship before: [HS.6](HS.6-docs-compliance-and-e2e-metadata.md).
- Order within this plan: schema (migration) → code (handler + copy) → content version bumps.
- Shared infra: Postgres migration runner (`RUN_MIGRATIONS=true`), metrics pipeline.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| HS.2 deploys the new `program` value before this migration lands → events silently dropped | M | **H** | Explicit ordering in §13; HS.2 FR-11 gates the flip behind a one-line constant; AC-1 is the go signal |
| Down migration run while `homeschool` rows exist → constraint fails or data must be discarded | L | M | FR-2 documents the required purge/remap and fails loudly rather than silently |
| Prompt edit changes study-buddy behaviour in an unmeasured way | M | M | §11 eval on existing fixtures before merge |
| Content version not bumped → reworded syllabi never reach existing installs | M | M | FR-12/FR-13 called out separately; AC-8 verifies on a provisioned install |
| Someone "cleans up" the old migration files' comments | M | L | §3 non-goal + HS.1 allowlist entry for `server/migrations/*.sql` |
| Analytics dashboards split the segment across two values post-cutover | H | L | FR-6 documents the mapping; note the cutover date alongside HS.2's GA4 note |
| Renaming `FFSelfPacedMode` by association | L | M | §3 non-goal: "self-paced" is a different product concept |

## 15. Rollout Plan

- Feature flag: none. The widened enum is strictly additive and the copy edits are not gate-worthy.
- Migration sequencing: **schema → code → content-version bump → (then HS.2 flips its constant).**
  No backfill step.
- Dogfood: apply 433 on staging, POST all five program values, confirm five rows; restart the API and
  confirm the intro-course resync.
- GA criteria: AC-1…AC-10 green on staging, then production; `onboarding_event_insert_failed_total`
  flat at zero for 24 h.
- Rollback: revert the code; leave the migration applied. A widened CHECK constraint is harmless to
  an older binary — it simply never sends the new values. Only run the down migration if the table
  must be restored exactly, and only after handling new-value rows per FR-2.

## 16. Test Plan

- **Unit** — `onboarding_http` handler table test over all five accepted values plus rejects;
  `disclosure` catalogue snapshot; `studybuddy` prompt assembly asserts the absence of the old
  substring; syllabus fixture validation (`marketplacecourses.ValidateAllCourses`).
- **Integration** (needs DB, `make test`) — insert each accepted `program` against the migrated
  schema and assert persistence; assert `'bogus'` is rejected at the handler before reaching the DB;
  assert the failure counter increments when an insert is forced to fail.
- **End-to-end** — Playwright/API: hit the public tracking endpoint with `homeschool`, then read back
  via a DB fixture; provision an install at `ContentVersion = 3` and assert resync to `4`.
- **Security** — confirm the endpoint still returns `204` on internal failure (no state leak) and
  that the new log line contains no IP/UA/referrer; re-run the rate-limit test.
- **Accessibility** — n/a.
- **Performance / load** — measure the constraint swap on a staging copy sized to production;
  if lock time exceeds ~200 ms, switch to `NOT VALID` + `VALIDATE`.
- **Manual exploratory** — read the AI-disclosure page and the three course overviews in the web app
  after resync.

## 17. Documentation & Training

- End-user docs: none.
- Admin/instructor docs: none (flag labels are HS.3).
- API reference: update the OpenAPI `program` enum and its deprecation note
  (`docs/api-changelog-*.md` if the endpoint is listed there).
- Internal runbook: add a note that `onboarding_events.program` values must be kept in sync between
  `onboarding_http.go` and the CHECK constraint, and that a mismatch is now alarmed by
  `onboarding_event_insert_failed_total`.

## 18. Open Questions

1. How long do we keep accepting `'self-learner'` from clients (FR-3)? Proposal: one full reporting
   period after HS.2 is live, then remove from the handler while keeping it in the CHECK constraint.
2. Should historical `'self-learner'` rows be *mapped* in the reporting layer (a view) rather than
   left for each query to handle? A view is cheap and removes a recurring footgun.
3. Does the study-buddy prompt change need a formal eval sign-off, or is the fixture comparison in
   §11 sufficient?
4. Is there an AI-disclosure changelog that must record the FR-7 wording change?
5. Are there non-English variants of the three syllabi in-tree at implementation time (FR-11)?
6. Is `433` still the next free migration number when this lands?

## 19. References

- Existing files this work touches: `server/internal/httpserver/onboarding_http.go`,
  `server/internal/repos/onboardingevent/onboardingevent.go`,
  `server/migrations/142_onboarding_events.sql` (read-only reference),
  `server/migrations/433_onboarding_program_homeschool.sql` (new),
  `server/internal/aidisclosure/disclosure.go`, `server/internal/service/studybuddy/{prompt,service}.go`,
  `server/internal/service/onboarding/onboarding.go`, `server/internal/service/coursereviews/service.go`,
  `server/internal/repos/learnergoals/learnergoals.go`, `server/internal/config/config.go`,
  `server/internal/service/introcourse/{content_version.go,content/en/syllabus.json}`,
  `server/internal/service/marketplacecourses/content/{ai-essentials,introduction-to-python}/**`.
- External standards: RFC 2119; PostgreSQL `ALTER TABLE … VALIDATE CONSTRAINT` locking semantics.
- Related plans: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md),
  [HS.2](HS.2-www-marketing-site-rebrand.md), [HS.3](HS.3-web-client-rebrand.md),
  [15.11 onboarding & diagnostic](../15-self-learner-specific/15.11-onboarding-diagnostic.md),
  [15.12 AI study buddy](../15-self-learner-specific/15.12-ai-study-buddy.md),
  [17.7 observability](../17-platform-performance-operability/) (metric registration
  pattern in `server/internal/telemetry`).

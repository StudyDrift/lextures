# CC.10 — Guidance, Assisted Fixes, Analytics & Rollout

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.10 |
| **Section** | Course Checklist |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) per phase; M (2–4w) overall |
| **Owner (proposed)** | Platform + web + instructional design |
| **Depends on** | CC.2, CC.3, CC.4, CC.5, CC.6, CC.7 |
| **Unblocks** | — (closes the section) |

---

## 1. Problem Statement

CC.1–CC.9 build a checklist that can tell an instructor what is wrong and take them to the fix. Three things
are still missing before it earns its place in the navigation. First, **guidance**: an instructor told "map
every assessment to an outcome" without knowing why or how will dismiss it. Second, **assisted fixes**: the
two rules with the largest backlogs (outcome mapping, rubrics on high-stakes work) have existing AI
affordances in the product that the checklist should route to instead of leaving a wall of manual work.
Third, **evidence that the rules are right**: without per-rule pass and dismissal telemetry we cannot know
whether a heuristic is wrong, and the tier-promotion gates specified in CC.3–CC.6 have nothing to gate on.
CC.10 delivers guidance content, the assisted-fix action slot, the analytics that make promotion decisions
possible, and the plan for taking the checklist from "informational" to "badged".

## 2. Goals

- Author **per-item help content** (`HelpRef` destinations) so every rule explains why it matters and how to
  satisfy it.
- Fill the **action slot** CC.7/CC.9 reserved with a small set of high-leverage assisted fixes that reuse
  existing AI paths, always with human approval.
- Ship the **event dictionary and reporting queries** that answer: which rules pass, which are dismissed,
  which are disagreed with, which targets fail to resolve.
- Execute the **tier-promotion programme** — the mechanism by which the nav badge starts showing numbers —
  with objective per-rule gates.
- Close the section with runbooks and a decision record for the "no feature flag" posture.

## 3. Non-Goals

- No new checklist rules (CC.3–CC.6 own the catalog).
- No new AI capability: CC.10 wires existing affordances, it does not build a model, prompt library or
  provider integration.
- No org-level rollup dashboard for administrators (deferred; see §18 Q2).
- No automated remediation without review — nothing writes to a course without an explicit human confirm.
- No push/email nudges in this plan (§18 Q3).

## 4. Personas & User Stories

- **As an instructor**, I want each item to link to a short explanation of why it matters, so that the
  checklist teaches me rather than nagging me.
- **As an instructor facing 11 unmapped assessments**, I want a "suggest mappings" action that proposes
  links I can approve or reject, so that a 40-minute chore becomes a 5-minute review.
- **As an instructor with a rubric-less capstone**, I want the checklist to hand me the existing
  Build-with-AI rubric flow, so that I do not have to find it.
- **As a product manager**, I want to know that `syllabus.late-policy` is dismissed as "disagree" by 30% of
  instructors, so that I can fix the heuristic instead of shipping a bad nag.
- **As an engineer**, I want to know when a checklist target stops resolving, so that a page refactor does
  not silently break navigation.
- **As a support lead**, I want a runbook for "an instructor says the checklist is wrong", so that the
  response is a rule fix rather than an apology.

## 5. Functional Requirements

### Guidance content

- **FR-1.** Every registered checklist item MUST have a `HelpRef` that resolves to a real help destination;
  a registry test MUST fail on a dangling reference.
- **FR-2.** Help content per item MUST cover: what the check looks at, why it matters (with the rubric
  citation), how to satisfy it, and when it is reasonable to dismiss it.
- **FR-3.** Item help MUST be reachable from the item row without leaving the page (a popover or side sheet
  on web, a sheet on mobile) and MUST also be a linkable URL for support.
- **FR-4.** The **sources** chips (`QM 3.1`, `OSCQR 45`, `NSQ C`, `WCAG 1.1.1`) MUST link to
  [course-design-research.md](course-design-research.md), which MUST carry the full rule-to-standard mapping
  for all four packs and short, non-infringing summaries with links to the canonical sources.

### Assisted fixes (action slot)

- **FR-5.** The item action slot MUST support an optional primary action declared in the registry as
  `Action { kind, labelKey, endpoint }`, rendered by CC.7 and CC.9. Items without an action render nothing.
- **FR-6.** `outcomes.assessment-mapping` MUST offer **"Suggest outcome mappings"**: an AI proposal of
  `(item → outcome, measurement_level)` links, presented as a reviewable list where each proposal is
  individually accepted or rejected, with nothing written until the instructor confirms.
- **FR-7.** `feedback.rubrics-on-high-stakes` MUST offer **"Build a rubric with AI"**, routing to the
  existing assignment rubric generation flow pre-scoped to the selected assignment.
- **FR-8.** `orientation.welcome-message` MUST offer **"Draft a welcome announcement"**, opening the
  announcement composer pre-filled with a draft the instructor edits and posts — never auto-posted.
- **FR-9.** `a11y.image-alt-text` MUST offer **"Suggest alt text"** only where the platform already has that
  capability, and MUST require per-image approval.
- **FR-10.** Every assisted fix MUST: honour the course's AI opt-out settings and the org's AI policy, be
  attributed in the AI-disclosure surface, respect the existing token budgets, and be a no-op (action hidden)
  when AI is unavailable for the course.
- **FR-11.** No assisted fix may write to the course without an explicit confirm; every write MUST be audited
  through the existing course audit trail with `source = checklist_assist`.
- **FR-12.** Assisted-fix failures MUST degrade to the manual path (navigate to the target) with a plain
  message, never a dead end.

### Analytics

- **FR-13.** An **event dictionary** MUST be published at `checklist-event-dictionary.md` covering every
  client and server event, its fields, and its retention.
- **FR-14.** Server MUST emit, per evaluation: per-item status counts (aggregated, not per course in the
  hot path) so pass-rate per rule is chartable.
- **FR-15.** Client MUST emit: `checklist_viewed`, `checklist_item_expanded`, `checklist_evidence_clicked`,
  `checklist_target_navigated{anchorId, resolved}`, `checklist_item_dismissed{itemId, reason}`,
  `checklist_item_restored`, `checklist_item_rechecked`, `checklist_refreshed`,
  `checklist_assist_started{itemId}`, `checklist_assist_accepted{itemId, acceptedCount, proposedCount}`.
- **FR-16.** Events MUST contain **no PII and no evidence content** — item IDs, statuses, counts and
  anchor IDs only. The two accommodation rules (CC.5 FR-21/FR-22) MUST be excluded from any event carrying
  counts.
- **FR-17.** A reporting document MUST provide the SQL for: pass rate per rule, dismissal rate per rule by
  reason, time-to-completion per rule, badge-count distribution, target-resolution failure rate, and
  assisted-fix acceptance rate.
- **FR-18.** An **analytics inventory** entry MUST be filed per the S05 convention used by the pinned-settings
  work, recording data collected, purpose, retention and lawful basis.
- **FR-19.** Alerting MUST cover: `checklist_target_navigated{resolved=false}` > 1% over 24 h; a rule's
  `disagree` dismissal rate > 20% over 7 days; snapshot miss ratio > 40% (from CC.2).

### Promotion programme

- **FR-20.** Every rule ships `recommended` (CC.3–CC.6 FR). Promotion to `essential` MUST require, per rule:
  ≥ 200 courses evaluated; manual plausibility review of a 20-finding sample; `disagree` dismissal < 10%;
  `done_elsewhere` dismissal < 15%; and, for accessibility rules, accessibility-owner sign-off.
- **FR-21.** Promotions MUST be batched into at most one release per two weeks and announced in-product via
  the existing banner system, because a promotion is what makes a badge number appear.
- **FR-22.** A **demotion path** MUST exist and be exercised at least once in staging: moving a rule back to
  `recommended`, or into `RETIRED_ITEM_IDS`, in a server-only release.

## 6. Non-Functional Requirements

- **Performance** — Analytics emission MUST be fire-and-forget and MUST NOT block rendering or the API.
  Server-side aggregation MUST be batched, not per-request. Assisted fixes run asynchronously with a
  progress state; they MUST NOT block the checklist page.
- **Security** — Assisted fixes reuse existing authenticated, authorised endpoints; no new write surface is
  created beyond the confirm-then-apply endpoints the underlying features already expose. Proposals are
  rendered as text, never as HTML.
- **Privacy & Compliance** — FR-16 (no PII, no evidence content) is a hard constraint. AI assists send course
  content to a provider; they MUST therefore respect `ai_processing_opt_out`, the org's provider allow-list,
  and the AI-disclosure obligations — the same path the existing Build-with-AI features use. The analytics
  inventory (FR-18) is required for the compliance evidence programme in `docs/plan/standards/`.
- **Accessibility** — Help popovers/sheets are focus-managed and dismissible with Escape; assisted-fix
  proposal lists are keyboard-operable with per-proposal accept/reject controls that have distinct
  accessible names; progress is announced politely.
- **Scalability** — Reporting queries MUST run against the events table with appropriate indexes and MUST be
  bounded by time range; the dashboards are scheduled, not live.
- **Reliability** — Analytics loss is acceptable (fire-and-forget); assisted fixes are idempotent per
  proposal so a retry cannot double-write.
- **Observability** — This plan *is* the observability plan for the section; it additionally defines the
  alerts in FR-19.
- **Maintainability** — Help content lives with the docs, not in code; the registry stores only a reference.
- **Internationalization** — Help content English-first with the existing translation pipeline; assisted-fix
  output must be generated in the course's language.
- **Backward compatibility** — Actions are optional in the registry; clients that do not know an action kind
  render nothing rather than erroring.

## 7. Acceptance Criteria

- **AC-1.** *Given* any registered item, *Then* the registry test confirms its `HelpRef` resolves.
- **AC-2.** *Given* an item row, *When* help is opened, *Then* the content covers what/why/how/when-to-dismiss
  and is reachable at a stable URL.
- **AC-3.** *Given* `outcomes.assessment-mapping` with 11 unmapped items, *When* "Suggest outcome mappings"
  runs, *Then* a reviewable proposal list appears and **no** `course_outcome_links` row exists until the
  instructor confirms.
- **AC-4.** *Given* the instructor accepts 7 of 11 proposals, *Then* exactly 7 links are written, the audit
  trail records `source = checklist_assist`, and the item recomputes to `20 / 24`.
- **AC-5.** *Given* a course with `ai_processing_opt_out = true`, *Then* no assisted-fix action renders on any
  item.
- **AC-6.** *Given* an assisted fix fails, *Then* the user is offered the manual target and no partial write
  occurred.
- **AC-7.** *Given* a dismissal, *Then* `checklist_item_dismissed` is emitted with `itemId` and `reason` and
  **no** course, user or evidence identifier beyond what the analytics inventory declares.
- **AC-8.** *Given* the accommodations rules, *Then* no analytics event is emitted for them at all
  (asserted).
- **AC-9.** *Given* the reporting document, *When* each query is run against a seeded events table, *Then* it
  returns the documented shape.
- **AC-10.** *Given* a rule whose `disagree` rate exceeds 20% for 7 days, *Then* the alert fires.
- **AC-11.** *Given* a rule meeting all FR-20 gates, *When* it is promoted, *Then* the badge count increases
  for affected courses and the in-product announcement is shown once per user.
- **AC-12.** *Given* a demotion in staging, *Then* the rule leaves the badge count within one snapshot TTL
  and no client release is required.

## 8. Data Model

No new tables beyond what CC.2 created. CC.10 uses:

- `course.course_checklist_events` (CC.2) as the source for dismissal-rate reporting.
- The existing product-analytics event pipeline for client events (no bespoke store).
- The existing course audit trail for assisted-fix writes, with a new `source` value `checklist_assist`
  (an enum/string addition, not a schema change if the column is free-text; otherwise a one-line migration).

Retention: client events follow the platform analytics retention; `course_checklist_events` follows the
400-day policy set in CC.2 §8.

## 9. API Surface

No new checklist routes. Assisted fixes call **existing** endpoints:

| Action | Endpoint reused |
|---|---|
| Suggest outcome mappings | A new proposal endpoint is required — `POST /api/v1/courses/{course_code}/outcomes/suggest-links` returning proposals only (no writes), then the existing outcome-link create endpoint per accepted proposal |
| Build a rubric with AI | Existing assignment rubric generation flow |
| Draft a welcome announcement | Existing feed message composer with an AI draft (no auto-post) |
| Suggest alt text | Existing alt-text assistance where available |

The one new endpoint (`suggest-links`) MUST be read-only, staff-guarded, rate-limited, documented in OpenAPI,
and MUST return proposals with a confidence and a rationale string for human review.

## 10. UI / UX

- **Help**: an "About this check" affordance on each item opens a popover (web) / sheet (mobile) with the
  four-part content and the source citations. Escape closes; focus returns to the trigger.
- **Action slot**: at most one primary action per item, rendered as a secondary button on the item row.
  Label is a verb ("Suggest mappings", "Build a rubric", "Draft announcement").
- **Proposal review**: a list where each proposal shows the item, the proposed outcome, the measurement
  level, a one-line rationale, and Accept / Reject. A header shows "7 of 11 selected" with Accept-all /
  Reject-all. A single confirm applies the accepted set. AI-generated content is labelled as such per the
  platform's AI-disclosure conventions.
- **Empty/failed states**: assist unavailable → action hidden, not disabled. Assist failed → inline message
  plus the manual target link.

## 11. AI / ML Considerations

- **Models & routing** — Reuse the platform's existing provider abstraction and per-org model policy; no new
  provider integration. Course-setup-class tasks already route through a configured default model.
- **Prompts** — Outcome-mapping proposals receive the outcome titles/descriptions and the assessment titles,
  instructions and (where short) rubric criteria — **not** learner data, submissions or grades. Welcome-draft
  receives course title, description and dates only.
- **Eval metric** — Acceptance rate per assist (`accepted / proposed`), tracked by FR-15. Target ≥ 60%
  acceptance for outcome mappings before the assist is promoted from opt-in to prominent; below 40% the
  assist is withdrawn rather than tuned indefinitely.
- **Fallback path** — Any failure, timeout, opt-out or budget exhaustion hides or fails the action and leaves
  the manual path (FR-12).
- **PII redaction** — Prompts MUST NOT include student names, submissions, grades, accommodations or
  enrollment data. A test asserts the prompt builders reject those fields.
- **Cost budget** — Assists are per-course, human-initiated and bounded (≤ 200 items per suggestion run);
  they inherit the existing token budgets and are counted in AI usage reporting.
- **Disclosure** — Every assisted output is labelled AI-generated and recorded in the AI-disclosure surface.

## 12. Integration Points

- Internal: `server/internal/service/coursechecklist` (action declarations),
  `server/internal/repos/courseoutcomes` (proposal endpoint), the existing rubric-generation and
  announcement-composer paths, `server/internal/aidisclosure`, AI provider/budget plumbing,
  `server/internal/telemetry`, the product-analytics pipeline.
- Clients: CC.7's action slot and help affordance; CC.9's equivalents.
- Docs: `docs/plan/checklist/course-design-research.md`, `checklist-event-dictionary.md`,
  `checklist-reporting.md`, `checklist-analytics-inventory.md`, `docs/runbooks/course-checklist.md`.

## 13. Dependencies & Sequencing

- Must ship after: CC.2 and at least CC.3; the assists depend on CC.4 (`outcomes.assessment-mapping`) and
  CC.5 (`feedback.rubrics-on-high-stakes`) being live.
- **Phase 1** (guidance + analytics) SHOULD ship with CC.7 so the first release is measurable.
- **Phase 2** (assisted fixes) after CC.4/CC.5.
- **Phase 3** (promotion programme) last, gated on Phase 1 data.
- CC.4's `outcomes.assessment-mapping` MUST NOT be promoted to `essential` until the mapping assist ships
  (CC.4 §15).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| AI-proposed outcome mappings are plausible but wrong, and instructors accept them wholesale | **H** | H | Per-proposal accept/reject (no accept-all default), rationale shown, confidence surfaced, acceptance-rate monitoring, audit trail with `checklist_assist` source so bad batches are traceable and reversible |
| Analytics accidentally captures evidence content (student names) | M | **H** | FR-16 hard rule; accommodations rules excluded entirely (AC-8); payload schema test |
| Promotion makes badges appear suddenly and feels like a regression | M | M | FR-21 batched promotions + in-product announcement; gates require low disagreement first |
| Help content rots as rules change | M | M | FR-1 dangling-reference test; help authored in the same PR as the rule |
| Assists increase AI spend unpredictably | M | M | Human-initiated only, bounded batch size, existing token budgets, usage reporting |
| The "no feature flag" posture leaves no fast lever if the feature is badly received | M | H | Documented levers: retire rules (server-only), demote tiers, `CHECKLIST_SNAPSHOT_TTL`, link-check kill switch; ADR records the trade-off explicitly |

## 15. Rollout Plan

**No feature flag** — recorded as a deliberate decision in the section ADR, with these levers instead:
rule retirement, tier demotion, the link-check env kill switch (CC.6), and TTL tuning (CC.2).

- **Phase 1 — Guidance & measurement** (with CC.7): help content for every shipped rule, event dictionary,
  client + server events, reporting queries, analytics inventory, alerts. All rules `recommended`; badge
  reads 0. Exit: dashboards populated for ≥ 200 courses.
- **Phase 2 — Assisted fixes** (after CC.4/CC.5): the four assists in FR-6–FR-9, opt-in placement, acceptance
  tracked. Exit: outcome-mapping acceptance ≥ 60%; zero PII findings in prompt audits.
- **Phase 3 — Promotion** (after Phase 1 data): promote rules meeting FR-20 gates in batches of ≤ 8, with an
  in-product announcement. Exit: badge live; support-ticket rate for "checklist is wrong" below an agreed
  threshold.
- **GA criteria for the section**: all four packs shipped; ≥ 20 rules promoted to `essential`; target
  resolution failure < 1%; no open privacy findings; runbooks published.
- **Rollback**: per-phase. Phase 3 is reversible by demotion within one snapshot TTL and needs no client
  release — this is the property that makes the no-flag posture acceptable.

## 16. Test Plan

- **Unit** — `HelpRef` resolution; action declaration validation; event payload schema (no PII, no evidence);
  accommodation-rule exclusion; prompt builders rejecting learner data.
- **Integration** — Proposal endpoint returns proposals and writes nothing; accepting N proposals writes
  exactly N links with audit rows; opt-out hides actions; budget exhaustion degrades gracefully.
- **End-to-end** — Playwright: run the mapping assist, accept a subset, confirm, verify the item's progress
  updates and the evidence rows shrink; verify no write occurred on cancel.
- **Security** — Authz on the proposal endpoint; rate limits; proposals rendered as text not HTML; audit
  trail completeness.
- **Accessibility** — Help popover and proposal list keyboard/screen-reader walkthroughs; accessible names on
  per-proposal controls; polite progress announcements.
- **Performance / load** — Assist latency budget and timeout behaviour; analytics emission adds no measurable
  render cost; reporting queries under a bounded time range on a large events table.
- **Manual exploratory** — Instructional-design review of all help content; AI-output review across three
  disciplines for the mapping assist; a deliberate demotion rehearsal in staging (FR-22).

## 17. Documentation & Training

- [course-design-research.md](course-design-research.md) — completed rule-to-standard mapping for all packs.
- `checklist-event-dictionary.md`, `checklist-reporting.md`, `checklist-analytics-inventory.md` — following
  the pinned-settings precedent.
- `docs/runbooks/course-checklist.md` — retire a rule, demote a tier, force-refresh, disable link checking,
  investigate "the checklist is wrong".
- `docs/adr/` — ADR recording the no-feature-flag decision, the levers that replace it, and the tiered
  promotion mechanism.
- Help-centre hub "Designing a good course" linking every item's guidance, with the rubric sources credited.
- Support enablement: what the checklist is, who sees it, how dismissal works, how to escalate a bad rule.

## 18. Open Questions

1. Should assisted fixes be gated on an org-level AI setting distinct from the general AI opt-out?
   Proposed: reuse the existing opt-out; revisit if orgs ask for finer control.
2. Is an admin-facing rollup ("checklist health across the department") in scope for section 18 (Admin
   Experience) rather than here? Proposed: yes, defer, and keep `Result` aggregatable per CC.1 §18 Q4.
3. Should we nudge by email/push ("your course starts in 3 days, 4 essential items remain")? Proposed: not
   in this section — it changes the feature from a tool into a nag and needs preference plumbing.
4. Should acceptance-rate thresholds withdraw an assist automatically, or trigger review? Proposed: trigger
   review; automatic withdrawal is surprising.
5. Do we publish the rule catalog publicly (marketing/trust surface) as evidence of quality tooling?
   Candidate for `docs/plan/20-docs-trust` follow-up.

## 19. References

- Precedent: [PS.4 suggested pins, telemetry & rollout](../../completed/settings/PS.4-suggested-pins-telemetry-and-rollout.md)
  and its companion [event dictionary](../../completed/settings/pinned-settings-event-dictionary.md),
  [reporting](../../completed/settings/pinned-settings-reporting.md) and
  [analytics inventory](../../completed/settings/pinned-settings-analytics-inventory.md) documents — CC.10
  follows the same four-document shape.
- Existing files this work touches: `server/internal/service/coursechecklist`,
  `server/internal/repos/courseoutcomes`, `server/internal/aidisclosure`, the rubric-generation and
  announcement-composer paths, `clients/web/src/pages/lms/course-checklist/`, mobile checklist features.
- External standards: [Quality Matters](https://www.qualitymatters.org/qa-resources/rubric-standards),
  [OSCQR](https://oscqr.suny.edu/), [NSQ](https://nsqol.org/the-standards/quality-online-courses/),
  [CAST UDL 3.0](https://udlguidelines.cast.org/).
- Related plans: all of [CC.1](../../completed/checklist/CC.1-checklist-registry-and-evaluation-engine.md) –
  [CC.9](CC.9-mobile-checklist-ios-and-android.md); AI provider plumbing in
  [`docs/completed/ai-providers/`](../../completed/ai-providers/);
  compliance evidence programme in [`../standards/`](../standards/).

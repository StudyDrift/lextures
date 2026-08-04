# CC — Course Checklist

A new **Checklist** entry directly under **Dashboard** inside a course, visible only to teachers-and-higher,
that answers one question: *is this course actually ready, and is it well designed?*

Items that are done are **crossed off**. Items that are not are **clickable** — either expanding a table of
the specific offending entities (the eleven assessments with no outcome mapping), or navigating straight to
the control that is wrong and **highlighting it**. Any item can be **dismissed** with a reason, which moves
it to the dismissed pile. The nav item carries a **badge** with the number of outstanding items, on web and
in both mobile apps. **There is no feature flag** — the checklist is always on for staff.

Every plan follows [_TEMPLATE.md](../_TEMPLATE.md).

## Plans

| ID | Plan | Layer | Effort | Depends on |
|---|---|---|---|---|
| CC.1 | [Checklist rule registry & evaluation engine](../../completed/checklist/CC.1-checklist-registry-and-evaluation-engine.md) ✅ | Server | M | — |
| CC.2 | [Checklist state, API & dismissals](../../completed/checklist/CC.2-checklist-state-api-and-dismissals.md) ✅ | Server | M | CC.1 |
| CC.3 | [Rule pack A — foundations, orientation, policies & people](../../completed/checklist/CC.3-rule-pack-foundations-and-orientation.md) ✅ (33 rules) | Server | M | CC.1, CC.2 |
| CC.4 | [Rule pack B — structure, content & outcome alignment](../../completed/checklist/CC.4-rule-pack-structure-outcomes-alignment.md) ✅ (22 rules) | Server | M | CC.1, CC.2 |
| CC.5 | [Rule pack C — assessment, grading, feedback & interaction](../../completed/checklist/CC.5-rule-pack-assessment-feedback-interaction.md) ✅ (26 rules) | Server | M | CC.1, CC.2 |
| CC.6 | [Rule pack D — accessibility, inclusive design & launch readiness](CC.6-rule-pack-accessibility-and-launch-readiness.md) (20 rules) | Server | M | CC.1, CC.2 |
| CC.7 | [Web: checklist page, nav entry & badge](CC.7-web-checklist-page-and-nav-badge.md) | Web | M | CC.2, CC.3 |
| CC.8 | [Deep-link & highlight targeting](CC.8-deep-link-and-highlight-targeting.md) | Web (+ shared) | M | CC.1, CC.7 |
| CC.9 | [Mobile: checklist on iOS & Android](CC.9-mobile-checklist-ios-and-android.md) | iOS + Android | M | CC.2, CC.7, CC.8 |
| CC.10 | [Guidance, assisted fixes, analytics & rollout](CC.10-analytics-guidance-and-rollout.md) | Cross-cutting | M | CC.2–CC.7 |

Supporting reference: **[course-design-research.md](course-design-research.md)** — the rubric-to-rule mapping
for all **101 rules**, and the sources they come from.

## Sequencing

```
CC.1 ──▶ CC.2 ──┬──▶ CC.3 ─┐
                ├──▶ CC.4 ─┤
                ├──▶ CC.5 ─┼──▶ CC.7 ──▶ CC.8 ──▶ CC.9
                └──▶ CC.6 ─┘        └──────────────┴──▶ CC.10
```

CC.1 and CC.2 are the critical path. The four rule packs are independent of each other and can be worked in
parallel once the snapshot loader change they share has landed. CC.7 needs at least CC.3 so the page is not
empty. CC.8 should land before any rule is promoted to `essential` — a badge that sends people to the wrong
place is worse than no badge.

## Design decisions worth knowing before reading

**Rules are code, not data.** A checklist item is a registry entry plus a pure evaluator function over an
in-memory course snapshot. Adding item #102 needs no migration, no table and no route — the same shape as
the [content-tools registry](../../completed/content_tools/CT.1-foundations-registry-and-data-model.md) and
the [pinned-settings registry](../../completed/settings/PS.1-settings-registry-and-addressable-controls.md).

**Every rule cites a standard.** The catalog is derived from Quality Matters, SUNY OSCQR, the National
Standards for Quality, CAST UDL 3.0 and WCAG 2.1 AA, plus operational go-live checklists for the things no
design rubric covers. A proposed rule with no source fails review.

**Deterministic, not AI.** All evaluators are heuristics, so evaluation is free, reproducible, offline-safe
and sends nothing to a model provider. AI appears only as optional, human-approved *fixes* in CC.10, reusing
affordances the product already has.

**No feature flag — but four levers.** The product decision is that the checklist is always on for staff.
The safety valves are structural instead: retire a rule (`RETIRED_ITEM_IDS`, server-only release), demote a
rule from `essential` to `recommended` (removes it from the badge within one snapshot TTL), tune
`CHECKLIST_SNAPSHOT_TTL`, and the one env kill switch on the outbound link checker. Every rule ships
`recommended` and is promoted only after per-rule dismissal telemetry clears the gates in
[CC.10 FR-20](CC.10-analytics-guidance-and-rollout.md).

**Teachers and higher.** Visibility requires the `course:{code}:item:create` capability (course owner,
teacher/instructor, designer) or an org/platform admin role. Students, TAs, observers, auditors, librarians
and parents get `403`, and the nav item is absent — not disabled — for them.

**Privacy by construction.** Rule pack B contains no learner data at all, so alignment evidence can be shown
to accreditors without a FERPA review. The two accommodation rules in pack C report counts and types only,
never a student. Nothing is cached to disk on mobile, and analytics carries item IDs and statuses only.

## What the checklist is not

It is not a gate — nothing is blocked. It is not QM certification, an OSCQR review, or a WCAG conformance
claim; it automates the machine-checkable subset of those standards and says so in its own help copy.

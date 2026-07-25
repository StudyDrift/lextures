# Adaptive Content Engine (ACE)

> **Tagline realized:** *"The learning environment that adapts."* ACE is the per-course loop that
> makes the *content itself* adapt to each learner: a **pre-assessment** measures where a student
> stands, the content is **rewritten or re-emphasized** for that student (introduce vs. reinforce vs.
> compress vs. remediate a misconception), and a **post-assessment** measures whether the adaptation
> actually helped. Every plan in this folder follows [`../_TEMPLATE.md`](../_TEMPLATE.md).

## Why this folder exists

Lextures already ships a deep adaptive-learning stack — but it adapts the *path* and the *questions*,
never the *teaching content*:

| Already shipped | What it adapts | Plan |
|---|---|---|
| Adaptive paths | Which whole module/item a learner sees next (skip / insert / unlock) | `../completed/01-adaptive-learning-core/1.4-adaptive-paths-across-modules.md` |
| Adaptive quizzes / CAT | Which *questions* a learner is asked | 1.6, `server/migrations/040_quiz_adaptive.sql` |
| Diagnostic / placement | Where a learner *starts* | `../completed/01-adaptive-learning-core/1.7-diagnostic-placement-assessments.md` |
| Misconception remediation | A remediation snippet *after a wrong answer* | `../completed/01-adaptive-learning-core/1.10-misconception-detection-remediation.md` |
| Recommendations | What to *do next* | `../completed/01-adaptive-learning-core/1.8-recommendations-engine.md` |

None of them rewrites a content page so that a struggling student gets more scaffolding and a
mastered student gets a compressed version, and none of them closes the loop with a
**post-assessment that proves the adaptation worked**. ACE is that missing layer. It reuses the
learner model (`course.learner_concept_states`), the concept graph, the misconception library, the
outcomes model, and the platform AI stack (`aiprovider` / `aigateway` / `contentpagegeneration` /
`analytics.ai_usage_log`) rather than reinventing them.

## Feature-flag philosophy (per-course, **not** global)

ACE is enabled **per course** by the instructor via a single boolean —
`course.courses.adaptive_content_enabled` (JSON `adaptiveContentEnabled`) — wired through the exact
same path as `adaptive_paths_enabled` and `misconception_detection_enabled`
(`server/internal/httpserver/course_features.go` → `models/course/types.go` →
`clients/web/src/lib/courses-api-schemas.ts`). There is **no required global platform on-switch**: a
course turns ACE on for itself. The only platform-level control is an **ops-only emergency
kill-switch** (`ADAPTIVE_CONTENT_KILL_SWITCH`, default *disengaged*) used solely for incident
response; when disengaged it never blocks a course. This directly honors the requirement that the
feature be flagged at the course level and not at the global platform level. Full rationale in
**[AC.1](../../completed/adaptive/AC.1-foundations-flag-and-data-model.md)** §15 and **[AC.8](../../completed/adaptive/AC.8-governance-safety-fairness-privacy.md)**.

## Conventions

- **File naming:** `AC.{N}-{kebab-slug}.md` (mirrors the `VC.`/`IQ.` per-course-feature folders).
- Every plan fills **all 19** template sections (no `…` placeholders) before it is "ready".
- **Backend:** service `server/internal/service/adaptivecontent/`, repo
  `server/internal/repos/adaptivecontent/`, HTTP `server/internal/httpserver/adaptive_content_*.go`,
  models `server/internal/models/adaptivecontent/`, routes under
  `/api/v1/courses/{course_code}/adaptive-content/*`.
- **Migrations** continue the global sequence. Highest existing on the working branch is `438_*`, so
  these plans reserve **`439_*` onward** (each story states its number). Renumber on merge if the
  sequence has advanced.
- **AI feature id** in `aigateway`: `adaptive_content` (plus per-axis sub-features), so every ACE
  model call is disclosed, budgeted, and logged to `analytics.ai_usage_log` like every other AI call.

## Severity legend

- **BLOCKER** — the loop cannot function / is unsafe to ship without it.
- **MAJOR** — the loop works but a market-critical capability or guardrail is missing.
- **MINOR** — parity, polish, or defence-in-depth.

## Story index

| ID | Plan | Severity | Depends on | Delivers |
|---|---|---|---|---|
| **AC.1** | [Foundations, course feature flag & data model](../../completed/adaptive/AC.1-foundations-flag-and-data-model.md) ✅ | BLOCKER | — | The `adaptive_content_enabled` flag, core tables, config API, kill-switch |
| **AC.2** | [Pre-assessment binding & adaptation profile](../../completed/adaptive/AC.2-pre-assessment-and-adaptation-profile.md) ✅ | BLOCKER | AC.1 | Entry-ticket binding + deterministic per-student adaptation profile |
| **AC.3** | [Adaptive content generation engine](../../completed/adaptive/AC.3-content-generation-engine.md) ✅ | BLOCKER | AC.1, AC.2 | AI variant generation with fidelity + safety checks |
| **AC.4** | [Generation pipeline, caching & cost controls](../../completed/adaptive/AC.4-generation-pipeline-caching-cost.md) ✅ | MAJOR | AC.3 | Async pre-warming, variant cache/dedup, per-course budgets |
| **AC.5** | [Instructor authoring & human-in-the-loop approval](../../completed/adaptive/AC.5-instructor-authoring-and-approval.md) ✅ | BLOCKER | AC.1, AC.3 | Course-editor config, guardrails, preview, approve/lock, fallback |
| **AC.6** | [Student runtime experience & transparency](../../completed/adaptive/AC.6-student-runtime-and-transparency.md) ✅ | BLOCKER | AC.2, AC.3, AC.5 | Serving variants, "adapted for you" disclosure, opt-out, a11y |
| **AC.7** | [Post-assessment, effectiveness & holdout experiments](../../completed/adaptive/AC.7-post-assessment-and-effectiveness.md) ✅ | BLOCKER | AC.2, AC.6 | Exit-ticket lift, mastery delta, control-group causal measurement |
| **AC.8** | [Governance, safety, fairness & privacy](../../completed/adaptive/AC.8-governance-safety-fairness-privacy.md) ✅ | BLOCKER | AC.3 | AI disclosure, FERPA/COPPA/EU-AI-Act, bias audit, oversight, DSAR |
| **AC.9** | [Analytics, reporting & operability](AC.9-analytics-reporting-and-operability.md) | MAJOR | AC.4, AC.7 | Instructor/admin dashboards, observability, alerts, rollout |

## Sequencing at a glance

```
AC.1 Foundations ─┬─► AC.2 Pre-assessment / profile ─┬─► AC.3 Generation engine ─┬─► AC.4 Pipeline / cache / cost ─┐
                  │                                    │                           │                                │
                  └────────────────────────────────────┘                           ├─► AC.5 Authoring / approval ───┤
                                                                                    │                                │
                                          AC.2 + AC.3 + AC.5 ──────────────────────►├─► AC.6 Student runtime ────────┤
                                                                                    │                                │
                                                          AC.2 + AC.6 ─────────────►└─► AC.7 Post-assessment / lift ──┤
                                                                                                                     │
                        AC.8 Governance/safety/fairness ── cross-cuts AC.3–AC.7; GATES GA ───────────────────────────┤
                                                                                                                     │
                        AC.9 Analytics/operability ── aggregates AC.4 (cost/cache) + AC.7 (lift) ────────────────────┘
```

## Relationship to the Standards folder

ACE is a high-risk, AI-driven, student-facing system, so it inherits obligations from
[`../standards/`](../standards/): **S06** (DPIA / algorithmic-impact assessment) and **S13** (EU AI
Act — education as high-risk AI) both apply to the generation engine; **S08** (children's privacy /
age assurance) constrains adaptation for minors; **S01/S02** (DSAR / retention) cover adaptation
profiles and stored variants as part of the education record. **[AC.8](../../completed/adaptive/AC.8-governance-safety-fairness-privacy.md)**
is the single owner that discharges those obligations for this feature and links back to each
standard.

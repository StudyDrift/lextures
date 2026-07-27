# Content Tools (CT) — interactive tools inside content sections

> **The idea:** a section body is currently *read-only prose*. A student can scroll past a whole
> page without producing a single piece of evidence that they learned anything. **Content Tools**
> add a **Tools** dropdown to the section toolbar so an author can drop an *interactive element*
> straight into the flow of the content — ask-the-AI, a two-question check, a prediction probe, a
> drag-to-sort, a runnable code cell — and every one of them **persists state per enrollment** as a
> JSON document. Every plan in this folder follows [`../_TEMPLATE.md`](../_TEMPLATE.md).

## Why this folder exists

Lextures already has *interactive activities* — but each one is a **whole module item** the student
must navigate away to: a quiz, an assignment, an H5P package, a vibe activity, a board, a live quiz
game. Nothing is interactive **in place, mid-paragraph, at the moment of confusion**.

| Already shipped | Granularity | Why it does not cover this |
|---|---|---|
| Quizzes / question bank (`course.questions`) | Whole module item | Separate page, separate attempt lifecycle, graded — too heavy for a 20-second comprehension check |
| H5P packages (`content.h5p_packages`) | Whole module item | Third-party authoring tool, uploaded `.h5p` zip, opaque state, no Lextures-native config |
| Vibe activities (`course.module_vibe_activities`) | Whole module item | Free-form AI-generated HTML in a sandboxed iframe; no typed state, no per-enrollment persistence |
| Boards / whiteboards (` ```board `) | Inline block | Real-time collaboration surface; no per-student state, no assessment semantics |
| AI tutor / study buddy (`course.tutor_sessions`) | Course-wide sidebar | Not bound to *this* section's content or *this* section's web links |
| Personal highlights (`course.content_page_user_markups`) | Whole page | Learner's own private markup, ungraded, un-prompted, invisible to the instructor |

Content Tools is the missing layer: **inline, typed, configurable, stateful, resettable**.

## The framework (what every tool gets for free)

Authors insert a tool from the section toolbar's new **Tools** dropdown (and from the `/` slash
menu). The block serializes into the section Markdown exactly like the existing ` ```board ` block:

````markdown
```lex-tool
{"instanceId":"9f0c…","toolId":"inline_questions","v":1}
```
````

The body carries only a pointer. Configuration lives server-side in
`course.content_tool_instances.config_json`; learner state lives in
`course.content_tool_states.state_json`, keyed **`(instance_id, enrollment_id)`** — so state follows
the *enrollment*, works identically for a student, a TA, or a teacher previewing as themselves, and
is wiped with the enrollment. Both are `JSONB`, validated against the JSON Schemas the tool declares
in its **manifest**. That is the whole extensibility contract:

> **Adding a tool = a manifest + a renderer bundle + (optionally) an action handler.
> No migration. No new table. No new route.**

Which is what makes hundreds of tools — and eventually a [tool marketplace](../../completed/content_tools/CT.9-tool-marketplace-and-third-party-tools.md) — tractable.

```
┌──────────────────────── section editor ────────────────────────┐
│  ≡ ≣ B I <> {} 🔗 ▦ Σ  🖼  [ Tools ▾ ]  Generate   ⌃ ⌄ 🗑        │   CT.2
│                          ├ Ask Questions            (CT.10)     │
│  Section heading         ├ Inline Questions         (CT.11)     │
│  Lorem ipsum…            ├ Predict & Reveal         (CT.12)     │
│  ```lex-tool             ├ Highlight & Annotate     (CT.13)     │
│  {"instanceId":"…"}      ├ … 100s more              (CT.9)      │
│  ```                     └ Browse all tools…                    │
└────────────────────────────────────────────────────────────────┘
             │ manifest (id, semver, configSchema, stateSchema, caps)   CT.1 / CT.5
             ▼
┌──────────── student runtime host ────────────┐   ┌──── instructor ────┐
│  renders tool · autosaves state · a11y · i18n│   │ state console      │  CT.4
│  PUT …/state {stateJson, revision}           │──▶│ per-enrollment     │
│  POST …/actions/{action}  (AI, grade, run)   │   │ RESET ↺            │
└──────────────────────────────────────────────┘   └────────────────────┘
       CT.3                    CT.6 (grounded AI + web-link ingestion)
```

## Feature-flag philosophy (per-course, **not** global)

Content Tools follows the [ACE](../adaptive/README.md) precedent: a single per-course boolean
`course.courses.content_tools_enabled` (JSON `contentToolsEnabled`) wired through
`server/internal/httpserver/course_features.go` → `models/course/types.go` →
`clients/web/src/lib/courses-api-schemas.ts`, alongside `adaptive_paths_enabled` and
`adaptive_content_enabled`. There is **no required global on-switch**; the only platform control is
an ops-only emergency kill-switch (`CONTENT_TOOLS_KILL_SWITCH`, default *disengaged*). Individual
tools are additionally gated by an **allowlist per course** (CT.1 §8) so a course can enable
Content Tools without enabling, say, the code sandbox.

## Conventions

- **File naming:** `CT.{N}-{kebab-slug}.md` (mirrors the `AC.`/`VC.`/`IQ.` per-course-feature folders).
- Every plan fills **all 19** template sections (no `…` placeholders) before it is "ready".
- **Backend:** service `server/internal/service/contenttools/`, repo `server/internal/repos/contenttools/`,
  models `server/internal/models/contenttools/`, HTTP `server/internal/httpserver/content_tools_*.go`,
  routes under `/api/v1/courses/{course_code}/content-tools/*`.
- **Web:** `clients/web/src/components/content-tools/` — `host/` (runtime shell), `registry/`
  (generated index), `tools/{tool-id}/` (one folder per tool, lazily code-split).
- **Tool ids** are stable `snake_case` strings (`ask_questions`, `inline_questions`, …) and are
  **never renamed** — they are the primary key of stored state.
- **Migrations** continue the global sequence. Highest on the working branch is `448_*`, so these
  plans reserve **`449_*` onward** (each story states its number). Renumber on merge if the sequence
  has advanced. **Tool stories CT.10–CT.23 introduce no migrations at all** — that is the design
  claim being tested.
- **AI feature ids** in `aigateway`: `content_tool` plus per-tool sub-ids (`content_tool_ask`,
  `content_tool_explain_back`, …), so every model call is disclosed, budgeted and logged to
  `analytics.ai_usage_log` like every other AI call.
- **Metrics** namespace `lextures_content_tool_*`, always labelled `{tool_id}`.
- **i18n**: new locale namespace `contentTools.json`; tool strings under `contentTools.tools.{toolId}.*`.

## Severity legend

- **BLOCKER** — the framework cannot function / is unsafe to ship without it.
- **MAJOR** — the framework works but a market-critical capability or guardrail is missing.
- **MINOR** — parity, polish, or a tool that is additive to an already-working shelf.

## Story index

### Platform — the framework itself

| ID | Plan | Severity | Effort | Depends on | Delivers |
|---|---|---|---|---|---|
| **CT.1** | [Foundations: tool registry, manifest contract & data model](../../completed/content_tools/CT.1-foundations-registry-and-data-model.md) | BLOCKER | M | — | `content_tools_enabled` flag, manifest contract, `content_tool_instances` / `content_tool_states` / `content_tool_events`, config API |
| **CT.2** | [Authoring: the Tools dropdown, insert flow & config panel](../../completed/content_tools/CT.2-authoring-tools-dropdown-and-config.md) | BLOCKER | M | CT.1 | Toolbar dropdown + slash command, ` ```lex-tool ` serialization, per-tool config forms, preview-as-student |
| **CT.3** | [Student runtime host & state persistence](../../completed/content_tools/CT.3-student-runtime-and-state-persistence.md) | DONE | M | CT.1, CT.2 | Renderer host, autosave with optimistic concurrency, offline queue, action dispatch, a11y baseline |
| **CT.4** | [Instructor state console & per-enrollment reset](../../completed/content_tools/CT.4-instructor-state-console-and-reset.md) | BLOCKER | S | CT.3 | Inspect any learner's tool state; reset one/many/all with snapshot, audit and grade side-effects |
| **CT.5** | [Tool SDK, sandboxing, versioning & migration](../../completed/content_tools/CT.5-tool-sdk-sandboxing-and-versioning.md) | DONE | M | CT.1, CT.3 | `@lextures/tool-sdk`, iframe sandbox + postMessage bridge, semver pinning, state migrations, bundle budgets |
| **CT.6** | [Grounded context service & web-link ingestion](../../completed/content_tools/CT.6-grounded-context-and-link-ingestion.md) | DONE | M | CT.1 | Context packs from the activity, SSRF-safe link fetch/extract/cache, `fetch_link` tool-calling for AI tools |
| **CT.7** | [Analytics, insights & gradebook bridge](../../completed/content_tools/CT.7-analytics-insights-and-gradebook.md) | DONE | M | CT.3 | Per-tool instructor insights, struggle detection, xAPI/Caliper emission, optional score passback |
| **CT.9** | [Tool marketplace & third-party tools](../../completed/content_tools/CT.9-tool-marketplace-and-third-party-tools.md) | DONE | L | CT.5,
### The shelf — tools an author can drop into a section

| ID | Tool | Mechanism it exploits | Severity | Migration? |
|---|---|---|---|---|
| **CT.10** | [Ask Questions](../../completed/content_tools/CT.10-tool-ask-questions.md) — grounded AI Q&A about *this* activity | Question-asking / just-in-time explanation | BLOCKER | none |
| **CT.11** | [Inline Questions](../../completed/content_tools/CT.11-tool-inline-questions.md) — 1–2 question check with a correct answer | Retrieval practice + immediate feedback | BLOCKER | none |
| **CT.12** | [Predict & Reveal](../../completed/content_tools/CT.12-tool-predict-and-reveal.md) — commit a prediction + confidence, *then* see the answer | Generation effect / hypercorrection | MAJOR | none |
| **CT.13** | [Highlight & Annotate](../../completed/content_tools/CT.13-tool-highlight-and-annotate.md) — tag passages against a prompt; instructor heat map | Active reading / attention direction | MAJOR | none |
| **CT.14** | [Sort & Sequence](../../completed/content_tools/CT.14-tool-sort-and-sequence.md) — drag items into categories or into order | Schema building / procedural fluency | MAJOR | none |
| **CT.15** | [Labeled Diagram & Hotspot](CT.15-tool-labeled-diagram-and-hotspot.md) — click/drag labels onto an image | Dual coding / spatial recall | MAJOR | none |
| **CT.16** | [Parameter Explorer](CT.16-tool-parameter-explorer.md) — sliders drive a live model + guided noticing prompts | Inquiry / variable isolation | MAJOR | none |
| **CT.17** | [Code Sandbox](CT.17-tool-code-sandbox.md) — runnable cell with instructor tests | Deliberate practice with feedback | MAJOR | none |
| **CT.18** | [Step-Through Worked Example](CT.18-tool-step-through-worked-example.md) — enter one step at a time, checked, with hints | Worked-example effect / faded scaffolding | MAJOR | none |
| **CT.19** | [Media Checkpoints](CT.19-tool-media-checkpoints.md) — questions injected at timestamps in a video/audio | Segmenting / interpolated testing | MAJOR | none |
| **CT.20** | [Explain It Back](CT.20-tool-explain-it-back.md) — free-text self-explanation with AI formative feedback | Self-explanation / elaborative interrogation | MAJOR | none |
| **CT.21** | [Class Pulse](CT.21-tool-class-pulse.md) — vote, then see the anonymised class distribution | Peer instruction / social proof | MINOR | none |
| **CT.22** | [Inline Discussion](CT.22-tool-inline-discussion.md) — a scoped thread anchored to this paragraph | Collaborative elaboration | MINOR | none |
| **CT.23** | [Flashcards & Spaced Recall](CT.23-tool-flashcards-and-spaced-recall.md) — inline deck that feeds the shipped SRS | Spacing + retrieval | MINOR | none |

## Sequencing at a glance

```
CT.1 Foundations ─┬─► CT.2 Authoring ──┬─► CT.3 Runtime ──┬─► CT.4 Reset console
                  │                    │                  ├─► CT.7 Analytics / gradebook
                  ├─► CT.6 Context ────┘                  ├─► CT.5 SDK / sandbox ──► CT.9 Marketplace
                  │      (grounding)                      │
                  └────────────────────────────────────────┴─►
           Tools CT.10 … CT.23 depend only on { CT.1, CT.2, CT.3 } (+ CT.6 for AI tools).
           They ship in parallel, in any order, by any number of teams.
```

The last line is the point of the architecture: **CT.1–CT.3 are the only serialised work.** Once the
host exists, tools are independent, parallelisable, migration-free units — which is exactly the
property a shelf of hundreds of tools (and a third-party marketplace) requires.

## First-party tool backlog (candidates for later stories)

Deliberately *not* planned yet, listed to prove the framework's headroom and to seed the marketplace
taxonomy. Each is one manifest + one renderer under the CT.1–CT.3 contract.

| Category | Candidates |
|---|---|
| **Reasoning** | Concept-map builder (reuses `service/conceptgraph`), branching scenario / case study, claim-evidence-reasoning scaffold, find-the-flaw ("debug this argument"), Socratic dialogue, decision-tree triage |
| **Quantitative** | Estimation / Fermi problem, numeric answer with tolerance, unit-conversion drill, graph sketcher, proof builder, chemical-equation balancer, statistics sampler |
| **Language** | Cloze / guided notes, vocabulary matching, sentence reordering, dictation, pronunciation record-and-compare (reuses `service/tts` + captions), translation practice, close-reading gloss |
| **Data** | Dataset explorer with guided questions, chart-reading probe, pivot/filter challenge, spreadsheet cell task |
| **Metacognition** | Muddiest-point capture, KWL chart, confidence calibration report, study-plan builder, goal commitment device |
| **Practice** | Timed retrieval sprint, error-analysis review of one's own past attempt, worked-example comparison (two solutions side by side), peer-review-in-place |
| **Spatial / creative** | Drawing response (reuses the shipped whiteboard block), timeline builder, map pin task, image-sequence storyboard |

## Relationship to other folders

- **[`../adaptive/` (ACE)](../adaptive/README.md)** — ACE rewrites *the prose* per learner; Content
  Tools adds *interaction* to whichever prose is served. A tool instance is rendered against the
  variant the learner actually received, and tool outcomes are a legitimate ACE evidence source
  (CT.7 §12). No ordering dependency in either direction.
- **[`../../completed/16.9-marketplace-plugin-system.md`](../../completed/16.9-marketplace-plugin-system.md)** —
  the *org-level OAuth app* extension point. CT.9 is a different unit (an in-content widget, not an
  app) but deliberately reuses 16.9's registration, consent and revocation machinery rather than
  inventing a second developer portal.
- **[`../standards/`](../standards/README.md)** — tools capture student work and (for AI tools)
  send it to a model, so **S01/S02** (DSAR, retention), **S06** (DPIA), **S08** (children's privacy),
  **S13** (EU AI Act) and **S20** (accessibility law) all attach.  discharges those obligations for this feature and links back to each standard.
- **[`../tech_debt/`](../tech_debt/README.md)** — CT.1–CT.3 add code to `internal/httpserver`, which
  TD.6 is decomposing. New Content Tools handlers MUST land in the new package layout if TD.6 has
  merged; if not, they land as `content_tools_*.go` files with no additions to `Deps` beyond one
  service pointer (TD.10 constraint).

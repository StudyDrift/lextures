# GA-M2 — Grade non-file (online text-entry) & image/scanned submissions

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](../../plan/grading-agent/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-M2 |
| **Section** | Grading Agent — Missing Features |
| **Severity** | BLOCKER |
| **Markets** | HE / K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | broad agent adoption |

## 1. Problem Statement

The agent can only grade a submission that has a **file attachment** whose bytes convert to non-empty
text. `gradableSubmissionsForAgent` drops every submission where `AttachmentFileID == nil`, and
`LoadSubmissionMarkdownsForSubmission` errors with "no submission text available" when there are no
file refs. Two extremely common Higher-Ed submission types are therefore ungradable:

1. **Online text-entry** submissions (student types an essay/short answer directly into the box) — no file at all.
2. **Image-only or scanned PDF** submissions (handwritten math, lab sketches, scanned problem sets) — a file exists but markitdown yields no extractable text.

For these, a batch run reports "No submissions with readable file attachments matched this scope," and
the instructor reasonably concludes the agent is broken. Any class that uses text-entry or handwritten
work cannot adopt the agent.

## 2. Goals

- Grade online text-entry submissions (body text stored on the submission, no file).
- Grade image/scanned submissions via a vision-capable model path (or explicit OCR), behind config.
- Give a clear, per-submission reason when something truly cannot be read, instead of silently excluding it.

## 3. Non-Goals

- Grading binary artifacts that are out of scope for text/vision (e.g., raw video) — those route to manual.
- Building a bespoke OCR engine; prefer a vision model or an existing OCR dependency.
- Changing the rubric/scoring contract.

## 4. Personas & User Stories

- **As an instructor**, I want the agent to read text typed into the submission box, so that essay/short-answer classes can use it.
- **As a math/physics TA**, I want the agent to read a scanned/handwritten PDF or photo, so that problem sets are gradable.
- **As an instructor**, I want a clear per-student note when a submission can't be read, so that I know exactly what to grade by hand.

## 5. Functional Requirements

- **FR-1.** The agent MUST grade submissions whose content is online text-entry (no file) by sourcing the submission body text.
- **FR-2.** `gradableSubmissionsForAgent` MUST include text-entry submissions, not only `AttachmentFileID != nil`.
- **FR-3.** When a file yields empty text and the platform has a vision-capable grader model configured, the system SHOULD send the image/PDF pages to the vision path instead of failing.
- **FR-4.** When neither text nor vision can read a submission, the system MUST record a per-submission `failed` result with a human-readable reason (surfaced in the review queue, [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md)) rather than excluding it from the run.
- **FR-5.** The Student Submission node MUST advertise which input modalities a given submission provides (text / file / image) in the dry-run log.
- **FR-6.** Vision usage MUST pass through the same AI-gateway opt-in / PII / cost accounting as text grading.

## 6. Non-Functional Requirements

- **Performance** — text-entry path adds no LLM cost beyond existing; vision path capped by page count (configurable, default ≤ 10 pages).
- **Security** — image bytes handled with the same storage auth as files; signed reads only.
- **Privacy & Compliance** — PII redaction applies to extracted/typed text; vision requests honor tenant BYOK and opt-in; FERPA-safe logging (no raw content in logs).
- **Accessibility** — n/a (backend); failure reasons localized.
- **Scalability** — vision requests are heavier; respect per-run concurrency and budget ([GA-M7](missing-7-cost-estimate-and-budget.md)).
- **Reliability** — a single unreadable submission never fails the whole run.
- **Observability** — metric: submissions by modality (text/file/vision/unreadable) per run.
- **Internationalization** — failure reasons under `gradingAgent.*`.
- **Backward compatibility** — file-text path unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a text-entry submission, *when* the agent runs, *then* it grades the typed body and writes a grade.
- **AC-2.** *Given* a scanned PDF and a configured vision model, *when* the agent runs, *then* it grades from the rendered pages.
- **AC-3.** *Given* an unreadable submission, *when* the agent runs, *then* a `failed` result with reason appears in the review queue and the run still completes.
- **AC-4.** *Given* no vision model configured, *when* an image-only submission is encountered, *then* the reason explicitly says vision grading is not enabled.
- **AC-5.** *Given* a mixed scope (text-entry + file + image), *when* the batch runs, *then* every submission is attempted and accounted for.

## 8. Data Model

- No new tables. Confirm the submission row exposes body text for online text-entry (`moduleassignmentsubmissions`); add a read accessor if missing.
- Optionally record per-result `input_modality TEXT` on `grading_agent_results` for analytics.
- Migration only if `input_modality` is added: `server/migrations/NNN_grading_agent_input_modality.sql`.

## 9. API Surface

- No new public routes. Internal `Service` gains a submission-content resolver that returns `{text, images[]}` from text-entry body and/or files.
- Dry-run WS log gains modality lines (existing `DryRunEvent` log type).

## 10. UI / UX

- Run scope copy updated: remove the implication that only file submissions count.
- Per-submission failure reasons render in the review queue and dry-run console.
- Settings: surface whether a vision-capable grader model is configured (links to AI settings).

## 11. AI / ML Considerations

- Vision path uses a vision-capable model id (per-tenant BYOK respected). Prompt mirrors text grading but with image parts.
- Page/image cap and downscaling to bound cost; PII redaction on any extracted text.
- Fallback order: text-entry body → file text → vision (if enabled) → `failed` with reason.

## 12. Integration Points

- `server/internal/service/gradingagent/submission_markdown.go` (content resolver), `service.go` (vision call path).
- `server/internal/httpserver/grading_agent_http.go` + `grading_agent_queue.go` (scope selection, per-submission failure recording).
- `moduleassignmentsubmissions` repo (body text accessor).
- AI gateway + openrouter client for vision.

## 13. Dependencies & Sequencing

- Best sequenced with [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md) so per-submission failures are reviewable.
- Vision requires a configured vision model (ties to existing AI provider settings).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Vision cost surprises instructors | M | H | Gate behind config + budget/estimate ([GA-M7](missing-7-cost-estimate-and-budget.md)) |
| OCR/vision accuracy on messy handwriting | M | M | Always route low-confidence to review ([GA-M4](missing-4-confidence-auto-hold-threshold.md)); never silently auto-apply |
| Large scanned PDFs blow token limits | M | M | Page cap + downscale; clear failure reason |

## 15. Rollout Plan

- Flags: `graderAgentTextEntryGrading` (low-risk, default on after dogfood), `graderAgentVisionGrading` (default off).
- Sequence: text-entry path → ship; vision path → dogfood → opt-in.
- Pilot: an essay course (text-entry) and a STEM course (scanned).
- Rollback: disable vision flag; text-entry path is additive and safe.

## 16. Test Plan

- **Unit** — content resolver fallback order; failure-reason mapping.
- **Integration** — text-entry submission graded end-to-end; unreadable submission produces reviewable failure.
- **E2E** — mixed-modality batch; counts reconcile.
- **Security** — signed image reads; PII redaction on typed text.
- **Performance** — vision page cap honored.

## 17. Documentation & Training

- Help-center: "Which submission types the grading agent can read."
- Admin doc: enabling vision grading and its cost profile.

## 18. Open Questions

1. Do we render scanned PDFs to images server-side, or rely on the model's native PDF support?
2. Should text-entry + attachments be concatenated, or graded as the student's "primary" content only?
3. Is there a per-assignment switch to force manual grading for image submissions regardless of vision availability?

## 19. References

- `server/internal/service/gradingagent/submission_markdown.go` (`LoadSubmissionMarkdownsForSubmission`).
- `server/internal/httpserver/grading_agent_http.go` (`gradableSubmissionsForAgent`, `resolveGraderAgentSubmissions`).
- `server/internal/service/gradingagent/service.go` (`Score`, PII redaction).
- Related: [GA-M1](../../completed/grading-agent/missing-1-persistent-review-queue.md), [GA-M4](missing-4-confidence-auto-hold-threshold.md), [GA-M7](missing-7-cost-estimate-and-budget.md).

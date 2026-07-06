# C36 — AI tutor, study buddy & diagnostics

> CLI parity plan. Source: `registerTutorRoutes` + `registerPersistentTutorRoutes` (`courses/{id}/tutor`), `registerStudyBuddyRoutes` (`courses/{id}/study-buddy`, `me/study-*`), `registerDiagnosticRoutes` (`diagnostic-attempts`, `courses/{id}/diagnostic-config`), `registerConceptRoutes` (`concepts`), `registerLearningPathRoutes` (`paths`, `me/paths`), `registerLearnerRoutes` (`learners`). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C36 |
| **Section** | Student experience |
| **Severity** | MINOR |
| **Markets** | SL / HE / K12 |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / CLI |
| **Depends on** | C25, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

The AI tutor, study-buddy, diagnostics, concept graph and adaptive learning paths — the platform's adaptive core — have no CLI. Power self-learners, researchers evaluating the tutor, and QA engineers cannot script tutor sessions, run diagnostics, or inspect learner models/paths for testing and evaluation.

## 2. Goals

- Hold a scriptable tutor session (send a turn, stream the reply).
- Run diagnostics and read learner concept mastery / adaptive path state.
- Enable batch evaluation of the tutor (eval harness feeding transcripts in, scoring out).

## 3. Non-Goals

- Replacing the rich chat UI.
- Training models.

## 4. Personas & User Stories

- **As a self-learner**, I want `tutor ask --course C "explain X"` in the terminal.
- **As a researcher**, I want `tutor eval --file prompts.jsonl` to batch-run tutor turns for evaluation.
- **As a learner**, I want `diagnostic run --course C` and `paths status` to see my adaptive path.
- **As a QA engineer**, I want `learners get --user U` to inspect the learner model.

## 5. Functional Requirements

- **FR-1.** MUST add `tutor ask <course> <prompt>` (streaming reply; `--session` to continue) and `tutor sessions list`.
- **FR-2.** MUST add `diagnostic run|attempts|config <course>` (`registerDiagnosticRoutes`).
- **FR-3.** SHOULD add `study-buddy ask|sessions` (`registerStudyBuddyRoutes`) and `paths get|status|next` (`registerLearningPathRoutes`).
- **FR-4.** SHOULD add `concepts list|get <course>` (`registerConceptRoutes`) and `learners get --user <u>` (learner model).
- **FR-5.** MAY add `tutor eval --file <jsonl> --out results.jsonl` for batch evaluation.

## 6. Non-Functional Requirements

- **Performance** — streaming responses rendered incrementally (SSE/WebSocket via C40 helper).
- **Security** — learner scope; tutor honors course AI opt-out and age-appropriate mode.
- **Privacy & Compliance** — tutor turns are learner data (FERPA/COPPA); AI disclosure surfaced; PII redaction server-side.
- **Cost** — `tutor eval` prints token usage; `--max-turns` cap.
- **Reliability** — session continuity via server session id; safe to resume.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course, *When* `tutor ask "..."`, *Then* a streamed reply renders and a session id is returned.
- **AC-2.** *Given* an opt-out course, *Then* tutor commands refuse with a clear message.
- **AC-3.** *Given* a prompts file, *When* `tutor eval`, *Then* per-prompt results + usage are written.

## 8. Data Model

- Client may persist a `--session` id transiently for multi-turn.

## 9. API Surface

- `registerTutorRoutes`/`registerPersistentTutorRoutes`; `registerStudyBuddyRoutes`; `registerDiagnosticRoutes`; `registerLearningPathRoutes`; `registerConceptRoutes`; `registerLearnerRoutes`.

## 10. UI / UX

- `lextures tutor ...`, `lextures study-buddy ...`, `lextures diagnostic ...`, `lextures paths ...`, `lextures learners ...`.
- Streaming rendered to stderr; final structured result to stdout under `--json`.

## 11. AI / ML Considerations

- Tutor is the flagship AI feature; CLI is a thin client. Disclosure, opt-out, age-mode, PII redaction all server-enforced; CLI surfaces them. Eval harness enables offline quality measurement.

## 12. Integration Points

- Server tutor/diagnostic/path handlers; streaming (SSE/WS helper, C40); AI provider settings (C21).

## 13. Dependencies & Sequencing

- After: C25/C21 (AI provider config), C40 (streaming).
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| No streaming client in CLI | M | M | Shared SSE/WS helper in C40 |
| Eval cost | M | M | `--max-turns`, usage printout, `--yes` for large runs |

## 15. Rollout Plan

- Ship `tutor ask` + diagnostics first, then study-buddy/paths/learners, then eval harness.
- Rollback: additive.

## 16. Test Plan

- **Unit** — session handling; opt-out gating.
- **Integration** — mock streaming server; diagnostic attempt.
- **E2E** — multi-turn tutor session.

## 17. Documentation & Training

- "Evaluate the AI tutor from the CLI" recipe.

## 18. Open Questions

1. Is tutor transport SSE or WebSocket?
2. Does an eval-friendly endpoint exist, or must we script `ask`?

## 19. References

- `registerTutorRoutes`, `registerDiagnosticRoutes`, `registerLearningPathRoutes`, `registerConceptRoutes`.
- Related: [C09](C09-ai-grading-agents.md), [C21](C21-platform-settings.md), [C40](C40-cli-framework.md).

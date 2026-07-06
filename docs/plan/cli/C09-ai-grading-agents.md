# C09 — AI grading agents

> CLI parity plan. Source: `courses/{id}/grader-agent-templates` (5), `grader-agents`, `course-grading-agents`, `grading_agent_dry_run_ws.go`, `course_grading_settings`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C09 |
| **Section** | Assessment & grading |
| **Severity** | MAJOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI / CLI |
| **Depends on** | C03, C06, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

AI grading agents (templates, per-course agents, dry-runs) are a differentiating feature with no CLI access. Course teams cannot version-control grader configs, run dry-runs in CI to validate a rubric-to-agent mapping, or batch-trigger AI grading — forcing manual UI configuration that can't be reviewed or reproduced.

## 2. Goals

- Manage grader-agent templates and per-course agents as version-controlled config.
- Run a grading agent **dry-run** and inspect results before applying.
- Trigger AI grading for an assignment and review/accept suggested scores.

## 3. Non-Goals

- Training or fine-tuning models (server/provider concern).
- Human grade entry (C06) — this plan produces suggestions that flow into it.

## 4. Personas & User Stories

- **As a course team**, I want `grader-agents set --file agent.json` so configs live in git.
- **As an instructor**, I want `grader-agents dry-run` to preview AI scores on sample submissions.
- **As a grader**, I want `grader-agents run <assignment>` then review before syncing to the gradebook.

## 5. Functional Requirements

- **FR-1.** MUST add `grader-templates list|get|create` (grader-agent-templates).
- **FR-2.** MUST add `grader-agents list|get|set|delete <course>` (`--file` config).
- **FR-3.** MUST add `grader-agents dry-run <course> --assignment <a> [--sample N]` consuming `grading_agent_dry_run_ws.go` (WS stream → printed results).
- **FR-4.** SHOULD add `grader-agents run <assignment>` and `grader-agents results <assignment>` (suggested scores; `--accept` to sync).
- **FR-5.** SHOULD surface AI disclosure/model info in output (ties to C29 AI disclosure).

## 6. Non-Functional Requirements

- **Performance** — dry-run streams incremental results over WebSocket; CLI renders progress.
- **Security** — AI-grading scope; provider keys never exposed.
- **Privacy & Compliance** — submissions sent to AI are FERPA/PII-sensitive; CLI MUST surface the AI-disclosure notice and honor course AI opt-out.
- **Reliability** — dry-run has no gradebook side effects; `run` is idempotent per submission.
- **Cost** — dry-run `--sample` limits token spend; command prints an estimated cost if the server returns one.
- **Observability** — print model id and token usage per run (from `ai_provider_usage`).

## 7. Acceptance Criteria

- **AC-1.** *Given* an agent config file, *When* `grader-agents set --file`, *Then* the config is stored and `get` matches.
- **AC-2.** *Given* a dry-run, *Then* streamed suggested scores print and the gradebook is unchanged.
- **AC-3.** *Given* an AI-opt-out course, *When* any grading agent command runs, *Then* it refuses with a clear message.

## 8. Data Model

- None client-side. Document agent config JSON schema.

## 9. API Surface

- `grader-agent-templates` CRUD; `courses/{c}/grader-agents` CRUD; dry-run WebSocket (`grading_agent_dry_run_ws.go`); run/results; grade-sync into C06.

## 10. UI / UX

- `lextures grader-templates ...`, `lextures grader-agents ...`.
- WebSocket dry-run renders a live progress line; `--json` collects final results.

## 11. AI / ML Considerations

- Model(s): server-selected via provider settings (C25/C29). CLI passes config, never prompts directly.
- PII redaction and opt-out enforced server-side; CLI surfaces disclosure text.
- Cost budget: `--sample` and printed usage.

## 12. Integration Points

- Server grader-agent + AI provider + disclosure handlers; WebSocket client (new in CLI — see C40 WS helper).
- Internal: new `cmd/grader_agents.go`.

## 13. Dependencies & Sequencing

- After: C03 (submissions), C06 (grade sync), C40 (WS helper).
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| CLI has no WebSocket client yet | H | M | Add shared WS helper in C40; fall back to polling if server offers REST dry-run |
| Cost overrun on large runs | M | M | Enforce `--sample`; print estimate; require `--yes` for full-class runs |

## 15. Rollout Plan

- Ship templates/agents CRUD + dry-run first, then run/accept.
- Rollback: additive; dry-run has no side effects.

## 16. Test Plan

- **Unit** — config parse; opt-out gating.
- **Integration** — WS dry-run stream parsing (mock WS server).
- **E2E** — set agent → dry-run → run → sync grade.

## 17. Documentation & Training

- "Version-control an AI grader and validate it in CI" recipe.

## 18. Open Questions

1. Is dry-run WebSocket-only, or is there a REST variant?
2. How is estimated cost surfaced by the server?

## 19. References

- `grading_agent_dry_run_ws.go`, grader-agent handlers; `ai_provider_usage.go`.
- Related: [C06](C06-gradebook-final-grades.md), [C25](C25-integrations-webhooks-bots.md), [C29](C29-compliance-privacy.md), [C40](C40-cli-framework.md).

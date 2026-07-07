# C09 — AI grading agents

> CLI parity plan. Source: `courses/{id}/grader-agent-templates`, `grader-agents`, `grading_agent_dry_run_ws.go`, `grading_agent_http.go`. Baseline: `clients/cli/cmd/grader_templates.go`, `grader_agents.go`, `grader_agents_logic.go`, `grader_agents_test.go`, `internal/wsclient/wsclient.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C09 |
| **Section** | Assessment & grading |
| **Severity** | MAJOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | COMPLETE |
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

- None client-side. Agent config JSON accepts either a legacy `prompt` or a `workflowGraph` (same shape as the web grader canvas). `grader-agents set --file` also accepts a `get` response wrapped in `{ "config": … }`.

## 9. API Surface

- `grader-agent-templates` CRUD; `courses/{c}/grader-agents` CRUD; dry-run WebSocket (`grading_agent_dry_run_ws.go`); run/results; grade-sync into C06.

## 10. UI / UX

- `lextures grader-templates list|get|create <course>`.
- `lextures grader-agents list|get|set|delete <course> --assignment <item-id>` (`set --file agent.json`).
- `lextures grader-agents dry-run <course> --assignment <a> [--sample N] [--submission id]`.
- `lextures grader-agents run <assignment> --course C [--scope ungraded|all] [--mode suggest|apply] [--wait] [--yes]`.
- `lextures grader-agents results <assignment> --course C [--run id] [--accept --yes]`.
- WebSocket dry-run renders a live progress line; `--json` collects final results.

## 11. AI / ML Considerations

- Model(s): server-selected via provider settings (C25/C29). CLI passes config, never prompts directly.
- PII redaction and opt-out enforced server-side; CLI surfaces disclosure text.
- Cost budget: `--sample` and printed usage.

## 12. Integration Points

- Server grader-agent + AI provider + disclosure handlers; WebSocket client in `clients/cli/internal/wsclient`.
- Internal: `clients/cli/cmd/grader_templates.go`, `grader_agents.go`, `grader_agents_logic.go`, `grader_agents_test.go`.

## 13. Dependencies & Sequencing

- After: C03 (submissions), C06 (grade sync), C40 (WS helper).
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| CLI has no WebSocket client yet | H | M | Added `internal/wsclient` for dry-run streaming |
| Cost overrun on large runs | M | M | `--sample`; run-estimate line; `--yes` for scope=all |

## 15. Rollout Plan

- Shipped templates/agents CRUD + dry-run + run/accept.
- Rollback: additive; dry-run has no side effects.

## 16. Test Plan

- **Unit** — config parse; opt-out gating (`grader_agents_test.go`).
- **Integration** — WS dry-run stream parsing (mock WS server).
- **E2E** — set agent → dry-run → run → sync grade (manual / future stack test).

## 17. Documentation & Training

- "Version-control an AI grader and validate it in CI" recipe:

```bash
# 1. Save agent config from git
lextures grader-agents set CS101 --assignment <item-id> --file agent.json

# 2. Preview on one submission (no gradebook writes)
lextures grader-agents dry-run CS101 --assignment <item-id> --sample 1

# 3. Run suggest mode for ungraded work, then inspect results
lextures grader-agents run <item-id> --course CS101 --mode suggest --wait
lextures grader-agents results <item-id> --course CS101

# 4. Accept held suggestions into the gradebook
lextures grader-agents results <item-id> --course CS101 --accept --yes
```

## 18. Open Questions

1. Dry-run is **WebSocket-only** (`GET …/grader-agent/dry-run/ws`); there is no REST dry-run variant.
2. Estimated cost is returned by `GET …/grader-agent/run-estimate` (`estimatedCostMinUsd` / `estimatedCostMaxUsd`, token fields).

## 19. References

- `grading_agent_dry_run_ws.go`, grader-agent handlers; `ai_provider_usage.go`.
- Related: [C06](C06-gradebook-final-grades.md), [C25](../../plan/cli/C25-integrations-webhooks-bots.md), [C29](../../plan/cli/C29-compliance-privacy.md), [C40](../../plan/cli/C40-cli-framework.md).
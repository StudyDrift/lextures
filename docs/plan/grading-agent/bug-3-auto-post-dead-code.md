# GA-B3 — Auto-post / confidence_floor dead code

> Implementation plan. Source: grading-agent audit (2026-06-24). See [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | GA-B3 |
| **Section** | Grading Agent — Bugs |
| **Severity** | MAJOR |
| **Bug size** | Medium |
| **Markets** | HE / K12 |
| **Status (today)** | BUG |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment / Grading squad |
| **Depends on** | — |
| **Unblocks** | [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md), [GA-M4](missing-4-confidence-auto-hold-threshold.md) |

## 1. Problem Statement

Two config columns are read but **never written**, leaving intended behavior unreachable:

- **`post_policy`** — the consumer gates auto-posting with
  `strings.TrimSpace(assignRow.PostingPolicy) == "automatic" && cfg.PostPolicy == "auto_post"`
  in three places. But `UpsertConfig` never sets `post_policy` (it is omitted from the INSERT column
  list), so it is always its DB default `'unposted'`. Therefore `cfg.PostPolicy == "auto_post"` is
  **always false** and AI grades are **always written with `posting = "manual"`** (unposted), even for
  assignments configured to post automatically. There is no UI/API to change it. Instructors who expect
  automatic-posting assignments to show AI grades to students will find grades silently withheld.
- **`confidence_floor`** — read into `ConfigRow.ConfidenceFloor` but used nowhere and never written;
  fully dead (the live confidence-hold lives only on the gate node). See [GA-M4](missing-4-confidence-auto-hold-threshold.md).

This is both a latent bug (a whole code branch is unreachable) and a feature gap (no posting control).

## 2. Goals

- Decide and implement the intended posting behavior: either make `post_policy` settable end-to-end, or remove the dead branch and document "AI grades are always unposted until reviewed."
- Remove or wire `confidence_floor` (wiring is [GA-M4](missing-4-confidence-auto-hold-threshold.md); this plan at minimum removes the dead read or connects it).
- Eliminate misleading unreachable code so behavior matches intent.

## 3. Non-Goals

- Building the full suggest-only/bulk experience ([GA-M3](missing-3-suggest-only-batch-and-bulk-review.md)) — this plan fixes the posting plumbing it depends on.
- Changing the gradebook posting model itself.

## 4. Personas & User Stories

- **As an instructor**, I want to choose whether AI grades post to students automatically, so that the behavior matches my assignment's posting policy.
- **As a maintainer**, I want no unreachable branches, so that the code reflects real behavior.

## 5. Functional Requirements

- **FR-1.** `post_policy` MUST be persisted by `UpsertConfig` (add to INSERT/UPDATE) and exposed via the config GET/PUT API, **or** the three `cfg.PostPolicy == "auto_post"` branches MUST be removed and the behavior documented as always-manual.
- **FR-2.** If retained, the agent MUST default to `draft`/`unposted` (safe), with `auto_post` opt-in; the effective posting still respects the assignment's own posting policy.
- **FR-3.** `confidence_floor` MUST either be wired ([GA-M4](missing-4-confidence-auto-hold-threshold.md)) or removed from `ConfigRow`/queries to avoid a dead field.
- **FR-4.** No grade's posting state changes silently for existing agents (existing default behavior preserved unless the instructor opts in).

## 6. Non-Functional Requirements

- **Privacy & Compliance** — default remains "do not auto-show AI grades to students" (FERPA-safe); auto-post is explicit and logged.
- **Reliability** — posting decision covered by tests for both policies.
- **Observability** — log the effective posting decision per applied grade.
- **Backward compatibility** — existing agents keep `unposted` default; no behavior change without opt-in.
- **Maintainability** — no unreachable branches remain.

## 7. Acceptance Criteria

- **AC-1.** *Given* an agent set to `auto_post` and an assignment that posts automatically, *when* a grade is applied, *then* it is posted (visible to the student per posting rules).
- **AC-2.** *Given* an agent left at default, *when* a grade is applied, *then* it is unposted (today's behavior).
- **AC-3.** *Given* the config API, *when* I set and reload posting policy, *then* it persists.
- **AC-4.** *Given* the codebase, *when* searched, *then* there is no `cfg.PostPolicy == "auto_post"` branch that can never be true (it is either reachable or removed).
- **AC-5.** *Given* `confidence_floor`, *when* searched, *then* it is either used or fully removed.

## 8. Data Model

- `grading_agent_configs.post_policy` already exists (default `'unposted'`); add to `UpsertConfigInput` and the upsert SQL.
- `confidence_floor` exists; wire ([GA-M4](missing-4-confidence-auto-hold-threshold.md)) or drop from `ConfigRow` and queries.
- Migration: none required for `post_policy` (column exists); a migration only if dropping `confidence_floor`.

## 9. API Surface

- `GET …/grader-agent` config gains `postPolicy`; `PUT …/grader-agent` accepts `postPolicy` (`draft`|`auto_post`).
- `graderAgentConfigToJSON` includes `postPolicy` (and `confidenceFloor` if wired).

## 10. UI / UX

- Posting control in the agent settings (pairs with [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md)'s posting note): "Post AI grades to students automatically" vs "Keep as draft until I post."
- Copy/i18n under `gradingAgent.settings.posting.*`.

## 11. AI / ML Considerations

- None; posting is a downstream gradebook concern.

## 12. Integration Points

- `server/internal/repos/gradingagent/repo.go` (`UpsertConfigInput.PostPolicy`, upsert SQL; `ConfidenceFloor`).
- `server/internal/httpserver/grading_agent_http.go` (config body + `graderAgentConfigToJSON`).
- `server/internal/httpserver/grading_agent_queue.go` (the three posting-derivation sites → centralize once via [GA-S1](simplify-1-unify-grade-write-paths.md)).

## 13. Dependencies & Sequencing

- Foundational for [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md) (posting) and [GA-M4](missing-4-confidence-auto-hold-threshold.md) (confidence_floor).
- Cleanest after [GA-S1](simplify-1-unify-grade-write-paths.md) so posting is decided in one place.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Enabling auto-post surprises students with unreviewed grades | M | H | Default stays draft; auto-post is explicit opt-in + recommend suggest-only first |
| Removing the branch loses a planned feature | M | M | Prefer wiring over deletion; if removed, document and keep the column |

## 15. Rollout Plan

- Flag: `graderAgentPostingPolicy` if wiring; otherwise a straight cleanup PR.
- Sequence: persist + expose `post_policy` → centralize posting decision → UI control.
- Rollback: revert to always-`unposted`.

## 16. Test Plan

- **Unit** — posting decision matrix (config × assignment posting policy).
- **Integration** — auto_post posts; default does not; config persists.
- **Static** — no unreachable `auto_post` branch; `confidence_floor` used or gone.

## 17. Documentation & Training

- Help-center: "When do AI grades become visible to students?"

## 18. Open Questions

1. Wire `post_policy` (recommended, enables [GA-M3]) or remove the branch and document always-manual?
2. Remove `confidence_floor` now or implement [GA-M4](missing-4-confidence-auto-hold-threshold.md) in the same change?

## 19. References

- `server/internal/httpserver/grading_agent_queue.go` (3× `cfg.PostPolicy == "auto_post"`).
- `server/internal/repos/gradingagent/repo.go` (`UpsertConfig` omits `post_policy`/`confidence_floor`; `ConfigRow.PostPolicy/ConfidenceFloor`).
- `server/migrations/290_grading_agent.sql` (`post_policy` default `'unposted'`, `confidence_floor`).
- Related: [GA-M3](missing-3-suggest-only-batch-and-bulk-review.md), [GA-M4](missing-4-confidence-auto-hold-threshold.md), [GA-S1](simplify-1-unify-grade-write-paths.md).

# C22 — SIS, SCIM & OneRoster

> CLI parity plan. Source: `registerSISRoutes`, `registerSCIMRoutes` (`/scim/*`), `oneroster_admin.go` + `/oneroster/v1p2/*`, `admin/lrs-config`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C22 |
| **Section** | Integrations & interoperability |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Integrations / CLI |
| **Depends on** | C11, C15, C18, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Continuous roster/user sync via SIS, SCIM and OneRoster is the backbone of K-12/HE deployments, yet the CLI cannot configure or trigger any of it. Integration engineers cannot bootstrap a OneRoster connection, run a sync, or inspect SCIM provisioning results from automation — forcing fragile manual setup.

## 2. Goals

- Configure SIS/OneRoster connections and trigger/monitor syncs.
- Inspect SCIM provisioning state and reconcile drift.
- Validate a OneRoster feed before enabling continuous sync.

## 3. Non-Goals

- Being the SCIM server (the platform is); the CLI is a management/diagnostic client.
- One-off CSV import (that's C15/C11).

## 4. Personas & User Stories

- **As an integration engineer**, I want `sis config set --file oneroster.json` then `sis test`.
- **As an engineer**, I want `sis sync run --wait` and `sis sync status` to operate rostering.
- **As an admin**, I want `scim users list` / `scim status` to verify IdP provisioning.
- **As an engineer**, I want `oneroster validate --url ...` before enabling.

## 5. Functional Requirements

- **FR-1.** MUST add `sis config get|set|test` and `sis sync run|status|history` (`registerSISRoutes`; `--wait` via C18/C40).
- **FR-2.** MUST add `oneroster pull|validate|status` bridging `/oneroster/v1p2/*` + `oneroster_admin.go`.
- **FR-3.** MUST add `scim status|users list|groups list` (`/scim/*`) for diagnostics/reconciliation.
- **FR-4.** SHOULD add `sis reconcile --dry-run` (show drift between SIS and current roster).
- **FR-5.** SHOULD add `lrs config get|set` (`admin/lrs-config`, ties to C26).

## 6. Non-Functional Requirements

- **Performance** — sync run is async/job-backed; `--wait` streams progress.
- **Security** — integration-admin scope; SIS/IdP secrets via file/stdin, redacted in output.
- **Privacy & Compliance** — rosters carry student PII (FERPA/COPPA); sync reports gated by `--yes`; data-residency respected.
- **Reliability** — sync idempotent; `reconcile --dry-run` has no side effects.
- **Observability** — sync report shows created/updated/deleted/errored counts.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a OneRoster config, *When* `sis test`, *Then* connectivity + credential validity print without exposing secrets.
- **AC-2.** *Given* a config, *When* `sis sync run --wait`, *Then* the job completes and a change summary prints.
- **AC-3.** *Given* IdP provisioning, *When* `scim users list`, *Then* provisioned users are shown.

## 8. Data Model

- None client-side. Document connection config JSON (endpoint, auth, mappings).

## 9. API Surface

- `registerSISRoutes` config/sync; `/oneroster/v1p2/*` + `oneroster_admin.go`; `/scim/*`; `admin/lrs-config`.

## 10. UI / UX

- `lextures sis ...`, `lextures oneroster ...`, `lextures scim ...`.
- Sync uses the shared `--wait` job primitive (C18/C40).

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server SIS/SCIM/OneRoster/LRS handlers; job queue (C18); roster (C11); provisioning (C15).

## 13. Dependencies & Sequencing

- After: C11, C15 (roster/user substrate), C18 (`--wait`), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Destructive sync (mass-delete) | M | H | `reconcile --dry-run` default; `sync run` requires `--yes` when deletions detected |
| Vendor OneRoster quirks | H | M | `validate` surfaces schema issues before enable |

## 15. Rollout Plan

- Ship config/test + validate first, then sync run/status, then SCIM diagnostics + reconcile.
- Rollback: additive.

## 16. Test Plan

- **Unit** — config parse; secret redaction; drift diff.
- **Integration** — sync job status; OneRoster validate against fixtures.
- **Security** — secrets never printed; deletion gating.
- **E2E** — configure→validate→sync→reconcile against a mock feed.

## 17. Documentation & Training

- "Bootstrap a OneRoster connection" and "Operate nightly SIS sync from cron" runbooks.

## 18. Open Questions

1. Is sync push, pull, or bidirectional?
2. Does the server expose SCIM read endpoints for diagnostics, or only inbound provisioning?

## 19. References

- `registerSISRoutes`, `registerSCIMRoutes`, `oneroster_admin.go`.
- Related: [C11](C11-enrollments-sections.md), [C15](C15-people-provisioning.md), [C18](C18-jobs-scheduler-backups.md), [C26](C26-xapi-lrs-engagement.md).

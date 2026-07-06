# C14 — Org & org-unit administration

> CLI parity plan. Source: `admin/orgs` (38), `admin_org_units.go` (`admin/org-units`, `orgs/{orgId}/terms`, `cross-list-groups`, `branding`, `role-grants`, `parent-links`, `settings`), `admin_orgs.go`. Baseline: `orgs list/get/create`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C14 |
| **Section** | Admin & governance |
| **Severity** | MAJOR |
| **Markets** | K12 / HE |
| **Status (today)** | PARTIAL (orgs list/get/create) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Admin / CLI |
| **Depends on** | C40 |
| **Unblocks** | C11, C15, C16 |

---

## 1. Problem Statement

Org administration is only partially exposed: the CLI can list/get/create orgs but cannot update them, manage org-units (the org hierarchy), terms, branding, role-grants, or org settings. Multi-tenant admins provisioning a district hierarchy or configuring terms must use the web console.

## 2. Goals

- Full org lifecycle (update, archive) and org-unit hierarchy management.
- Term management (create academic terms/sessions).
- Org branding and settings as version-controlled config.
- Org-scoped role grants (delegated admin).

## 3. Non-Goals

- Platform-wide RBAC role definitions (see C16).
- User provisioning (see C15).

## 4. Personas & User Stories

- **As a super-admin**, I want `orgs update` and `org-units create` to build a district tree.
- **As a registrar**, I want `terms create --org O --name Fall2026 --start --end`.
- **As a brand admin**, I want `orgs branding set --org O --file brand.json`.
- **As a delegated admin**, I want `orgs role-grants add --org O --user U --role admin`.

## 5. Functional Requirements

- **FR-1.** MUST add `orgs update|archive <id>` and `orgs settings get|set <id>`.
- **FR-2.** MUST add `org-units list|create|update|delete` and `org-units move` (reparent).
- **FR-3.** MUST add `terms list|create|update|delete --org <id>`.
- **FR-4.** MUST add `orgs branding get|set <id>` (`--file`, logo asset upload).
- **FR-5.** SHOULD add `orgs role-grants list|add|remove <id>` and `orgs parent-links` management.
- **FR-6.** MAY add `orgs cross-list-groups` (shared with C11).

## 6. Non-Functional Requirements

- **Performance** — org tree fetch p95 < 1 s.
- **Security** — super-admin / org-admin scope; delegated grants respect hierarchy.
- **Privacy & Compliance** — org settings may hold data-residency/compliance flags (ties to C29).
- **Reliability** — settings set idempotent; branding asset upload resumable.
- **Backward compatibility** — existing `orgs list/get/create` unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* an org, *When* `org-units create --parent P`, *Then* the unit appears under P.
- **AC-2.** *Given* an org, *When* `terms create`, *Then* the term is listed.
- **AC-3.** *Given* a brand file, *When* `orgs branding set`, *Then* `branding get` matches.

## 8. Data Model

- None client-side. Document org settings/branding JSON.

## 9. API Surface

- `admin/orgs` CRUD; `admin/org-units`; `orgs/{orgId}/terms|branding|role-grants|settings|parent-links|cross-list-groups`.

## 10. UI / UX

- Extend `orgsCmd`; add `org-units`, `terms` groups.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server org/org-unit/term/branding handlers; branding asset store.
- Internal: `clients/cli/cmd/orgs.go`.

## 13. Dependencies & Sequencing

- After: C40.
- Before: C11/C15/C16 (org + units + terms are the container for people/roster/roles).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Org-unit reparent has invariants | M | M | Rely on server validation; surface 409s clearly |
| Branding asset upload path | M | L | Reuse `files upload` signed-URL flow |

## 15. Rollout Plan

- Ship org update/settings + org-units + terms first, then branding + role-grants.
- Rollback: additive.

## 16. Test Plan

- **Unit** — settings/branding parse.
- **Integration** — org-unit reparent; term CRUD.
- **E2E** — build a 3-level org tree with terms.

## 17. Documentation & Training

- "Provision a district hierarchy" recipe.

## 18. Open Questions

1. Are terms org-scoped or platform-scoped?
2. Is org archive a soft delete like courses?

## 19. References

- `clients/cli/cmd/orgs.go`; `admin_org_units.go`, `admin_orgs.go`.
- Related: [C15](C15-people-provisioning.md), [C16](C16-roles-permissions.md), [C11](C11-enrollments-sections.md).

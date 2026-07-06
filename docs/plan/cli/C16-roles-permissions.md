# C16 — Roles & permissions (RBAC)

> CLI parity plan. Source: `registerSettingsRoutes` (`settings` RBAC, 23 routes), `rbac_settings.go`, `orgs/{orgId}/role-grants`, `me/permissions`, `me/org-role-capabilities`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C16 |
| **Section** | Admin & governance |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Admin / CLI |
| **Depends on** | C14, C40 |
| **Unblocks** | C15 |

---

## 1. Problem Statement

Role and permission management (RBAC) — the definition of roles, their capability grants, and assignments — is entirely UI-driven. Security teams cannot version-control their permission model, audit who has what, or reproduce role setups across tenants, which is a frequent compliance and onboarding requirement.

## 2. Goals

- Define/inspect roles and their permission sets as version-controlled config.
- Grant/revoke roles to users at platform and org scope.
- Audit effective permissions for a given user ("can user X do Y?").

## 3. Non-Goals

- Org hierarchy management (see C14).
- Impersonation (see C19).

## 4. Personas & User Stories

- **As a security admin**, I want `roles export > roles.json` to snapshot the RBAC model.
- **As a security admin**, I want `roles apply --file roles.json` to reproduce it in another tenant.
- **As an admin**, I want `roles grant --user U --role admin --org O`.
- **As an auditor**, I want `permissions check --user U --capability course:manage`.

## 5. Functional Requirements

- **FR-1.** MUST add `roles list|get|create|update|delete` mapping to `settings`/`rbac_settings.go`.
- **FR-2.** MUST add `roles permissions list|add|remove <role>` (capability grants).
- **FR-3.** MUST add `roles grant|revoke --user <u> --role <r> [--org <o>]` (`role-grants`).
- **FR-4.** MUST add `permissions check --user <u> --capability <cap>` (`me/permissions`, `org-role-capabilities`).
- **FR-5.** SHOULD add `roles export|apply --file roles.json` (declarative RBAC sync with `--dry-run`).

## 6. Non-Functional Requirements

- **Performance** — role/permission list p95 < 500 ms.
- **Security** — `rbac:manage` scope required; the command MUST NOT let a user grant capabilities they lack (server-enforced) and surfaces 403 clearly.
- **Privacy & Compliance** — role assignments are audit-relevant (SOC 2) → operations logged server-side.
- **Reliability** — `apply` idempotent; `--dry-run` shows the diff.
- **Observability** — `apply` prints created/updated/removed grants.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a role file, *When* `roles apply --dry-run`, *Then* a permission diff prints and nothing changes.
- **AC-2.** *Given* a user, *When* `roles grant --role admin`, *Then* `permissions check` returns allowed for admin caps.
- **AC-3.** *Given* insufficient scope, *When* any mutate runs, *Then* exit 2 with a permission message.

## 8. Data Model

- None client-side. Document roles.json schema (role → capabilities).

## 9. API Surface

- `settings` RBAC routes (roles/capabilities CRUD); `orgs/{orgId}/role-grants`; `me/permissions`, `me/org-role-capabilities` for checks.

## 10. UI / UX

- `lextures roles ...`, `lextures permissions ...`.
- `apply` diff output; `--json` for CI gating.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server RBAC handlers; audit log (C19).
- Internal: new `cmd/roles.go`.

## 13. Dependencies & Sequencing

- After: C14 (orgs exist for scoped grants), C40.
- Before: C15 (assign roles during provisioning).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Declarative apply could lock out admins | M | H | `apply` refuses to remove the caller's own admin grant without `--force`; dry-run default |
| Capability taxonomy large | M | M | `roles capabilities list` to enumerate; validate against it |

## 15. Rollout Plan

- Ship read/check + grant/revoke first, then declarative apply.
- Rollback: additive; apply reversible via re-apply of prior snapshot.

## 16. Test Plan

- **Unit** — roles.json parse; diff computation; self-lockout guard.
- **Integration** — grant/revoke; permission check.
- **Security** — scope enforcement; self-lockout prevention.
- **E2E** — export→apply into a second tenant→verify.

## 17. Documentation & Training

- "Version-control your RBAC model" recipe.

## 18. Open Questions

1. Are roles global or org-scoped, or both?
2. Is there a canonical capability list endpoint?

## 19. References

- `rbac_settings.go`, `registerSettingsRoutes`; `me.go` permissions.
- Related: [C14](C14-org-administration.md), [C15](C15-people-provisioning.md), [C19](C19-audit-impersonation-search.md).

# C17 — Licenses, entitlements & marketplace

> CLI parity plan. Source: `admin_license.go` (`admin/licenses`), `me/entitlements`, `admin/marketplace`, `registerMarketplaceRoutes`, `admin/revenue`, `registerRevenueShareRoutes`. Baseline: `clients/cli/cmd/licenses.go`, `entitlements.go`, `marketplace.go`, `revenue.go`, `licenses_entitlements_logic.go`, `licenses_entitlements_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C17 |
| **Section** | Admin & governance |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Admin / CLI |
| **Depends on** | C14, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

License activation/seat management, entitlements, marketplace listings and revenue-share are UI-only. Admins cannot check seat consumption, apply a license key, or audit entitlements from scripts — routine tasks for procurement and renewal.

## 2. Goals

- Apply/inspect license keys and seat consumption.
- Audit user/org entitlements.
- Manage marketplace listings and view revenue-share data (for creators/orgs).

## 3. Non-Goals

- Payment collection (see C30 billing/payments).
- Contract negotiation.

## 4. Personas & User Stories

- **As an admin**, I want `licenses apply --key ...` and `licenses status` to see seat usage.
- **As a procurement lead**, I want `entitlements list --org O` for renewal planning.
- **As a creator**, I want `marketplace list` and `revenue report` to track earnings.

## 5. Functional Requirements

- **FR-1.** MUST add `licenses status|apply|list` (`admin_license.go`; `--key`).
- **FR-2.** MUST add `entitlements list [--org|--user]` (`me/entitlements`, admin entitlements).
- **FR-3.** SHOULD add `marketplace list|get|publish|unpublish` (`registerMarketplaceRoutes`).
- **FR-4.** SHOULD add `revenue report --org O` / `revenue share` (`registerRevenueShareRoutes`).
- **FR-5.** MAY add `licenses seats --by-org` breakdown.

## 6. Non-Functional Requirements

- **Performance** — status p95 < 500 ms.
- **Security** — license/billing admin scope; license keys never echoed after apply.
- **Privacy & Compliance** — revenue data is financial; export gated by `--yes`.
- **Reliability** — apply idempotent (same key twice = no double-count).
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a valid key, *When* `licenses apply`, *Then* seats increase and `status` reflects it.
- **AC-2.** *Given* an org, *When* `entitlements list`, *Then* entitlements print.
- **AC-3.** *Given* a creator, *When* `revenue report`, *Then* earnings print (gated by `--yes` on export).

## 8. Data Model

- None client-side.

## 9. API Surface

- `admin/licenses` status/apply/list; `me/entitlements` + admin entitlements; `admin/marketplace`; `admin/revenue` + revenue-share.

## 10. UI / UX

- `lextures licenses ...`, `lextures entitlements ...`, `lextures marketplace ...`, `lextures revenue ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server license/marketplace/revenue handlers; billing (C30).

## 13. Dependencies & Sequencing

- After: C14, C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| License key leakage in shell history | M | M | Support `--key-file`/stdin; never print key back |

## 15. Rollout Plan

- Ship licenses + entitlements first, then marketplace/revenue.
- Rollback: additive.

## 16. Test Plan

- **Unit** — key redaction; seat math display.
- **Integration** — apply/status; entitlements list.
- **E2E** — apply license → verify seats.

## 17. Documentation & Training

- "Check seat consumption before renewal" recipe.

## 18. Open Questions

1. Is licensing org-scoped or platform-scoped?

## 19. References

- `admin_license.go`, `registerMarketplaceRoutes`, `registerRevenueShareRoutes`.
- Related: [C14](C14-org-administration.md), [C30](C30-billing-payments-tax.md).

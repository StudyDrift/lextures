# C30 — Billing, payments, tax & revenue

> CLI parity plan. Source: `billing_http.go` (`billing`), `payments_http.go` (`payments`, `checkout`, `invoices`), `tax_http.go` (`orgs/{orgId}/tax`), `registerRevenueShareRoutes` (`revenue`), `affiliate`, `me/transactions`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C30 |
| **Section** | Commerce |
| **Severity** | MAJOR |
| **Markets** | SL / HE (CE) |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / CLI |
| **Depends on** | C14, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Billing, payments, invoices, tax configuration and revenue reporting are UI-only. Finance teams and self-learner-marketplace operators cannot pull invoices, reconcile transactions, or configure tax (Stripe Tax/VAT/GST) via automation — routine month-end finance work.

## 2. Goals

- Read billing/subscription state and export invoices/transactions for reconciliation.
- Configure org tax settings (VAT/GST/Stripe Tax).
- Pull revenue/payout and affiliate reports.

## 3. Non-Goals

- Taking payments interactively (Stripe hosted checkout is a browser flow).
- Being a full accounting system.

## 4. Personas & User Stories

- **As a finance admin**, I want `invoices list|export --org O --month 2026-06` for reconciliation.
- **As a finance admin**, I want `tax set --org O --file tax.json` (VAT id, nexus).
- **As an operator**, I want `revenue report --from --to` and `payments transactions export`.
- **As a creator**, I want `affiliate report` for referral earnings.

## 5. Functional Requirements

- **FR-1.** MUST add `billing status --org <o>` and `billing subscription get`.
- **FR-2.** MUST add `invoices list|get|export` and `payments transactions list|export` (`me/transactions`, `admin/transactions`).
- **FR-3.** MUST add `tax get|set --org <o>` (`tax_http.go`).
- **FR-4.** SHOULD add `revenue report` (`registerRevenueShareRoutes`) and `affiliate report`.
- **FR-5.** MAY add `checkout link create` (generate a hosted checkout link for a product).

## 6. Non-Functional Requirements

- **Performance** — invoice/transaction export paginated/streamed.
- **Security** — finance/billing scope; no card data ever handled by CLI (PCI: server/Stripe only).
- **Privacy & Compliance** — financial records; export gated by `--yes`; PII in invoices handled per policy.
- **Reliability** — exports idempotent; reconcilable by transaction id.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a month, *When* `invoices export --month`, *Then* invoices export as CSV/JSON.
- **AC-2.** *Given* a tax file, *When* `tax set`, *Then* `tax get` reflects VAT/nexus config.
- **AC-3.** *Given* a range, *When* `revenue report`, *Then* revenue-share figures print.

## 8. Data Model

- None client-side. Document tax config JSON.

## 9. API Surface

- `billing_http.go`; `payments_http.go` (payments/checkout/invoices); `tax_http.go`; `registerRevenueShareRoutes`; `affiliate`; `me/transactions`.

## 10. UI / UX

- `lextures billing ...`, `lextures invoices ...`, `lextures payments ...`, `lextures tax ...`, `lextures revenue ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server billing/payment/tax handlers; Stripe (server-side only).

## 13. Dependencies & Sequencing

- After: C14 (org scope), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| PCI scope creep | L | H | CLI never touches card data; checkout is a link only |
| Financial data exposure | M | H | `--yes` gate; no stdout dumps of full ledgers |

## 15. Rollout Plan

- Ship invoices/transactions export + billing status first, then tax config, then revenue/affiliate.
- Rollback: additive.

## 16. Test Plan

- **Unit** — month/range filters; export gating.
- **Integration** — invoice list; tax set/get.
- **E2E** — export a month of invoices → reconcile.

## 17. Documentation & Training

- "Month-end invoice reconciliation" runbook.

## 18. Open Questions

1. Is tax configuration org-scoped only, or also platform-level defaults?

## 19. References

- `billing_http.go`, `payments_http.go`, `tax_http.go`, `registerRevenueShareRoutes`.
- Related: [C14](C14-org-administration.md), [C17](C17-licenses-entitlements.md).

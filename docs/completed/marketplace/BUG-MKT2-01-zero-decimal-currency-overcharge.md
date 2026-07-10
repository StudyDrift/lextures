# BUG-MKT2-01 — Zero-decimal currencies (JPY) are charged 100× the displayed price

> Bug report. Follows [../../plan/_TEMPLATE.md](../../plan/_TEMPLATE.md), bug-tuned. Source features: [MKT2 — Marketplace Listing & Pricing Settings](./MKT2-course-marketplace-listing-settings.md), [MKT4 — Course Purchase & Entitlement Flow](./MKT4-course-purchase-entitlement-flow.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | BUG-MKT2-01 |
| **Section** | Marketplace — Commerce (pricing + checkout) |
| **Severity** | MAJOR |
| **Markets** | SL / HE (any tenant selling paid courses priced in JPY) |
| **Status (today)** | FIXED — currency-aware minor-unit conversion in web/server/mobile; migration 371 backfills JPY listings and flags overcharged entitlements |
| **Estimated effort** | S (1w) — currency-aware conversion + a validation guard + backfill audit |
| **Owner (proposed)** | Platform / Billing |
| **Depends on** | — |
| **Unblocks** | Correct multi-currency pricing (MKT2 FR-3), safe checkout (MKT4) |
| **Components** | `clients/web/src/lib/marketplace-price.ts`, `server/internal/service/paymentprovider/stripe.go`, `server/internal/repos/course/catalog_listing.go` |
| **Intentional?** | **No.** MKT2 offers a 14-currency selector **including JPY** (FR-3) and codifies a flat "major × 100 = cents" model (AC-3). Zero-decimal currencies are never mentioned or excluded — the 2-decimal assumption is applied universally, which is the defect. |

## 1. Problem Statement

The marketplace lets an instructor price a paid course in any of 14 currencies, **including JPY**. The pricing code universally treats the stored `price_cents` as "major units × 100" and, at checkout, passes `price_cents` straight to Stripe's `UnitAmount`. Stripe's `unit_amount` is denominated in the currency's **smallest unit**, and for **zero-decimal currencies** (JPY, and Stripe's other zero-decimal set — KRW, VND, CLP, etc.) the smallest unit *is* the whole currency unit. A course an instructor prices at **¥1,000** is stored as `price_cents = 100000` and charged by Stripe as **¥100,000** — a silent **100× overcharge** — while every UI surface (storefront badge, checkout, receipt, invoice) divides by 100 and displays the intended **¥1,000**. The customer's card is charged 100×, the recorded entitlement amount and revenue-share payout are inflated 100×, and refunds reconcile against the wrong figure.

## 2. Goals

- Amounts sent to Stripe MUST equal the price the instructor set and the learner saw, for **every** supported currency.
- Zero-decimal currencies MUST round-trip correctly through price entry, storage, display, checkout, receipts, and revenue share.
- Existing JPY (and any other zero-decimal) listings MUST be audited and corrected; no customer is silently overcharged.

## 3. Non-Goals

- No change to 2-decimal currency behaviour (USD/EUR/GBP/etc. are already correct).
- No new currencies added.
- No three-decimal currency support (BHD/KWD/etc. are not in the selectable list; out of scope unless added later).
- No change to free-claim (`price_cents = 0`) path.

## 4. Personas & User Stories

- **As an instructor pricing in JPY**, I want the price I enter (¥1,000) to be exactly what my learner is charged, not ¥100,000.
- **As a learner buying a ¥1,000 course**, I want my card charged ¥1,000 and my receipt to match the charge.
- **As a finance/ops admin**, I want recorded entitlement amounts, payouts, and refunds to equal the real Stripe charge so reconciliation and revenue share are correct.

## 5. Functional Requirements

- **FR-1.** Price conversion MUST be **currency-aware**: for zero-decimal currencies, a major-unit input MUST map 1:1 to the Stripe amount (no ×100), and display MUST NOT divide by 100.
- **FR-2.** A single source of truth for a currency's exponent (0 vs. 2 decimals) MUST be defined and shared by web (entry/format) and server (Stripe amount), so entry, storage, Stripe `UnitAmount`, display, and receipts all agree.
- **FR-3.** The Stripe line-item amount MUST be derived from the stored price via the currency's exponent, not by assuming `price_cents / ... ×100` semantics.
- **FR-4.** Server-side validation MUST enforce currency-appropriate minimums (Stripe's per-currency minimum charge) and reject fractional minor units for zero-decimal currencies.
- **FR-5.** A migration/audit MUST identify existing zero-decimal listings (`price_currency IN (zero-decimal set) AND price_cents > 0`) and correct them, plus reconcile any already-completed JPY purchases.
- **FR-6.** Until a correct implementation ships, the currency selector SHOULD hide/disable zero-decimal currencies (interim mitigation) rather than silently overcharge.

## 6. Non-Functional Requirements

- **Correctness (financial)** — charged amount == displayed amount == recorded amount, for all currencies; verified by tests per currency class.
- **Security/Trust** — no silent overcharge; chargebacks and refund disputes are a legal/reputational risk.
- **Observability** — log the currency + computed Stripe amount at checkout creation; alert on any charge where recorded amount ≠ displayed price.
- **Backward compatibility** — storage format decision (keep `price_cents` as Stripe smallest-unit vs. reinterpret) MUST include a backfill; document the chosen invariant.
- **Internationalization** — `Intl.NumberFormat` already respects currency fraction digits for **display**; the bug is the manual ×100/÷100, not `Intl`.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor sets a JPY price of ¥1,000, *When* saved, *Then* the stored value and the Stripe `unit_amount` both represent **¥1,000** (Stripe `unit_amount = 1000`), not 100000.
- **AC-2.** *Given* a learner checks out that course, *When* the Stripe session is created, *Then* the charge is **¥1,000** and the receipt/invoice/entitlement all read ¥1,000.
- **AC-3.** *Given* a USD price of $19.99, *Then* behaviour is unchanged (`unit_amount = 1999`, charge $19.99).
- **AC-4.** *Given* a zero-decimal currency, *When* a fractional amount (e.g. "1000.50") is entered, *Then* it is rejected with a clear message.
- **AC-5.** *Given* the audit runs, *Then* every pre-fix zero-decimal listing is corrected and every completed zero-decimal purchase is flagged for reconciliation/refund.

## 8. Data Model

- No new tables. Affects `course.courses.price_cents` / `price_currency` interpretation and `billing` entitlement `amount_paid_cents` for zero-decimal currencies.
- Define the invariant explicitly (recommended: **`price_cents` stores the Stripe smallest-unit amount** — i.e. for JPY it already *is* the yen count — and the web must stop multiplying by 100 for zero-decimal currencies).
- Backfill: `UPDATE` zero-decimal listings that were entered under the buggy ×100 assumption (divide by 100); reconcile completed purchases (`payments`/entitlements) for those courses.

## 9. API Surface

- No route changes. `PUT /api/v1/courses/{code}/catalog-listing` (`priceCents`/`priceCurrency`) semantics are clarified: server must validate `priceCents` against the currency exponent.
- Stripe Checkout `unit_amount` derivation changes (server).

## 10. UI / UX

- Marketplace settings fee editor: currency-aware minor-unit handling; hide the "cents" affordance for zero-decimal currencies; show the correct minimum.
- Storefront badge / checkout / My Purchases / invoice: values become correct automatically once the ÷100 is made currency-aware (they already route through `formatMarketplacePrice` / `Intl`).

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- `clients/web/src/lib/marketplace-price.ts` — `majorUnitsToPriceCents` (`:25`, `Math.round(value*100)` at `:32`), `priceCentsToMajorUnits` (`:36`), `formatMarketplacePrice` (`:41`, `priceCents/100` at `:52`), `MARKETPLACE_CURRENCIES` includes `jpy` (`:11`), `validateMarketplaceAmount` min-charge is currency-blind (`:63`).
- `server/internal/service/paymentprovider/stripe.go` — `UnitAmount: stripe.Int64(int64(req.PriceCents))` (`:72`) — the charge.
- `server/internal/service/paymentprovider/checkout.go` — passes `price.PriceCents` / `price.Currency` (`:82`–`:83`).
- `server/internal/repos/course/catalog_listing.go` — allowed-currency set **includes `jpy`** (`:36`).
- `server/internal/service/billing/invoice_pdf.go` — `formatCurrency(cents, currency)` also assumes 2 decimals for receipts.
- `server/internal/service/billing/revenue_share.go` — payout amounts inherit the inflated figure.
- **Mobile clients share the display half of the bug** (so the overcharge is silent on every surface): iOS `clients/ios/Lextures/Core/LMS/PathsLogic.swift:28` (`Decimal(cents) / 100`) and Android `clients/android/app/src/main/kotlin/com/lextures/android/core/lms/PathsLogic.kt:23` (`formatter.format(cents / 100.0)`) both divide by 100 for every currency. Any currency-exponent fix MUST land in web + iOS + Android together.

## 13. Dependencies & Sequencing

- Ship the interim mitigation (FR-6: disable JPY selection) immediately; the full currency-aware fix + backfill follows.
- No dependency on other in-flight epics.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Live customers already overcharged in JPY | L–M | **H** | Audit `payments`/entitlements for zero-decimal currencies; proactively refund the 99× excess |
| Backfill misinterprets which listings were entered pre-fix | M | H | Gate backfill on `price_currency ∈ zero-decimal ∧ price_cents % 100 == 0` heuristics + manual review; snapshot before update |
| Storage-invariant flip double-corrects display | M | M | Change entry (×100), display (÷100), and Stripe amount **together**, with per-currency tests |
| Future 3-decimal currency reintroduces the bug | L | M | Drive all conversions from a single currency-exponent table |

## 15. Rollout Plan

- **Step 0 (now):** hide/disable zero-decimal currencies in the selector (stops new bad listings).
- **Step 1:** introduce shared currency-exponent helper; make web entry/format and server Stripe amount currency-aware; per-currency unit tests.
- **Step 2:** migration to correct existing zero-decimal listings; reconciliation report for completed purchases; refunds where charged.
- **Step 3:** re-enable zero-decimal currencies.
- Rollback: revert the selector change; conversion changes are behind tests.

## 16. Test Plan

- **Unit** — `majorUnitsToPriceCents`/`formatMarketplacePrice` for a 2-decimal (USD) and a 0-decimal (JPY) currency; Stripe `unit_amount` derivation per currency class.
- **Integration** — create checkout session for a JPY course; assert Stripe `LineItems[0].PriceData.UnitAmount == 1000` for a ¥1,000 course.
- **E2E** — list a ¥1,000 course → badge shows ¥1,000 → checkout → (test-mode) charge is ¥1,000 → receipt ¥1,000.
- **Regression** — USD/EUR/GBP unchanged.
- **Audit script test** — seed a mispriced JPY listing + completed purchase; assert the audit flags and corrects them.

## 17. Documentation & Training

- Document the `price_cents` invariant (Stripe smallest-unit) in the billing runbook and MKT2/MKT4 plans.
- Ops runbook: "Reconcile zero-decimal overcharges."

## 18. Root Cause & Evidence

**Entry (web) multiplies by 100 for every currency** — `clients/web/src/lib/marketplace-price.ts:25`:

```ts
export function majorUnitsToPriceCents(amount: string): number | null {
  ...
  return Math.round(value * 100)   // :32 — ×100 regardless of currency
}
```

**Display (web) divides by 100 for every currency** — `marketplace-price.ts:41`:

```ts
export function formatMarketplacePrice(priceCents, currency, ...) {
  if (priceCents <= 0) return freeLabel
  return new Intl.NumberFormat(locale, { style:'currency', currency: currency.toUpperCase() })
    .format(priceCents / 100)       // :52 — ÷100 regardless of currency
}
```

For JPY these two cancel out **for display** (enter 1000 → store 100000 → show ¥1,000), which is exactly why the overcharge is silent.

**Charge (server) sends the stored value straight to Stripe** — `server/internal/service/paymentprovider/stripe.go:72`:

```go
PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
    Currency:   stripe.String(currency),
    UnitAmount: stripe.Int64(int64(req.PriceCents)),   // :72 — 100000 for a ¥1,000 course
},
```

Stripe interprets `unit_amount` as the smallest unit; JPY has **no** subunit, so `100000` = **¥100,000**. There is **no** zero-decimal handling anywhere in `server/` (grep for `zero.?decimal|jpy|krw|smallest.?unit` returns only unrelated MIME/allow-list hits), and JPY is an accepted currency at `server/internal/repos/course/catalog_listing.go:36` and offered in the web selector at `marketplace-price.ts:11`.

**Net effect for a ¥1,000 JPY course:** badge/checkout/receipt/entitlement all read **¥1,000**; Stripe charges **¥100,000**; `AmountPaidCents` is recorded as 100000 and revenue-share pays out on the inflated base.

**Contradicts the plan (proves it is a bug, not a decision):**

- MKT2 **FR-3**: "a currency selector (default … `usd`) and an amount input in major units, stored as `price_cents`" — currency is user-selectable and the JPY option is present.
- MKT2 **AC-3**: "19.99 USD … persists as `price_cents = 1999`" — the plan's only worked example encodes the flat ×100 model and never addresses zero-decimal currencies.
- MKT2 **§14 Risk** only anticipates "currency mismatch with **Stripe account currency**," not zero-decimal denomination — the failure mode was unconsidered, i.e. unintentional.

## 19. References

- Buggy files: `clients/web/src/lib/marketplace-price.ts` (`:11`, `:25`, `:32`, `:36`, `:52`, `:63`); `server/internal/service/paymentprovider/stripe.go:72`; `server/internal/service/paymentprovider/checkout.go:82`; `server/internal/repos/course/catalog_listing.go:36`; `server/internal/service/billing/invoice_pdf.go:194`.
- Plans: [MKT2](../../completed/marketplace/MKT2-course-marketplace-listing-settings.md) (§8, FR-3, AC-3, §14), [MKT4](../../completed/marketplace/MKT4-course-purchase-entitlement-flow.md).
- External: Stripe "Zero-decimal currencies" (JPY, KRW, VND, CLP, …) — `unit_amount` is in the currency's smallest unit; ISO 4217 currency exponents.

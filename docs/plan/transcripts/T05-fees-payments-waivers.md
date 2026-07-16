# T05 — Transcript Fees, Payments & Waivers

> Implementation plan. Per-order/per-recipient/rush fees, fee waivers, and refunds via existing Stripe billing. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T05 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MAJOR |
| **Markets** | HE · SL |
| **Status (today)** | MISSING — transcript requests are free; there is no fee schedule, checkout, waiver, or refund. Most registrars charge per official transcript and need to collect it at order time. |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce/Billing squad + Registrar/SIS |
| **Depends on** | T02 (orders), T03 (payment gate), [15.3 Stripe billing](../../completed/15-self-learner-specific/15.3-billing-stripe.md) |
| **Unblocks** | T06 (deliver only after payment), T12 (revenue analytics) |

---

## 1. Problem Statement

Institutions typically charge for official transcripts (a flat fee, plus rush and per-recipient
surcharges) and offer fee waivers for eligible students. Lextures charges nothing and has no
checkout for transcripts, so it can't replace an existing paid transcript service. This story adds
a configurable fee schedule, a checkout that reuses the shipped Stripe integration, fee waivers,
and refunds — with payment status wired as a hard gate in the order lifecycle.

## 2. Goals

- Let orgs configure a **fee schedule**: base fee, rush surcharge, per-recipient fee, per-delivery-method surcharge, free allotment.
- Compute an itemized **order total** and collect payment at checkout via existing Stripe billing.
- Support **fee waivers** (eligibility-based or admin-granted, e.g. fee-waiver codes) that zero or reduce the total.
- Support **refunds** (full/partial) when an order is canceled or fails, and reflect them in state.
- Make **payment satisfied** a hard gate before any delivery (T03/T06).

## 3. Non-Goals

- General subscription/course commerce (owned by [15.3](../../completed/15-self-learner-specific/15.3-billing-stripe.md)); this reuses its Stripe plumbing.
- Tax handling beyond what [15.13 tax compliance](../../completed/15-self-learner-specific/15.13-tax-compliance.md) provides — integrate, don't reimplement.
- Payouts/revenue share to third parties.

## 4. Personas & User Stories

- **As a registrar/admin**, I want to set transcript fees (base, rush, per recipient) so that we recover costs.
- **As a student**, I want to see an itemized total before I pay so that there are no surprises.
- **As an eligible student**, I want a fee waiver applied so that cost is not a barrier.
- **As a registrar**, I want to grant a waiver or issue a refund so that I can handle exceptions.
- **As finance**, I want transcript revenue reconciled with Stripe so that the books match.

## 5. Functional Requirements

- **FR-1.** Orgs MUST configure a fee schedule: `base_fee`, `rush_fee`, `per_recipient_fee`, per-delivery-method surcharges, currency, and an optional `free_allotment` (N free per student per period).
- **FR-2.** The system MUST compute an itemized total for an order (base + per-item recipient/rush/method surcharges − waivers − free allotment).
- **FR-3.** Checkout MUST reuse the existing Stripe integration (PaymentIntent/Checkout) and store the payment reference on the order.
- **FR-4.** The system MUST support waivers: admin-granted per order, one-time **waiver codes**, and rule-based eligibility (e.g. financial-aid flag); a fully waived order skips payment.
- **FR-5.** Payment MUST be a hard gate: T03 MUST NOT advance to `processing` until the order is `paid` or `waived`; T06 MUST re-check before delivery.
- **FR-6.** The system MUST support refunds (full/partial) via Stripe on cancel/failure and update payment state and order state accordingly.
- **FR-7.** All monetary amounts MUST be stored in minor units with explicit currency; zero-decimal currencies handled (see migration `371` precedent).
- **FR-8.** The system MUST be idempotent against Stripe webhooks (no double-charge/double-refund) and reconcile async payment confirmations.
- **FR-9.** Receipts MUST be generated and available to the student; refund receipts on refund.
- **FR-10.** Free allotment and waivers MUST be auditable (who/why/when).

## 6. Non-Functional Requirements

- **Performance** — total computation < 100ms; checkout redirect/confirm p95 within Stripe norms.
- **Security** — no card data touches Lextures (Stripe-hosted); webhook signatures verified; waiver-grant RBAC-gated.
- **Privacy & Compliance** — PCI scope minimized (SAQ-A); tax/VAT via 15.13; refunds logged.
- **Accessibility** — checkout summary and receipts WCAG 2.1 AA.
- **Scalability** — reuse existing billing tables/queues; webhook processing idempotent and queued.
- **Reliability** — payment state derived from Stripe truth; reconcile job repairs drift; no delivery without confirmed payment.
- **Observability** — `transcript_payment_total{result}`, `transcript_refund_total`, `transcript_waiver_applied_total`, revenue gauge.
- **Maintainability** — fee computation pure and unit-tested; single source for money math.
- **Internationalization** — currency/locale formatting; multi-currency per org.
- **Backward compatibility** — orgs with no fee schedule default to $0 (free), preserving current behavior.

## 7. Acceptance Criteria

- **AC-1.** *Given* base $10 + $5/recipient and an order to 2 recipients with rush ($3), *When* totaled, *Then* the itemized total is correct and shown before payment.
- **AC-2.** *Given* an unpaid, non-waived order, *When* T03 tries to reach `processing`, *Then* it is blocked pending payment.
- **AC-3.** *Given* a valid waiver code, *When* applied, *Then* the total is reduced/zeroed, and a $0 order proceeds without Stripe checkout.
- **AC-4.** *Given* a paid order that is canceled, *When* refunded, *Then* Stripe issues the refund once (idempotent) and the order reflects `refunded`.
- **AC-5.** *Given* a duplicate Stripe webhook, *When* processed, *Then* no double state change occurs.
- **AC-6.** *Given* a completed payment, *When* the receipt is requested, *Then* an itemized receipt is available.

## 8. Data Model

Migration `382_transcript_fees.sql` (indicative):

```sql
CREATE TABLE transcripts.fee_schedule (
    org_id            UUID PRIMARY KEY REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    currency          TEXT NOT NULL DEFAULT 'usd',
    base_fee          INT  NOT NULL DEFAULT 0,     -- minor units
    rush_fee          INT  NOT NULL DEFAULT 0,
    per_recipient_fee INT  NOT NULL DEFAULT 0,
    method_surcharges JSONB NOT NULL DEFAULT '{}', -- {postal_mail: 200, ...}
    free_allotment    INT  NOT NULL DEFAULT 0,     -- free official transcripts per student per period
    allotment_period  TEXT NOT NULL DEFAULT 'lifetime' CHECK (allotment_period IN ('lifetime','year','term')),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transcripts.waiver_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    code        TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('full','percent','amount')),
    value       INT,                                -- percent (0-100) or minor-unit amount
    max_uses    INT,
    used_count  INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_by  UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    UNIQUE (org_id, code)
);

-- Payment state on the order (extends T02 orders):
ALTER TABLE transcripts.orders
    ADD COLUMN payment_status TEXT NOT NULL DEFAULT 'unpaid'
        CHECK (payment_status IN ('unpaid','pending','paid','waived','refunded','partially_refunded','free')),
    ADD COLUMN payment_ref    TEXT,                 -- Stripe PaymentIntent id
    ADD COLUMN waiver_id      UUID REFERENCES transcripts.waiver_codes(id),
    ADD COLUMN amount_refunded INT NOT NULL DEFAULT 0;
```

- Reuse existing billing/webhook tables from 15.3 for Stripe event bookkeeping.

## 9. API Surface

- `GET/PUT /api/v1/admin/transcripts/fees` — fee schedule (RBAC).
- `GET/POST /api/v1/admin/transcripts/waiver-codes` — manage waiver codes.
- `POST /api/v1/admin/transcripts/orders/{id}/waive` — admin grant waiver.
- `POST /api/v1/admin/transcripts/orders/{id}/refund` — full/partial refund.
- `GET  /api/v1/transcripts/orders/{id}/quote` — itemized total (with optional `waiverCode`).
- `POST /api/v1/transcripts/orders/{id}/checkout` — create Stripe PaymentIntent/session; returns client secret/URL.
- `GET  /api/v1/transcripts/orders/{id}/receipt` — receipt PDF/JSON.
- Stripe webhook handler extended for transcript payments (idempotent). OpenAPI updated.

## 10. UI / UX

- **Order checkout step** (after consent T04): itemized summary (base, per-recipient, rush, method surcharge, waiver, total), waiver-code field, Stripe payment element, free/waived path skips payment.
- **Registrar fee config** (T12 console): fee schedule form, waiver codes, per-order waive/refund actions.
- **Receipts**: student order detail shows paid amount + downloadable receipt; refund shows refunded amount.
- States: computing quote, invalid/expired waiver code, payment processing, payment failed (retry), refunded.
- Accessibility: summary table + payment element labeled; keyboard/SR flows.
- i18n + multi-currency formatting.

## 11. AI / ML Considerations

None.

## 12. Integration Points

- **Internal:** [15.3 Stripe billing](../../completed/15-self-learner-specific/15.3-billing-stripe.md) plumbing/webhooks, [15.13 tax](../../completed/15-self-learner-specific/15.13-tax-compliance.md), T02 orders, T03 payment gate, T06 delivery guard, T12 revenue analytics.
- **External:** Stripe (PaymentIntents, Refunds, Webhooks).
- **Emissions:** `transcript.payment.succeeded/refunded`, `transcript.waiver.applied`.

## 13. Dependencies & Sequencing

- After: T02, T03, 15.3. Before: T06 (deliver only when paid/waived), T12 revenue.
- Shared infra: Stripe, webhook queue, receipt rendering.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Double charge/refund from webhook retries | M | H | Idempotency keys + event dedupe (reuse 15.3 patterns); reconcile job |
| Deliver before payment confirmed (async) | M | H | Delivery guard requires `paid`/`waived`/`free`; async confirm advances state |
| Money math / zero-decimal currency bugs | M | H | Minor-unit integers only; currency-aware; golden tests (see migration 371 fix) |
| Waiver-code abuse | M | M | max_uses, expiry, per-student limits, audit |

## 15. Rollout Plan

- Flag `ff_transcripts`; fees behind `transcripts.fees_enabled` (default off → current free behavior).
- Sequence: fee schema → quote/computation → checkout (Stripe) → waivers → refunds → enable per org.
- Pilot: one org sets real fees; run test charges + a refund in Stripe test mode, then live.
- Rollback: set `fees_enabled` off → orders become free; existing paid orders unaffected.

## 16. Test Plan

- **Unit** — fee computation matrix; waiver kinds; currency/minor-unit math.
- **Integration** — checkout → PaymentIntent → webhook → `paid`; refund path; waiver zero-total path.
- **E2E** — student pays and gets receipt; admin waives; admin refunds.
- **Security** — webhook signature; RBAC on fee/waiver/refund; no card data stored.
- **Accessibility** — checkout + receipt axe.
- **Reliability** — duplicate-webhook idempotency; reconcile job repairs drift.

## 17. Documentation & Training

- Registrar/finance: configuring fees, waivers, refunds; reconciliation with Stripe.
- Student help: transcript costs, fee waivers, receipts.
- API reference for fees/checkout/refund endpoints.

## 18. Open Questions

1. Free allotment semantics (per academic year vs. term vs. lifetime) — org-configurable default?
2. Does Lextures take a platform fee per transcript, or is all revenue the institution's?
3. Automatic waiver eligibility source (financial-aid flag, free/reduced-lunch status for K-12)?

## 19. References

- Existing: [15.3 Stripe billing](../../completed/15-self-learner-specific/15.3-billing-stripe.md), migration `371_zero_decimal_currency_fix.sql`, `372_email_provider_ses` (receipts).
- Related plans: [T02](../../completed/transcripts/T02-recipient-directory-and-orders.md), [T03](../../completed/transcripts/T03-order-lifecycle-fulfillment-holds.md), [T06](T06-electronic-delivery-standards.md), [15.13 tax](../../completed/15-self-learner-specific/15.13-tax-compliance.md).

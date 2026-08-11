# What a coupon costs you (creators)

When a learner buys your course with a coupon, the platform charges the **discounted** amount. Revenue share and platform fees are calculated on that charged amount — not on the list price. You absorb the cost of your own promotion; the fee percentage itself does not change.

## Worked example

| Line | Amount |
|---|---|
| List price | $40.00 (4000¢) |
| Coupon `LAUNCH25` (25% off) | −$10.00 (1000¢) |
| Charged to learner (before tax) | $30.00 (3000¢) |

If revenue share is enabled at, say, 10% platform fee on course sales:

- Platform fee ≈ 10% × **$30.00** = $3.00
- Creator net ≈ $27.00 (before payment-processor fees)

Not 10% × $40.00. Tax (when collection is enabled) is also computed by the payment provider on the discounted line item.

## Caps and seats

- **Total redemptions** — once the cap is reached, new learners see “no seats left”.
- **Per learner** — defaults to one use; a refund releases the seat back to the pool (the learner may or may not be able to reuse the same code depending on their per-user count of *redeemed* rows after release).
- **Reservations** — when someone starts checkout, a seat is held for a short TTL so two people cannot claim the last seat at once. Abandoned checkouts free the seat when the TTL expires.

## 100% off codes

A 100% code (or a discount that clamps below the payment provider's minimum charge) grants access with no card charge. That still consumes a seat under your caps and is reported with acquisition source `coupon`, so finance can distinguish it from a course that is listed free for everyone.

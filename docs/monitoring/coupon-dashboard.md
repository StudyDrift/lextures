# Course coupons — Grafana dashboard & alerts (MKTC.7)

Metric namespace: `lextures_*` (see `server/internal/telemetry`).

## Panels (recommended)

| Panel | Query sketch | Notes |
|---|---|---|
| Apply attempts by result | `sum by (result) (rate(lextures_coupon_apply_total[5m]))` | Includes `ok`, `not_found`, `rate_limited`, reasons |
| Cool-downs | `rate(lextures_coupon_apply_cooldown_total[5m])` | FR-2 |
| Discount given / day | `increase(lextures_coupon_discount_cents_total[1d]) / 100` | Minor units → currency display in legend |
| Free grants | `rate(lextures_coupon_free_grant_total[5m])` | 100%-off path |
| Clamped-to-free | `rate(lextures_coupon_clamped_to_free_total[5m])` | Residual below Stripe min |
| Redemptions completed | `rate(lextures_coupon_redeemed_total[5m])` | Paid webhook + free grant |
| Reservation expiry | `rate(lextures_coupon_reservation_expired_total[5m])` | Sweeper |
| Releases by reason | `sum by (reason) (rate(lextures_coupon_release_total[5m]))` | `expired`, `refund`, `ops_manual`, … |
| Admin API | `sum by (route, result) (rate(lextures_coupon_admin_request_total[5m]))` | `summary`, `export_csv`, … |
| Creates by type | `sum by (discount_type) (rate(lextures_coupon_created_total[5m]))` | percent / fixed |
| Web redirects (mobile) | `sum by (platform) (rate(lextures_coupon_web_redirect_total[5m]))` | MKTC.6/7 |

## Alerts

| Alert | Condition | Runbook section |
|---|---|---|
| Enumeration burst | `not_found` share of `coupon_apply_total` &gt; 30% over 15m | [Alert: enumeration](../runbooks/coupons.md#alert-enumeration) |
| Redemption velocity | &gt; 50 redemptions of one coupon in 10m (log-derived or labeled counter if added) | [Alert: velocity](../runbooks/coupons.md#alert-velocity) |
| Ledger drift | `coupon_redeemed_total` vs coupon-tagged `marketplace_purchase_completed` diverge over 1h | [Alert: ledger drift](../runbooks/coupons.md#alert-ledger-drift) |
| Checkout breakage | reservation expiry rate &gt; 50% of reserves over 1h | [Alert: reservation expiry](../runbooks/coupons.md#alert-reservation-expiry) |

Example Prometheus rules (drop into the observability stack):

```yaml
groups:
  - name: lextures-coupons
    rules:
      - alert: CouponEnumerationBurst
        expr: |
          (
            sum(rate(lextures_coupon_apply_total{result="not_found"}[15m]))
            /
            clamp_min(sum(rate(lextures_coupon_apply_total[15m])), 1e-9)
          ) > 0.30
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: High coupon not_found rate (possible enumeration)
          runbook: docs/runbooks/coupons.md#alert-enumeration

      - alert: CouponReservationExpiryHigh
        expr: |
          (
            sum(rate(lextures_coupon_reservation_expired_total[1h]))
            /
            clamp_min(sum(rate(lextures_coupon_reserve_total[1h])), 1e-9)
          ) > 0.50
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: High coupon reservation expiry rate
          runbook: docs/runbooks/coupons.md#alert-reservation-expiry
```

Velocity (per-coupon) is best implemented from Postgres or a labeled counter once operators need it; until then use the SQL in the runbook during triage.

## Ownership

Commerce / growth on-call owns these alerts; security is paged only for sustained enumeration after runbook step 1 confirms no product incident (e.g. bad share-link campaign).

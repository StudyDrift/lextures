# Course coupons — operations runbook

Plan: **MKTC.1** (data model + discount engine) · **MKTC.2** (creator management API) ·
**MKTC.3** (learner preview / checkout / redemption) · **MKTC.4** (web creator manager) ·
**MKTC.7** (abuse prevention, analytics, dashboards).

## Purpose

Inspect seat usage for a coupon, free a stuck reservation, reconcile the denormalized
`redeemed_count` against the ledger, respond to abuse alerts, and disable the feature safely.

## Tables

| Table | Role |
|---|---|
| `billing.course_coupons` | Creator codes (percent/fixed, window, caps) |
| `billing.coupon_redemptions` | Authoritative seat ledger (`reserved` / `redeemed` / `released`) |
| `billing.coupon_attempts` | Failed apply attempts (30-day retention; hashed codes; IP prefix) |

Consumed seats = rows in (`reserved`, `redeemed`) whose `expires_at` is null or in the future.
Expired `reserved` rows do **not** burn a seat even before the sweeper runs.

## Inspect seat usage

```sql
-- Coupon row + denormalized display counter
SELECT id, course_id, code, status, max_redemptions, redeemed_count, starts_at, ends_at
FROM billing.course_coupons
WHERE course_id = :course_id
ORDER BY created_at DESC;

-- Authoritative consumed seats (one query for a page of coupons)
SELECT coupon_id,
       COUNT(*) FILTER (
         WHERE status = 'reserved'
           AND (expires_at IS NULL OR expires_at > NOW())
       ) AS reserved,
       COUNT(*) FILTER (WHERE status = 'redeemed') AS redeemed
FROM billing.coupon_redemptions
WHERE coupon_id = ANY(:coupon_ids)
GROUP BY coupon_id;
```

## Manually release a stuck reservation

```sql
-- Find open reservations for a learner / session
SELECT id, coupon_id, user_id, status, checkout_session_id, expires_at, reserved_at
FROM billing.coupon_redemptions
WHERE status = 'reserved'
  AND (user_id = :user_id OR checkout_session_id = :session_id);

-- Prefer the app helper (keeps metrics):
--   billing.ReleaseCouponReservation(ctx, pool, redemptionID, "ops_manual")
-- Or SQL (does not touch redeemed_count — only valid for reserved rows):
UPDATE billing.coupon_redemptions
SET status = 'released', released_at = NOW(), expires_at = NULL
WHERE id = :redemption_id AND status = 'reserved';
```

The sweeper job `scheduled.coupon_reservation_sweep` (every 5 minutes) releases rows past
`expires_at`. Correctness never depends on it — seat counts already exclude expired reservations.

## Reconciliation: `redeemed_count` vs ledger

```sql
SELECT c.id, c.code, c.redeemed_count AS denorm,
       COALESCE(l.redeemed, 0) AS ledger
FROM billing.course_coupons c
LEFT JOIN (
  SELECT coupon_id, COUNT(*)::int AS redeemed
  FROM billing.coupon_redemptions
  WHERE status = 'redeemed'
  GROUP BY coupon_id
) l ON l.coupon_id = c.id
WHERE c.redeemed_count IS DISTINCT FROM COALESCE(l.redeemed, 0);
```

Any non-empty result means a bug or a partial manual edit. Prefer fixing via the repo
`RedeemCoupon` / `ReleaseCouponReservation` path rather than hand-updating both columns.

## Discount arithmetic

Server-side only: `server/internal/service/coupons` (`ApplyDiscount`, `Evaluate`). Clients never
send amounts. See also the short note in `docs/marketplace-courses-authoring.md`.

## DSAR / erasure

- `billing.coupon_redemptions.user_id` → `ON DELETE CASCADE` (erasure removes the learner's ledger rows).
- `billing.course_coupons.created_by` → `ON DELETE SET NULL` (coupon codes stay for the course).
- Access/portability export includes `couponRedemptions` in the GDPR DSAR archive.

## How to archive a leaked code

1. Confirm the course and code:

```sql
SELECT id, code, status, course_id
FROM billing.course_coupons
WHERE code = upper(:code) AND status <> 'archived'
ORDER BY created_at DESC;
```

2. Prefer the creator API (keeps audit + metrics):

```http
DELETE /api/v1/courses/{course_code}/coupons/{coupon_id}
```

That soft-archives (`status = 'archived'`). Existing reservations still honour the price they
were quoted (MKTC.3); new applies fail as inactive. To pause without archiving (reversible):

```http
PATCH /api/v1/courses/{course_code}/coupons/{coupon_id}
{ "status": "disabled" }
```

3. SQL fallback (does not write admin audit):

```sql
UPDATE billing.course_coupons
SET status = 'archived', updated_at = NOW()
WHERE id = :coupon_id;
```

## Who can manage coupons on a course

Authorisation matches marketplace listing settings: capability
`course:{course_code}:item:create` (owner, teacher, designer). TAs, students, observers, and
parents do **not** hold it. Platform admins only reach the routes if they already have that
course-scoped grant (no separate cross-course admin surface in MKTC.2).

Feature flags (both required; routes 404 otherwise):

| Flag | Default | Surface |
|---|---|---|
| `ffCourseMarketplace` | ON | Settings → Global platform |
| `ffCourseCoupons` | **ON** by default (MKTC.7 GA); explicit OFF preserved | Settings → Global platform |

```sql
SELECT ff_course_marketplace, ff_course_coupons
FROM settings.platform_app_settings
WHERE id = 1;
```

## Metrics

| Metric | Labels |
|---|---|
| `lextures_coupon_reserve_total` | `result` |
| `lextures_coupon_redeem_total` | `result` |
| `lextures_coupon_release_total` | `reason` |
| `lextures_coupon_reservation_expired_total` | — |
| `lextures_coupon_admin_request_total` | `route`, `result` |
| `lextures_coupon_created_total` | `discount_type` |
| `lextures_coupon_status_changed_total` | `to` |
| `lextures_coupon_apply_total` | `result` (preview outcomes) |
| `lextures_coupon_checkout_created_total` | `discounted` |
| `lextures_coupon_redeemed_total` | — (completed paid or free grant) |
| `lextures_coupon_discount_cents_total` | — |
| `lextures_coupon_free_grant_total` | — |
| `lextures_coupon_clamped_to_free_total` | — |
| `lextures_coupon_apply_cooldown_total` | — (MKTC.7 cool-down hits) |
| `lextures_coupon_web_redirect_total` | `platform` |

Dashboard/alerts: [docs/monitoring/coupon-dashboard.md](../monitoring/coupon-dashboard.md).

## Attempt log (enumeration)

```sql
-- Recent failed applies for a course (hashes only — no raw not_found codes)
SELECT user_id, reason, ip_prefix, code_hash, created_at
FROM billing.coupon_attempts
WHERE course_id = :course_id
  AND created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC
LIMIT 200;

-- not_found share last 15 minutes (operator triage; metrics preferred)
SELECT reason, COUNT(*)
FROM billing.coupon_attempts
WHERE created_at > NOW() - INTERVAL '15 minutes'
GROUP BY reason;
```

Retention job: `scheduled.coupon_attempts_retention` (daily) deletes rows older than 30 days.

## Kill a leaked code (fast)

1. Pause (reversible): `PATCH .../coupons/{id}` with `{"status":"disabled"}`.
2. Archive: `DELETE .../coupons/{id}` (soft archive).
3. Confirm new applies fail (`inactive` / not listed).
4. Optionally lower `coupon_max_percent_off` platform-wide if percent abuse is systemic.

Open reservations keep the quoted price until expiry or payment (MKTC.3).

## Disable the feature (rollback)

Instant, no deploy:

```sql
UPDATE settings.platform_app_settings
SET ff_course_coupons = false
WHERE id = 1;
```

Or Settings → Global platform → turn off **Course coupons**. Redeemed entitlements remain valid.
In-flight reservations can still complete. Default in code is **ON** (MKTC.7 GA); tenants with an
explicit DB `false` stay off until they re-enable.

## Discount ceiling

```sql
SELECT coupon_max_percent_off FROM settings.platform_app_settings WHERE id = 1;
-- NULL or 100 = uncapped. Example: cap at 50%:
UPDATE settings.platform_app_settings SET coupon_max_percent_off = 50 WHERE id = 1;
```

Creates with `percentOff` above the ceiling return `422`.

## Alert response

### Alert: enumeration

<a id="alert-enumeration"></a>

1. Confirm `not_found` rate on Grafana / Prometheus.
2. Query `billing.coupon_attempts` for the top `user_id` / `ip_prefix`.
3. Cool-down should already throttle per (user, course). If many accounts, consider temporary
   platform disable or IP block at the edge.
4. Check whether a legitimate campaign is driving bad share links (learners mistyping).

### Alert: velocity

<a id="alert-velocity"></a>

```sql
SELECT coupon_id, COUNT(*) AS n
FROM billing.coupon_redemptions
WHERE status = 'redeemed'
  AND redeemed_at > NOW() - INTERVAL '10 minutes'
GROUP BY coupon_id
HAVING COUNT(*) > 50;
```

If a single unlimited code is leaking: **disable/archive** immediately (see kill switch above).

### Alert: ledger drift

<a id="alert-ledger-drift"></a>

Compare `lextures_coupon_redeemed_total` increase to marketplace purchases with coupon metadata
over 1h. Run the entitlement reconciliation query under "Reconcile ledger vs Stripe". Prefer
fixing via `RedeemCoupon` / webhook replay, not hand-updating counters.

### Alert: reservation expiry

<a id="alert-reservation-expiry"></a>

High `coupon_reservation_expired_total` vs reserves → checkout abandonment or webhook delay.
Check payment webhook worker lag and Stripe Dashboard for stuck sessions.

## Learner says the discount did not apply

1. Confirm the code and course:

```sql
SELECT id, code, status, starts_at, ends_at, max_redemptions, redeemed_count
FROM billing.course_coupons
WHERE course_id = :course_id AND code = upper(:code) AND status <> 'archived';
```

2. Check the learner's ledger rows:

```sql
SELECT id, status, checkout_session_id, provider_event_id,
       list_price_cents, discount_cents, charged_cents,
       reserved_at, expires_at, redeemed_at, released_at
FROM billing.coupon_redemptions
WHERE user_id = :user_id AND course_id = :course_id
ORDER BY reserved_at DESC;
```

3. Stripe session metadata (Dashboard → Checkout Session): expect `coupon_id`,
   `coupon_code`, `coupon_discount_cents`, `list_price_cents`, and
   `allow_promotion_codes = false`. `unit_amount` must equal `charged_cents`.

4. Webhook delivery: payment webhook worker must have processed
   `checkout.session.completed` for that session. On success the redemption is
   `redeemed` and linked to `entitlement_id`. If the reservation was swept before
   the webhook, MKTC.3 falls back to inserting a `redeemed` row from metadata.

Common reasons (typed tokens from preview): `expired`, `exhausted`, `already_used`,
`inactive`, `not_started`, `owned`, `currency_mismatch`, `not_found`.

## Reconcile ledger vs Stripe for a discounted sale

```sql
-- Every redeemed row must point at a live entitlement for the same (user, course)
SELECT r.id, r.user_id, r.course_id, r.entitlement_id, r.charged_cents, r.provider_event_id
FROM billing.coupon_redemptions r
LEFT JOIN billing.user_entitlements e ON e.id = r.entitlement_id
WHERE r.status = 'redeemed'
  AND (e.id IS NULL OR e.user_id <> r.user_id OR e.course_id <> r.course_id);
```

Empty result is healthy. For a single sale, compare `r.charged_cents` to Stripe
`amount_subtotal` (pre-tax) and `e.amount_paid_cents` (often total including tax).

Revenue share is computed on the **charged** amount (`session.AmountTotal`), not list price.

## Release a stuck reservation (learner abandoned checkout)

Prefer:

```text
billing.ReleaseCouponReservation(ctx, pool, redemptionID, "ops_manual")
-- or by session:
billing.ReleaseCouponReservationBySession(ctx, pool, sessionID, "ops_manual")
```

SQL fallback for `reserved` only (does not adjust `redeemed_count`):

```sql
UPDATE billing.coupon_redemptions
SET status = 'released', released_at = NOW(), expires_at = NULL
WHERE id = :redemption_id AND status = 'reserved';
```

TTL sweeper `scheduled.coupon_reservation_sweep` (every 5 minutes) releases rows past
`expires_at`. Correctness never depends on it — seat counts already exclude expired
reservations.

## Creator says the Copy share link button does nothing

Symptoms: click produces no toast, clipboard stays empty, or icon never flips to a check.

1. **Insecure origin** — `navigator.clipboard.writeText` requires a secure context
   (HTTPS or `localhost`). HTTP on a LAN IP fails. Use the app on HTTPS or
   `http://localhost:5173`.
2. **Permission denied** — Some browsers / enterprise policies block clipboard.
   The UI must open a fallback dialog with a selectable URL. If it does not, hard-refresh
   and re-test; if still silent, file a web bug (FR-6).
3. **Confirm the URL shape** — Server `shareUrl` is
   `{PublicWebOrigin}/marketplace/{slug}?coupon={CODE}` (slug falls back to course code).
   `publicShareUrl` is only set when the course is `is_public`.
4. **Flag / permission** — Panel is absent when `ffCourseCoupons` is off or the user
   lacks `course:{code}:item:create`. No copy control means no list loaded.

Workaround for the creator: open the row actions → not needed for the URL; ask them to
use the fallback dialog’s input (select all + Cmd/Ctrl+C), or copy `shareUrl` from
`GET /api/v1/courses/{code}/coupons` via browser network tab.
